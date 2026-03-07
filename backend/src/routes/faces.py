from typing import Annotated
from fastapi import APIRouter, Depends, WebSocket
from pymongo.asynchronous.database import AsyncDatabase
from ..db.conn import get_db
from ..face_detection.handler import handle_face_detection_ws

router = APIRouter(prefix="/face")


@router.websocket("/detection")
async def face_detection_ws(
    websocket: WebSocket,
    patient_id: str,
    db: Annotated[AsyncDatabase, Depends(get_db)]
) -> None:
    await handle_face_detection_ws(websocket, patient_id, db)
