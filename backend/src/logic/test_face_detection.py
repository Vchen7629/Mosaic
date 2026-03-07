import cv2
import time
from ..face_detection.detector import FaceDetector
from ..face_detection.service import DETECTION_INTERVAL


def test_continuous():
    detector = FaceDetector()
    cap = cv2.VideoCapture(0)
    last_detection_time = 0.0
    latest_results = []

    while True:
        ret, frame = cap.read()
        if not ret:
            break

        now = time.time()
        if now - last_detection_time >= DETECTION_INTERVAL:
            latest_results = detector.detect(frame)
            last_detection_time = now

        for person_id, (top, right, bottom, left), _ in latest_results:
            cv2.rectangle(frame, (left, top), (right, bottom), (0, 0, 255), 2)
            label = person_id if person_id else "Unknown"
            cv2.putText(frame, label, (left, top - 10), cv2.FONT_HERSHEY_SIMPLEX, 0.5, (0, 255, 0), 1)

        cv2.imshow(f"{len(latest_results)} face(s) detected", frame)
        if cv2.waitKey(1) & 0xFF == ord('q'):
            break

    cap.release()
    cv2.destroyAllWindows()


if __name__ == "__main__":
    test_continuous()
