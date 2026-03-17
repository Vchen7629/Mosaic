from psycopg_pool import ConnectionPool
from ..core.settings import _LOGS_DIR
from ..gen import audio_transcription_pb2
from ..gen import audio_transcription_pb2_grpc
from ..db.queries import save_conversation
from ..transcription.log import ConvoLogHandler
from ..transcription.handler import transcribe_handler
import threading
import numpy as np

class AudioTranscriptionServicer(
    audio_transcription_pb2_grpc.AudioTranscriptionServiceServicer
):
    def __init__(self, db_pool: ConnectionPool) -> None:
        self._convo_loggers: dict[str, ConvoLogHandler] = {}
        self._lock = threading.Lock()
        self._db_pool: ConnectionPool = db_pool

    def TranscribeAudio(self, request, _context):
        chunk = np.array(request.audio_bytes, dtype=np.float32)
        print(f"[audio_transcription] got chunk {chunk}")
        text = transcribe_handler(chunk)
        if text:
            self._get_log_writer(request.profile_id).write(text)

        return audio_transcription_pb2.TranscribeAudioResponse(success=text is not None)

    def SaveTranscript(self, request, _context):
        """gRPC handler that saves the current transcript to db"""
        try:
            logger = self._convo_loggers.get(request.profile_id)
            if logger is None:
                return audio_transcription_pb2.SaveTranscriptResponse(success=False)
            
            logger.close()
            convo_text = logger.read()

            with self._db_pool.connection() as conn:
                save_conversation(conn, request.profile_id, convo_text, request.visitor_id)
        except Exception:
            return audio_transcription_pb2.SaveTranscriptResponse(success=False)

        logger.delete()
        del self._convo_loggers[request.profile_id]

        return audio_transcription_pb2.SaveTranscriptResponse(success=True)

    def _get_log_writer(self, profile_id: str) -> ConvoLogHandler:
        """
        Creates the log file with profile_id specific name
        and registers the log writer for the patient to dict
        """
        with self._lock:
            if profile_id not in self._convo_loggers.keys():
                _LOGS_DIR.mkdir(parents=True, exist_ok=True)
                path = _LOGS_DIR / f"recording_{profile_id}.txt"
                self._convo_loggers[profile_id] = ConvoLogHandler(path)
            return self._convo_loggers[profile_id]