from collections import deque
from concurrent.futures import ProcessPoolExecutor
from typing import Callable
from .detector import FaceDetector
from .process_worker import detect as _detect, noop
import cv2
import logging
import multiprocessing as mp
import time
import threading

logger = logging.getLogger(__name__)

_mp_ctx = mp.get_context("spawn")

PATIENT_PERSON_ID = "__patient__"
DETECTION_INTERVAL = 2.5


class FaceDetectionService(FaceDetector):
    """Manages webcam streaming, threading, and in-memory known face cache."""

    def __init__(self) -> None:
        super().__init__()
        self._running = False
        self._thread: threading.Thread | None = None
        self._pool: ProcessPoolExecutor | None = None
        self.latest_faces: deque[tuple[str | None, tuple]] = deque(maxlen=32)
        self._on_face_detected: Callable[[str | None, list], None] | None = None
        self.patient_id: str | None = None
        self.known_faces: dict[str, list[float]] = {}  # name -> embedding
        self._patient_embedding: list[float] | None = None
        self.pending_unknown_encoding: list[float] | None = None  # awaiting user confirmation

    def startup(self) -> None:
        """Start the detection worker process. Call once from app lifespan."""
        self._pool = ProcessPoolExecutor(max_workers=1, mp_context=_mp_ctx)
        # Force the worker to spawn now so dlib/face_recognition are imported
        # before the first detection request arrives.
        self._pool.submit(noop).result()
        logger.info("[face_detection] Detection worker started")

    def shutdown(self) -> None:
        """Shut down the detection worker process. Call from app lifespan cleanup."""
        if self._pool:
            self._pool.shutdown(wait=False)
            self._pool = None

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
        self._thread = None  # daemon thread exits naturally
        self.pending_unknown_encoding = None

    def _run(self) -> None:
        last_detection_time = 0.0
        for frame in self._stream_frames():
            now = time.time()
            if now - last_detection_time >= DETECTION_INTERVAL:
                last_detection_time = now
                if not self._pool:
                    continue
                try:
                    results = self._pool.submit(
                        _detect, frame, list(self._known_embeddings)
                    ).result()
                except Exception as e:
                    print(f"[face_detection] detection error: {e}", flush=True)
                    continue
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
