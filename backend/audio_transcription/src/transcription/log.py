from datetime import datetime
from pathlib import Path
import logging
import threading

logger = logging.getLogger(__name__)

class ConvoLogHandler:
    """                                                                                                                                                                      
    Handles reading and writing transcription log files.                                                                                                                     
    Writes in progress so if it crashes mid transcription                                                                                                                    
    we still have the log containing transcribed text so far.                                                                                                                
    """
    def __init__(self, path: Path) -> None:
        self._path = path
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
    
    def read(self) -> str:
        """read the convo log for the patient"""
        with open(self._path, "r", encoding = "utf-8") as f:
            return f.read()
    
    def delete(self) -> None:
        """Delete the log file, used after transcript is saved to db"""
        self._path.unlink(missing_ok=True)

    def close(self) -> None:
        """Close the log file after finishing transcription"""
        with self._lock:
            try:
                self._file.close()
            except Exception:
                pass
