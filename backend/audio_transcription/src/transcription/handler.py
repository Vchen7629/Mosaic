from .model import get_model
from typing import Optional
import logging
import numpy as np

logger = logging.getLogger(__name__)

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
        if "CUDA" in str(e):
            logger.warning("CUDA error detected, clearing model cache for reload on next call")
            get_model.cache_clear()
        return None
