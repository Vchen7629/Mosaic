from concurrent.futures import ThreadPoolExecutor
from unittest.mock import MagicMock, patch
from psycopg_pool import ConnectionPool
from src.grpc.servicer import AudioTranscriptionServicer
from src.transcription.handler import TranscriptionHandler
from src.gen import audio_transcription_pb2_grpc
import grpc
import pytest


def _start_server(mock_model, queue_size: int, tmp_path):
    with (
        patch("src.transcription.handler.settings") as mock_settings,
        patch("src.transcription.handler.get_model", return_value=mock_model),
        patch("src.transcription.handler.WhisperDuration"),
        patch("src.transcription.handler.TranscribeQueueDepth"),
        patch("src.transcription.handler.ErrorsTotal"),
        patch("src.grpc.servicer._LOGS_DIR", tmp_path),
    ):
        mock_settings.transcribe_queue_max_size = queue_size

        handler = TranscriptionHandler()
        db_pool = MagicMock(spec=ConnectionPool)
        mock_cache = MagicMock()

        server = grpc.server(ThreadPoolExecutor(max_workers=10))
        audio_transcription_pb2_grpc.add_AudioTranscriptionServiceServicer_to_server(
            AudioTranscriptionServicer(db_pool, handler, mock_cache), server
        )
        port = server.add_insecure_port("[::]:0")
        server.start()

        yield f"localhost:{port}"

        server.stop(grace=None)


@pytest.fixture
def grpc_server(mock_model, tmp_path):
    yield from _start_server(mock_model, queue_size=100, tmp_path=tmp_path)


@pytest.fixture
def small_queue_grpc_server(mock_model, tmp_path):
    yield from _start_server(mock_model, queue_size=1, tmp_path=tmp_path)


@pytest.fixture
def grpc_client(grpc_server):
    channel = grpc.insecure_channel(grpc_server)
    yield audio_transcription_pb2_grpc.AudioTranscriptionServiceStub(channel)
    channel.close()
