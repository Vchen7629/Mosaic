from psycopg_pool import ConnectionPool
from typing import Optional, Any, TYPE_CHECKING
if TYPE_CHECKING:
    from grpc import ServicerContext
from ..core.logging import logger
from ..core.settings import _LOGS_DIR
from ..core.metrics import ErrorsTotal
from ..core.metrics import ConvoSaveDuration
from ..gen import audio_transcription_pb2
from ..gen import audio_transcription_pb2_grpc
from ..db.queries import save_conversation
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
        self, db_pool: ConnectionPool, transcription_handler: TranscriptionHandler
    ) -> None:
        self._convo_loggers: dict[int, ConvoLogHandler] = {}
        self._lock = threading.Lock()
        self._db_pool: ConnectionPool = db_pool
        self._transcription_handler = transcription_handler

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
            self._get_log_writer(request.profile_id).write(text)

        return audio_transcription_pb2.TranscribeAudioResponse(success=text is not None)

    def SaveTranscript(
        self,
        request: audio_transcription_pb2.SaveTranscriptRequest,
        context: "ServicerContext[Any, Any]",  # pyrefly: ignore  # noqa: ARG002
    ) -> Optional[audio_transcription_pb2.SaveTranscriptResponse]:
        """gRPC handler that saves the current transcript to db"""
        logger.debug(
            "called savetranscript to save transcript", profile_id=request.profile_id
        )
        try:
            profile_logger = self._convo_loggers.get(request.profile_id)
            if profile_logger is None:
                logger.error(
                    "profile_logger for profile_id is none",
                    profile_id=request.profile_id,
                )
                return audio_transcription_pb2.SaveTranscriptResponse(success=False)

            profile_logger.close()
            convo_text = profile_logger.read()

            convo_start = time.monotonic()
            with self._db_pool.connection() as conn:
                save_conversation(
                    conn, request.profile_id, convo_text, request.visitor_ids
                )
            ConvoSaveDuration.observe((time.monotonic() - convo_start) * 1000)
        except Exception as e:
            ErrorsTotal.labels(operation="save_conversation").inc()
            logger.error("failed to save conversation", err=str(e))
            return audio_transcription_pb2.SaveTranscriptResponse(success=False)

        profile_logger.delete()
        del self._convo_loggers[request.profile_id]

        return audio_transcription_pb2.SaveTranscriptResponse(success=True)

    def _get_log_writer(self, profile_id: int) -> ConvoLogHandler:
        """
        Creates the log file with profile_id specific name
        and registers the log writer for the patient to dict
        """
        with self._lock:
            if profile_id not in self._convo_loggers.keys():
                _LOGS_DIR.mkdir(parents=True, exist_ok=True)
                path = _LOGS_DIR / f"recording_{profile_id}.txt"
                self._convo_loggers[profile_id] = ConvoLogHandler(path)
                logger.debug("created convo logger", profile_id=profile_id)
            return self._convo_loggers[profile_id]
