from functools import lru_cache
from ..core.logging import logger
from ..core.settings import settings
import torch
import whisper


@lru_cache(maxsize=1)
def get_model():
    device = "cuda" if torch.cuda.is_available() else "cpu"
    logger.info(
        "Loading Whisper audio model", model_size=settings.whisper_mode, device=device
    )
    return whisper.load_model(settings.whisper_mode, device=device)
