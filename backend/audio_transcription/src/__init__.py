import sys
import os

sys.path.insert(0, os.path.join(os.path.dirname(__file__), '..', 'gen'))

from gen import audio_transcription_pb2, audio_transcription_pb2_grpc
