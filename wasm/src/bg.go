// +build js,wasm

package bg

import (
	"syscall/js"

	"github.com/will-rowe/baby-groot/src/pipeline"
)

// GrootWASM
type GrootWASM struct {
	info  *pipeline.Info
	fastq chan []byte

	inBuf1     []uint8
	inBuf2     []uint8
	initMemCb  js.Func
	initMem2Cb js.Func

	indexLoaderCb   js.Func
	inputCheckerCb  js.Func
	grootCb       	js.Func
	closeFASTQchanCb	js.Func
	shutdownCb    	js.Func

	console js.Value
	done    chan struct{}
	inputCheck bool
	running bool
	results bool
}

// New returns a new instance of GrootWASM
func New() *GrootWASM {
	return &GrootWASM{
		console: js.Global().Get("console"),
		fastq: make(chan []byte, pipeline.BUFFERSIZE),
		done:    make(chan struct{}),
	}
}

// Start sets up all the callbacks and waits for the close signal to be sent from the browser.
func (s *GrootWASM) Start() {

	// the call back for the mem pointers
	s.setupInitMem1Cb()
	js.Global().Set("initFASTQmem", s.initMemCb)
	s.setupInitMem2Cb()
	js.Global().Set("initIndexMem", s.initMem2Cb)

	// the call back for downloading and loading the index
	s.setupOnIndexLoad()
	js.Global().Get("document").
		Call("getElementById", "indexLoader").
		Call("addEventListener", "click", s.indexLoaderCb)
	
	// the call back for checking the input
	s.setupInputCheckerCb()
	js.Global().Get("document").
		Call("getElementById", "inputCheck").
		Call("addEventListener", "click", s.inputCheckerCb)

	// the call back for running GROOT!
	s.setupGrootCb()
	js.Global().Get("document").
		Call("getElementById", "startIcon").
		Call("addEventListener", "click", s.grootCb)

	// the call back for closing the input channel
	s.setupCloseFASTQchanCb()
	js.Global().Set("closeFASTQchan", s.closeFASTQchanCb)

	// the call back for shutting down the app
	s.setupShutdownCb()
	js.Global().Get("document").
		Call("getElementById", "close").
		Call("addEventListener", "click", s.shutdownCb)

	<-s.done
	s.statusUpdate("Shutting down GROOT app...")
	s.indexLoaderCb.Release()
	s.inputCheckerCb.Release()
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
