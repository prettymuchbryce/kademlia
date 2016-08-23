BIN ?= bin/${NAME}

all: ${BIN}

${BIN}:
	go build -o $@

run:
	go run *.go

clean:
	rm -f ${BIN}

debug:
	GORACE="history_size=7 halt_on_error=1" go run --race *.go

test:
	go test ./... -v -race -p 1

dev:
	go get github.com/skelterjohn/rerun
	rerun $(shell go list)

install:
	go get -t -v ./...
