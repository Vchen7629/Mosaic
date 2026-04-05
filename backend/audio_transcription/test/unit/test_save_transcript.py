from unittest.mock import patch
from unittest.mock import MagicMock
from src.grpc.servicer import AudioTranscriptionServicer
from src.gen import audio_transcription_pb2


SESSION_TOKEN = "test-session-token"
VISITOR_IDS = [1, 2]
PROFILE_ID = 42
CONVO_TEXT = "Hello, how are you?"


def _make_request(session_token: str = SESSION_TOKEN, visitor_ids=None):
    return audio_transcription_pb2.SaveTranscriptRequest(
        session_token=session_token,
        visitor_ids=visitor_ids if visitor_ids is not None else VISITOR_IDS,
    )


def _inject_logger(
    servicer: AudioTranscriptionServicer, session_token: str = SESSION_TOKEN
) -> MagicMock:
    """Inject a mock ConvoLogHandler into the servicer's _convo_loggers dict."""
    mock_logger = MagicMock()
    mock_logger.read.return_value = CONVO_TEXT
    servicer._convo_loggers[session_token] = mock_logger
    return mock_logger


def test_returns_false_and_logs_when_profile_logger_is_none(servicer):
    # No TranscribeAudio call was made for this session, so no logger exists
    response = servicer.SaveTranscript(_make_request(), MagicMock())

    assert response.success is False
    assert SESSION_TOKEN not in servicer._convo_loggers


def test_uses_cache_profile_id_on_cache_hit(servicer):
    _inject_logger(servicer)

    with (
        patch(
            "src.grpc.servicer.fetch_profile_id_from_cache", return_value=PROFILE_ID
        ) as mock_cache_fetch,
        patch("src.grpc.servicer.fetch_profile_id_from_session") as mock_db_fetch,
        patch("src.grpc.servicer.save_conversation"),
        patch("src.grpc.servicer.ConvoSaveDuration"),
    ):
        response = servicer.SaveTranscript(_make_request(), MagicMock())

    assert response.success is True
    mock_cache_fetch.assert_called_once_with(servicer._cache, SESSION_TOKEN)
    mock_db_fetch.assert_not_called()


def test_falls_back_to_db_and_logs_on_cache_miss(servicer):
    _inject_logger(servicer)

    mock_conn = MagicMock()
    servicer._db_pool.connection.return_value.__enter__.return_value = mock_conn

    with (
        patch(
            "src.grpc.servicer.fetch_profile_id_from_cache", return_value=None
        ) as mock_cache_fetch,
        patch(
            "src.grpc.servicer.fetch_profile_id_from_session", return_value=PROFILE_ID
        ) as mock_db_fetch,
        patch("src.grpc.servicer.save_conversation"),
        patch("src.grpc.servicer.ConvoSaveDuration"),
    ):
        response = servicer.SaveTranscript(_make_request(), MagicMock())

    assert response.success is True
    mock_cache_fetch.assert_called_once_with(servicer._cache, SESSION_TOKEN)
    mock_db_fetch.assert_called_once_with(mock_conn, SESSION_TOKEN)


def test_returns_false_when_both_cache_and_db_miss(servicer):
    _inject_logger(servicer)

    with (
        patch("src.grpc.servicer.fetch_profile_id_from_cache", return_value=None),
        patch("src.grpc.servicer.fetch_profile_id_from_session", return_value=None),
    ):
        response = servicer.SaveTranscript(_make_request(), MagicMock())

    assert response.success is False


def test_catches_exception_increments_error_metric_and_returns_false(servicer):
    _inject_logger(servicer)

    with (
        patch("src.grpc.servicer.fetch_profile_id_from_cache", return_value=PROFILE_ID),
        patch(
            "src.grpc.servicer.save_conversation", side_effect=Exception("db exploded")
        ),
        patch("src.grpc.servicer.ConvoSaveDuration"),
        patch("src.grpc.servicer.ErrorsTotal") as mock_errors,
    ):
        mock_label = MagicMock()
        mock_errors.labels.return_value = mock_label

        response = servicer.SaveTranscript(_make_request(), MagicMock())

    assert response.success is False
    mock_errors.labels.assert_called_once_with(operation="save_conversation")
    mock_label.inc.assert_called_once()


def test_deletes_profile_logger_from_dict_on_success(servicer):
    _inject_logger(servicer)
    assert SESSION_TOKEN in servicer._convo_loggers

    with (
        patch("src.grpc.servicer.fetch_profile_id_from_cache", return_value=PROFILE_ID),
        patch("src.grpc.servicer.save_conversation"),
        patch("src.grpc.servicer.ConvoSaveDuration"),
    ):
        response = servicer.SaveTranscript(_make_request(), MagicMock())

    assert response.success is True
    assert SESSION_TOKEN not in servicer._convo_loggers


def test_calls_logger_delete_on_success(servicer):
    mock_logger = _inject_logger(servicer)

    with (
        patch("src.grpc.servicer.fetch_profile_id_from_cache", return_value=PROFILE_ID),
        patch("src.grpc.servicer.save_conversation"),
        patch("src.grpc.servicer.ConvoSaveDuration"),
    ):
        servicer.SaveTranscript(_make_request(), MagicMock())

    mock_logger.delete.assert_called_once()


def test_logger_not_deleted_on_exception(servicer):
    mock_logger = _inject_logger(servicer)

    with (
        patch("src.grpc.servicer.fetch_profile_id_from_cache", return_value=PROFILE_ID),
        patch("src.grpc.servicer.save_conversation", side_effect=Exception("fail")),
        patch("src.grpc.servicer.ConvoSaveDuration"),
        patch("src.grpc.servicer.ErrorsTotal"),
    ):
        response = servicer.SaveTranscript(_make_request(), MagicMock())

    assert response.success is False
    mock_logger.delete.assert_not_called()
    assert SESSION_TOKEN in servicer._convo_loggers


def test_passes_visitor_ids_and_convo_text_to_save_conversation(servicer):
    _inject_logger(servicer)

    with (
        patch("src.grpc.servicer.fetch_profile_id_from_cache", return_value=PROFILE_ID),
        patch("src.grpc.servicer.save_conversation") as mock_save,
        patch("src.grpc.servicer.ConvoSaveDuration"),
    ):
        servicer.SaveTranscript(_make_request(visitor_ids=VISITOR_IDS), MagicMock())

    mock_save.assert_called_once()
    _, call_profile_id, call_convo_text, call_visitor_ids = mock_save.call_args.args
    assert call_profile_id == PROFILE_ID
    assert call_convo_text == CONVO_TEXT
    assert call_visitor_ids == VISITOR_IDS
