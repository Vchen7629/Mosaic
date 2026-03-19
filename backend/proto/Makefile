all: face_detection_gen audio_transcription_gen conversation_briefing_gen

face_detection_gen:
	protoc -I face_detection face_detection/face_detection.proto \
			--go_out=../client/gen/face_detection \
			--go-grpc_out=../client/gen/face_detection \
			--go_opt=paths=source_relative \
			--go-grpc_opt=paths=source_relative
	protoc -I face_detection face_detection/face_detection.proto \
		--go_out=../face_detection/gen \
		--go_opt=Mface_detection.proto=mosaic-face-detection.com/gen \
		--go_opt=paths=source_relative \
		--go-grpc_out=../face_detection/gen \
		--go-grpc_opt=Mface_detection.proto=mosaic-face-detection.com/gen \
		--go-grpc_opt=paths=source_relative

audio_transcription_gen:
	protoc -I audio_transcription audio_transcription/audio_transcription.proto \
			--go_out=../client/gen/audio_transcription \
			--go-grpc_out=../client/gen/audio_transcription \
			--go_opt=paths=source_relative \
			--go-grpc_opt=paths=source_relative
	uv run python -m grpc_tools.protoc -I audio_transcription audio_transcription/audio_transcription.proto \
			--python_out=../audio_transcription/src/gen \
			--grpc_python_out=../audio_transcription/src/gen \
			--mypy_out=../audio_transcription/src/gen \
			--mypy_grpc_out=../audio_transcription/src/gen
	uv run python -c "f='../audio_transcription/src/gen/audio_transcription_pb2_grpc.py'; content=open(f).read().replace('import audio_transcription_pb2 as', 'from . import audio_transcription_pb2 as'); open(f,'w').write(content)"

conversation_briefing_gen:
	protoc -I conversation_briefing conversation_briefing/conversation_briefing.proto \
			--go_out=../client/gen/conversation_briefing \
			--go-grpc_out=../client/gen/conversation_briefing \
			--go_opt=paths=source_relative \
			--go-grpc_opt=paths=source_relative
	protoc -I conversation_briefing conversation_briefing/conversation_briefing.proto \
		--go_out=../conversation_briefing/gen \
		--go_opt=Mconversation_briefing.proto=mosaic-conversation-briefing.com/gen \
		--go_opt=paths=source_relative \
		--go-grpc_out=../conversation_briefing/gen \
		--go-grpc_opt=Mconversation_briefing.proto=mosaic-conversation-briefing.com/gen \
		--go-grpc_opt=paths=source_relative