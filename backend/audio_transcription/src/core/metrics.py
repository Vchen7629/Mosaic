import os
import psutil
from prometheus_client import Counter, Gauge, Histogram

_process = psutil.Process(os.getpid())

process_cpu_seconds_total = Gauge(
    "process_cpu_service_seconds_total",
    "Total user and system CPU time spent in seconds",
)

process_resident_memory_bytes = Gauge(
    "process_resident_service_memory_bytes",
    "Resident memory size in bytes",
)

WhisperDuration = Histogram(
    "audio_transcription_whisper_duration_milliseconds",
    "Time to transcribe an audio chunk using Whisper",
    buckets=[5, 10, 25, 50, 100, 250, 500, 1000, 2000, 5000],
)

ConvoSaveDuration = Histogram(
    "audio_transcription_db_save_duration_milliseconds",
    "Time to save a conversation to DB",
    buckets=[5, 10, 25, 50, 100, 250, 500, 1000, 2000, 5000],
)

DBWritesTotal = Counter(
    "audio_transcription_db_writes_total",
    "Total DB write operations by Operation",
    ["operation"],
)

ErrorsTotal = Counter(
    "audio_transcription_errors_total",
    "Total errors by Operation",
    ["operation"],
)

TranscribeQueueDepth = Gauge(
    "audio_transcription_queue_depth",
    "Number of audio chunks waiting to be transcribed",
)

