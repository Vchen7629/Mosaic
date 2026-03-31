import pytest
from src.db.conversation_queries import save_conversation
import psycopg


def test_insert_one_valid_convo(
    db_connection: psycopg.Connection,
    seed_profile_table: int,
    seed_visitor_face_embeddings_table: int,
) -> None:
    """Should add the convo to db properly"""
    profileID = seed_profile_table(db_connection)
    visitorID = seed_visitor_face_embeddings_table(db_connection, profileID, "visitor")

    save_conversation(db_connection, profileID, "placeholdText", [visitorID])

    with db_connection.cursor() as cursor:
        query = """
            SELECT convo_text FROM conversation_records 
            WHERE profile_id = %(profile_id)s
            AND visitor_id = %(visitor_id)s
        """

        cursor.execute(query, {"profile_id": profileID, "visitor_id": visitorID})

        result = cursor.fetchone()

        assert result[0] == "placeholdText"


def test_insert_multiple_convo_for_same_profile_visitor_no_issues(
    db_connection: psycopg.Connection,
    seed_profile_table: int,
    seed_visitor_face_embeddings_table: int,
) -> None:
    """Should add the convos to db properly"""
    profileID = seed_profile_table(db_connection)
    visitorID = seed_visitor_face_embeddings_table(db_connection, profileID, "visitor")

    for i in range(5):
        save_conversation(db_connection, profileID, f"placeholdText{i}", [visitorID])

    with db_connection.cursor() as cursor:
        query = """
            SELECT convo_text FROM conversation_records 
            WHERE profile_id = %(profile_id)s
            AND visitor_id = %(visitor_id)s
        """

        cursor.execute(query, {"profile_id": profileID, "visitor_id": visitorID})

        result = cursor.fetchall()

        assert result[0][0] == "placeholdText0"
        assert result[1][0] == "placeholdText1"
        assert result[4][0] == "placeholdText4"


def test_insert_nonexistant_patient_or_visitor_id(
    db_connection: psycopg.Connection,
) -> None:
    """non existant profile id or visitor id should raise error"""
    with pytest.raises(Exception):
        save_conversation(db_connection, 23, "placeholdText", 23)
