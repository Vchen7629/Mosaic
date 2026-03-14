from pydantic_settings import BaseSettings
from pydantic_settings import SettingsConfigDict
from pathlib import Path
import os

PROJECT_ROOT = Path(__file__).parent.parent.parent
ENV_FILE = PROJECT_ROOT / ".env"
PRODUCTION_MODE = os.getenv("PRODUCTION", "True") == "True"
_LOGS_DIR = Path(__file__).parents[2] / "logs"

class Settings(BaseSettings):
    """
    Centralized Settings class that contains all of the application settings,
    Single source of truth so functions and files can import it
    """

    log_level: str = "INFO"
    log_dir: str = ""  # Will be set in __init__ based on production mode

    # --- gRPC Server Settings ---
    grpc_server_port: int = 50051
    max_workers: int = 10 

    # --- Audio Settings ---
    whisper_mode: str = "small"

settings = Settings()