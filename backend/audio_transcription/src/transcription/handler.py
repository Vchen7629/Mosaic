from .model import get_model
from ..core.logging import logger
from ..core.metrics import ErrorsTotal
from ..core.metrics import WhisperDuration
from ..core.metrics import TranscribeQueueDepth
from ..core.settings import settings
from typing import Optional
from concurrent.futures import Future
import queue
import threading
import time
import numpy as np
import torch

class QueueFullError(Exception):
    pass

class TranscriptionHandler:
    def __init__(self) -> None:
        """Starts the background GPU worker thread and initializes the bounded work queue."""
        self._work_queue: queue.Queue[tuple[np.ndarray, Future]] = queue.Queue(maxsize=settings.transcribe_queue_max_size)
        self._worker_thread = threading.Thread(target=self._gpu_worker, daemon=True)
        self._worker_thread.start()

    def transcribe(self, chunk: np.ndarray) -> Optional[str]:
        """
        Normalizes the audio chunk, enqueues it for GPU inference, and blocks
        until the result is ready. Each caller gets their own Future so results
        are never mixed between concurrent callers.

        Args:
            chunk: raw float32 audio samples from a single client

        Raises:
            QueueFullError: if the queue is at capacity and the chunk is rejected

        Returns:
            transcribed text, or None if the audio was silent/unintelligible
        """
        chunk = self._normalize(chunk)
        fut: Future = Future()
        try:
            TranscribeQueueDepth.inc()
            self._work_queue.put_nowait((chunk, fut))
        except queue.Full:
            TranscribeQueueDepth.dec()
            ErrorsTotal.labels(operation="whisper_queue_full").inc()
            logger.warning("Transcription queue full, rejecting chunk")
            raise QueueFullError("Transcription queue is full")

        return fut.result()

    def _gpu_worker(self) -> None:
        """Background thread that serially drains the work queue and runs GPU inference.
        Serial execution ensures ctranslate2/CUDA is never called concurrently."""
        while True:
            chunk, fut = self._work_queue.get()
            try:
                text = self._run_inference(chunk)
                fut.set_result(text)
            except Exception as e:
                ErrorsTotal.labels(operation="whisper_transcribe").inc()
                logger.error("Transcription error", err=str(e))
                fut.set_result(None)
            finally:
                TranscribeQueueDepth.dec()
    
    def _run_inference(self, chunk: np.ndarray) -> Optional[str]:
        """Runs Whisper inference on the chunk. Called only from _gpu_worker."""
        with torch.no_grad():
            whisper_start = time.monotonic()
            segments, _ = get_model().transcribe(chunk, language="en", vad_filter=False)
            text = " ".join(seg.text for seg in segments).strip()
            WhisperDuration.observe((time.monotonic() - whisper_start) * 1000)
        
        return text or None

    def _normalize(self, chunk: np.ndarray) -> np.ndarray:
        """Clamps non-finite values and scales the chunk to [-1.0, 1.0]."""
        chunk = chunk.astype(np.float32)
        if not np.isfinite(chunk).all():
            logger.debug("Non-finite values in audio chunk. Clamping...")
            chunk = np.nan_to_num(chunk, nan=0.0, posinf=1.0, neginf=-1.0)

        max_val = np.abs(chunk).max()
        if max_val > 1.0:
            chunk /= max_val

        return chunk