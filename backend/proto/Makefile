all: face_detection_gen audio_transcription_gen

face_detection_gen:
	protoc -I face_detection face_detection/face_detection.proto \
			--go_out=../client/gen/face_detection \
			--go-grpc_out=../client/gen/face_detection \
			--go_opt=paths=source_relative \
			--go-grpc_opt=paths=source_relative
	uv run python -m grpc_tools.protoc -I face_detection face_detection/face_detection.proto \
			--python_out=../face_detection_service/gen \
			--grpc_python_out=../face_detection_service/gen

audio_transcription_gen:
	protoc -I audio_transcription audio_transcription/audio_transcription.proto \
			--go_out=../client/gen/audio_transcription \
			--go-grpc_out=../client/gen/audio_transcription \
			--go_opt=paths=source_relative \
			--go-grpc_opt=paths=source_relative
	uv run python -m grpc_tools.protoc -I audio_transcription audio_transcription/audio_transcription.proto \
			--python_out=../audio_transcription/gen \
			--grpc_python_out=../audio_transcription/gen