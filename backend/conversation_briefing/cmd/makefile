build: 
test_all: integration unit

PKGS := ../internal/db/... ../internal/handler/... ../internal/service/...

coverage:
	go test -tags=unit,integration ${PKGS} -v -coverprofile=coverage.out -covermode=atomic

lint: 
	go fmt . 
	go fmt ${PKGS} .
	go fmt ../internal/observability/... .

integration:
	go test -tags integration ${PKGS}

unit:
	go test -tags unit ${PKGS}