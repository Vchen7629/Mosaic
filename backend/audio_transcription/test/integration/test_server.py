from unittest.mock import MagicMock, patch
from urllib.request import urlopen
from urllib.error import HTTPError
from src.server import start_metrics_server
import socket
import time


def _free_port() -> int:
    with socket.socket() as s:
        s.bind(("", 0))
        return s.getsockname()[1]


def _wait_for_port(port: int, timeout: float = 2.0) -> None:
    deadline = time.monotonic() + timeout
    while time.monotonic() < deadline:
        try:
            with socket.create_connection(("127.0.0.1", port), timeout=0.1):
                return
        except OSError:
            time.sleep(0.05)
    raise TimeoutError(f"port {port} not ready")


def test_ready_returns_200_when_db_healthy():
    mock_pool = MagicMock()
    mock_pool.connection.return_value.__enter__.return_value = MagicMock()
    port = _free_port()

    with patch("src.server._collect_process_metrics"):
        start_metrics_server(port, mock_pool)

    _wait_for_port(port)
    response = urlopen(f"http://127.0.0.1:{port}/ready")
    assert response.status == 200
    assert response.read() == b"ok"


def test_ready_returns_503_when_db_unhealthy():
    mock_pool = MagicMock()
    mock_pool.connection.side_effect = Exception("db down")
    port = _free_port()

    with patch("src.server._collect_process_metrics"):
        start_metrics_server(port, mock_pool)

    _wait_for_port(port)
    try:
        urlopen(f"http://127.0.0.1:{port}/ready")
        assert False, "expected HTTPError"
    except HTTPError as e:
        assert e.code == 503
        assert e.read() == b"db not ready"


def test_metrics_endpoint_returns_prometheus_output():
    mock_pool = MagicMock()
    port = _free_port()

    with patch("src.server._collect_process_metrics"):
        start_metrics_server(port, mock_pool)

    _wait_for_port(port)
    response = urlopen(f"http://127.0.0.1:{port}/metrics")
    assert response.status == 200
    assert b"# HELP" in response.read()
