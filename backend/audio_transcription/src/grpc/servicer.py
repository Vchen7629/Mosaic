from psycopg_pool import ConnectionPool
from valkey import Valkey
from typing import Optional, Any, TYPE_CHECKING

if TYPE_CHECKING:
    from grpc import ServicerContext  # pyrefly: ignore
from ..core.logging import logger
from ..core.settings import _LOGS_DIR
from ..core.metrics import ErrorsTotal
from ..core.metrics import ConvoSaveDuration
from ..gen import audio_transcription_pb2
from ..gen import audio_transcription_pb2_grpc
from ..db.conversation_queries import save_conversation
from ..db.session_queries import fetch_profile_id_from_session
from ..cache.session import fetch_profile_id_from_cache
from ..transcription.convo_log import ConvoLogHandler
from ..transcription.handler import QueueFullError
from ..transcription.handler import TranscriptionHandler
import grpc
import time
import threading
import numpy as np


class AudioTranscriptionServicer(
    audio_transcription_pb2_grpc.AudioTranscriptionServiceServicer
):
    def __init__(
        self,
        db_pool: ConnectionPool,
        transcription_handler: TranscriptionHandler,
        cache: Valkey,
    ) -> None:
        self._convo_loggers: dict[str, ConvoLogHandler] = {}
        self._lock = threading.Lock()
        self._db_pool: ConnectionPool = db_pool
        self._transcription_handler = transcription_handler
        self._cache: Valkey = cache

    def TranscribeAudio(
        self,
        request: audio_transcription_pb2.TranscribeAudioRequest,
        context: "ServicerContext[Any, Any]",  # pyrefly: ignore
    ) -> Optional[audio_transcription_pb2.TranscribeAudioResponse]:
        """gRPC handler to take in requests to this server, process and send response"""
        chunk = np.array(request.audio_bytes, dtype=np.float32)
        logger.debug("got audio chunk to transcribe", chunk_size=len(chunk))
        try:
            text = self._transcription_handler.transcribe(chunk)
        except QueueFullError:
            context.abort(
                grpc.StatusCode.RESOURCE_EXHAUSTED,  # pyrefly: ignore
                "Transcription queue is full, retry later",
            )
            return
        if text:
            logger.debug("writing audio transcribe to text to internal log", text=text)
            self._get_log_writer(request.session_token).write(text)

        return audio_transcription_pb2.TranscribeAudioResponse(success=text is not None)

    def SaveTranscript(
        self,
        request: audio_transcription_pb2.SaveTranscriptRequest,
        context: "ServicerContext[Any, Any]",  # pyrefly: ignore  # noqa: ARG002
    ) -> Optional[audio_transcription_pb2.SaveTranscriptResponse]:
        """gRPC handler that saves the current transcript to db"""
        session_token = request.session_token
        logger.debug(
            "called savetranscript to save transcript", session_token=session_token
        )
        try:
            profile_logger = self._convo_loggers.get(session_token)
            if profile_logger is None:
                logger.error(
                    "profile_logger for session_token is none",
                    session_token=session_token,
                )
                return audio_transcription_pb2.SaveTranscriptResponse(success=False)

            profile_logger.close()
            convo_text = profile_logger.read()

            profile_id = fetch_profile_id_from_cache(self._cache, session_token)
            if profile_id is None:
                logger.debug(
                    "cache miss for session_token, falling back to db",
                    session_token=session_token,
                )
                with self._db_pool.connection() as conn:
                    profile_id = fetch_profile_id_from_session(conn, session_token)

            if profile_id is None:
                logger.error(
                    "could not resolve profile_id for session_token",
                    session_token=session_token,
                )
                return audio_transcription_pb2.SaveTranscriptResponse(success=False)

            convo_start = time.monotonic()
            with self._db_pool.connection() as conn:
                save_conversation(
                    conn, profile_id, convo_text, list(request.visitor_ids)
                )
            ConvoSaveDuration.observe((time.monotonic() - convo_start) * 1000)
        except Exception as e:
            ErrorsTotal.labels(operation="save_conversation").inc()
            logger.error("failed to save conversation", err=str(e))
            return audio_transcription_pb2.SaveTranscriptResponse(success=False)

        profile_logger.delete()
        del self._convo_loggers[session_token]

        return audio_transcription_pb2.SaveTranscriptResponse(success=True)

    def _get_log_writer(self, session_token: str) -> ConvoLogHandler:
        """
        Creates the log file with session_token specific name
        and registers the log writer for the session to dict
        """
        with self._lock:
            if session_token not in self._convo_loggers:
                _LOGS_DIR.mkdir(parents=True, exist_ok=True)
                path = _LOGS_DIR / f"recording_{session_token}.txt"
                self._convo_loggers[session_token] = ConvoLogHandler(path)
                logger.debug("created convo logger", session_token=session_token)
            return self._convo_loggers[session_token]
