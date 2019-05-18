// +build js,wasm

package bg

import (
	"bytes"
	"image"
	"reflect"
	"syscall/js"
	"time"
	"unsafe"

	"github.com/will-rowe/baby-groot/src/pipeline"
	"github.com/anthonynsimon/bild/imgio"
)

// GrootWASM
type GrootWASM struct {
	info *pipeline.Info
	fastq bytes.Buffer
	inBuf1                             []uint8
	inBuf2                             []uint8
	initMemCb           			   js.Func
	initMem2Cb           			   js.Func


	outBuf                             bytes.Buffer /// delete


	onFastqLoadCb					   js.Func
	onIndexLoadCb					   js.Func
	grootCb                            js.Func
	shutdownCb  					   js.Func

	console 							js.Value
	done    							chan struct{}
	running 							bool
}

// New returns a new instance of GrootWASM
func New() *GrootWASM {
	return &GrootWASM{
		console: js.Global().Get("console"),
		done:    make(chan struct{}),
	}
}

// Start sets up all the callbacks and waits for the close signal to be sent from the browser.
func (s *GrootWASM) Start() {

	// the call back for the mem pointers
	s.setupInitMem1Cb()
	js.Global().Set("initMem1", s.initMemCb)
	s.setupInitMem2Cb()
	js.Global().Set("initMem2", s.initMem2Cb)

	// the call back for loading the fastq file
	s.setupOnFastqLoadCb()
	js.Global().Set("loadFastq", s.onFastqLoadCb)

	// the call back for loading the index file
	s.setupOnIndexLoadCb()
	js.Global().Set("loadIndex", s.onIndexLoadCb)


	// the call back for running GROOT!
	s.setupGrootCb()
	js.Global().Get("document").
		Call("getElementById", "start").
		Call("addEventListener", "click", s.grootCb)

	// the call back for shutting down the app
	s.setupShutdownCb()
	js.Global().Get("document").
		Call("getElementById", "close").
		Call("addEventListener", "click", s.shutdownCb)

	<-s.done
	s.statusUpdate("Shutting down GROOT app...")
	s.onFastqLoadCb.Release()
	s.onIndexLoadCb.Release()
	s.grootCb.Release()
	s.shutdownCb.Release()
	s.statusUpdate("GROOT has shut the app down!")
	s.iconUpdate("close")
}

// setupShutdownCb
func (s *GrootWASM) setupShutdownCb() {
	s.shutdownCb = js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		s.done <- struct{}{}
		return nil
	})
}

// statusUpdate calls the statusUpdate javascript function, which prints a message to the webpage
func (s *GrootWASM) statusUpdate(msg string) {
	js.Global().Call("statusUpdate", "status", msg)
}

// iconUpdate calls the iconUpdate javascript function, which changes an icon to a tick
func (s *GrootWASM) iconUpdate(icon string) {
	js.Global().Call("iconUpdate", icon)
}


// updateImage writes the image to a byte buffer and then converts it to base64.
// Then it sets the value to the src attribute of the target image.
func (s *GrootWASM) updateImage(img *image.RGBA, start time.Time) {
	enc := imgio.JPEGEncoder(90)
	err := enc(&s.outBuf, img)
	if err != nil {
		s.statusUpdate(err.Error())
		return
	}

	out := s.outBuf.Bytes()
	hdr := (*reflect.SliceHeader)(unsafe.Pointer(&out))
	ptr := uintptr(unsafe.Pointer(hdr.Data))
	js.Global().Call("displayImage", ptr, len(out))
	s.console.Call("statusUpdate", "time taken:", time.Now().Sub(start).String())
	s.outBuf.Reset()
}
