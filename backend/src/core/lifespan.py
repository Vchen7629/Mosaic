from fastapi import FastAPI
from contextlib import asynccontextmanager
from ..db.conn import db_instance
from ..audio.model import get_model
from ..face_detection.service import face_detection_service
import logging

logger = logging.getLogger(__name__)

@asynccontextmanager
async def lifespan(app: FastAPI):
    await db_instance.connect()

    try:
        get_model()
    except Exception as e:
        logger.error(f"Failed to load Whisper model: {e}")
        await db_instance.disconnect()
        return

    try:
        face_detection_service.startup()
    except Exception as e:
        logger.error(f"Failed to start face detection worker: {e}")
        await db_instance.disconnect()
        return

    yield

    face_detection_service.shutdown()
    await db_instance.disconnect()
