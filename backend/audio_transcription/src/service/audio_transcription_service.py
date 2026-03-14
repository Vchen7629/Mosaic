from .transcribe_handler import transcribe_handler
from ..core.settings import _LOGS_DIR
from .. import audio_transcription_pb2
from .. import audio_transcription_pb2_grpc
from .transcription_log import LogWriter
import threading
import numpy as np

class AudioTranscriptionServicer(
    audio_transcription_pb2_grpc.AudioTranscriptionServiceServicer
):
    def __init__(self) -> None:
        self._log_writers: dict[str, LogWriter] = {}
        self._lock = threading.Lock()

    def TranscribeAudio(self, request, _context):
        chunk = np.array(request.audio_bytes, dtype=np.float32)
        print(f"[audio_transcription] got chunk {chunk}")
        text = transcribe_handler(chunk)
        if text:
            self._get_log_writer(request.patient_id).write(text)

        return audio_transcription_pb2.TranscribeAudioResponse(success=text is not None)

    def SaveTranscript(self, request, _context):
        """gRPC handler that saves the current transcript to db"""
        print("Invoked Save!")
        return audio_transcription_pb2.SaveTranscriptResponse(success=True)

    def _get_log_writer(self, patient_id: str) -> LogWriter:
        """
        Creates the log file with patient_id specific name
        and registers the log writer for the patient to dict
        """
        with self._lock:
            if patient_id not in self._log_writers.keys():
                _LOGS_DIR.mkdir(parents=True, exist_ok=True)
                path = _LOGS_DIR / f"recording_{patient_id}.txt"
                self._log_writers[patient_id] = LogWriter(path)
            return self._log_writers[patient_id]