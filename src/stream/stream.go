/*
	the stream package contains a streaming implementation based on the Gopher Academy article by S. Lampa - Patterns for composable concurrent pipelines in Go (https://blog.gopheracademy.com/advent-2015/composable-pipelines-improvements/)
*/
package stream

import (
	"bufio"
	"compress/gzip"
	"errors"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"

	"github.com/will-rowe/baby-groot/src/graph"
	"github.com/will-rowe/baby-groot/src/lshForest"
	"github.com/will-rowe/baby-groot/src/minhash"
	"github.com/will-rowe/baby-groot/src/misc"
	"github.com/will-rowe/baby-groot/src/seqio"
)

// buffersize is the size of the buffer used by the channels
const buffersize = 128

// process is the interface used by pipeline
type process interface {
	Run()
}

// Pipeline is the base type, which takes any types that satisfy the process interface
type Pipeline struct {
	Processes []process
}

// NewPipeline is the pipeline constructor
func NewPipeline() *Pipeline {
	return &Pipeline{}
}

// AddProcess is a method to add a single process to the pipeline
func (pl *Pipeline) AddProcess(proc process) {
	pl.Processes = append(pl.Processes, proc)
}

// AddProcesses is a method to add multiple processes to the pipeline
func (pl *Pipeline) AddProcesses(procs ...process) {
	for _, proc := range procs {
		pl.AddProcess(proc)
	}
}

// Run is a method that starts the pipeline
func (pl *Pipeline) Run() {
	// each pipeline process is run in a Go routines, except the last process which is run in the foreground
	for i, proc := range pl.Processes {
		if i < len(pl.Processes)-1 {
			go proc.Run()
		} else {
			proc.Run()
		}
	}
}

// DataStreamer is a pipeline process that streams data from STDIN/file
type DataStreamer struct {
	process
	Output    chan []byte
	InputFile []string
}

// NewDataStreamer is the constructor
func NewDataStreamer() *DataStreamer {
	return &DataStreamer{Output: make(chan []byte, buffersize)}
}

// Run is the method to run this process, which satisfies the pipeline interface
func (proc *DataStreamer) Run() {
	var scanner *bufio.Scanner
	// if an input file path has not been provided, scan the contents of STDIN
	if len(proc.InputFile) == 0 {
		scanner = bufio.NewScanner(os.Stdin)
		for scanner.Scan() {
			// important: copy content of scan to a new slice before sending, this avoids race conditions (as we are using multiple go routines) from concurrent slice access
			proc.Output <- append([]byte(nil), scanner.Bytes()...)
		}
		if scanner.Err() != nil {
			log.Fatal(scanner.Err())
		}
	} else {
		for i := 0; i < len(proc.InputFile); i++ {
			fh, err := os.Open(proc.InputFile[i])
			misc.ErrorCheck(err)
			defer fh.Close()
			// handle gzipped input
			splitFilename := strings.Split(proc.InputFile[i], ".")
			if splitFilename[len(splitFilename)-1] == "gz" {
				gz, err := gzip.NewReader(fh)
				misc.ErrorCheck(err)
				defer gz.Close()
				scanner = bufio.NewScanner(gz)
			} else {
				scanner = bufio.NewScanner(fh)
			}
			for scanner.Scan() {
				proc.Output <- append([]byte(nil), scanner.Bytes()...)
			}
			if scanner.Err() != nil {
				log.Fatal(scanner.Err())
			}
		}
	}
	close(proc.Output)
}

/*
  A process to generate a FASTQ read from a stream of bytes
*/
type FastqHandler struct {
	process
	Input  chan []byte
	Output chan seqio.FASTQread
}

func NewFastqHandler() *FastqHandler {
	return &FastqHandler{Output: make(chan seqio.FASTQread, buffersize)}
}

// Run is the method to run this process, which satisfies the pipeline interface
func (proc *FastqHandler) Run() {
	defer close(proc.Output)
	var l1, l2, l3, l4 []byte
	// grab four lines and create a new FASTQread struct from them - perform some format checks and trim low quality bases
	for line := range proc.Input {
		if l1 == nil {
			l1 = line
		} else if l2 == nil {
			l2 = line
		} else if l3 == nil {
			l3 = line
		} else if l4 == nil {
			l4 = line
			// create fastq read
			newRead, err := seqio.NewFASTQread(l1, l2, l3, l4)
			if err != nil {
				log.Fatal(err)
			}
			// send on the new read and reset the line stores
			proc.Output <- newRead
			l1, l2, l3, l4 = nil, nil, nil, nil
		}
	}
}

/*
  A process to quality check FASTQ reads - trimming and discarding them according to user supplied cut offs
*/
type FastqChecker struct {
	process
	Input         chan seqio.FASTQread
	Output        chan seqio.FASTQread
	WindowSize    int
	MinReadLength int
}

func NewFastqChecker() *FastqChecker {
	return &FastqChecker{Output: make(chan seqio.FASTQread, buffersize)}
}

// Run is the method to run this process, which satisfies the pipeline interface
func (proc *FastqChecker) Run() {
	defer close(proc.Output)
	log.Printf("now streaming reads...")
	var wg sync.WaitGroup
	// count the number of reads and their lengths as we go
	rawCount, lengthTotal := 0, 0
	for read := range proc.Input {
		rawCount++
		// tally the length so we can report the mean
		lengthTotal += len(read.Seq)
		// send the read onwards for mapping
		proc.Output <- read
	}
	wg.Wait()
	// check we have received reads & print stats
	if rawCount == 0 {
		misc.ErrorCheck(errors.New("no fastq reads received"))
	}
	log.Printf("\tnumber of reads received from input: %d\n", rawCount)
	meanRL := float64(lengthTotal) / float64(rawCount)
	log.Printf("\tmean read length: %.0f\n", meanRL)
	// check the length is within +/-10 bases of the graph window
	if meanRL < float64(proc.WindowSize-10) || meanRL > float64(proc.WindowSize+10) {
		misc.ErrorCheck(fmt.Errorf("mean read length is outside the graph window size (+/- 10 bases)"))
	}
}

/*
  A process to query the LSH database, perform full MinHash comparisons on top hits and returns putative graph mapping locations
*/
type DbQuerier struct {
	process
	Input       chan seqio.FASTQread
	Db          lshForest.GROOTindex
	CommandInfo *misc.IndexInfo
	GraphStore  graph.GraphStore
	BloomFilter bool
}

func NewDbQuerier() *DbQuerier {
	return &DbQuerier{}
}

// Run is the method to run this process, which satisfies the pipeline interface
func (proc *DbQuerier) Run() {
	// if requested, set up a bloom filter to prevent unique k-mers being included in sketches
	var bf *minhash.BloomFilter
	if proc.BloomFilter {
		bf = minhash.NewDefaultBloomFilter()
	}
	// record the number of reads processed by the DbQuerier
	readTally, mappedTally, multiMappedTally := 0, 0, 0
	var wg sync.WaitGroup
	collectionChan := make(chan seqio.FASTQread)
	for read := range proc.Input {
		wg.Add(1)
		go func(read seqio.FASTQread) {
			defer wg.Done()
			mapped := false
			// get sketch for read
			readSketch, err := read.RunMinHash(proc.CommandInfo.Ksize, proc.CommandInfo.SigSize, proc.CommandInfo.KMVsketch, bf)
			misc.ErrorCheck(err)
			// query the LSH forest
			for _, result := range proc.Db.Query(readSketch) {
				mapped = true
				// convert the stringified db match for this mapping to the constituent parts (graph, node, offset)
				alignment, err := proc.Db.GetKey(result)
				misc.ErrorCheck(err)

				// attach the mapping info to the read
				read.Alignments = append(read.Alignments, alignment)
				// project the sketch of this read onto the graph and increment the k-mer count for each segment in the projection's subpaths
				// this also updates the segment coverage information, using a bit vector to indicate when a base is covered
				misc.ErrorCheck(proc.GraphStore[alignment.GraphID].IncrementSubPath(alignment.SubPath, alignment.OffSet, len(read.Seq), proc.CommandInfo.Ksize))

			}
			// BABY GROOT - re-evaluate channel usage
			if mapped == true {
				collectionChan <- read
			}
		}(read)
		readTally++
	}
	// close the channel once all the queries are done
	go func() {
		wg.Wait()
		close(collectionChan)
	}()
	// collect the mapped reads
	for mappedRead := range collectionChan {
		mappedTally++
		if len(mappedRead.Alignments) > 1 {
			multiMappedTally++
		}
	}

	// log some stuff
	if readTally == 0 {
		misc.ErrorCheck(fmt.Errorf("no reads passed quality-based trimming"))
	} else {
		log.Printf("\tnumber of reads received for alignment post QC: %d\n", readTally)
	}
	if mappedTally == 0 {
		misc.ErrorCheck(fmt.Errorf("no reads could be seeded against the reference graphs"))
	} else {
		log.Printf("\ttotal number of mapped reads: %d\n", mappedTally)
		log.Printf("\t\tuniquely mapped: %d\n", (mappedTally - multiMappedTally))
		log.Printf("\t\tmultimapped: %d\n", multiMappedTally)
	}
}
