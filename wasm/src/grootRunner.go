// +build js,wasm

package bg

import (
    "bufio"
    "io"
    "fmt"
    "syscall/js"
    "sync"

    "github.com/will-rowe/baby-groot/src/pipeline"
)

// setupGrootCb sets up the GROOT callback and runs GROOT when everything is set
func (s *GrootWASM) setupGrootCb() {
	s.grootCb = js.FuncOf(func(this js.Value, args []js.Value) interface{} {

	// check files have been loaded
	//if s.fastq == nil {
	//	s.statusUpdate("please specify a fastq file!")
	//	return nil
	//}
	if s.info == nil {
		s.statusUpdate("please specify an index file!")
		return nil
    }
        
	// start or stop GROOT?
	if s.running == true {
		s.running = false
		s.statusUpdate("stopped GROOT!")
           js.Global().Call("stopSpinner")
           js.Global().Call("stopLogo")
		return nil
	}
	s.running = true
	s.statusUpdate("running GROOT...")
	js.Global().Call("startSpinner")
	js.Global().Call("startLogo")
	
	// call the method to run GROOT
	s.runGroot()

		return nil
	})
}

// runGroot runs GROOT sketch and haplotype
func (s *GrootWASM) runGroot() {
    // set up the pipeline
    sketchingPipeline := pipeline.NewPipeline()
    fastqHandler := pipeline.NewFastqHandler(s.info)
    fastqChecker := pipeline.NewFastqChecker(s.info)
    readMapper := pipeline.NewDbQuerier(s.info)
    graphPruner := pipeline.NewGraphPruner(s.info)

    // set up a channel to deliver the FASTQ data
    inputFastqData := make(chan []byte)

    // connect the pipeline
    fastqHandler.ConnectChan(inputFastqData)
    fastqChecker.Connect(fastqHandler)
    readMapper.Connect(fastqChecker)
    graphPruner.Connect(readMapper)
    sketchingPipeline.AddProcesses(fastqHandler, fastqChecker, readMapper, graphPruner)

    // feed in the data
    reader := bufio.NewReader(&s.fastq)
    var wg sync.WaitGroup
    go func() {
        wg.Add(1)
        for {
            line, err := reader.ReadBytes('\n')
            if err != nil {
                if err == io.EOF {
                    break
                } else {
                    fmt.Println(err)
                }
            }
            fmt.Println(string(line))
            inputFastqData <- append([]byte(nil), line...)
        }
        wg.Done()
    }()
    go func() {
        wg.Wait()
        close(inputFastqData)
    }()

    sketchingPipeline.Run()

    // check that the right number of reads mapped
    readStats := readMapper.CollectReadStats()
    s.console.Call("log", "total number of test reads = ", readStats[0])
    s.console.Call("log", "number which mapped = ", readStats[1])

    // print the paths
    foundPaths := graphPruner.CollectOutput()
    for _, path := range foundPaths {
        fmt.Print(path)
	}


}