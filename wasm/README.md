# WASP

This is the Web ASembly Port for BABY-GROOT.

> this is work in progress...

To check out the current web version that is produced by this repo, go to [https://will-rowe.github.io/baby-groot](https://will-rowe.github.io/baby-groot/wasm/index.html)

Browser support is restricted to Chrome, Firefox and Opera for now.

## Issues

* can't load large index
  * runs out of memory when allocating
* premature terminations aren't graceful
  * they just print to the console, no notifications for user
  * in some cases, the application doesn't shut down
* doesn't handle GZIPed input yet
* can't handle multiple read files


## Running locally

To build the WASM binary and run the development server:

``` bash
make

go run dev-server.go
```

## Commands used to build index

```
cd assets/groot-files

go run ../../../main.go index -m . -i . -k 31 -s 100 -j 1.0 -w 100

```
