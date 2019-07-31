// +build js,wasm

package bg

import (
	"fmt"
	"syscall/js"
	"time"

	"../src/pipeline"
)

// setupGrootCb sets up the GROOT callback and runs GROOT when everything is set
func (s *GrootWASM) setupGrootCb() {
	s.grootCb = js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		go func() {
			if !s.inputCheck {
				s.statusUpdate("problem with input!")
				return
			}

			// stop GROOT?
			if s.running == true {
				s.running = false
				s.statusUpdate("stopped GROOT!")
				js.Global().Call("stopSpinner")
				js.Global().Call("stopLogo")
				return
			}

			// start GROOT notifications
			s.running = true
			s.statusUpdate("running GROOT...")
			js.Global().Call("startRecord")
			js.Global().Call("startLogo")
			startTime := time.Now()

			// set up the pipeline
			sketchingPipeline := pipeline.NewPipeline()
			wasmStreamer := pipeline.NewWASMstreamer()
			fastqHandler := pipeline.NewFastqHandler(s.info)
			fastqChecker := pipeline.NewFastqChecker(s.info)
			readMapper := pipeline.NewDbQuerier(s.info)
			graphPruner := pipeline.NewGraphPruner(s.info, true)
			emPathFinder := pipeline.NewEMpathFinder(s.info)
			haploParser := pipeline.NewHaplotypeParser(s.info)

			// connect the pipeline
			wasmStreamer.ConnectChan(s.fastq)
			fastqHandler.ConnectWASM(wasmStreamer)
			fastqChecker.Connect(fastqHandler)
			readMapper.Connect(fastqChecker)
			graphPruner.Connect(readMapper)
			emPathFinder.ConnectPruner(graphPruner)
			haploParser.Connect(emPathFinder)
			sketchingPipeline.AddProcesses(wasmStreamer, fastqHandler, fastqChecker, readMapper, graphPruner, emPathFinder, haploParser)

			// start the stream and send data to the pipeline
			go js.Global().Call("fastqStreamer", s.fastqFiles)

			// run the pipeline
			fmt.Println("starting the pipeline")
			sketchingPipeline.Run()
			fmt.Println("pipeline finished")

			// collect the output
			readStats := readMapper.CollectReadStats()
			//foundPaths := graphPruner.CollectOutput()
			//foundHaplotypes := haploParser.CollectOutput()
			fmt.Println("readStats: ", readStats)

			// print some updates
			if readStats[1] == 0 {
				js.Global().Call("stopRecord")
				js.Global().Call("stopLogo")
				s.iconUpdate("startIcon")
				s.statusUpdate("no reads mapped to graphs :(")
				return
			}
			s.statusUpdate(fmt.Sprintf("mapped reads = %d/%d", readStats[1], readStats[0]))

			// get the results
			s.results = false
			for _, g := range s.info.Store {
				paths, abundances := g.GetEMpaths()
				if len(paths) != 0 {
					s.results = true
					fmt.Printf("\tgraph %d has %d called alleles after EM", g.GraphID, len(paths))
					for i, path := range paths {
						js.Global().Call("addResults", path, abundances[i])
					}
				}
			}

			// report any results
			js.Global().Call("stopRecord")
			js.Global().Call("stopLogo")
			s.iconUpdate("startIcon")
			if s.results == false {
				s.statusUpdate("no results found :(")
			} else {
				s.statusUpdate("GROOT finished!")
				secs := time.Since(startTime).Seconds()
				mins := time.Since(startTime).Minutes()
				timer := fmt.Sprintf("%.0fmins %.0fsecs", mins, secs)
				js.Global().Call("updateTimer", timer)
				js.Global().Call("toggleDiv", "resultsModal")
			}
		}()
		return nil
	})
}

//
func (s *GrootWASM) printSeqs() {
	for _, g := range s.info.Store {
		seqs, err := g.Graph2Seqs()
		if err != nil {
			s.statusUpdate(fmt.Sprintf("%v", err))
		}
		for id, seq := range seqs {
			fmt.Printf(">%v\n%v\n", string(g.Paths[id]), string(seq))
		}
	}
}
