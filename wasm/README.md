# WASP

This is the Web ASembly Port for BABY-GROOT.

> this is work in progress...

To check out the current web version that is produced by this repo, go to [https://will-rowe.github.io/baby-groot](https://will-rowe.github.io/baby-groot/wasm/index.html)

Browser support is restricted to Chrome, Firefox and Opera for now.

## Issues

* can't load large index
  * runs out of memory when allocating
  * a 150Mb groot index takes less than 800Mb RAM to load from disk, yet this is resulting in memalloc failure in WASM
* premature terminations aren't graceful
  * they just print to the console, no notifications for user
  * in some cases, the application doesn't shut down
* doesn't handle GZIPed input yet
* sometimes it doesn't progress past the read mapping - not sure what causes this and no errors detected (all test reads map and graphs are updated)

## Running locally

To build the WASM binary and run the development server:

``` bash
make

go run dev-server.go
```
