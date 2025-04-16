
.PHONY: default
default: build

.PHONY: build
build:
	go build .

.PHONY: local-release
local-release:
	goreleaser build --snapshot --clean

.PHONY: test
test:
	go test -count=1 ./...
