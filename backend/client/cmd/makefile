test_all: unit integration

PKGS := ../internal/handler/... ../internal/middleware/... ../internal/service/...

coverage:
	go test -tags=unit,integration ${PKGS} -v -coverprofile=coverage.out -covermode=atomic

lint: 
	go fmt . 
	go fmt ../internal/db/... .
	go fmt ../internal/handler/... .
	go fmt ../internal/observability/... .
	go fmt ../internal/service/... .

integration:
	go test -tags integration ${PKGS}

unit:
	go test -tags unit ${PKGS}

race_test:
	go test -tags integration ../internal/handler/...
	set CGO_ENABLED=1 && go test -race -tags integration ../internal/service/...