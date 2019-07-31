package pipeline

/*
 this part of the pipeline will convert multiple sequence alignments to variation graphs, sketch the traversals and index them
*/

import (
	"fmt"
	"log"
	"sync"

	"github.com/biogo/biogo/seq/multi"
	"github.com/will-rowe/baby-groot/src/graph"
	"github.com/will-rowe/baby-groot/src/lshforest"
	"github.com/will-rowe/baby-groot/src/misc"
	"github.com/will-rowe/gfa"
)

// MSAconverter is a pipeline process that converts a list of MSAs to GFAs
type MSAconverter struct {
	info   *Info
	input  []string
	output chan *graph.GrootGraph
}

// NewMSAconverter is the constructor
func NewMSAconverter(info *Info) *MSAconverter {
	return &MSAconverter{info: info, output: make(chan *graph.GrootGraph, BUFFERSIZE)}
}

// Connect is the method to connect the MSAconverter to some data source
func (proc *MSAconverter) Connect(input []string) {
	proc.input = input
}

// Run is the method to run this process, which satisfies the pipeline interface
func (proc *MSAconverter) Run() {
	var wg sync.WaitGroup
	wg.Add(len(proc.input))

	// load each MSA outside of the go-routines to prevent 'too many open files' error on OSX
	for i, msaFile := range proc.input {
		msa, err := gfa.ReadMSA(msaFile)
		misc.ErrorCheck(err)
		go func(msaID int, msa *multi.Multi) {
			// convert the MSA to a GFA instance
			newGFA, err := gfa.MSA2GFA(msa)
			misc.ErrorCheck(err)
			// create a GrootGraph
			grootGraph, err := graph.CreateGrootGraph(newGFA, msaID)
			if err != nil {
				misc.ErrorCheck(err)
			}
			proc.output <- grootGraph
			wg.Done()
		}(i, msa)
	}
	wg.Wait()
	close(proc.output)
}

// GraphSketcher is a pipeline process that windows graph traversals and sketches them
type GraphSketcher struct {
	info   *Info
	input  chan *graph.GrootGraph
	output chan *lshforest.Key
}

// NewGraphSketcher is the constructor
func NewGraphSketcher(info *Info) *GraphSketcher {
	return &GraphSketcher{info: info, output: make(chan *lshforest.Key, BUFFERSIZE)}
}

// Connect is the method to connect the MSAconverter to some data source
func (proc *GraphSketcher) Connect(previous *MSAconverter) {
	proc.input = previous.output
}

// Run is the method to run this process, which satisfies the pipeline interface
func (proc *GraphSketcher) Run() {
	defer close(proc.output)

	// after sketching all the received graphs, add the graphs to a store and save it
	graphChan := make(chan *graph.GrootGraph)
	graphStore := make(graph.Store)

	// receive the graphs to be sketched
	var wg sync.WaitGroup
	for newGraph := range proc.input {
		wg.Add(1)
		go func(grootGraph *graph.GrootGraph) {

			// create sketch for each window in the graph
			for window := range grootGraph.WindowGraph(proc.info.WindowSize, proc.info.KmerSize, proc.info.SketchSize, proc.info.KMVsketch) {

				// send the windows for this graph onto the next process
				proc.output <- window
			}
			// this graph is sketched, now send it on to be saved in the current process
			graphChan <- grootGraph
			wg.Done()
		}(newGraph)
	}
	go func() {
		wg.Wait()
		close(graphChan)
	}()

	// collect the graphs
	for sketchedGraph := range graphChan {
		graphStore[sketchedGraph.GraphID] = sketchedGraph
	}

	// check some graphs have been sketched
	if len(graphStore) == 0 {
		misc.ErrorCheck(fmt.Errorf("could not create any graphs"))
	}
	log.Printf("\tnumber of groot graphs built: %d", len(graphStore))

	// add the graphs to the pipeline info
	proc.info.Store = graphStore
}

// SketchIndexer is a pipeline process that adds sketches to the LSH Forest
type SketchIndexer struct {
	info  *Info
	input chan *lshforest.Key
}

// NewSketchIndexer is the constructor
func NewSketchIndexer(info *Info) *SketchIndexer {
	return &SketchIndexer{info: info}
}

// Connect is the method to connect the MSAconverter to some data source
func (proc *SketchIndexer) Connect(previous *GraphSketcher) {
	proc.input = previous.output
}

// Run is the method to run this process, which satisfies the pipeline interface
func (proc *SketchIndexer) Run() {

	// get the index ready
	index := lshforest.NewLSHforest(proc.info.SketchSize, proc.info.JSthresh)
	sketchCount := 0
	for window := range proc.input {
		sketchCount++

		// add the sketch to the lshforest index
		misc.ErrorCheck(index.Add(window))
	}
	numHF, numBucks := index.Settings()
	log.Printf("\tnumber of LSH Forest buckets: %d\n", numBucks)
	log.Printf("\tnumber of hash functions per bucket: %d\n", numHF)
	log.Printf("\tnumber of sketches added to the LSH Forest index: %d\n", sketchCount)

	// add the index to the pipeline info
	proc.info.AttachDB(index)
}
