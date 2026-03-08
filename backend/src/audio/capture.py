from ..core.settings import _LOGS_DIR
from collections import deque
from typing import Any, Optional
from ..audio.audio_input_device import get_input_device
from .transcription import LogWriter, Transcriber
from ..core.settings import settings
import logging
import queue
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
        self._pending_lock = threading.Lock()
        self._pending_count = 0
        self._all_done = threading.Event()
        self._all_done.set()

    @property
    def is_recording(self) -> bool:
        return self._active.is_set()

    def start(self) -> str:
        if self._active.is_set():
            logger.debug("Already recording")
            return ""

        _LOGS_DIR.mkdir(parents=True, exist_ok=True)
        log_path = _LOGS_DIR / "recording.txt"
        log_path.write_text("")  # clear on start

        self._writer = LogWriter(str(log_path))
        self._transcriber = Transcriber()
        self._pending_count = 0
        self._all_done.set()
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
            self._thread.join(timeout=5.0)  # just waits for the loop to exit, not Whisper
            if self._thread.is_alive():
                logger.warning("Capture loop did not exit within 5s")
            self._thread = None

        logger.info("Recording stopped, transcription finishing in background")

    def wait_for_transcript(self, timeout: float = 300.0) -> str:
        self._all_done.wait(timeout=timeout)
        if self._writer:
            self._writer.close()
            self._writer = None
        try:
            return (_LOGS_DIR / "recording.txt").read_text(encoding="utf-8")
        except Exception:
            return ""

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
            device_info = sd.query_devices(device_index)
            print(f"[audio] Using device [{device_index}]: {device_info['name']}", flush=True)
        except RuntimeError as e:
            print(f"[audio] ERROR: Cannot start capture: {e}", flush=True)
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
                print(f"[audio] InputStream opened, recording...", flush=True)
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
                        self._dispatch_process(chunk)

                    if not drained:
                        threading.Event().wait(0.01)

                # Flush remaining partial audio (non-blocking)
                if blocks and self._transcriber and self._writer:
                    chunk = np.concatenate(list(blocks))
                    print(f"[audio] Flushing {len(chunk)} remaining frames", flush=True)
                    self._dispatch_process(chunk)

        except Exception as e:
            print(f"[audio] ERROR in InputStream: {e}", flush=True)
            logger.error(f"Capture error: {e}")
        finally:
            print(f"[audio] Capture loop exited", flush=True)
            logger.info("Capture loop exited")

    def _dispatch_process(self, chunk: np.ndarray) -> None:
        with self._pending_lock:
            self._pending_count += 1
            self._all_done.clear()
        threading.Thread(target=self._process, args=(chunk,), daemon=True).start()

    def _process(self, chunk: np.ndarray) -> None:
        try:
            if not self._transcriber or not self._writer:
                return
            text = self._transcriber.transcribe(chunk)
            print(f"[audio] Transcription: {repr(text)}", flush=True)
            if text:
                logger.debug(f"Transcribed: {text}")
                self._writer.write(text)
        finally:
            with self._pending_lock:
                self._pending_count -= 1
                if self._pending_count == 0:
                    self._all_done.set()
