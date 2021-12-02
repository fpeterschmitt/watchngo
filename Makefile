.PHONY: release
release:
	echo build release watchngo
	go build -o watchngo -ldflags="-s -w" ./cmd/watchngo/

.PHONY: dev
dev:
	echo build dev watchngo
	go build -o watchngo.dev ./cmd/watchngo/

.PHONY: install
install:
	echo install watchngo at ${GOPATH}/bin/watchngo
	go install -ldflags="-s -w" ./cmd/watchngo/

.PHONY: vet
vet:
	go vet ./...
