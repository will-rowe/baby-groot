package pipeline

// theBoss is used to orchestrate the minions
type theBoss struct {
	inputReads       chan []byte // the boss uses this channel to receive data from the main sketching pipeline
	finish           chan bool   // the boss uses this channel to stop the minions
	minionRegister   []*minion   // a slice of all the minions controlled by this boss
	readCount        int         // the total number of reads the minions received
	mappedCount      int         // the total number of reads that were successful mapped to at least one graph
	multimappedCount int         // the total number of reads that had multiple mappings
}

// stopWork is a method to initiate a controlled shut down of the boss and minions
func (theBoss *theBoss) stopWork() {

	// close the channel sending sequences to the minions
	close(theBoss.inputReads)

	// stop the Boss's go routine and close channels
	theBoss.finish <- true

	// send the finish signal to the minions and collect the number of reads they mapped
	for _, minion := range theBoss.minionRegister {
		receivedReads, mappedReads, multimappedReads := minion.finish()

		theBoss.readCount += receivedReads
		theBoss.mappedCount += mappedReads
		theBoss.multimappedCount += multimappedReads
	}

}

// mapReads is a function to start off the minions to map reads, the function returns their boss
func mapReads(runtimeInfo *Info) (*theBoss, error) {

	// create a boss to orchestrate the minions
	boss := &theBoss{
		inputReads:  make(chan []byte),
		finish:      make(chan bool),
		readCount:   0,
		mappedCount: 0,
	}

	// minionQueue is where a minion will put their input channel if they are available to do some work
	minionQueue := make(chan chan []byte)

	// set up the minion pool
	boss.minionRegister = make([]*minion, runtimeInfo.NumProc)
	for id := 0; id < runtimeInfo.NumProc; id++ {

		// create a minion
		minion := newMinion(id, runtimeInfo, uint(runtimeInfo.Index.KmerSize), uint(runtimeInfo.Index.SketchSize), runtimeInfo.Index.KMVsketch, minionQueue)

		// start it running
		minion.start()

		// add it to the boss's register of running minions
		boss.minionRegister[id] = minion
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

				// wait for a minion to be available
				freeMinion := <-minionQueue

				// put the read in the minion's input channel
				freeMinion <- read

			// stop the minions working when the boss receives word
			case <-boss.finish:
				break
			}
		}
	}()

	return boss, nil
}
