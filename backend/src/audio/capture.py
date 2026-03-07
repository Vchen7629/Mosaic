from ..core.settings import _LOGS_DIR
from collections import deque
from datetime import datetime
from pathlib import Path
from typing import Any, Optional
from src.audio.audio_input_device import get_input_device
from src.audio.transcription import LogWriter, Transcriber
from src.core.settings import settings
import logging
import queue
import time
import threading
import numpy as np
import sounddevice as sd

logger = logging.getLogger(__name__)


def _build_chunk(blocks: deque[np.ndarray], frames_needed: int) -> Optional[np.ndarray]:
    """Pull exactly `frames_needed` samples from `blocks` into one array."""
    parts, remaining = [], frames_needed
    while remaining > 0 and blocks:
        block = blocks.popleft()
        if len(block) <= remaining:
            parts.append(block)
            remaining -= len(block)
        else:
            parts.append(block[:remaining])
            blocks.appendleft(block[remaining:])
            remaining = 0
    return np.concatenate(parts) if parts else None


class AudioRecorder:
    def __init__(self) -> None:
        self._queue: queue.Queue = queue.Queue()
        self._active = threading.Event()
        self._thread: Optional[threading.Thread] = None
        self._writer: Optional[LogWriter] = None
        self._transcriber: Optional[Transcriber] = None

    @property
    def is_recording(self) -> bool:
        return self._active.is_set()

    def start(self) -> str:
        if self._active.is_set():
            logger.debug("Already recording")
            return ""

        _LOGS_DIR.mkdir(parents=True, exist_ok=True)
        log_path = _LOGS_DIR / f"recording.txt"

        self._writer = LogWriter(str(log_path))
        self._transcriber = Transcriber()
        self._active.set()
        self._thread = threading.Thread(target=self._loop, daemon=True)
        self._thread.start()

        logger.info(f"Recording started → {log_path}")
        return str(log_path)

    def stop(self) -> None:
        if not self._active.is_set():
            logger.debug("Already stopped")
            return

        self._active.clear()
        if self._thread:
            self._thread.join(timeout=5.0)
            if self._thread.is_alive():
                logger.warning("Capture thread did not exit cleanly within timeout")
            self._thread = None

        if self._writer:
            self._writer.close()
            self._writer = None

        logger.info("Recording stopped")

    def _drain_queue(self) -> None:
        while not self._queue.empty():
            try:
                self._queue.get_nowait()
            except queue.Empty:
                break

    def _callback(self, indata: np.ndarray, frames: int, time: Any, status: str) -> None:
        if status:
            logger.debug(status)
            if "overflow" in str(status).lower():
                self._drain_queue()
                return
        self._queue.put(indata.flatten().copy())

    def _loop(self) -> None:
        try:
            device_index = get_input_device()
        except RuntimeError as e:
            logger.error(f"Cannot start capture: {e}")
            return

        frames_needed = int(settings.chunk_duration_s * settings.sample_rate)
        blocks: deque[np.ndarray] = deque()
        pending = 0

        try:
            with sd.InputStream(
                device=device_index,
                channels=1,
                samplerate=settings.sample_rate,
                blocksize=settings.blocksize,
                callback=self._callback,
            ):
                logger.info(f"Capture started ({settings.chunk_duration_s}s chunks)")
                while self._active.is_set():
                    drained = False
                    while not self._queue.empty():
                        block = self._queue.get_nowait()
                        blocks.append(block)
                        pending += len(block)
                        drained = True

                    while pending >= frames_needed:
                        chunk = _build_chunk(blocks, frames_needed)
                        if chunk is None:
                            break
                        pending -= frames_needed
                        threading.Thread(target=self._process, args=(chunk,), daemon=True).start()

                    if not drained:
                        time.sleep(0.01)
        except Exception as e:
            logger.error(f"Capture error: {e}")
        finally:
            logger.info("Capture loop exited")

    def _process(self, chunk: np.ndarray) -> None:
        if not self._active.is_set() or not self._transcriber or not self._writer:
            return
        text = self._transcriber.transcribe(chunk)
        if text:
            logger.debug(f"Transcribed: {text}")
            self._writer.write(text)