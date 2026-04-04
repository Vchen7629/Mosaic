test_all: unit integration

PKGS := ../internal/handler/... ../internal/observability/... ../internal/service/...

coverage:
	go test -tags=unit,integration ${PKGS} -v -coverprofile=coverage.out -covermode=atomic

format: 
	go fmt ${PKGS}

integration:
	go test -tags integration ${PKGS}

unit:
	go test -tags unit ${PKGS}

race_test:
	go test -tags integration ../internal/handler/...
	set CGO_ENABLED=1 && go test -race -tags integration ../internal/service/...