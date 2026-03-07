from fastapi import FastAPI
from contextlib import asynccontextmanager
from dotenv import load_dotenv
from src.audio.model import get_model

load_dotenv()

@asynccontextmanager
async def lifespan(app: FastAPI):
    try:
        get_model()
    except Exception as e:
        logger.error(f"Failed to load Whisper model: {e}")
        return

    yield