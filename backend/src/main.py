from fastapi import FastAPI
from fastapi.middleware.cors import CORSMiddleware
from .core.lifespan import lifespan
from .core.settings import settings
from .routes.audio import router as audio_router
from .routes.faces import router as face_router
from .routes.conversation import router
import logging
import uvicorn

logger = logging.getLogger(__name__)

app = FastAPI(lifespan=lifespan)

app.add_middleware(
    CORSMiddleware,
    allow_origins=["http://localhost:1420", "tauri://localhost"],
    allow_methods=["*"],
    allow_headers=["*"],
)

app.include_router(audio_router)
app.include_router(face_router)
app.include_router(router)

@app.get("/")
async def root():
    return {"message": "Hello World"}

if __name__ == "__main__":
    # This block allows running with `python -m backend.src.main`
    logger.info(f"Starting backend service at {settings.backend_url}")
    uvicorn.run(app, host=settings.backend_host, port=settings.backend_port)
