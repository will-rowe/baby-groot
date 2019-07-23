package pipeline

import (
	"github.com/will-rowe/baby-groot/src/minhash"
	"github.com/will-rowe/baby-groot/src/misc"
)

// minion is the base data type
type minion struct {
	id               int
	info             *Info
	kmerSize         uint
	sketchSize       uint
	kmvSketch        bool
	minionQueue      chan chan []byte
	inputChannel     chan []byte
	stop             chan struct{}
	readCount        int
	mappedCount      int
	multimappedCount int
}

// newMinion is the constructor function
func newMinion(id int, runtimeInfo *Info, kmerSize, sketchSize uint, kmvSketch bool, minionQueue chan chan []byte) *minion {
	return &minion{
		id:               id,
		info:             runtimeInfo,
		kmerSize:         kmerSize,
		sketchSize:       sketchSize,
		kmvSketch:        kmvSketch,
		minionQueue:      minionQueue,
		inputChannel:     make(chan []byte),
		stop:             make(chan struct{}),
		readCount:        0,
		mappedCount:      0,
		multimappedCount: 0,
	}
}

// start is a method to start the minion running
func (minion *minion) start() {
	go func() {
		for {

			// when the minion is available for work, place its data channel in the queue
			minion.minionQueue <- minion.inputChannel

			// wait for work or stop signal
			select {

			// the minion has receieved some data from the boss
			case sequence := <-minion.inputChannel:

				// increment the read count for this read
				minion.readCount++

				// get sketch for read TODO: I'm ignoring the bloom filter for now
				mapped := 0
				readSketch, err := minhash.GetReadSketch(sequence, minion.kmerSize, minion.sketchSize, minion.kmvSketch)
				misc.ErrorCheck(err)

				// query the LSH forest
				for _, result := range minion.info.db.Query(readSketch) {
					mapped++

					// convert the stringified db match for this mapping to the constituent parts (graph, node, offset)
					mapping, err := minion.info.db.GetKey(result)

					// TODO: I'd rather not use the error checker here, should I send errors down a channel instead and let the boss deal with it
					misc.ErrorCheck(err)

					// project the sketch of this read onto the graph and increment the k-mer count for each segment in the projection's subpaths
					// this also updates the segment coverage information, using a bit vector to indicate when a base is covered
					misc.ErrorCheck(minion.info.Store[mapping.GraphID].IncrementSubPath(mapping.SubPath, mapping.OffSet, len(sequence), int(minion.kmerSize)))
				}

				if mapped > 0 {
					minion.mappedCount++
				}

				if mapped > 1 {
					minion.multimappedCount++
				}

			// end the minion go function if a stop signal has been sent
			case <-minion.stop:
				return
			}
		}
	}()
}

// finish is a method to close down a minion, after checking it isn't currently working on something. It returns the number of reads it has processed, how many mapped and how many had multiple mappings
func (minion *minion) finish() (int, int, int) {
	close(minion.stop)
	return minion.readCount, minion.mappedCount, minion.multimappedCount
}
