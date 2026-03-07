from fastapi import FastAPI
from src.core.lifespan import lifespan
from src.core.settings import settings
from src.routes.audio import router as audio_router
import logging
import uvicorn

logger = logging.getLogger(__name__)

app = FastAPI(lifespan=lifespan)

app.include_router(audio_router)

@app.get("/")
async def root():
    return {"message": "Hello World"}

if __name__ == "__main__":
    # This block allows running with `python -m backend.src.main`
    logger.info(f"Starting backend service at {settings.backend_url}")
    uvicorn.run(app, host=settings.backend_host, port=settings.backend_port)
