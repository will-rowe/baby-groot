// +build js,wasm

package bg

import (
	"fmt"
	"syscall/js"
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
//TODO: need to check this is correct
func (GrootWASM *GrootWASM) getFiles(this js.Value, args []js.Value) interface{} {
	fmt.Printf("loading fastq file list...\n")
	files := make([]interface{}, len(args))
	for i, val := range args {
		files[i] = val
	}
	if len(files) != 0 {
		GrootWASM.fastqFiles = files
		fmt.Printf("\tfound %d files\n", len(files))
	} else {
		fmt.Println("no input files found")
	}
	return nil
}

// munchFASTQ is linked with the JavaScript function of the same name
func (GrootWASM *GrootWASM) munchFASTQ(this js.Value, args []js.Value) interface{} {
	fastqBuffer := make([]byte, args[1].Int())
	_ = js.CopyBytesToGo(fastqBuffer, args[0])
	GrootWASM.fastqInput <- fastqBuffer
	return nil
}

// closeFASTQchan
func (GrootWASM *GrootWASM) closeFASTQchan(this js.Value, i []js.Value) interface{} {
	fmt.Println("closing the FASTQ stream")
	close(GrootWASM.fastqInput)
	return nil
}
