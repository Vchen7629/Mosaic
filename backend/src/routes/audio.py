from ..audio.transcription import LogReader
from fastapi import Depends
from fastapi import APIRouter
from typing import Annotated
from ..audio import get_audio_recorder
from ..audio.capture import AudioRecorder

router = APIRouter(prefix="/audio")


@router.get(path="/start")
async def start_recording(recorder: Annotated[AudioRecorder, Depends(get_audio_recorder)]):
    log_path = recorder.start()
    return {"log_path": log_path}

@router.get(path="/stop")
async def stop_recording(recorder: Annotated[AudioRecorder, Depends(get_audio_recorder)]):
    recorder.stop()

    transcript = LogReader().read()

    LogReader().delete()

    return {"transcript": transcript}
