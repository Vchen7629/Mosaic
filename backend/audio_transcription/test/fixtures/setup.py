from typing import Generator
from pgvector.psycopg import register_vector
from psycopg_pool import ConnectionPool
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
        cursor.execute("TRUNCATE TABLE profiles, visitor_face_embeddings, conversation_records RESTART IDENTITY CASCADE;")
    conn.commit()

    yield conn

    if not conn.closed:
        if conn.info.transaction_status != psycopg.pq.TransactionStatus.IDLE:
            conn.rollback()
        conn.close()


@pytest.fixture
def db_pool(postgres_container: PostgresContainer) -> Generator[ConnectionPool, None, None]:
    """
    Create a connection pool for testing pool-based operations. clean up tables after each test
    """
    # Convert SQLAlchemy URL to PostgreSQL URI for psycopg
    connection_url = postgres_container.get_connection_url().replace("+psycopg2", "")
    pool = ConnectionPool(conninfo=connection_url, min_size=1, max_size=5, open=True)
    with pool.connection() as conn:
        # Rollback any failed transaction first
        if conn.info.transaction_status != psycopg.pq.TransactionStatus.IDLE:
            conn.rollback()

        with conn.cursor() as cursor:
            cursor.execute(
                "TRUNCATE TABLE product_sentiment, raw_comments RESTART IDENTITY CASCADE;"
            )
        conn.commit()

    yield pool

    pool.close()