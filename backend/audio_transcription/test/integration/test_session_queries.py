import psycopg
from src.db.session_queries import fetch_profile_id_from_session


def test_fetch_profile_id_returns_correct_id(
    db_connection: psycopg.Connection,
    seed_profile_table,
    seed_session_table,
) -> None:
    """Should return the profile_id for a known session_token"""
    profile_id = seed_profile_table(db_connection)
    seed_session_table(db_connection, profile_id, "test-token-abc")

    result = fetch_profile_id_from_session(db_connection, "test-token-abc")

    assert result == profile_id


def test_fetch_profile_id_returns_none_for_unknown_token(
    db_connection: psycopg.Connection,
) -> None:
    """Should return None when session_token does not exist"""
    result = fetch_profile_id_from_session(db_connection, "nonexistent-token")

    assert result is None


def test_fetch_profile_id_returns_correct_id_among_multiple_sessions(
    db_connection: psycopg.Connection,
    seed_profile_table,
    seed_session_table,
) -> None:
    """Should return the right profile_id when multiple sessions exist"""
    profile_id_1 = seed_profile_table(db_connection)
    profile_id_2 = seed_profile_table(db_connection)
    seed_session_table(db_connection, profile_id_1, "token-1")
    seed_session_table(db_connection, profile_id_2, "token-2")

    assert fetch_profile_id_from_session(db_connection, "token-1") == profile_id_1
    assert fetch_profile_id_from_session(db_connection, "token-2") == profile_id_2
