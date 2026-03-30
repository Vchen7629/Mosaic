from typing import Optional
from faster_whisper import WhisperModel as FasterWhisperModel
from ..core.logging import logger
from ..core.settings import settings
import torch
import threading


class WhisperModel:
    def __init__(self) -> None:
        self._model: Optional[FasterWhisperModel] = None
        self._lock = threading.Lock()

    def get(self) -> FasterWhisperModel:
        if self._model is not None:
            return self._model
        with self._lock:
            if self._model is None:
                device = "cuda" if torch.cuda.is_available() else "cpu"
                logger.info(
                    "Loading Whisper audio model",
                    model_size=settings.whisper_model,
                    device=device,
                )
                self._model = FasterWhisperModel(
                    settings.whisper_model,
                    device=device,
                    compute_type=settings.whisper_compute_type,
                )
        return self._model


_whisper = WhisperModel()


def get_model() -> FasterWhisperModel:
    return _whisper.get()
