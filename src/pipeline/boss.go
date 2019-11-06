package pipeline

import (
	"sync"

	"github.com/will-rowe/baby-groot/src/lshforest"
	"github.com/will-rowe/baby-groot/src/minhash"
	"github.com/will-rowe/baby-groot/src/misc"
)

// theBoss is used to orchestrate the minions
type theBoss struct {
	info                *Info          // the runtime info for the pipeline
	graphMinionRegister []*graphMinion // used to keep a record of the graph minions
	reads               chan []byte    // the boss uses this channel to receive data from the main sketching pipeline
	receivedReadCount   int            // the number of reads the boss is sent during it's lifetime
	mappedCount         int            // the total number of reads that were successful mapped to at least one graph
	multimappedCount    int            // the total number of reads that had multiple mappings
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

	// launch the graph minions (one minion per graph in the index)
	var graphWG sync.WaitGroup
	graphWG.Add(len(boss.info.Store))
	boss.graphMinionRegister = make([]*graphMinion, len(boss.info.Store))
	for graphID, graph := range boss.info.Store {

		// create, start and register the graph minion
		minion := newGraphMinion(graphID, graph, &graphWG)
		minion.start()
		boss.graphMinionRegister[graphID] = minion
	}

	// launch the sketching minions (one per CPU)
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

				// get sketch for read
				readSketch, err := minhash.GetReadSketch(read, uint(boss.info.KmerSize), uint(boss.info.SketchSize), false)
				misc.ErrorCheck(err)

				// get the number of k-mers in the sequence
				readLength := len(read)
				kmerCount := float64(readLength-boss.info.KmerSize) + 1

				// query the LSH ensemble
				hits, err := boss.info.db.Query(readSketch, readLength+boss.info.KmerSize-1)
				if err != nil {
					panic(err)
				}
				for _, hit := range hits {

					// make a copy of this graphWindow
					graphWindow := &lshforest.Key{
						GraphID:        hit.GraphID,
						Node:           hit.Node,
						OffSet:         hit.OffSet,
						ContainedNodes: hit.ContainedNodes, // don't need to deep copy this as we don't edit it
						Freq:           kmerCount,          // add the k-mer count of the read in this window
					}

					// send the window on for graph augmentation
					boss.graphMinionRegister[hit.GraphID].inputChannel <- graphWindow

				}

				// update counts
				receivedReads++
				if len(hits) > 0 {
					mappedCount++
				}
				if len(hits) > 1 {
					multimappedCount++
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
	for _, graphMinion := range boss.graphMinionRegister {
		close(graphMinion.inputChannel)
	}
	graphWG.Wait()
	return boss, nil
}
