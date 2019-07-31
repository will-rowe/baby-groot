// +build js,wasm

package bg

import (
	"fmt"
	"reflect"
	"syscall/js"
	"unsafe"

	"github.com/will-rowe/baby-groot/src/lshforest"
	"github.com/will-rowe/baby-groot/src/pipeline"
)

// closeFASTQchan
func (s *GrootWASM) setupCloseFASTQchanCb() {
	s.closeFASTQchanCb = js.FuncOf(func(this js.Value, i []js.Value) interface{} {
		fmt.Println("closing FASTQ stream")
		close(s.fastq)
		return nil
	})
	return
}

// setupInitMem1Cb handles the memory for the fastq stream
func (s *GrootWASM) setupInitMem1Cb() {
	s.initMemCb = js.FuncOf(func(this js.Value, args []js.Value) interface{} {

		// create the buffer and a pointer for this chunk of fastq data
		length := args[0].Int()
		fastqBuffer := make([]uint8, length)
		hdr := (*reflect.SliceHeader)(unsafe.Pointer(&fastqBuffer))
		ptr := uintptr(unsafe.Pointer(hdr.Data))

		// link the pointer with the data
		js.Global().Call("gotMem", ptr)

		// send the chunk of fastq data on to GROOT
		s.fastq <- fastqBuffer
		return nil
	})
	return
}

// setupInitMem2Cb gets the initial memory for the graphs file
func (s *GrootWASM) setupInitMem2Cb() {

	// The length of the array buffer is passed.
	// Then the buf slice is initialized to that length.
	// And a pointer to that slice is passed back to the browser.
	s.initMem2Cb = js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		length := args[0].Int()
		s.console.Call("log", "initialising memory for the GROOT graphs (file size: ", length, ")")
		s.inBuf2 = make([]uint8, length)
		hdr := (*reflect.SliceHeader)(unsafe.Pointer(&s.inBuf2))
		ptr := uintptr(unsafe.Pointer(hdr.Data))
		s.console.Call("log", "setting web assembly linear memory for gaphs")
		js.Global().Call("gotMem", ptr)
		return nil
	})
	return
}

// setupInitMem3Cb gets the initial memory for the index file
func (s *GrootWASM) setupInitMem3Cb() {

	// The length of the array buffer is passed.
	// Then the buf slice is initialized to that length.
	// And a pointer to that slice is passed back to the browser.
	s.initMem3Cb = js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		length := args[0].Int()
		s.console.Call("log", "initialising memory for the GROOT index (file size: ", length, ")")
		s.inBuf3 = make([]uint8, length)
		hdr := (*reflect.SliceHeader)(unsafe.Pointer(&s.inBuf3))
		ptr := uintptr(unsafe.Pointer(hdr.Data))
		s.console.Call("log", "setting web assembly linear memory for lsh forest")
		js.Global().Call("gotMem", ptr)
		return nil
	})
	return
}

// setupFastqFiles is the callback to get a list of FASTQs for GROOT
func (s *GrootWASM) setupFastqFiles() {
	s.fastqFilesCb = js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		files := make([]interface{}, len(args))
		for i, val := range args {
			files[i] = val
		}
		if len(files) != 0 {
			s.fastqFiles = files
			fmt.Println("fastq files ready for streaming")
		} else {
			fmt.Println("no input files found")
		}
		return nil
	})
	return
}

// setupInputCheckerCb is the callback to check the input is correct
func (s *GrootWASM) setupInputCheckerCb() {
	s.inputCheckerCb = js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		js.Global().Get("document").
			Call("getElementById", "spinner").
			Call("setAttribute", "hidden", "")

		// check the input first
		if len(s.fastqFiles) == 0 {
			s.statusUpdate("no FASTQ files selected!")
			return nil
		}
		if len(s.inBuf2) == 0 {
			s.statusUpdate("can't find graphs")
			return nil
		}
		if len(s.inBuf3) == 0 {
			s.statusUpdate("can't find index")
			return nil
		}

		// read the index files
		if err := s.info.LoadFromBytes(s.inBuf2); err != nil {
			s.statusUpdate("failed to load GROOT graphs!")
			fmt.Println(err)
			return nil
		}
		lshf := lshforest.NewLSHforest(s.info.SketchSize, s.info.JSthresh)
		if err := lshf.LoadFromBytes(s.inBuf3); err != nil {
			s.statusUpdate("failed to load GROOT index!")
			fmt.Println(err)
			return nil
		}

		// set the number of available processors to 1
		s.info.NumProc = 1
		s.info.AttachDB(lshf)
		fmt.Println("index loaded")

		/////////////////////////////////////////////////
		// TODO: have these parameters set by the user
		s.info.Sketch = pipeline.SketchCmd{
			MinKmerCoverage: 1.0,
			BloomFilter:     false,
			Fasta:           false,
		}
		s.info.Haplotype = pipeline.HaploCmd{
			Cutoff:        0.001,
			MaxIterations: 10000,
			MinIterations: 50,
			HaploDir:      ".",
		}
		/////////////////////////////////////////////////

		if s.info == nil {
			s.statusUpdate("index didn't load!")
			return nil
		}

		s.inputCheck = true
		s.iconUpdate("inputIcon")
		s.statusUpdate("input is set")
		return nil
	})
	return
}
