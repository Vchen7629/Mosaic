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
