from concurrent.futures import ThreadPoolExecutor
from .service.audio_transcription_service import AudioTranscriptionServicer
from .core.settings import settings
from gen import audio_transcription_pb2_grpc
import grpc
import signal

def handle_shutdown(server, _sig, _frame):
    server.stop(grace=5)

def serve():
    server = grpc.server(ThreadPoolExecutor(max_workers=settings.max_workers))
    audio_transcription_pb2_grpc.add_AudioTranscriptionServiceServicer_to_server(
        AudioTranscriptionServicer(), server
    )
    server.add_insecure_port(f'[::]:{settings.grpc_server_port}')
    print(f"Server running on [::]:{settings.grpc_server_port}")
    server.start()

    signal.signal(signal.SIGINT, lambda sig, frame: handle_shutdown(server, sig, frame))
    signal.signal(signal.SIGTERM, lambda sig, frame: handle_shutdown(server, sig, frame))

    server.wait_for_termination()

if __name__ == "__main__":
    serve()