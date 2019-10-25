// +build js,wasm

package bg

import (
	"syscall/js"

	"github.com/will-rowe/baby-groot/src/pipeline"
)

// GrootWASM
type GrootWASM struct {
	info       *pipeline.Info
	fastqFiles []interface{}
	fastq      chan []byte

	inBuf1 []uint8
	inBuf2 []uint8
	inBuf3 []uint8

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
		info:    new(pipeline.Info),
		console: js.Global().Get("console"),
		fastq:   make(chan []byte, pipeline.BUFFERSIZE),
		done:    make(chan struct{}),
	}
}

// Start sets up all the callbacks and waits for the close signal to be sent from the browser.
func (s *GrootWASM) Start() {
	defer s.releaseCallbacks()

	// the call backs for loading the data
	js.Global().Set("getFiles", js.FuncOf(s.getFiles))
	js.Global().Set("loadFASTQ", js.FuncOf(s.loadFASTQ))
	js.Global().Set("loadGraphs", js.FuncOf(s.loadGraphs))
	js.Global().Set("loadIndex", js.FuncOf(s.loadIndex))
	js.Global().Set("inputCheck", js.FuncOf(s.inputCheck))
	js.Global().Set("closeFASTQchan", js.FuncOf(s.closeFASTQchan))

	// set up the callbacks for start/stopping the GROOT app
	s.setupGrootCb()
	js.Global().Get("document").
		Call("getElementById", "startIcon").
		Call("addEventListener", "click", s.grootCb)

	s.setupShutdownCb()
	js.Global().Get("document").
		Call("getElementById", "close").
		Call("addEventListener", "click", s.shutdownCb)

	// use this blocking channel to wait keep the app alive until the shutdown is initiated
	<-s.done
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

// releaseCallbacks
func (s *GrootWASM) releaseCallbacks() {
	s.grootCb.Release()
	s.shutdownCb.Release()
	s.statusUpdate("GROOT has shut the app down!")
	s.iconUpdate("close")
}
