all: integration unit

integration:
	go test -tags integration ../internal/db/...

unit:
	go test -tags unit ../internal/db/...