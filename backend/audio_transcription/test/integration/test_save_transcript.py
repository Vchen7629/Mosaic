from unittest.mock import patch
from unittest.mock import MagicMock
from psycopg_pool import ConnectionPool
from concurrent.futures import ThreadPoolExecutor
from src.gen import audio_transcription_pb2
from src.gen import audio_transcription_pb2_grpc
from src.grpc.servicer import AudioTranscriptionServicer
from src.transcription.handler import TranscriptionHandler
from test.fixtures.transcription import make_chunk
import grpc
import pytest
import psycopg


SESSION_TOKEN = "save-transcript-session"


def _transcribe_request(
    session_token: str = SESSION_TOKEN,
) -> audio_transcription_pb2.TranscribeAudioRequest:
    return audio_transcription_pb2.TranscribeAudioRequest(
        audio_bytes=make_chunk().tolist(),
        session_token=session_token,
    )


def _save_request(
    session_token: str = SESSION_TOKEN, visitor_ids=None
) -> audio_transcription_pb2.SaveTranscriptRequest:
    return audio_transcription_pb2.SaveTranscriptRequest(
        session_token=session_token,
        visitor_ids=visitor_ids or [],
    )


def _start_server_with_db(
    mock_model, db_pool: ConnectionPool, mock_cache: MagicMock, tmp_path
):
    with (
        patch("src.transcription.handler.settings") as mock_settings,
        patch("src.transcription.handler.get_model", return_value=mock_model),
        patch("src.transcription.handler.WhisperDuration"),
        patch("src.transcription.handler.TranscribeQueueDepth"),
        patch("src.transcription.handler.ErrorsTotal"),
        patch("src.grpc.servicer._LOGS_DIR", tmp_path),
    ):
        mock_settings.transcribe_queue_max_size = 100
        handler = TranscriptionHandler()
        servicer = AudioTranscriptionServicer(db_pool, handler, mock_cache)

        server = grpc.server(ThreadPoolExecutor(max_workers=10))
        audio_transcription_pb2_grpc.add_AudioTranscriptionServiceServicer_to_server(
            servicer, server
        )
        port = server.add_insecure_port("[::]:0")
        server.start()

        yield f"localhost:{port}"

        server.stop(grace=None)


@pytest.fixture
def real_db_pool(postgres_container) -> ConnectionPool:
    connection_url = postgres_container.get_connection_url().replace("+psycopg2", "")
    with psycopg.connect(connection_url) as conn:
        with conn.cursor() as cursor:
            cursor.execute(
                "TRUNCATE TABLE conversation_records, sessions, visitor_face_embeddings, profiles RESTART IDENTITY CASCADE;"
            )
        conn.commit()

    pool = ConnectionPool(conninfo=connection_url, min_size=1, max_size=5, open=True)
    yield pool
    pool.close()


@pytest.fixture
def mock_cache() -> MagicMock:
    return MagicMock()


@pytest.fixture
def save_transcript_server(mock_model, real_db_pool, mock_cache, tmp_path):
    yield from _start_server_with_db(mock_model, real_db_pool, mock_cache, tmp_path)


@pytest.fixture
def save_transcript_client(save_transcript_server):
    channel = grpc.insecure_channel(save_transcript_server)
    yield audio_transcription_pb2_grpc.AudioTranscriptionServiceStub(channel)
    channel.close()


def test_full_pipeline_cache_hit_saves_conversation_to_db(
    save_transcript_client,
    mock_cache,
    real_db_pool,
    seed_profile_table,
    seed_visitor_face_embeddings_table,
    seed_session_table,
):
    with real_db_pool.connection() as conn:
        profile_id = seed_profile_table(conn)
        visitor_id = seed_visitor_face_embeddings_table(conn, profile_id, "alice")
        seed_session_table(conn, profile_id, SESSION_TOKEN)

    mock_cache.get.return_value = str(profile_id).encode()

    save_transcript_client.TranscribeAudio(_transcribe_request())
    response = save_transcript_client.SaveTranscript(
        _save_request(visitor_ids=[visitor_id])
    )

    assert response.success is True

    with real_db_pool.connection() as conn:
        with conn.cursor() as cursor:
            cursor.execute(
                "SELECT convo_text FROM conversation_records WHERE profile_id = %s AND visitor_id = %s",
                (profile_id, visitor_id),
            )
            result = cursor.fetchone()

    assert result is not None
    assert isinstance(result[0], str)


def test_full_pipeline_cache_miss_falls_back_to_db_session_lookup(
    save_transcript_client,
    mock_cache,
    real_db_pool,
    seed_profile_table,
    seed_visitor_face_embeddings_table,
    seed_session_table,
):
    with real_db_pool.connection() as conn:
        profile_id = seed_profile_table(conn)
        visitor_id = seed_visitor_face_embeddings_table(conn, profile_id, "bob")
        seed_session_table(conn, profile_id, SESSION_TOKEN)

    # Force cache miss
    mock_cache.get.return_value = None

    save_transcript_client.TranscribeAudio(_transcribe_request())
    response = save_transcript_client.SaveTranscript(
        _save_request(visitor_ids=[visitor_id])
    )

    assert response.success is True

    with real_db_pool.connection() as conn:
        with conn.cursor() as cursor:
            cursor.execute(
                "SELECT COUNT(*) FROM conversation_records WHERE profile_id = %s",
                (profile_id,),
            )
            count = cursor.fetchone()[0]

    assert count == 1


def test_save_transcript_without_prior_transcribe_returns_false(
    save_transcript_client,
):
    # No TranscribeAudio call for this session
    response = save_transcript_client.SaveTranscript(
        _save_request(session_token="no-transcribe-session")
    )

    assert response.success is False


def test_save_transcript_returns_false_when_session_not_in_cache_or_db(
    save_transcript_client,
    mock_cache,
):
    # Session token has no matching record in cache or DB
    mock_cache.get.return_value = None

    save_transcript_client.TranscribeAudio(_transcribe_request())
    response = save_transcript_client.SaveTranscript(_save_request())

    assert response.success is False


def test_full_pipeline_saves_convo_for_multiple_visitors(
    save_transcript_client,
    mock_cache,
    real_db_pool,
    seed_profile_table,
    seed_visitor_face_embeddings_table,
    seed_session_table,
):
    with real_db_pool.connection() as conn:
        profile_id = seed_profile_table(conn)
        visitor_id_1 = seed_visitor_face_embeddings_table(conn, profile_id, "carol")
        visitor_id_2 = seed_visitor_face_embeddings_table(conn, profile_id, "dave")
        seed_session_table(conn, profile_id, SESSION_TOKEN)

    mock_cache.get.return_value = str(profile_id).encode()

    save_transcript_client.TranscribeAudio(_transcribe_request())
    response = save_transcript_client.SaveTranscript(
        _save_request(visitor_ids=[visitor_id_1, visitor_id_2])
    )

    assert response.success is True

    with real_db_pool.connection() as conn:
        with conn.cursor() as cursor:
            cursor.execute(
                "SELECT COUNT(*) FROM conversation_records WHERE profile_id = %s",
                (profile_id,),
            )
            count = cursor.fetchone()[0]

    assert count == 2


def test_second_save_transcript_for_same_session_returns_false(
    save_transcript_client,
    mock_cache,
    real_db_pool,
    seed_profile_table,
    seed_visitor_face_embeddings_table,
    seed_session_table,
):
    """After a successful save, the logger is deleted so a second save should fail."""
    with real_db_pool.connection() as conn:
        profile_id = seed_profile_table(conn)
        visitor_id = seed_visitor_face_embeddings_table(conn, profile_id, "eve")
        seed_session_table(conn, profile_id, SESSION_TOKEN)

    mock_cache.get.return_value = str(profile_id).encode()

    save_transcript_client.TranscribeAudio(_transcribe_request())
    first_response = save_transcript_client.SaveTranscript(
        _save_request(visitor_ids=[visitor_id])
    )
    second_response = save_transcript_client.SaveTranscript(
        _save_request(visitor_ids=[visitor_id])
    )

    assert first_response.success is True
    assert second_response.success is False
