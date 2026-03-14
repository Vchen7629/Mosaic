from functools import lru_cache
from ..core.settings import settings
import torch
import whisper
import logging

logger = logging.getLogger(__name__)

@lru_cache(maxsize=1)
def get_model():
    device = "cuda" if torch.cuda.is_available() else "cpu"
    logger.info(f"Loading Whisper '{settings.whisper_mode}' on {device}")
    return whisper.load_model(settings.whisper_mode, device=device)