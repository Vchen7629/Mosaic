build: 
test_all: integration unit

lint: 
	go fmt . 
	go fmt ../internal/db/... .
	go fmt ../internal/handler/... .
	go fmt ../internal/observability/... .
	go fmt ../internal/service/... .

integration:
	go test -tags integration ../internal/db/...
	go test -tags integration ../internal/handler/...

unit:
	go test -tags unit ../internal/db/...
	go test -tags unit ../internal/handler/...
	go test -tags unit ../internal/service/...