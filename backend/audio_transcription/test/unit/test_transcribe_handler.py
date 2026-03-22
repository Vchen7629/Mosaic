import threading
import time
from unittest.mock import MagicMock
import numpy as np
import pytest
from src.transcription.handler import QueueFullError


def test_concurrent_requests_all_complete(handler, make_chunk):
    results = [None] * 10

    def call(i):
        results[i] = handler.transcribe(make_chunk())

    threads = [threading.Thread(target=call, args=(i,)) for i in range(10)]
    for t in threads:
        t.start()
    for t in threads:
        t.join(timeout=10)

    assert all(r == "hello" for r in results)


def test_worker_never_runs_inference_concurrently(
    handler, mock_model, make_segment, make_chunk
) -> None:
    active = []
    overlap = []
    lock = threading.Lock()

    def fake_transcribe(chunk, **kwargs):
        with lock:
            if active:
                overlap.append(True)
            active.append(1)
        time.sleep(0.02)
        with lock:
            active.pop()
        return iter([make_segment("ok")]), MagicMock()

    mock_model.transcribe = fake_transcribe

    threads = [
        threading.Thread(target=handler.transcribe, args=(make_chunk(),))
        for _ in range(8)
    ]
    for t in threads:
        t.start()
    for t in threads:
        t.join(timeout=10)

    assert not overlap


def test_results_not_mixed_between_callers(
    handler, mock_model, make_segment, make_chunk
) -> None:
    call_count = 0
    lock = threading.Lock()

    def fake_transcribe(chunk, **kwargs):
        nonlocal call_count
        with lock:
            call_count += 1
            label = f"result_{call_count}"
        time.sleep(0.01)
        return iter([make_segment(label)]), MagicMock()

    mock_model.transcribe = fake_transcribe
    results = [None] * 5

    threads = [
        threading.Thread(
            target=lambda i=i: results.__setitem__(i, handler.transcribe(make_chunk())),
            args=(),
        )
        for i in range(5)
    ]
    for t in threads:
        t.start()
    for t in threads:
        t.join(timeout=10)

    assert len(set(results)) == 5


def test_raises_when_queue_saturated(
    small_queue_handler, mock_model, make_segment, make_chunk
) -> None:
    blocked = threading.Event()
    unblock = threading.Event()

    def blocking_transcribe(chunk, **kwargs):
        blocked.set()
        unblock.wait(timeout=5)
        return iter([make_segment("x")]), MagicMock()

    mock_model.transcribe = blocking_transcribe

    occupier = threading.Thread(
        target=small_queue_handler.transcribe, args=(make_chunk(),)
    )
    occupier.start()
    blocked.wait(timeout=3)

    filler = threading.Thread(
        target=small_queue_handler.transcribe, args=(make_chunk(),)
    )
    filler.start()
    time.sleep(0.05)

    with pytest.raises(QueueFullError):
        small_queue_handler.transcribe(make_chunk())

    unblock.set()
    occupier.join(timeout=5)
    filler.join(timeout=5)


def test_clamps_loud_audio(handler):
    result = handler._normalize(np.full(100, 5.0, dtype=np.float32))
    assert np.abs(result).max() <= 1.0


def test_replaces_non_finite_values(handler):
    chunk = np.array([float("nan"), float("inf"), float("-inf"), 0.5], dtype=np.float32)
    assert np.isfinite(handler._normalize(chunk)).all()


def test_leaves_valid_audio_unchanged(handler):
    chunk = np.array([0.1, -0.2, 0.3], dtype=np.float32)
    np.testing.assert_array_almost_equal(handler._normalize(chunk), chunk)


def test_returns_none_on_empty_transcription(
    handler, mock_model, make_segment, make_chunk
) -> None:
    mock_model.transcribe.side_effect = lambda *a, **kw: (
        iter([make_segment("   ")]),
        MagicMock(),
    )
    assert handler.transcribe(make_chunk()) is None


def test_returns_none_on_inference_exception(handler, mock_model, make_chunk) -> None:
    mock_model.transcribe.side_effect = RuntimeError("GPU error")
    assert handler.transcribe(make_chunk()) is None


def test_joins_multiple_segments(handler, mock_model, make_segment, make_chunk) -> None:
    mock_model.transcribe.side_effect = lambda *a, **kw: (
        iter([make_segment("hello"), make_segment("world")]),
        MagicMock(),
    )
    assert handler.transcribe(make_chunk()) == "hello world"
