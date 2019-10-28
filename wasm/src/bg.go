// +build js,wasm

package bg

import (
	"syscall/js"

	"github.com/will-rowe/baby-groot/src/pipeline"
)

// GrootWASM
type GrootWASM struct {
	info        *pipeline.Info
	fastqFiles  []interface{}
	fastqInput  chan []byte
	graphBuffer []uint8
	indexBuffer []uint8

	grootCb    js.Func
	shutdownCb js.Func

	console      js.Value
	done         chan struct{}
	inputChecked bool
	running      bool
	results      bool
}

// New returns a new instance of GrootWASM
func New() *GrootWASM {
	return &GrootWASM{
		info:       new(pipeline.Info),
		console:    js.Global().Get("console"),
		fastqInput: make(chan []byte, pipeline.BUFFERSIZE),
		done:       make(chan struct{}),
	}
}

// Start sets up all the callbacks and waits for the close signal to be sent from the browser.
func (GrootWASM *GrootWASM) Start() {
	defer GrootWASM.releaseCallbacks()

	// the call backs for loading the data
	js.Global().Set("getFiles", js.FuncOf(GrootWASM.getFiles))
	js.Global().Set("munchFASTQ", js.FuncOf(GrootWASM.munchFASTQ))
	js.Global().Set("loadGraphs", js.FuncOf(GrootWASM.loadGraphs))
	js.Global().Set("loadIndex", js.FuncOf(GrootWASM.loadIndex))
	js.Global().Set("inputCheck", js.FuncOf(GrootWASM.inputCheck))
	js.Global().Set("closeFASTQchan", js.FuncOf(GrootWASM.closeFASTQchan))

	// set up the callbacks for start/stopping the GROOT app
	GrootWASM.setupGrootCb()
	js.Global().Get("document").
		Call("getElementById", "startIcon").
		Call("addEventListener", "click", GrootWASM.grootCb)

	GrootWASM.setupShutdownCb()
	js.Global().Get("document").
		Call("getElementById", "close").
		Call("addEventListener", "click", GrootWASM.shutdownCb)

	// use this blocking channel to keep the app alive until the shutdown is initiated
	<-GrootWASM.done
}

// setupShutdownCb
func (GrootWASM *GrootWASM) setupShutdownCb() {
	GrootWASM.shutdownCb = js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		GrootWASM.done <- struct{}{}
		return nil
	})
}

// statusUpdate calls the statusUpdate javascript function, which prints a message to the webpage
func (GrootWASM *GrootWASM) statusUpdate(msg string) {
	js.Global().Call("statusUpdate", "status", msg)
}

// iconUpdate calls the iconUpdate javascript function, which changes an icon to a tick
func (GrootWASM *GrootWASM) iconUpdate(icon string) {
	js.Global().Call("iconUpdate", icon)
}

// releaseCallbacks
func (GrootWASM *GrootWASM) releaseCallbacks() {
	GrootWASM.grootCb.Release()
	GrootWASM.shutdownCb.Release()
	GrootWASM.statusUpdate("GROOT has shut the app down!")
	GrootWASM.iconUpdate("close")
}
