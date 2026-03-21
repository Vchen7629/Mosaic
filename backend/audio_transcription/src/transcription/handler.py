from .model import get_model
from ..core.logging import logger
from ..core.metrics import ErrorsTotal
from ..core.metrics import WhisperDuration
from typing import Optional
import time
import numpy as np


def transcribe_handler(chunk: np.ndarray) -> Optional[str]:
    """
    handler that handles infinite values, does clamping so it doesnt
    go past 1, and transcribes the audio to text using whisper

    Args:
        chunk: the audio chunk containing audio bytes

    Returns:
        the transcribed audio in text using whisper or empty string
    """
    chunk = chunk.astype(np.float32)
    if not np.isfinite(chunk).all():
        logger.debug("Non-finite values in audio chunk. Clamping...")
        chunk = np.nan_to_num(chunk, nan=0.0, posinf=1.0, neginf=-1.0)
    max_val = np.abs(chunk).max()
    if max_val > 1.0:
        chunk /= max_val

    try:
        whisper_start = time.monotonic()
        segments, _ = get_model().transcribe(chunk, language="en", vad_filter=False)
        WhisperDuration.observe((time.monotonic() - whisper_start) * 1000)
        text = " ".join(seg.text for seg in segments).strip()
        return text or None
    except Exception as e:
        ErrorsTotal.labels(operation="whisper_transcribe").inc()
        logger.error("Transcription error", err=str(e))
        if "CUDA" in str(e):
            logger.error(
                "CUDA error detected, keeping cached model — GPU memory may be insufficient for reload"
            )
        return None
