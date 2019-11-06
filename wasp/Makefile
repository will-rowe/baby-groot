BINARY='baby-groot.wasm'

all: build index

dev: build index server

build:
	npm run build --prefix assets/js/
	GOOS=js GOARCH=wasm CGO_ENABLED=1 go build -o ${BINARY} -ldflags "-s -w" ./cmd

index:
	go build ../main.go
	./baby-groot get -d arg-annot
	

server:
	go run dev-server.go