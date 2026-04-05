from unittest.mock import patch
from unittest.mock import MagicMock
from src.db.connection_pool import create_connection_pool
import pytest


def _mock_settings(**overrides):
    s = MagicMock()
    s.DB_HOST = "testhost"
    s.DB_PORT = 5432
    s.DB_NAME = "testdb"
    s.DB_USER = "testuser"
    s.DB_PASS = "testpass"
    s.POOL_MIN_SIZE = 1
    s.POOL_MAX_SIZE = 5
    s.KEEP_ALIVES_COUNT = 1
    s.KEEP_ALIVES_IDLE_S = 30
    s.KEEP_ALIVES_RETRY_INTERVAL_S = 10
    s.KEEP_ALIVES_RETRY_COUNT = 5
    for k, v in overrides.items():
        setattr(s, k, v)
    return s


@pytest.fixture
def mock_pool():
    pool = MagicMock()
    mock_conn = MagicMock()
    mock_cursor = MagicMock()
    pool.connection.return_value.__enter__.return_value = mock_conn
    mock_conn.cursor.return_value.__enter__.return_value = mock_cursor
    return pool


def test_creates_pool_with_conninfo_from_settings(mock_pool):
    mock_settings = _mock_settings()

    with (
        patch("src.db.connection_pool.settings", mock_settings),
        patch(
            "src.db.connection_pool.ConnectionPool", return_value=mock_pool
        ) as mock_cls,
    ):
        create_connection_pool()

    expected_conninfo = (
        "host=testhost port=5432 dbname=testdb user=testuser password=testpass "
    )
    mock_cls.assert_called_once_with(
        conninfo=expected_conninfo,
        min_size=1,
        max_size=5,
        open=True,
        kwargs={
            "keepalives": 1,
            "keepalives_idle": 30,
            "keepalives_interval": 10,
            "keepalives_count": 5,
        },
    )


def test_returns_pool_on_success(mock_pool):
    with (
        patch("src.db.connection_pool.settings", _mock_settings()),
        patch("src.db.connection_pool.ConnectionPool", return_value=mock_pool),
    ):
        result = create_connection_pool()

    assert result is mock_pool


def test_logs_health_check_success(mock_pool):
    with (
        patch("src.db.connection_pool.settings", _mock_settings()),
        patch("src.db.connection_pool.ConnectionPool", return_value=mock_pool),
        patch("src.db.connection_pool.logger") as mock_logger,
    ):
        create_connection_pool()

    mock_logger.debug.assert_called_once_with("health check for db succeeded")


def test_closes_pool_and_reraises_on_health_check_failure():
    pool = MagicMock()
    pool.connection.side_effect = Exception("connection refused")

    with (
        patch("src.db.connection_pool.settings", _mock_settings()),
        patch("src.db.connection_pool.ConnectionPool", return_value=pool),
        patch("src.db.connection_pool.logger"),
    ):
        with pytest.raises(Exception, match="connection refused"):
            create_connection_pool()

    pool.close.assert_called_once()


def test_logs_error_on_health_check_failure():
    pool = MagicMock()
    pool.connection.side_effect = Exception("timeout")

    with (
        patch("src.db.connection_pool.settings", _mock_settings()),
        patch("src.db.connection_pool.ConnectionPool", return_value=pool),
        patch("src.db.connection_pool.logger") as mock_logger,
    ):
        with pytest.raises(Exception):
            create_connection_pool()

    mock_logger.error.assert_called_once_with(
        "health check for database failed", err="timeout"
    )
