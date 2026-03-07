import cv2
import numpy as np
import face_recognition

KNOWN_FACE_DISTANCE_THRESHOLD = 0.6

class FaceDetector:
    """Core face detection and matching against known embeddings."""

    def __init__(self) -> None:
        self._known_embeddings: list[dict] = []

    def detect(self, frame) -> list[tuple[str | None, tuple, list]]:
        """Returns (person_id, face_location, encoding) for each face found in frame."""
        rgb = cv2.cvtColor(frame, cv2.COLOR_BGR2RGB)
        locations = face_recognition.face_locations(rgb, model="cnn")
        encodings = face_recognition.face_encodings(rgb, locations)
        return [
            (self._match_face(enc), loc, enc.tolist())
            for loc, enc in zip(locations, encodings)
        ]

    def _match_face(self, encoding) -> str | None:
        if not self._known_embeddings:
            return None
        known_encs = [np.array(e["embedding"]) for e in self._known_embeddings]
        distances = face_recognition.face_distance(known_encs, encoding)
        min_idx = int(np.argmin(distances))
        if distances[min_idx] < KNOWN_FACE_DISTANCE_THRESHOLD:
            return str(self._known_embeddings[min_idx]["person_id"])
        return None