package pipeline

import (
	"sync"

	"github.com/will-rowe/baby-groot/src/lshforest"
	"github.com/will-rowe/baby-groot/src/minhash"
	"github.com/will-rowe/baby-groot/src/misc"
)

// theBoss is used to orchestrate the minions
type theBoss struct {
	info                *Info                   // the runtime info for the pipeline
	graphMinionRegister map[uint32]*graphMinion // used to keep a record of the graph minions
	reads               chan []byte             // the boss uses this channel to receive data from the main sketching pipeline
	receivedReadCount   int                     // the number of reads the boss is sent during it's lifetime
	mappedCount         int                     // the total number of reads that were successful mapped to at least one graph
	multimappedCount    int                     // the total number of reads that had multiple mappings
}

// launchGraphMinions is a method to set up the graph minions which are used to augment the graphs with mapping results
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

// mapReads is a function to start off the minions to map reads, the minions to augement graphs, and to return their boss
func mapReads(runtimeInfo *Info, inputChan chan []byte) (*theBoss, error) {

	// create a boss to orchestrate the minions and collect stats
	boss := &theBoss{
		info:              runtimeInfo,
		reads:             inputChan,
		receivedReadCount: 0,
		mappedCount:       0,
		multimappedCount:  0,
	}

	// launch the graph minions
	boss.launchGraphMinions()

	// launch the sketching minions
	var wg sync.WaitGroup
	wg.Add(runtimeInfo.NumProc)
	countChan := make(chan [3]int, runtimeInfo.NumProc)
	for i := 0; i < runtimeInfo.NumProc; i++ {
		go func(workerNum int) {
			defer wg.Done()

			// keep a track of what this minion does
			receivedReads := 0
			mappedCount := 0
			multimappedCount := 0

			// start the main processing loop
			for {

				// pull reads from queue until done
				read, ok := <-boss.reads
				if !ok {

					// send back the stats from this minion
					countChan <- [3]int{receivedReads, mappedCount, multimappedCount}
					return
				}

				// get sketch for read TODO: I'm ignoring the bloom filter for now
				readSketch, err := minhash.GetReadSketch(read, uint(boss.info.KmerSize), uint(boss.info.SketchSize), false)
				misc.ErrorCheck(err)

				// get the number of k-mers in the sequence
				kmerCount := float64(len(read)) - float64(boss.info.KmerSize) + 1

				// query the LSH forest
				mapped := 0
				for _, hit := range boss.info.db.Query(readSketch) {
					mapped++

					// convert the stringified db match for this mapping to the constituent parts (graph, node, offset)
					mappingInfo, err := boss.info.db.GetKey(hit)
					if err != nil {
						panic(err)
					}

					// make a copy of this graphWindow
					graphWindow := &lshforest.Key{
						GraphID: mappingInfo.GraphID,
						Node:    mappingInfo.Node,
						OffSet:  mappingInfo.OffSet,
						SubPath: mappingInfo.SubPath, // don't need to deep copy this as we don't edit it
						Freq:    kmerCount,           // add the mapping frequency
					}

					// look up the graph minion responsible for this graph window
					graphMinion, ok := boss.graphMinionRegister[graphWindow.GraphID]
					if !ok {
						panic("can't find a graph minion")
					}

					// send the window on for graph augmentation
					graphMinion.inputChannel <- graphWindow

					// update counts
					receivedReads++
					if mapped > 0 {
						mappedCount++
					}
					if mapped > 1 {
						multimappedCount++
					}
				}
			}
		}(i)
	}

	// wait for the sketching minions
	wg.Wait()
	close(countChan)

	// get the counts
	for count := range countChan {
		boss.receivedReadCount += count[0]
		boss.mappedCount += count[1]
		boss.multimappedCount += count[2]
	}

	// close down the graph minions
	for _, minion := range boss.graphMinionRegister {
		minion.finish()
	}
	return boss, nil
}
