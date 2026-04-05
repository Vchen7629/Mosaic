from unittest.mock import patch
from unittest.mock import MagicMock
from src.server import serve
from src.server import handle_shutdown
import pytest


def test_handle_shutdown_stops_server_with_grace_period():
    mock_server = MagicMock()
    handle_shutdown(mock_server, 2, None)
    mock_server.stop.assert_called_once_with(grace=5)


def test_serve_closes_db_pool_and_cache_on_shutdown():
    mock_db_pool = MagicMock()
    mock_cache = MagicMock()

    with (
        patch("src.server.grpc.server") as mock_grpc_server_factory,
        patch("src.server.ThreadPoolExecutor"),
        patch("src.server.create_connection_pool", return_value=mock_db_pool),
        patch("src.server.TranscriptionHandler"),
        patch("src.server.Valkey", return_value=mock_cache),
        patch("src.server.start_metrics_server"),
        patch("src.server.signal.signal"),
        patch("src.server.logger"),
        patch("src.server.settings"),
    ):
        mock_grpc_server = MagicMock()
        mock_grpc_server_factory.return_value = mock_grpc_server
        mock_grpc_server.wait_for_termination.return_value = None

        serve()

    mock_db_pool.close.assert_called_once()
    mock_cache.close.assert_called_once()


def test_serve_closes_resources_even_when_exception_raised():
    mock_db_pool = MagicMock()
    mock_cache = MagicMock()

    with (
        patch("src.server.grpc.server") as mock_grpc_server_factory,
        patch("src.server.ThreadPoolExecutor"),
        patch("src.server.create_connection_pool", return_value=mock_db_pool),
        patch("src.server.TranscriptionHandler"),
        patch("src.server.Valkey", return_value=mock_cache),
        patch("src.server.start_metrics_server"),
        patch("src.server.signal.signal"),
        patch("src.server.logger"),
        patch("src.server.settings"),
    ):
        mock_grpc_server = MagicMock()
        mock_grpc_server_factory.return_value = mock_grpc_server
        mock_grpc_server.wait_for_termination.side_effect = RuntimeError("unexpected")

        with pytest.raises(RuntimeError):
            serve()

    mock_db_pool.close.assert_called_once()
    mock_cache.close.assert_called_once()
