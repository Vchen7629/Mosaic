import json
import asyncio
import logging

from fastapi import WebSocket, WebSocketDisconnect
from pymongo.asynchronous.database import AsyncDatabase

from ..db.embeddings import get_patient_doc, add_interacted_user
from .service import face_detection_service, PATIENT_PERSON_ID

logger = logging.getLogger(__name__)


async def handle_face_detection_ws(websocket: WebSocket, patient_id: str, db: AsyncDatabase) -> None:
    await websocket.accept()

    patient_doc = await get_patient_doc(db, patient_id)
    if patient_doc:
        face_detection_service.load_from_patient_doc(patient_doc)
    else:
        logger.warning(f"No patient doc found for {patient_id}, starting with empty embeddings")
    loop = asyncio.get_event_loop()

    connected = True

    def on_face_detected(person_id: str | None, encoding: list) -> None:
        if not connected or person_id == PATIENT_PERSON_ID:
            return
        print(f"[face_detection] face detected: person_id={person_id}")
        event = {"type": "known_face", "name": person_id} if person_id else {"type": "unknown_face"}
        asyncio.run_coroutine_threadsafe(websocket.send_json(event), loop)

    started = face_detection_service.start(on_face_detected=on_face_detected)
    print(f"[face_detection] start() called, started={started}, patient_id={patient_id}")

    try:
        while True:
            data = json.loads(await websocket.receive_text())
            if data.get("action") == "add_face":
                await _handle_new_face_detected(data, websocket, db, patient_id)
    except (WebSocketDisconnect, Exception):
        connected = False
        face_detection_service.stop()
        print(f"[face_detection] stopped for patient {patient_id}")


async def _handle_new_face_detected(data, websocket: WebSocket, db: AsyncDatabase, patient_id: str) -> None:
    name = data.get("name", "").strip()
    encoding = face_detection_service.pending_unknown_encoding

    if name and encoding:
        face_detection_service.add_known_face(name, encoding)
        face_detection_service.pending_unknown_encoding = None
        await add_interacted_user(db, patient_id, name, encoding)
        await websocket.send_json({"type": "face_added", "name": name})
        logger.info(f"New face added: {name}")
    else:
        logger.warning(f"add_face failed: name={repr(name)}, has_encoding={encoding is not None}")
        await websocket.send_json({"type": "error", "message": "No pending face encoding — try again"})
