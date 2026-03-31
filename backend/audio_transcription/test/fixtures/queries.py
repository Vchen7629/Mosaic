from unittest.mock import MagicMock
from pgvector.psycopg import register_vector
import pytest
import psycopg
import numpy as np


@pytest.fixture
def mock_conn() -> MagicMock:
    return MagicMock(spec=psycopg.Connection)


@pytest.fixture
def mock_cache() -> MagicMock:
    return MagicMock()


@pytest.fixture
def seed_profile_table():
    def _seed(db_conn: psycopg.Connection) -> int:
        """Create a new profile and return the id for downstream use"""
        with db_conn.cursor() as cursor:
            cursor.execute("INSERT INTO profiles DEFAULT VALUES RETURNING id")
            result = cursor.fetchone()
            return result[0]

    return _seed


@pytest.fixture
def seed_visitor_face_embeddings_table():
    def _seed(db_conn: psycopg.Connection, profile_id: int, visitor_name: str) -> int:
        """Create a new visitor face record and return the id for downstream use"""
        register_vector(db_conn)

        with db_conn.cursor() as cursor:
            query = """
                INSERT INTO visitor_face_embeddings
                (profile_id, visitor_name, face_embedding)
                VALUES (%(profile_id)s, %(visitor_name)s, %(face_embedding)s)
                RETURNING id
            """

            face_embedding = np.zeros(128, dtype=np.float32)

            cursor.execute(
                query,
                {
                    "profile_id": profile_id,
                    "visitor_name": visitor_name,
                    "face_embedding": face_embedding,
                },
            )
            result = cursor.fetchone()
            return result[0]

    return _seed


@pytest.fixture
def seed_session_table():
    def _seed(db_conn: psycopg.Connection, profile_id: int, session_token: str) -> None:
        """Insert a session row linking profile_id to session_token"""
        with db_conn.cursor() as cursor:
            cursor.execute(
                "INSERT INTO sessions (profile_id, session_token) VALUES (%s, %s)",
                (profile_id, session_token),
            )
        db_conn.commit()

    return _seed
