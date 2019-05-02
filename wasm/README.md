# WASM

This is a place to test out WASM for GROOT

I'm starting by trying the [WebAssembly Framework](https://github.com/maxence-charriere/app) by Maxence Charriere - which looks awesome.

```bash
# Get the goapp CLI tool:
go get -u github.com/maxence-charriere/app/cmd/goapp

# Builds a server ready to serve the wasm app and its resources:
goapp build -v

# Launches the server and app in the default browser:
goapp run -v -b default

# Clean up when done testing:
goapp clean -v
```