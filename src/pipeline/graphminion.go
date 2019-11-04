package pipeline

import (
	"sync"

	"github.com/will-rowe/baby-groot/src/graph"
	"github.com/will-rowe/baby-groot/src/lshforest"
	"github.com/will-rowe/baby-groot/src/misc"
)

// graphMinion holds a graph and is responsible for augmenting the paths when new mapping data arrives
type graphMinion struct {
	id           uint32
	kmerSize     int
	graph        *graph.GrootGraph
	inputChannel chan *lshforest.Key

	// control over the minion
	stop chan struct{}
	sync.RWMutex
}

// newGraphMinion is the constructor function
func newGraphMinion(id uint32, kmerSize int, graph *graph.GrootGraph) *graphMinion {
	return &graphMinion{
		id:           id,
		kmerSize:     kmerSize,
		graph:        graph,
		inputChannel: make(chan *lshforest.Key, BUFFERSIZE),
		stop:         make(chan struct{}),
	}
}

// start is a method to start the graphMinion running
func (graphMinion *graphMinion) start() {
	go func() {
		for {

			// wait for work or stop signal
			select {

			// some mapping data has arrived
			case mappingData := <-graphMinion.inputChannel:

				if mappingData == nil {
					continue
				}

				// record work being done
				graphMinion.Lock()

				// augment the graph using the mapping data
				// project the sketch of this read onto the graph and increment the k-mer count for each node in the projection's subpaths
				misc.ErrorCheck(graphMinion.graph.IncrementSubPath(mappingData.SubPath, mappingData.Freq))

				// work done
				graphMinion.Unlock()

			// end the sketchingMinion go function if a stop signal has been sent
			case <-graphMinion.stop:
				return
			}
		}
	}()
}

// finish is a method to properly stop and close down a graphMinion
func (graphMinion *graphMinion) finish() {

	// close down the input channel
	close(graphMinion.inputChannel)

	// break out of the graphMinion's go routine
	close(graphMinion.stop)

	return
}
