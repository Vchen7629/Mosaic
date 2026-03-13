face_detection_gen:
	protoc -I . face_detection.proto \
			--go_out=../client/gen \
			--go-grpc_out=../client/gen \
			--go_opt=paths=source_relative \
			--go-grpc_opt=paths=source_relative
	uv run python -m grpc_tools.protoc -I . face_detection.proto \
			--python_out=../face_detection_service/gen \
			--grpc_python_out=../face_detection_service/gen

audio_transcription_gen: