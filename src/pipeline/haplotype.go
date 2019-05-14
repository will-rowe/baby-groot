package pipeline

/*
 this part of the pipeline will read in the weighted graphs from the sketch command, find paths in them and return the haplotypes
*/

import (
	"fmt"
	"log"
	"os"
	"sync"

	"github.com/will-rowe/baby-groot/src/graph"
	"github.com/will-rowe/baby-groot/src/misc"
	"github.com/will-rowe/gfa"
	"github.com/will-rowe/hulk/src/version"
)

// GFAreader is a pipeline process that reads in the weighted GFAs
type GFAreader struct {
	info   *Info
	input  []string
	output chan *graph.GrootGraph
}

// NewGFAreader is the constructor
func NewGFAreader(info *Info) *GFAreader {
	return &GFAreader{info: info, output: make(chan *graph.GrootGraph, BUFFERSIZE)}
}

// Connect is the method to connect the GFAreader to some data source
func (proc *GFAreader) Connect(input []string) {
	proc.input = input
}

// Run is the method to run this process, which satisfies the pipeline interface
func (proc *GFAreader) Run() {
	var wg sync.WaitGroup
	for i, gfaFile := range proc.input {
		gfaObj, err := graph.LoadGFA(gfaFile)
		misc.ErrorCheck(err)
		wg.Add(1)
		go func(gfaID int, g *gfa.GFA) {
			defer wg.Done()
			grootGraph, err := graph.CreateGrootGraph(g, gfaID)
			if err != nil {
				log.Fatal(err)
			}
			proc.output <- grootGraph
		}(i, gfaObj)
	}
	wg.Wait()
	close(proc.output)
}

// EMpathFinder is a pipeline process to identify graph paths using Expectation Maximization
type EMpathFinder struct {
	info   *Info
	input  chan *graph.GrootGraph
	output chan *graph.GrootGraph
}

// NewEMpathFinder is the constructor
func NewEMpathFinder(info *Info) *EMpathFinder {
	return &EMpathFinder{info: info, output: make(chan *graph.GrootGraph)}
}

// Connect is the method to connect the MCMCpathFinder to the output of a GFAreader
func (proc *EMpathFinder) Connect(previous *GFAreader) {
	proc.input = previous.output
}

// Run is the method to run this process, which satisfies the pipeline interface
func (proc *EMpathFinder) Run() {
	var wg sync.WaitGroup

	// collect the weighted graphs
	for inputGraph := range proc.input {
		wg.Add(1)

		// add each graph to the graphStore
		proc.info.Store[inputGraph.GraphID] = inputGraph

		// concurrently process the graphs
		go func(g *graph.GrootGraph) {
			defer wg.Done()

			// run the EM
			err := g.RunEM()
			misc.ErrorCheck(err)

			// process the EM results
			err = g.ProcessEMpaths(proc.info.Haplotype.Cutoff, proc.info.Sketch.TotalKmers)

			// send the graph to the next process
			proc.output <- g
		}(inputGraph)
	}
	wg.Wait()
	close(proc.output)
}

/*
// MCMCpathFinder is a pipeline process to identify graph paths using MCMC
type MCMCpathFinder struct {
	info   *Info
	input  chan *graph.GrootGraph
	output chan *graph.GrootGraph
}

// NewMCMCpathFinder is the constructor
func NewMCMCpathFinder(info *Info) *MCMCpathFinder {
	return &MCMCpathFinder{info: info, output: make(chan *graph.GrootGraph)}
}

// Connect is the method to connect the MCMCpathFinder to the output of a GFAreader
func (proc *MCMCpathFinder) Connect(previous *GFAreader) {
	proc.input = previous.output
}

// Run is the method to run this process, which satisfies the pipeline interface
func (proc *MCMCpathFinder) Run() {
	chain := markov.NewChain(proc.info.Haplotype.ChainOrder)
	for graph := range proc.input {
		// add the graph to the graphStore
		proc.info.Store[graph.GraphID] = graph
		// build the markov model
		err := graph.BuildMarkovChain(chain)
		misc.ErrorCheck(err)
	}

	// once all the graphs have been received and added to the combined model, start on the path finding
	var wg sync.WaitGroup
	for _, storedGraph := range proc.info.Store {
		wg.Add(1)
		go func(g *graph.GrootGraph, c *markov.Chain) {
			defer wg.Done()

				err := g.FindMarkovPaths(c, proc.info.Haplotype.BootStraps, proc.info.Haplotype.ScalingFactor)
				misc.ErrorCheck(err)

				err = g.ProcessMarkovPaths(proc.info.Haplotype.ProbabilityThreshold)
				misc.ErrorCheck(err)

				paths, err := g.GetMarkovPaths()
				misc.ErrorCheck(err)

				log.Printf("\tgraph %d has %d markov paths passing thresholds", g.GraphID, len(paths))
				for _, path := range paths {
					log.Printf("\t- [%v]", path)
				}

				// replace the old paths with the new MCMC derived paths
				err = g.PathReplace()
				misc.ErrorCheck(err)

			// send the graph to the next process
			proc.output <- g
		}(storedGraph, chain)
	}
	wg.Wait()
	close(proc.output)
}
*/

// HaplotypeParser is a pipeline process to parse the paths produced by the MCMCpathFinder process
type HaplotypeParser struct {
	info   *Info
	input  chan *graph.GrootGraph
	output []string
}

// NewHaplotypeParser is the constructor
func NewHaplotypeParser(info *Info) *HaplotypeParser {
	return &HaplotypeParser{info: info}
}

// Connect is the method to connect the HaplotypeParser to the output of a MCMCpathFinder
func (proc *HaplotypeParser) Connect(previous *EMpathFinder) {
	proc.input = previous.output
}

// CollectOutput is a method to return what paths are found via MCMC
func (proc *HaplotypeParser) CollectOutput() []string {
	return proc.output
}

// Run is the method to run this process, which satisfies the pipeline interface
func (proc *HaplotypeParser) Run() {
	counter := 0
	counter2 := 0
	meanEMiterations := 0
	keptPaths := []string{}
	log.Println("processing haplotypes...")
	for graph := range proc.input {
		meanEMiterations += graph.EMiterations

		// check graph has some paths left
		if len(graph.Paths) == 0 {
			continue
		}

		// remove dead ends
		misc.ErrorCheck(graph.RemoveDeadPaths())

		// print
		paths := graph.PrintEMpaths()
		log.Printf("\tgraph %d has %d called alleles after EM", graph.GraphID, len(paths))
		for _, path := range paths {
			log.Printf("\t- [%v]", path)
		}
		counter2 += len(paths)

		// write the graph
		graph.GrootVersion = version.VERSION
		fileName := fmt.Sprintf("%v/groot-graph-%d-haplotype.gfa", proc.info.Haplotype.HaploDir, graph.GraphID)
		graphWritten, err := graph.SaveGraphAsGFA(fileName)
		misc.ErrorCheck(err)
		counter += graphWritten

		// write the sequences
		seqs, err := graph.Graph2Seqs()
		misc.ErrorCheck(err)
		fileName += ".fna"
		fh, err := os.Create(fileName)
		defer fh.Close()
		misc.ErrorCheck(err)
		for id, seq := range seqs {
			fmt.Fprintf(fh, ">%v\n%v\n", string(graph.Paths[id]), string(seq))
			keptPaths = append(keptPaths, string(graph.Paths[id]))
		}
	}
	log.Printf("\tmean number of EM iterations: %d\n", meanEMiterations/counter)
	proc.output = keptPaths
	log.Printf("saved graphs to \"./%v/\"...", proc.info.Haplotype.HaploDir)
	log.Printf("\tnumber of graphs written to disk: %d\n", counter)
	log.Printf("saved haplotype sequences to \"./%v/\"...", proc.info.Haplotype.HaploDir)
	log.Printf("\tnumber of sequences written to disk: %d\n", counter2)
	log.Println("finished")

}
