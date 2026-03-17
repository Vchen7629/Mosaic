from ..core.settings import _LOGS_DIR
from datetime import datetime
import logging
import threading

logger = logging.getLogger(__name__)

_LOGS_DIR.mkdir(parents=True, exist_ok=True)
log_path = _LOGS_DIR / f"recording.txt"

class LogWriter:
    """
    Log writer class used to write transcriptions in progress 
    to a txt log file. This way if it crashes mid transcription
    We still have the log containing transcribed text so far
    """
    def __init__(self, path: str) -> None:
        self._file = open(path, "w", encoding="utf-8")
        self._lock = threading.Lock()

    def write(self, text: str) -> None:
        """Write to the internal log file with timestamp: text"""
        timestamp = datetime.now().strftime("%H:%M:%S")
        with self._lock:
            try:
                self._file.write(f"[{timestamp}] {text}\n")
                self._file.flush()
            except Exception as e:
                logger.error(f"Failed to write transcript: {e}")

    def close(self) -> None:
        """Close the log file after finishing transcription"""
        with self._lock:
            try:
                self._file.close()
            except Exception:
                pass

class LogReader:
    """
    Reader class to read all text from the transcription
    log to be used to be saved in the database after user
    clicks stop recording
    """
    def __init__(self) -> None:
        _LOGS_DIR.mkdir(parents=True, exist_ok=True)
        self._path = _LOGS_DIR / f"recording.txt"

    def read(self) -> str:
        """Read log file and return all the text"""
        with open(self._path, "r", encoding="utf-8") as f:
            return f.read()

    def delete(self) -> bool:
        """
        Delete log file, used after transcript is saved
        to database
        """
        self._path.unlink(missing_ok=True)

