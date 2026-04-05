from unittest.mock import patch
from urllib.parse import urlparse
from testcontainers.postgres import PostgresContainer
from psycopg_pool import ConnectionPool
from src.db.connection_pool import create_connection_pool
import pytest


def test_create_connection_pool_succeeds_and_returns_working_pool(
    postgres_container: PostgresContainer,
) -> None:
    connection_url = postgres_container.get_connection_url().replace("+psycopg2", "")

    parsed = urlparse(connection_url)

    mock_settings = patch("src.db.connection_pool.settings")
    with mock_settings as s:
        s.DB_HOST = parsed.hostname
        s.DB_PORT = parsed.port
        s.DB_NAME = parsed.path.lstrip("/")
        s.DB_USER = parsed.username
        s.DB_PASS = parsed.password
        s.POOL_MIN_SIZE = 1
        s.POOL_MAX_SIZE = 5
        s.KEEP_ALIVES_COUNT = 1
        s.KEEP_ALIVES_IDLE_S = 30
        s.KEEP_ALIVES_RETRY_INTERVAL_S = 10
        s.KEEP_ALIVES_RETRY_COUNT = 5

        pool = create_connection_pool()

    try:
        with pool.connection() as conn:
            with conn.cursor() as cursor:
                cursor.execute("SELECT 1")
                result = cursor.fetchone()
        assert result == (1,)
    finally:
        pool.close()


def test_create_connection_pool_raises_on_unreachable_db() -> None:
    def _fast_pool(*args, **kwargs):
        kwargs["reconnect_timeout"] = 1
        kwargs["timeout"] = 1
        return ConnectionPool(*args, **kwargs)

    mock_settings = patch("src.db.connection_pool.settings")
    with (
        mock_settings as s,
        patch("src.db.connection_pool.ConnectionPool", side_effect=_fast_pool),
    ):
        s.DB_HOST = "127.0.0.1"
        s.DB_PORT = 1
        s.DB_NAME = "nonexistent"
        s.DB_USER = "nobody"
        s.DB_PASS = "wrong"
        s.POOL_MIN_SIZE = 1
        s.POOL_MAX_SIZE = 5
        s.KEEP_ALIVES_COUNT = 1
        s.KEEP_ALIVES_IDLE_S = 30
        s.KEEP_ALIVES_RETRY_INTERVAL_S = 10
        s.KEEP_ALIVES_RETRY_COUNT = 5

        with pytest.raises(Exception):
            create_connection_pool()
