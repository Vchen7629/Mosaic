from unittest.mock import patch, MagicMock
from src.transcription.model import WhisperModel, get_model
import threading
import pytest


@pytest.fixture
def whisper():
    return WhisperModel()


@pytest.mark.parametrize("cuda_available,expected_device", [
    (False, "cpu"),
    (True, "cuda"),
])
def test_get_loads_model_with_correct_device(whisper, cuda_available, expected_device):
    mock_model = MagicMock()

    with (
        patch("src.transcription.model.torch.cuda.is_available", return_value=cuda_available),
        patch("src.transcription.model.settings") as mock_settings,
        patch("src.transcription.model.FasterWhisperModel", return_value=mock_model) as mock_cls,
        patch("src.transcription.model.logger"),
    ):
        mock_settings.whisper_model = "medium"
        mock_settings.whisper_compute_type = "float16"
        result = whisper.get()

    mock_cls.assert_called_once_with("medium", device=expected_device, compute_type="float16")
    assert result is mock_model


def test_get_returns_cached_model_and_logs_once(whisper):
    mock_model = MagicMock()

    with (
        patch("src.transcription.model.torch.cuda.is_available", return_value=False),
        patch("src.transcription.model.settings") as mock_settings,
        patch("src.transcription.model.FasterWhisperModel", return_value=mock_model) as mock_cls,
        patch("src.transcription.model.logger") as mock_logger,
    ):
        mock_settings.whisper_model = "medium"
        first = whisper.get()
        second = whisper.get()

    assert first is second
    mock_cls.assert_called_once()
    mock_logger.info.assert_called_once()


def test_get_is_safe_for_concurrent_access(whisper):
    load_count = 0
    results = [None] * 10

    def slow_load(*args, **kwargs):
        nonlocal load_count
        load_count += 1
        return MagicMock()

    with (
        patch("src.transcription.model.torch.cuda.is_available", return_value=False),
        patch("src.transcription.model.settings"),
        patch("src.transcription.model.FasterWhisperModel", side_effect=slow_load),
        patch("src.transcription.model.logger"),
    ):
        threads = [
            threading.Thread(target=lambda i=i: results.__setitem__(i, whisper.get()))
            for i in range(10)
        ]
        for t in threads:
            t.start()
        for t in threads:
            t.join(timeout=5)

    assert load_count == 1
    assert all(r is results[0] for r in results)


def test_get_model_delegates_to_whisper_singleton():
    mock_model = MagicMock()

    with patch("src.transcription.model._whisper") as mock_whisper:
        mock_whisper.get.return_value = mock_model
        result = get_model()

    assert result is mock_model
    mock_whisper.get.assert_called_once()
