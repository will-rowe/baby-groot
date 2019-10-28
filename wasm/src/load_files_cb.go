// +build js,wasm

package bg

import (
	"fmt"
	"syscall/js"

	"github.com/will-rowe/baby-groot/src/lshforest"
	"github.com/will-rowe/baby-groot/src/pipeline"
)

// loadGraphs is linked with the JavaScript function of the same name
func (GrootWASM *GrootWASM) loadGraphs(this js.Value, args []js.Value) interface{} {
	fmt.Printf("loading graphs...\n")
	fmt.Printf("\tname: %v\n", args[0])
	fmt.Printf("\tsize: %d\n", args[2].Int())
	GrootWASM.graphBuffer = make([]byte, args[2].Int())
	size := js.CopyBytesToGo(GrootWASM.graphBuffer, args[1])
	if size != 0 {
		fmt.Printf("\tdone\n")
	}
	return nil
}

// loadIndex is linked with the JavaScript function of the same name
func (GrootWASM *GrootWASM) loadIndex(this js.Value, args []js.Value) interface{} {
	fmt.Printf("loading index...\n")
	fmt.Printf("\tname: %v\n", args[0])
	fmt.Printf("\tsize: %d\n", args[2].Int())
	GrootWASM.indexBuffer = make([]byte, args[2].Int())
	size := js.CopyBytesToGo(GrootWASM.indexBuffer, args[1])
	if size != 0 {
		fmt.Printf("\tdone\n")
	}
	return nil
}

// getFiles is linked with the JavaScript function of the same name
func (GrootWASM *GrootWASM) getFiles(this js.Value, args []js.Value) interface{} {
	fmt.Printf("loading fastq file list...\n")
	files := make([]interface{}, len(args))
	for i, val := range args {
		files[i] = val
	}
	if len(files) != 0 {
		GrootWASM.fastqFiles = files
		fmt.Printf("\tfound %d files\n\tdone\n", len(files))
	} else {
		fmt.Println("no input files found")
	}
	return nil
}

// setupInputCheckerCb is the callback to check the input is correct
func (GrootWASM *GrootWASM) inputCheck(this js.Value, args []js.Value) interface{} {
	js.Global().Get("document").
		Call("getElementById", "spinner").
		Call("setAttribute", "hidden", "")

	// check the input first
	if len(GrootWASM.fastqFiles) == 0 {
		GrootWASM.statusUpdate("no FASTQ files selected!")
		return nil
	}
	if len(GrootWASM.graphBuffer) == 0 {
		GrootWASM.statusUpdate("can't find graphs")
		return nil
	}
	if len(GrootWASM.indexBuffer) == 0 {
		GrootWASM.statusUpdate("can't find index")
		return nil
	}

	// read the index files
	if err := GrootWASM.info.LoadFromBytes(GrootWASM.graphBuffer); err != nil {
		GrootWASM.statusUpdate("failed to load GROOT graphs!")
		fmt.Println(err)
		return nil
	}
	lshf := lshforest.NewLSHforest(GrootWASM.info.SketchSize, GrootWASM.info.JSthresh)
	if err := lshf.LoadFromBytes(GrootWASM.indexBuffer); err != nil {
		GrootWASM.statusUpdate("failed to load GROOT index!")
		fmt.Println(err)
		return nil
	}

	// set the number of available processors to 1
	GrootWASM.info.NumProc = 1
	GrootWASM.info.AttachDB(lshf)
	fmt.Println("input checked - GROOT happy to launch")

	/////////////////////////////////////////////////
	// TODO: have these parameters set by the user
	GrootWASM.info.Sketch = pipeline.SketchCmd{
		MinKmerCoverage: 1.0,
		BloomFilter:     false,
		Fasta:           false,
	}
	GrootWASM.info.Haplotype = pipeline.HaploCmd{
		Cutoff:        0.001,
		MaxIterations: 10000,
		MinIterations: 50,
		HaploDir:      ".",
	}
	/////////////////////////////////////////////////

	if GrootWASM.info == nil {
		GrootWASM.statusUpdate("index didn't load!")
		return nil
	}
	GrootWASM.inputChecked = true
	GrootWASM.iconUpdate("inputIcon")
	GrootWASM.statusUpdate("input is set")
	return nil
}

// munchFASTQ is linked with the JavaScript function of the same name
func (GrootWASM *GrootWASM) munchFASTQ(this js.Value, args []js.Value) interface{} {
	fastqBuffer := make([]byte, args[2].Int())
	_ = js.CopyBytesToGo(fastqBuffer, args[1])
	GrootWASM.fastqInput <- fastqBuffer
	return nil
}

// closeFASTQchan
func (GrootWASM *GrootWASM) closeFASTQchan(this js.Value, i []js.Value) interface{} {
	fmt.Println("closing the FASTQ stream")
	close(GrootWASM.fastqInput)
	return nil
}
