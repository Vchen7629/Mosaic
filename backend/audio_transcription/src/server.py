from .db.connection_pool import create_connection_pool
from psycopg_pool import ConnectionPool
from prometheus_client import make_wsgi_app
from wsgiref.simple_server import make_server
from concurrent.futures import ThreadPoolExecutor
from grpc_health.v1 import health as grpc_health
from grpc_health.v1 import health_pb2_grpc
from .grpc.servicer import AudioTranscriptionServicer
from .core.logging import logger
from .core.settings import settings
from .gen import audio_transcription_pb2_grpc
import grpc
import signal
import threading

def handle_shutdown(server, _sig, _frame):
    server.stop(grace=5)

def start_metrics_server(port: int, db_pool: ConnectionPool) -> None:
    """Seperate wsgi server that handles metrics and ready endpoint"""
    metrics_app = make_wsgi_app()

    def app(environ, start_response):
        if environ["PATH_INFO"] == "/ready":
            try:
                with db_pool.connection() as conn:
                    conn.execute("SELECT 1")
                start_response("200 OK", [("Content-Type", "text/plain")])
                return [b"ok"]
            except Exception:
                start_response("503 Service Unavailable", [("Content-Type", "text/plain")])
                return [b"db not ready"]
        return metrics_app(environ, start_response)

    server = make_server("", port, app)
    thread = threading.Thread(target=server.serve_forever, daemon=True)
    thread.start()

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
    start_metrics_server(settings.metrics_server_port, db_pool)
    logger.info("metrics server listening", port=settings.metrics_server_port)

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
