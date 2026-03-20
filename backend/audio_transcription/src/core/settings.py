from pydantic_settings import BaseSettings
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
    log_format: str = "json"
    log_dir: str = ""  # Will be set in __init__ based on production mode

    # --- gRPC Server Settings ---
    grpc_server_port: int = 50051
    max_workers: int = 10

    metrics_server_port: int = 9091

    # --- Audio Settings ---
    whisper_mode: str = "small"

    # --- db connection cfg ---
    DB_HOST: str = "localhost"
    DB_PORT: int = 5432
    DB_NAME: str = "pg"
    DB_USER: str = "user"
    DB_PASS: str = "password"
    POOL_MIN_SIZE: int = 1
    POOL_MAX_SIZE: int = 10
    KEEP_ALIVES_COUNT: int = 1
    KEEP_ALIVES_IDLE_S: int = 30
    KEEP_ALIVES_RETRY_INTERVAL_S: int = 10
    KEEP_ALIVES_RETRY_COUNT: int = 5


settings = Settings()
