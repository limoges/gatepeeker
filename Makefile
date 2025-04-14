
.PHONY: default
default: build

.PHONY: build
build:
	go build .

.PHONY: test
test:
	go test -count=1 ./...
