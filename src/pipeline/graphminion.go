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
	graph        *graph.GrootGraph
	inputChannel chan *lshforest.Key
	wg           *sync.WaitGroup
}

// newGraphMinion is the constructor function
func newGraphMinion(id uint32, graph *graph.GrootGraph, wg *sync.WaitGroup) *graphMinion {
	return &graphMinion{
		id:           id,
		graph:        graph,
		inputChannel: make(chan *lshforest.Key, BUFFERSIZE),
		wg:           wg,
	}
}

// start is a method to start the graphMinion running
func (graphMinion *graphMinion) start() {
	go func() {
		defer graphMinion.wg.Done()
		for {

			// pull reads from queue until done
			mappingData, ok := <-graphMinion.inputChannel
			if !ok {
				return
			}
			if mappingData == nil {
				continue
			}

			// increment the nodes contained in the mapping window
			misc.ErrorCheck(graphMinion.graph.IncrementSubPath(mappingData.ContainedNodes, mappingData.Freq))
		}
	}()
}
