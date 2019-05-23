// +build js,wasm

package bg

import (
	//"bufio"
	//"bytes"
	//"io"
	"reflect"
	"syscall/js"
	"unsafe"

	"github.com/will-rowe/baby-groot/src/pipeline"
)

// closeFASTQchan
func (s *GrootWASM) setupCloseFASTQchanCb() {
	s.closeFASTQchanCb = js.FuncOf(func(this js.Value, i []js.Value) interface{} {
		close(s.fastq)
		return nil
	})
	return	
}

// setupInitMem1Cb handles the memory for the fastq stream
func (s *GrootWASM) setupInitMem1Cb() {
	s.initMemCb = js.FuncOf(func(this js.Value, i []js.Value) interface{} {
		// create the buffer and a pointer for this chunk of fastq data
		length := i[0].Int()
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
	return
}

// setupOnIndexLoad is the callback to load the index
func (s *GrootWASM) setupOnIndexLoad() {
	s.indexLoaderCb = js.FuncOf(func(this js.Value, args []js.Value) interface{} {

		// check the index downloaded
		if len(s.inBuf2) == 0 {
			js.Global().Call("toggleDiv", "inputModal")
			s.statusUpdate("index file didn't download!")
			return nil
		}

		// read the INDEX
		s.info = new(pipeline.Info)
		if err := s.info.LoadFromBytes(s.inBuf2); err != nil {
			s.statusUpdate("does not look like a GROOT index!")
			return nil
		}

		// update the user
		js.Global().Call("setButtonColour", "indexLoader", "#80ff00")
		js.Global().Call("setButtonText", "indexLoader", "loaded!")

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

		return nil
	})
	return
}


// setupInputCheckerCb is the callback to check the input is correct
func (s *GrootWASM) setupInputCheckerCb() {
	s.inputCheckerCb = js.FuncOf(func(this js.Value, args []js.Value) interface{} {


		//TODO: check fastq data streamer is ready to go
		/*
				// check for the FASTQ file
				if len(s.inBuf1) == 0 {
					s.statusUpdate("no FASTQ file selected!")
					return nil
				}
				// read the FASTQ
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
*/


		js.Global().Call("toggleDiv", "inputModal")
		if s.info == nil {
			s.statusUpdate("index file didn't load!")
			return nil
		}

		s.iconUpdate("inputIcon")
		s.statusUpdate("input is set")
		s.inputCheck = true
		return nil
	})
	return
}