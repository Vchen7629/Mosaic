from pydantic_settings import BaseSettings
from typing import List
from pathlib import Path
import os

PROJECT_ROOT = Path(__file__).parent.parent.parent
ENV_FILE = PROJECT_ROOT / ".env"
PRODUCTION_MODE = os.getenv("PRODUCTION", "True") == "True"

class Settings(BaseSettings):
    """
    Centralized Settings class that contains all of the application settings,
    Single source of truth so functions and files can import it
    """

    log_level: str = "INFO"
    log_dir: str = ""  # Will be set in __init__ based on production mode

    # --- API Settings ---
    backend_url: str = "http://127.0.0.1:8000"
    backend_host: str = "127.0.0.1"
    backend_port: int = 8000

    # --- Audio Settings ---
    chunk_duration_s: float = 10.0  # Duration of each audio chunk to process
    block_duration_sec: float = 10.0
    blocksize: int = 4096
    sample_rate: int = 16000  # 16kHz, as required by Whisper
    device_name_hints: dict[str, List[str]] = {
        "Darwin": ["BlackHole 2ch"],
        "Windows": ["Stereo Mix", "Wave Out Mix", "CABLE Output", "VB-Audio"],
        "Linux": ["pipewire", "monitor", "loopback", "default"],
    }

    silence_threshold: float = 0.0002
    max_hash_history: int = 3

    whisper_mode: str = "small"

settings = Settings()