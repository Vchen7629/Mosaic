from collections import deque
from typing import Callable
from .detector import FaceDetector
import cv2
import time
import threading

PATIENT_PERSON_ID = "__patient__"
DETECTION_INTERVAL = 2.5


class FaceDetectionService(FaceDetector):
    """Manages webcam streaming, threading, and in-memory known face cache."""

    def __init__(self) -> None:
        super().__init__()
        self._running = False
        self._thread: threading.Thread | None = None
        self.latest_faces: deque[tuple[str | None, tuple]] = deque(maxlen=32)
        self._on_face_detected: Callable[[str | None, list], None] | None = None
        self.patient_id: str | None = None
        self.known_faces: dict[str, list[float]] = {}  # name -> embedding
        self._patient_embedding: list[float] | None = None
        self.pending_unknown_encoding: list[float] | None = None  # awaiting user confirmation

    def load_from_patient_doc(self, patient_doc: dict) -> None:
        self.patient_id = patient_doc.get("user_id")
        self._patient_embedding = patient_doc.get("face_embedding")
        self.known_faces = {
            u["name"]: u["face_embedding"]
            for u in (patient_doc.get("interacted_users") or [])
            if u.get("face_embedding")
        }
        self._sync_known_embeddings()

    def add_known_face(self, name: str, embedding: list[float]) -> None:
        self.known_faces[name] = embedding
        self._sync_known_embeddings()

    def _sync_known_embeddings(self) -> None:
        self._known_embeddings = [
            {"person_id": name, "embedding": emb}
            for name, emb in self.known_faces.items()
        ]
        if self._patient_embedding:
            self._known_embeddings.append({"person_id": PATIENT_PERSON_ID, "embedding": self._patient_embedding})

    def start(self, on_face_detected: Callable[[str | None, list], None] | None = None) -> bool:
        if self._running:
            return False
        self._on_face_detected = on_face_detected
        self._running = True
        self._thread = threading.Thread(target=self._run, daemon=True)
        self._thread.start()
        return True

    def stop(self) -> None:
        self._running = False
        if self._thread:
            self._thread.join(timeout=5)
            self._thread = None
        self.pending_unknown_encoding = None

    def _run(self) -> None:
        last_detection_time = 0.0
        for frame in self._stream_frames():
            now = time.time()
            if now - last_detection_time >= DETECTION_INTERVAL:
                last_detection_time = now
                results = self.detect(frame)
                print(f"[face_detection] running detection... found {len(results)} face(s)")
                for person_id, face_location, encoding in results:
                    self.latest_faces.append((person_id, face_location))
                    if person_id is None:
                        self.pending_unknown_encoding = encoding
                    if self._on_face_detected:
                        self._on_face_detected(person_id, encoding)

    def _stream_frames(self):
        cap = cv2.VideoCapture(0)
        if not cap.isOpened():
            raise RuntimeError("Could not open webcam")
        try:
            while self._running:
                ret, frame = cap.read()
                if not ret:
                    break
                yield frame
        finally:
            cap.release()


face_detection_service = FaceDetectionService()
