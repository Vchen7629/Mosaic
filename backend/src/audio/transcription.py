from src.audio.model import get_model
from src.core.settings import settings
from typing import Any, Optional
from collections import deque
from datetime import datetime
import hashlib
import logging
import threading
import numpy as np

logger = logging.getLogger(__name__)

class LogWriter:
    def __init__(self, path: str) -> None:
        self._file = open(path, "w", encoding="utf-8")
        self._lock = threading.Lock()

    def write(self, text: str) -> None:
        timestamp = datetime.now().strftime("%H:%M:%S")
        with self._lock:
            try:
                self._file.write(f"[{timestamp}] {text}\n")
                self._file.flush()
            except Exception as e:
                logger.error(f"Failed to write transcript: {e}")

    def close(self) -> None:
        with self._lock:
            try:
                self._file.close()
            except Exception:
                pass

class Transcriber:
    def __init__(self) -> None:
        self._recent_hashes: deque[str] = deque(maxlen=settings.max_hash_history)

    def transcribe(self, chunk: np.ndarray) -> Optional[str]:
        if not self._filter_silence_and_loud(chunk):
            return None

        audio_hash = hashlib.md5(chunk.tobytes()).hexdigest()
        if not self._filter_duplicate_audio_hash(audio_hash):
            return None
        
        self._recent_hashes.append(audio_hash)

        chunk = chunk.astype(np.float32)
        if not np.isfinite(chunk).all():
            logger.debug("Non-finite values in audio chunk. Clamping...")
            chunk = np.nan_to_num(chunk, nan=0.0, posinf=1.0, neginf=-1.0)
        max_val = np.abs(chunk).max()
        if max_val > 1.0:
            chunk /= max_val

        try:
            result = get_model().transcribe(
                chunk,
                language="en",
                task="transcribe",
                beam_size=1,
                best_of=1,
                logprob_threshold=None,
                no_speech_threshold=0.9,
                fp16=False,
            )
            return result.get("text", "").strip() or None
        except Exception as e:
            logger.error(f"Transcription error: {e}")
            return None

    def _filter_silence_and_loud(self, chunk: np.ndarray) -> bool:
        rms = float(np.sqrt(np.mean(chunk ** 2)))
        if rms < settings.silence_threshold:
            logger.debug(f"Silence (rms={rms:.4f}), skipping")
            return False
        if rms > 0.5:
            logger.debug(f"Audio too loud (rms={rms:.4f}), skipping")
            return False

        return True
    
    def _filter_duplicate_audio_hash(self, audio_hash: Any) -> bool:
        if audio_hash in self._recent_hashes:
            logger.debug("Duplicate chunk, skipping")
            return False

        return True