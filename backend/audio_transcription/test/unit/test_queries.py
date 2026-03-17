from unittest.mock import MagicMock
from src.db.queries import save_conversation
import pytest
import psycopg

@pytest.fixture
def mock_conn() -> MagicMock:
    return MagicMock(spec=psycopg.Connection)

@pytest.mark.parametrize(
    argnames="profile_id,convo_text,visitor_id", 
    argvalues=[
        (-1, "hi", 2), (None, "hi", 2), # profile_id invalid cases
        (1, None, 2), (1, "", 2),       # convo_text invalid cases
        (1, "hi", -5), (1, "hi", None)
    ]
)
def test_save_conversation_invalid_inputs(
    mock_conn: MagicMock, profile_id: int, convo_text: str, visitor_id: int
) -> None:
    """Tests that it handles invalid input params properly"""
    with pytest.raises(ValueError):
        save_conversation(mock_conn, profile_id, convo_text, visitor_id)

