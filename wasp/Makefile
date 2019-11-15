BINARY='baby-groot.wasm'
INDEX='assets/groot-files/index'

all: build index

dev: build server

build:
	npm run build --prefix assets/js/
	GOOS=js GOARCH=wasm CGO_ENABLED=1 go build -o ${BINARY} -ldflags "-s -w" ./cmd

index:
	go run ../main.go get -d arg-annot
	go run ../main.go index -m arg-annot* -i ${INDEX} -k 31 -s 30 -w 100
	rm -r arg-annot*

server:
	go run dev-server.go

clean:
	rm ${BINARY}
	rm -r ${INDEX}