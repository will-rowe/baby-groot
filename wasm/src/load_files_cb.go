// +build js,wasm

package bg

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"reflect"
	"sync"
	"syscall/js"
	"unsafe"

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
		js.Global().Call("gotMem", ptr)
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
		js.Global().Call("gotMem", ptr)
		return nil
	})
}

// setupOnFileSelectionCb is the callback to load the fastq and index files
func (s *GrootWASM) setupOnFileSelectionCb() {
	s.fileSelectionCb = js.FuncOf(func(this js.Value, args []js.Value) interface{} {

		// TODO: need a spinner for big files
		js.Global().Call("toggleDiv", "inputModal")
		s.statusUpdate("one moment please...")


		// check for the FASTQ file
		if len(s.inBuf1) == 0 {
			s.statusUpdate("no FASTQ file selected!")
			return nil
		}

		// check for the INDEX file
		if len(s.inBuf2) == 0 {
			s.statusUpdate("no INDEX file selected!")
			return nil
		}

		// read the FASTQ
		s.fastq = make(chan []byte)
		var wg sync.WaitGroup
		wg.Add(1)
		var b bytes.Buffer
		b.Write(s.inBuf1)
		reader := bufio.NewReader(&b)
		go func() {
			for {
				line, err := reader.ReadBytes('\n')
				if err != nil {
					if err == io.EOF {
						break
					} else {
						s.statusUpdate(fmt.Sprintf("%v\n", err))
					}
				}
				s.fastq <- append([]byte(nil), line...)
			}
			wg.Done()
		}()
		go func() {
			wg.Wait()
			close(s.fastq)
		}()

		// read the INDEX
		s.info = new(pipeline.Info)
		if err := s.info.LoadFromBytes(s.inBuf2); err != nil {
			s.statusUpdate("does not look like a GROOT index!")
			return nil
		}

		/////////////////////////////////////////////////
		// TODO: have these parameters set by the user
		s.info.Sketch = &pipeline.SketchCmd{
			MinKmerCoverage: 1,
			MinBaseCoverage: 1.0,
			BloomFilter:     false,
			Fasta:           false,
		}
		s.info.Haplotype = &pipeline.HaploCmd{
			Cutoff:        0.05,
			MaxIterations: 10000,
			MinIterations: 50,
			HaploDir:      ".",
		}
		/////////////////////////////////////////////////


		s.ready = true
		s.iconUpdate("inputIcon")
		s.statusUpdate("input is set")
		return nil
	})
}
