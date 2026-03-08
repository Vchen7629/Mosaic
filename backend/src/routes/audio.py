from fastapi import Depends
from fastapi import APIRouter
from typing import Annotated
from ..audio import get_audio_recorder
from ..audio.capture import AudioRecorder
from ..audio.transcription import LogReader
import asyncio

router = APIRouter(prefix="/audio")


@router.get(path="/start")
async def start_recording(recorder: Annotated[AudioRecorder, Depends(get_audio_recorder)]):
    log_path = recorder.start()
    return {"log_path": log_path}

@router.get(path="/stop")
async def stop_recording(recorder: Annotated[AudioRecorder, Depends(get_audio_recorder)]):
    recorder.stop()
    transcript = await asyncio.to_thread(recorder.wait_for_transcript, 300.0)

    LogReader().delete()
    return {"transcript": transcript}
