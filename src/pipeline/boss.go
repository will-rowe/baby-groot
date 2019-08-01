package pipeline

import (
	"fmt"
	"log"
	"sync"

	"github.com/will-rowe/baby-groot/src/lshforest"
	"github.com/will-rowe/baby-groot/src/misc"
)

// theBoss is used to orchestrate the minions
type theBoss struct {
	info            *Info       // the runtime info for the pipeline
	reads      chan []byte // the boss uses this channel to receive data from the main sketching pipeline
	receivedReadCount int
	wg sync.WaitGroup

	// the following fields are used by the sketching minions
	sketchingMinionRegister []*sketchingMinion // a slice of all the sketching minions controlled by this boss
	readCount               int                // the total number of reads the sketching minions received
	mappedCount             int                // the total number of reads that were successful mapped to at least one graph
	multimappedCount        int                // the total number of reads that had multiple mappings

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

	// check read counts TODO: this shouldn't be necessary
	if theBoss.receivedReadCount != theBoss.readCount {
		fmt.Println(theBoss.receivedReadCount, theBoss.readCount)
		panic("boss didn't process all the reads")
	}
}

// mapReads is a function to start off the minions to map reads, the minions to augement graphs, and to return their boss
func mapReads(runtimeInfo *Info, inputChan chan []byte) (*theBoss, error) {

	// create a boss to orchestrate the minions
	boss := &theBoss{
		info:            runtimeInfo,
		reads:      	 inputChan,
		receivedReadCount: 0,
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

		// add it to the boss's register of running minions
		boss.sketchingMinionRegister[id] = minion

		// start it running
		minion.start()
	}
	if len(boss.sketchingMinionRegister) == 0 {
		return nil, fmt.Errorf("the boss didn't make any sketching minions - check number of available processors")
	}

	// start mapping reads
	for read := range boss.reads {

		// wait for a minion to be available
		freeMinion := <-minionQueue

		// put the read in the minion's input channel
		freeMinion <- read

		// tell the boss a read is being processed
		boss.wg.Add(1)

		// print current memory usage every 100,000 reads
		boss.receivedReadCount++
		if boss.info.Profiling {
			if (boss.receivedReadCount % 100000) == 0 {
				log.Printf("\tprocessed %d reads -> current memory usage %v", boss.receivedReadCount, misc.PrintMemUsage())
			}
		}
	}

	// close the channels
	boss.wg.Wait()
	boss.stopMinions()

	return boss, nil
}
