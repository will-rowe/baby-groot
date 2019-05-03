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
	"github.com/will-rowe/baby-groot/src/seqio"
	"github.com/will-rowe/gfa"
)

// MSAconverter is a pipeline process that streams data from STDIN/file
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
	// load each MSA outside of the go-routines to prevent 'too many open files' error on OSX
	for i, msaFile := range proc.input {
		msa, err := gfa.ReadMSA(msaFile)
		misc.ErrorCheck(err)
		wg.Add(1)
		go func(msaID int, msa *multi.Multi) {
			defer wg.Done()
			// convert the MSA to a GFA instance
			newGFA, err := gfa.MSA2GFA(msa)
			misc.ErrorCheck(err)
			// create a GrootGraph
			grootGraph, err := graph.CreateGrootGraph(newGFA, msaID)
			if err != nil {
				misc.ErrorCheck(err)
			}
			proc.output <- grootGraph
		}(i, msa)
	}
	wg.Wait()
	close(proc.output)
}

// GraphSketcher is a pipeline process that windows graph traversals and sketches them
type GraphSketcher struct {
	info   *Info
	input  chan *graph.GrootGraph
	output chan *seqio.Key
}

// NewGraphSketcher is the constructor
func NewGraphSketcher(info *Info) *GraphSketcher {
	return &GraphSketcher{info: info, output: make(chan *seqio.Key, BUFFERSIZE)}
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
	graphStore := make(graph.GraphStore)

	// receive the graphs to be sketched
	var wg sync.WaitGroup
	for newGraph := range proc.input {
		wg.Add(1)
		go func(grootGraph *graph.GrootGraph) {
			defer wg.Done()
			keyChecker := make(map[string]string)
			// create sketch for each window in the graph
			for window := range grootGraph.WindowGraph(proc.info.WindowSize, proc.info.KmerSize, proc.info.SketchSize, proc.info.KMVsketch) {

				// there may be multiple copies of the same window key
				// - one graph+node+offset can have several subpaths to window
				// - or windows can be derived from identical regions of the graph that multiple sequences share
				// the keyCheck map will keep track of seen window keys
				window.StringifiedKey = fmt.Sprintf("g%dn%do%d", window.GraphID, window.Node, window.OffSet)
				newSubPath := misc.Stringify(window.SubPath)
				if seenSubPath, ok := keyChecker[window.StringifiedKey]; ok {

					// check if the subpath is identical
					if newSubPath != seenSubPath {
						// if it is different, we have multiple unique traversals from one node, so adjust the new key and send the window
						window.StringifiedKey = fmt.Sprintf("g%dn%do%dp%d", window.GraphID, window.Node, window.OffSet, window.Ref)
					} else {
						// if it is identical, we don't need another sketch from the same subpath
						continue
					}
				} else {
					keyChecker[window.StringifiedKey] = newSubPath
				}

				// send the windows for this graph onto the next process
				proc.output <- window
			}
			// this graph is sketched, now send it on to be saved in the current process
			graphChan <- grootGraph
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
	// save them
	misc.ErrorCheck(graphStore.Dump(proc.info.IndexDir + "/index.graph"))
	log.Printf("\tsaved groot graphs")

}

// SketchIndexer is a pipeline process that adds sketches to the LSH Forest
type SketchIndexer struct {
	info  *Info
	input chan *seqio.Key
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
	log.Printf("\tnumber of hash functions per bucket: %d\n", numHF)
	log.Printf("\tnumber of buckets: %d\n", numBucks)
	log.Printf("\tnumber of sketches added: %d\n", sketchCount)
	misc.ErrorCheck(index.Dump(proc.info.IndexDir + "/index.sketches"))
	log.Printf("\tsaved the sketches")
}
