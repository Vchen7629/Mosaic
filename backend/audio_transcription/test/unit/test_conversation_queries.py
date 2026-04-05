from unittest.mock import patch
from unittest.mock import MagicMock
from src.db.conversation_queries import save_conversation
import pytest
import psycopg


@pytest.mark.parametrize(
    argnames="profile_id,convo_text,visitor_id",
    argvalues=[
        (-1, "hi", [2, 5]),
        (None, "hi", [2]),  # profile_id invalid cases
        (1, None, [2, 5]),
        (1, "", [2, 5]),  # convo_text invalid cases
        (1, "hi", [-5, 2]),
        (1, "hi", [None, 2]),
        (1, "hi", [None]),
    ],
)
def test_save_conversation_invalid_inputs(
    profile_id: int, convo_text: str, visitor_id: int
) -> None:
    """Tests that it handles invalid input params properly"""
    with pytest.raises(ValueError):
        mock_conn = MagicMock(spec=psycopg.Connection)
        save_conversation(mock_conn, profile_id, convo_text, visitor_id)


def test_save_conversation_rolls_back_and_raises_on_db_error() -> None:
    mock_conn = MagicMock(spec=psycopg.Connection)
    mock_cursor = mock_conn.cursor.return_value.__enter__.return_value
    # non-retryable error so retry_with_backoff fails immediately without sleeping
    mock_cursor.execute.side_effect = psycopg.ProgrammingError("constraint violation")

    with patch("src.db.conversation_queries.DBWritesTotal"):
        with pytest.raises(psycopg.ProgrammingError):
            save_conversation(mock_conn, 1, "hello", [1])

    mock_conn.rollback.assert_called_once()
