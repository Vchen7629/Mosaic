from typing import Optional
from src.core.settings import settings
import logging
import platform
import sounddevice as sd

logger = logging.getLogger(__name__)


def get_input_device() -> Optional[int]:
    """Return the best available input device index, or None for system default."""
    system = platform.system()
    hints = [h for h in settings.device_name_hints.get(system, []) if h not in ("default", "")]
    devices = sd.query_devices()

    for index, device in enumerate(devices):
        if device["max_input_channels"] > 0:
            for hint in hints:
                if hint.lower() in device["name"].lower():
                    logger.info(f"Found audio device [{index}]: {device['name']}")
                    return index

    try:
        default_idx = sd.default.device[0]
        info = sd.query_devices(default_idx)
        if info["max_input_channels"] > 0:
            logger.info(f"Using default input device [{default_idx}]: {info['name']}")
            return default_idx
    except Exception:
        pass

    for index, device in enumerate(devices):
        if device["max_input_channels"] > 0:
            logger.info(f"Using fallback input device [{index}]: {device['name']}")
            return index

    raise RuntimeError("No input audio device found.")