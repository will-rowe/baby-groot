// +build js,wasm

package bg

import (
	"fmt"
	"syscall/js"

	"github.com/will-rowe/baby-groot/src/lshforest"
	"github.com/will-rowe/baby-groot/src/pipeline"
)

// loadGraphs is linked with the JavaScript function of the same name
func (s *GrootWASM) loadGraphs(this js.Value, args []js.Value) interface{} {
	fmt.Printf("loading graphs...\n")
	fmt.Printf("\tname: %v\n", args[0])
	fmt.Printf("\tsize: %d\n", args[2].Int())
	s.inBuf2 = make([]byte, args[2].Int())
	size := js.CopyBytesToGo(s.inBuf2, args[1])
	if size != 0 {
		fmt.Printf("\tdone\n")
	}
	return nil
}

// loadIndex is linked with the JavaScript function of the same name
func (s *GrootWASM) loadIndex(this js.Value, args []js.Value) interface{} {
	fmt.Printf("loading index...\n")
	fmt.Printf("\tname: %v\n", args[0])
	fmt.Printf("\tsize: %d\n", args[2].Int())
	s.inBuf3 = make([]byte, args[2].Int())
	size := js.CopyBytesToGo(s.inBuf3, args[1])
	if size != 0 {
		fmt.Printf("\tdone\n")
	}
	return nil
}

// getFiles is linked with the JavaScript function of the same name
func (s *GrootWASM) getFiles(this js.Value, args []js.Value) interface{} {
	fmt.Printf("loading fastq file list...\n")
	files := make([]interface{}, len(args))
	for i, val := range args {
		files[i] = val
	}
	if len(files) != 0 {
		s.fastqFiles = files
		fmt.Printf("\tfound %d files\n\tdone\n", len(files))
	} else {
		fmt.Println("no input files found")
	}
	return nil
}

// setupInputCheckerCb is the callback to check the input is correct
func (s *GrootWASM) inputCheck(this js.Value, args []js.Value) interface{} {
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
	fmt.Println("input checked - GROOT happy to launch")

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
	s.inputChecked = true
	s.iconUpdate("inputIcon")
	s.statusUpdate("input is set")
	return nil
}

// loadFASTQ is linked with the JavaScript function of the same name
func (s *GrootWASM) loadFASTQ(this js.Value, args []js.Value) interface{} {
	fmt.Printf("reading a fastq\n")
	fastqBuffer := make([]byte, args[2].Int())
	_ = js.CopyBytesToGo(fastqBuffer, args[1])
	fmt.Println(string(fastqBuffer))
	s.fastq <- fastqBuffer
	return nil
}

// closeFASTQchan
func (s *GrootWASM) closeFASTQchan(this js.Value, i []js.Value) interface{} {
	fmt.Println("closing the FASTQ stream")
	close(s.fastq)
	return nil
}
