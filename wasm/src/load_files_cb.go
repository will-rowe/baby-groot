// +build js,wasm

package bg

import (
	"bytes"
	"reflect"
	"syscall/js"
	"unsafe"

	"github.com/segmentio/objconv/msgpack"
	"github.com/will-rowe/baby-groot/src/pipeline"
)

// setupInitMem1Cb gets the initial memory for the fastq file
func (s *GrootWASM) setupInitMem1Cb() {
	// The length of the image array buffer is passed.
	// Then the buf slice is initialized to that length.
	// And a pointer to that slice is passed back to the browser.
	s.initMemCb = js.FuncOf(func(this js.Value, i []js.Value) interface{} {
		length := i[0].Int()
		s.console.Call("log", "length:", length)
		s.inBuf1 = make([]uint8, length)
		hdr := (*reflect.SliceHeader)(unsafe.Pointer(&s.inBuf1))
		ptr := uintptr(unsafe.Pointer(hdr.Data))
		s.console.Call("log", "ptr:", ptr)
		js.Global().Call("gotMem1", ptr)
		return nil
	})
}

// setupInitMem2Cb gets the initial memory for the index file
func (s *GrootWASM) setupInitMem2Cb() {
	// The length of the image array buffer is passed.
	// Then the buf slice is initialized to that length.
	// And a pointer to that slice is passed back to the browser.
	s.initMem2Cb = js.FuncOf(func(this js.Value, i []js.Value) interface{} {
		length := i[0].Int()
		s.console.Call("log", "length:", length)
		s.inBuf2 = make([]uint8, length)
		hdr := (*reflect.SliceHeader)(unsafe.Pointer(&s.inBuf2))
		ptr := uintptr(unsafe.Pointer(hdr.Data))
		s.console.Call("log", "ptr:", ptr)
		js.Global().Call("gotMem2", ptr)
		return nil
	})
}

// setupOnFastqLoadCb is the callback to load the fastq file from the buffer
func (s *GrootWASM) setupOnFastqLoadCb() {
	s.onFastqLoadCb = js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		var b bytes.Buffer
		s.fastq = b
		// how to we want to use the buffer for sending fastq to groot?
		s.fastq.Write(s.inBuf1)
		s.statusUpdate("fastq file selected")
		s.iconUpdate("opener1")
		return nil
	})
}

// setupOnIndexLoadCb is the callback to load the index from the buffer
func (s *GrootWASM) setupOnIndexLoadCb() {
	s.onIndexLoadCb = js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		s.info = &pipeline.Info{}
		msgpack.Unmarshal(s.inBuf2, s.info)
		if len(s.info.DbDump) <= 1 {
			s.statusUpdate("does not look like a GROOT index!")
			return nil
		}
		s.statusUpdate("index file selected")
		s.iconUpdate("opener2")
		return nil
	})
}