all: 
test_all: integration unit

PKGS := ../internal/db/... ../internal/handler/... ../internal/service/...

lint: 
	go fmt . 
	go fmt ${PKGS} .
	go fmt ../internal/observability/... .
	go fmt ../internal/test/... .
	
coverage:
	go test -tags=unit,integration ${PKGS} -v -coverprofile=coverage.out -covermode=atomic

integration:
	go test -tags integration ${PKGS}

unit:
	go test -tags unit ${PKGS}