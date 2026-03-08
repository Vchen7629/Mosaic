from functools import lru_cache
from ..audio.capture import AudioRecorder

@lru_cache(maxsize=1)
def get_audio_recorder() -> AudioRecorder:
    return AudioRecorder()