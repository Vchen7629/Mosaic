from typing import Generator
from pgvector.psycopg import register_vector
from testcontainers.postgres import PostgresContainer
import pytest
import psycopg
import numpy as np


@pytest.fixture(scope="session")
def postgres_container() -> Generator[PostgresContainer, None, None]:
    """
    Start a PostgreSQL container for integration tests.
    This fixture is session-scoped so one container is shared across all tests.
    """
    with PostgresContainer("pgvector/pgvector:pg17") as postgres:
        # Set up the schema
        # Convert SQLAlchemy URL to PostgreSQL URI for psycopg
        connection_url = postgres.get_connection_url().replace("+psycopg2", "")
        with psycopg.connect(connection_url) as conn:
            with conn.cursor() as cursor:
                cursor.execute("""
                    CREATE EXTENSION IF NOT EXISTS vector;

                    CREATE TABLE profiles ( id SERIAL PRIMARY KEY );

                    CREATE TABLE IF NOT EXISTS visitor_face_embeddings (
                        id SERIAL PRIMARY KEY,
                        profile_id INTEGER REFERENCES profiles(id),
                        visitor_name VARCHAR NOT NULL,
                        face_embedding vector(128) NOT NULL
                    );

                    CREATE TABLE IF NOT EXISTS sessions (
                        profile_id    INT PRIMARY KEY REFERENCES profiles(id),
                        session_token TEXT NOT NULL UNIQUE
                    );

                    CREATE TABLE IF NOT EXISTS conversation_records (
                        id SERIAL PRIMARY KEY,
                        profile_id INTEGER REFERENCES profiles(id),
                        visitor_id INTEGER REFERENCES visitor_face_embeddings(id),
                        created_at TIMESTAMP DEFAULT NOW(),
                        convo_text VARCHAR NOT NULL
                    );
                """)
            conn.commit()

        yield postgres


@pytest.fixture
def db_connection(
    postgres_container: PostgresContainer,
) -> Generator[psycopg.Connection, None, None]:
    """
    Create a fresh database connection for each test.
    Cleans up tables before each test to ensure isolation.
    """
    connection_url = postgres_container.get_connection_url().replace("+psycopg2", "")
    conn = psycopg.connect(connection_url)

    # Cleanup BEFORE test to ensure clean state
    with conn.cursor() as cursor:
        cursor.execute(
            "TRUNCATE TABLE profiles, visitor_face_embeddings, sessions, conversation_records RESTART IDENTITY CASCADE;"
        )
    conn.commit()

    yield conn

    if not conn.closed:
        if conn.info.transaction_status != psycopg.pq.TransactionStatus.IDLE:
            conn.rollback()
        conn.close()


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
