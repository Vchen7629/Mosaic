from ..core.logging import logger
from ..core.settings import settings
import torch
import whisper
import threading

class WhisperModel:
    def __init__(self):
        self._model = None
        self._lock = threading.Lock()

    def get(self):
        if self._model is not None:
            return self._model
        with self._lock:
            if self._model is None:
                device = "cuda" if torch.cuda.is_available() else "cpu"
                logger.info("Loading Whisper audio model", model_size=settings.whisper_mode, device=device)
                self._model = whisper.load_model(settings.whisper_mode, device=device)
        return self._model

_whisper = WhisperModel()

def get_model():
    return _whisper.get()