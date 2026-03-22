from unittest.mock import MagicMock, patch
from src.transcription.handler import TranscriptionHandler
import numpy as np
import pytest


def make_chunk(size: int = 40000) -> np.ndarray:
    return np.random.default_rng(0).uniform(-0.5, 0.5, size).astype(np.float32)


def make_segment(text: str) -> MagicMock:
    seg = MagicMock()
    seg.text = text
    return seg


def _handler_with_queue_size(mock_model: MagicMock, queue_size: int):
    with (
        patch("src.transcription.handler.settings") as mock_settings,
        patch("src.transcription.handler.get_model", return_value=mock_model),
        patch("src.transcription.handler.WhisperDuration"),
        patch("src.transcription.handler.TranscribeQueueDepth"),
        patch("src.transcription.handler.ErrorsTotal"),
    ):
        mock_settings.transcribe_queue_max_size = queue_size
        yield TranscriptionHandler()


@pytest.fixture
def mock_model() -> MagicMock:
    model = MagicMock()
    model.transcribe.side_effect = lambda *a, **kw: (
        iter([make_segment("hello")]),
        MagicMock(),
    )
    return model


@pytest.fixture
def handler(mock_model: MagicMock):
    yield from _handler_with_queue_size(mock_model, queue_size=100)


@pytest.fixture
def small_queue_handler(mock_model: MagicMock):
    yield from _handler_with_queue_size(mock_model, queue_size=1)
