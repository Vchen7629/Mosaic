import threading
import time
from unittest.mock import MagicMock
import pytest
import grpc

from src.gen.audio_transcription_pb2 import TranscribeAudioRequest
from src.gen import audio_transcription_pb2_grpc
from test.fixtures.transcription import make_segment, make_chunk


def _request(profile_id: int = 1) -> TranscribeAudioRequest:
    return TranscribeAudioRequest(
        audio_bytes=make_chunk().tolist(), profile_id=profile_id
    )


def test_concurrent_requests_all_succeed(grpc_client):
    results = [None] * 10

    def call(i):
        results[i] = grpc_client.TranscribeAudio(_request(profile_id=i + 1))

    threads = [threading.Thread(target=call, args=(i,)) for i in range(10)]
    for t in threads:
        t.start()
    for t in threads:
        t.join(timeout=15)

    assert all(r is not None and r.success for r in results)


def test_concurrent_requests_from_different_profiles(grpc_client):
    results = {}
    lock = threading.Lock()

    def call(profile_id):
        resp = grpc_client.TranscribeAudio(_request(profile_id=profile_id))
        with lock:
            results[profile_id] = resp

    profile_ids = list(range(1, 6))
    threads = [threading.Thread(target=call, args=(pid,)) for pid in profile_ids]
    for t in threads:
        t.start()
    for t in threads:
        t.join(timeout=15)

    assert set(results.keys()) == set(profile_ids)
    assert all(r.success for r in results.values())


def test_returns_resource_exhausted_when_queue_full(
    mock_model, small_queue_grpc_server
) -> None:
    blocked = threading.Event()
    unblock = threading.Event()

    def blocking_transcribe(*a, **kw):
        blocked.set()
        unblock.wait(timeout=5)
        return iter([make_segment("x")]), MagicMock()

    mock_model.transcribe = blocking_transcribe

    channel = grpc.insecure_channel(small_queue_grpc_server)
    client = audio_transcription_pb2_grpc.AudioTranscriptionServiceStub(channel)

    occupier = threading.Thread(target=client.TranscribeAudio, args=(_request(),))
    occupier.start()
    blocked.wait(timeout=3)

    filler = threading.Thread(target=client.TranscribeAudio, args=(_request(),))
    filler.start()
    time.sleep(0.05)

    with pytest.raises(grpc.RpcError) as exc_info:
        client.TranscribeAudio(_request())

    assert exc_info.value.code() == grpc.StatusCode.RESOURCE_EXHAUSTED

    unblock.set()
    occupier.join(timeout=5)
    filler.join(timeout=5)
    channel.close()
