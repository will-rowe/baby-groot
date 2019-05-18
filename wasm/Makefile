BINARY='bg.wasm'

all: build

build:
	GOOS=js GOARCH=wasm CGO_ENABLED=1 go build -o ${BINARY} -ldflags "-s -w" ./cmd/bg
