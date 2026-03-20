from .db.connection_pool import create_connection_pool
from concurrent.futures import ThreadPoolExecutor
from grpc_health.v1 import health as grpc_health
from grpc_health.v1 import health_pb2_grpc
from .grpc.servicer import AudioTranscriptionServicer
from .core.logging import logger
from .core.settings import settings
from .gen import audio_transcription_pb2_grpc
import grpc
import signal


def handle_shutdown(server, _sig, _frame):
    server.stop(grace=5)


def serve():
    server = grpc.server( # pyrefly: ignore
        ThreadPoolExecutor(max_workers=settings.max_workers)
    )
    db_pool = create_connection_pool()

    audio_transcription_pb2_grpc.add_AudioTranscriptionServiceServicer_to_server(
        AudioTranscriptionServicer(db_pool), server
    )
    health_servicer = grpc_health.HealthServicer()
    health_pb2_grpc.add_HealthServicer_to_server(health_servicer, server)

    server.add_insecure_port(f"[::]:{settings.grpc_server_port}")
    server.start()
    logger.info("gRPC server running on", port=f"[::]:{settings.grpc_server_port}")

    signal.signal(signal.SIGINT, lambda sig, frame: handle_shutdown(server, sig, frame))
    signal.signal(
        signal.SIGTERM, lambda sig, frame: handle_shutdown(server, sig, frame)
    )

    try:
        server.wait_for_termination()
    finally:
        db_pool.close()
        logger.info("server stopped")


if __name__ == "__main__":
    serve()
