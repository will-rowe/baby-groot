package pipeline

import (
	"fmt"
	"sync"

	"github.com/will-rowe/baby-groot/src/lshforest"
)

// theBoss is used to orchestrate the minions
type theBoss struct {
	info            *Info       // the runtime info for the pipeline
	inputReads      chan []byte // the boss uses this channel to receive data from the main sketching pipeline
	finishSketching chan bool   // the boss uses this channel to stop the sketching minions

	// the following fields are used by the sketching minions
	sketchingMinionRegister []*sketchingMinion // a slice of all the sketching minions controlled by this boss
	readCount               int                // the total number of reads the sketching minions received
	mappedCount             int                // the total number of reads that were successful mapped to at least one graph
	multimappedCount        int                // the total number of reads that had multiple mappings
	wg                      sync.WaitGroup     // records the number of reads currently being mapped by the sketching minions

	// the following fields are used by the graph minions
	graphMinionRegister map[uint32]*graphMinion // used to keep a record of the graph minions
}

// launchGraphMinions is a method to set up the graph minions
func (theBoss *theBoss) launchGraphMinions() {

	// one minion per graph in the index
	theBoss.graphMinionRegister = make(map[uint32]*graphMinion, len(theBoss.info.Store))
	for graphID, graph := range theBoss.info.Store {

		// create the minion
		minion := newGraphMinion(graphID, theBoss.info.KmerSize, graph)

		// start the minion
		minion.start()

		// register it
		theBoss.graphMinionRegister[graphID] = minion
	}
}

// stopMinions is a method to initiate a controlled shut down of the boss and minions
func (theBoss *theBoss) stopMinions() {

	// wait until all the reads have been processed
	theBoss.wg.Wait()

	// close the channel sending sequences to the minions
	close(theBoss.inputReads)

	// wait for work to finish and stop the Boss's go routine
	theBoss.finishSketching <- true

	// send the finish signal to the sketching minions
	for _, minion := range theBoss.sketchingMinionRegister {
		receivedReads, mappedReads, multimappedReads := minion.finish()

		// record the number of reads each minion processed
		theBoss.readCount += receivedReads
		theBoss.mappedCount += mappedReads
		theBoss.multimappedCount += multimappedReads

		// collect the mappings and send to the graph minions
		// TODO: replace panics with error handling
		if len(minion.mappingMap) != 0 {
			for graphWindow, frequency := range minion.mappingMap {

				// convert the stringified db match for this mapping to the constituent parts (graph, node, offset)
				mappingInfo, err := minion.info.db.GetKey(graphWindow)
				if err != nil {
					panic(err)
				}

				// make a copy of this graphWindow
				graphWindow := &lshforest.Key{
					GraphID: mappingInfo.GraphID,
					Node:    mappingInfo.Node,
					OffSet:  mappingInfo.OffSet,
					SubPath: mappingInfo.SubPath, // don't need to deep copy this as we don't edit it
					Freq:    frequency,           // add the mapping frequency
				}

				// look up the graph minion responsible for this graph window
				graphMinion, ok := theBoss.graphMinionRegister[graphWindow.GraphID]
				if !ok {
					panic("can't find graph minion")
				}

				// send the window on for graph augmentation
				graphMinion.inputChannel <- graphWindow
			}
		}
	}

	// close down the graph minions
	for _, minion := range theBoss.graphMinionRegister {
		minion.finish()
	}

}

// mapReads is a function to start off the minions to map reads, the minions to augement graphs, and to return their boss
func mapReads(runtimeInfo *Info) (*theBoss, error) {

	// create a boss to orchestrate the minions
	boss := &theBoss{
		info:            runtimeInfo,
		inputReads:      make(chan []byte, BUFFERSIZE),
		finishSketching: make(chan bool),
		readCount:       0,
		mappedCount:     0,
	}

	// launch the graph minions
	boss.launchGraphMinions()

	// minionQueue is where a minion will put their input channel if they are available to do some work
	minionQueue := make(chan chan []byte)

	// set up the minion pool
	boss.sketchingMinionRegister = make([]*sketchingMinion, runtimeInfo.NumProc)
	for id := 0; id < runtimeInfo.NumProc; id++ {

		// create a minion
		minion := newSketchingMinion(id, runtimeInfo, uint(runtimeInfo.KmerSize), uint(runtimeInfo.SketchSize), runtimeInfo.KMVsketch, minionQueue, &boss.wg)

		// start it running
		minion.start()

		// add it to the boss's register of running minions
		boss.sketchingMinionRegister[id] = minion
	}
	if len(boss.sketchingMinionRegister) == 0 {
		return nil, fmt.Errorf("the boss didn't make any sketching minions - check number of processors available")
	}

	// start processing the sequences
	go func() {
		for {
			select {

			// if there's a sequence to be processed, send it to a minion
			case read := <-boss.inputReads:

				//  TODO: add some checking at this stage, before sending the read on
				if len(read) == 0 {
					continue
				}
				boss.wg.Add(1)

				// wait for a minion to be available
				freeMinion := <-minionQueue

				// put the read in the minion's input channel
				freeMinion <- read

			// stop the minions working when the boss receives word
			case <-boss.finishSketching:

				break
			}
		}
	}()

	return boss, nil
}
