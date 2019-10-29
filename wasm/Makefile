BINARY='baby-groot.wasm'

all: build

build:
	npm run build --prefix  assets/js/
	GOOS=js GOARCH=wasm CGO_ENABLED=1 go build -o ${BINARY} -ldflags "-s -w" ./cmd/bg