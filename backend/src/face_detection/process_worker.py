"""
Face detection worker – runs in a dedicated subprocess.

`detect` is the picklable unit of work submitted per video frame.
face_recognition / dlib are imported at module level so they are loaded once
per worker process.
"""
import cv2
import face_recognition
import numpy as np
from typing import Optional

KNOWN_FACE_DISTANCE_THRESHOLD = 0.6


def noop() -> None:
    """Picklable no-op used to force worker spawn during startup."""


def detect(frame: np.ndarray, known_embeddings: list) -> list:
    """
    Returns a list of (person_id, face_location, encoding) for every face found.
    person_id is None when no match is found in known_embeddings.
    """
    rgb = cv2.cvtColor(frame, cv2.COLOR_BGR2RGB)
    locations = face_recognition.face_locations(rgb, model="cnn")
    encodings = face_recognition.face_encodings(rgb, locations)
    return [
        (_match(enc, known_embeddings), loc, enc.tolist())
        for loc, enc in zip(locations, encodings)
    ]


def _match(encoding: np.ndarray, known_embeddings: list) -> Optional[str]:
    if not known_embeddings:
        return None
    distances = face_recognition.face_distance(
        [np.array(e["embedding"]) for e in known_embeddings], encoding
    )
    min_idx = int(np.argmin(distances))
    if distances[min_idx] < KNOWN_FACE_DISTANCE_THRESHOLD:
        return str(known_embeddings[min_idx]["person_id"])
    return None
