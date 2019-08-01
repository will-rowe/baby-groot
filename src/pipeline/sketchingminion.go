package pipeline

import (
	"sync"

	"github.com/will-rowe/baby-groot/src/minhash"
	"github.com/will-rowe/baby-groot/src/misc"
)

// sketchingMinion is the base data type
type sketchingMinion struct {
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
	mappingMap       map[string]float64
	wg 				*sync.WaitGroup
}

// newSketchingMinion is the constructor function
func newSketchingMinion(id int, runtimeInfo *Info, kmerSize, sketchSize uint, kmvSketch bool, minionQueue chan chan []byte, wg *sync.WaitGroup) *sketchingMinion {
	return &sketchingMinion{
		id:               id,
		info:             runtimeInfo,
		kmerSize:         kmerSize,
		sketchSize:       sketchSize,
		kmvSketch:        kmvSketch,
		minionQueue:      minionQueue,
		inputChannel:     make(chan []byte, BUFFERSIZE),
		stop:             make(chan struct{}),
		readCount:        0,
		mappedCount:      0,
		multimappedCount: 0,
		mappingMap:       make(map[string]float64),
		wg: wg,
	}
}

// start is a method to start the sketchingMinion running
func (sketchingMinion *sketchingMinion) start() {
	go func() {
		for {

			// when the sketchingMinion is available for work, place its data channel in the queue
			sketchingMinion.minionQueue <- sketchingMinion.inputChannel

			// wait for work or stop signal
			select {

			// the sketchingMinion has receieved some data from the boss
			case sequence := <-sketchingMinion.inputChannel:

				// increment the read count for this read
				sketchingMinion.readCount++

				// get sketch for read TODO: I'm ignoring the bloom filter for now
				mapped := 0
				readSketch, err := minhash.GetReadSketch(sequence, sketchingMinion.kmerSize, sketchingMinion.sketchSize, sketchingMinion.kmvSketch)
				misc.ErrorCheck(err)

				// query the LSH forest
				for _, result := range sketchingMinion.info.db.Query(readSketch) {
					mapped++

					// record the graph window and increment by the number of k-mers in the sequence
					kmerCount := float64(len(sequence)) - float64(sketchingMinion.kmerSize) + 1
					sketchingMinion.mappingMap[result] += kmerCount
				}

				// record if the read produced any mappings, if so was it a multimapper
				if mapped > 0 {
					sketchingMinion.mappedCount++
				}
				if mapped > 1 {
					sketchingMinion.multimappedCount++
				}

				// tell the boss that a read has been processed
				sketchingMinion.wg.Done()

			// end the sketchingMinion go function if a stop signal has been sent
			case <-sketchingMinion.stop:
				return
			}
		}
	}()
}

// finish is a method to properly stop and close down a sketchingMinion
func (sketchingMinion *sketchingMinion) finish() (int, int, int) {

	// close down the input channel
	close(sketchingMinion.inputChannel)

	// break out of the sketchingMinion's go routine
	close(sketchingMinion.stop)

	// return the counts from this sketchingMinion
	return sketchingMinion.readCount, sketchingMinion.mappedCount, sketchingMinion.multimappedCount
}
