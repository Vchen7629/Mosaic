from unittest.mock import MagicMock, patch
from src.db.session_queries import fetch_profile_id_from_session
from src.cache.session import fetch_profile_id_from_cache
import psycopg
import pytest


def test_fetch_profile_id_from_session_returns_profile_id() -> None:
    mock_conn = MagicMock(spec=psycopg.Connection)
    mock_cursor = mock_conn.cursor.return_value.__enter__.return_value
    mock_cursor.fetchone.return_value = (42,)

    result = fetch_profile_id_from_session(mock_conn, "some-token")

    assert result == 42


def test_fetch_profile_id_from_session_returns_none_when_not_found() -> None:
    mock_conn = MagicMock(spec=psycopg.Connection)
    mock_cursor = mock_conn.cursor.return_value.__enter__.return_value
    mock_cursor.fetchone.return_value = None

    result = fetch_profile_id_from_session(mock_conn, "nonexistent-token")

    assert result is None


def test_fetch_profile_id_from_session_increments_error_and_raises_on_db_error() -> (
    None
):
    mock_conn = MagicMock(spec=psycopg.Connection)
    mock_cursor = mock_conn.cursor.return_value.__enter__.return_value
    mock_cursor.execute.side_effect = Exception("db error")

    with patch("src.db.session_queries.ErrorsTotal") as mock_errors:
        mock_label = MagicMock()
        mock_errors.labels.return_value = mock_label

        with pytest.raises(Exception, match="db error"):
            fetch_profile_id_from_session(mock_conn, "some-token")

        mock_errors.labels.assert_called_once_with("fetch_profile_id_from_session")
        mock_label.inc.assert_called_once()
        mock_conn.rollback.assert_called_once()


def test_fetch_profile_id_from_cache_returns_profile_id() -> None:
    mock_cache = MagicMock()
    mock_cache.get.return_value = b"7"

    result = fetch_profile_id_from_cache(mock_cache, "some-token")

    assert result == 7
    mock_cache.get.assert_called_once_with("session:some-token")


def test_fetch_profile_id_from_cache_returns_none_on_cache_miss() -> None:
    mock_cache = MagicMock()
    mock_cache.get.return_value = None

    result = fetch_profile_id_from_cache(mock_cache, "missing-token")

    assert result is None


def test_fetch_profile_id_from_cache_returns_none_and_increments_error_on_bad_value() -> (
    None
):
    mock_cache = MagicMock()
    mock_cache.get.return_value = b"not-an-int"

    with patch("src.cache.session.ErrorsTotal") as mock_errors:
        mock_label = MagicMock()
        mock_errors.labels.return_value = mock_label

        result = fetch_profile_id_from_cache(mock_cache, "bad-token")

        assert result is None
        mock_errors.labels.assert_called_once_with("fetch_profile_id_from_cache")
        mock_label.inc.assert_called_once()
