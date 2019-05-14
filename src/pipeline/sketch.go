package pipeline

/*
 this part of the pipeline will process reads, sketch them, map them and then project them onto variation graphs
*/

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
	"github.com/will-rowe/baby-groot/src/minhash"
	"github.com/will-rowe/baby-groot/src/misc"
	"github.com/will-rowe/baby-groot/src/seqio"
)

// DataStreamer is a pipeline process that streams data from STDIN/file
type DataStreamer struct {
	info   *Info
	input  []string
	output chan []byte
}

// NewDataStreamer is the constructor
func NewDataStreamer(info *Info) *DataStreamer {
	return &DataStreamer{info: info, output: make(chan []byte, BUFFERSIZE)}
}

// Connect is the method to connect the DataStreamer to some data source
func (proc *DataStreamer) Connect(input []string) {
	proc.input = input
}

// Run is the method to run this process, which satisfies the pipeline interface
func (proc *DataStreamer) Run() {
	defer close(proc.output)
	var scanner *bufio.Scanner
	// if an input file path has not been provided, scan the contents of STDIN
	if len(proc.input) == 0 {
		scanner = bufio.NewScanner(os.Stdin)
		for scanner.Scan() {
			// important: copy content of scan to a new slice before sending, this avoids race conditions (as we are using multiple go routines) from concurrent slice access
			proc.output <- append([]byte(nil), scanner.Bytes()...)
		}
		if scanner.Err() != nil {
			log.Fatal(scanner.Err())
		}
	} else {
		for i := 0; i < len(proc.input); i++ {
			fh, err := os.Open(proc.input[i])
			misc.ErrorCheck(err)
			defer fh.Close()
			// handle gzipped input
			splitFilename := strings.Split(proc.input[i], ".")
			if splitFilename[len(splitFilename)-1] == "gz" {
				gz, err := gzip.NewReader(fh)
				misc.ErrorCheck(err)
				defer gz.Close()
				scanner = bufio.NewScanner(gz)
			} else {
				scanner = bufio.NewScanner(fh)
			}
			for scanner.Scan() {
				proc.output <- append([]byte(nil), scanner.Bytes()...)
			}
			if scanner.Err() != nil {
				log.Fatal(scanner.Err())
			}
		}
	}
}

// FastqHandler is a pipeline process to convert a pipeline to the FASTQ type
type FastqHandler struct {
	info   *Info
	input  chan []byte
	output chan *seqio.FASTQread
}

// NewFastqHandler is the constructor
func NewFastqHandler(info *Info) *FastqHandler {
	return &FastqHandler{info: info, output: make(chan *seqio.FASTQread, BUFFERSIZE)}
}

// Connect is the method to join the input of this process with the output of a DataStreamer
func (proc *FastqHandler) Connect(previous *DataStreamer) {
	proc.input = previous.output
}

// Run is the method to run this process, which satisfies the pipeline interface
func (proc *FastqHandler) Run() {
	defer close(proc.output)
	var l1, l2, l3, l4 []byte
	if proc.info.Sketch.Fasta {
		for line := range proc.input {
			if len(line) == 0 {
				break
			}
			// check for chevron
			if line[0] == 62 {
				if l1 != nil {
					// store current fasta entry (as FASTQ read)
					l1[0] = 64
					newRead, err := seqio.NewFASTQread(l1, l2, nil, nil)
					if err != nil {
						log.Fatal(err)
					}
					// send on the new read and reset the line stores
					proc.output <- newRead
				}
				l1, l2 = line, nil
			} else {
				l2 = append(l2, line...)
			}
		}
		// flush final fasta
		l1[0] = 64
		newRead, err := seqio.NewFASTQread(l1, l2, nil, nil)
		if err != nil {
			log.Fatal(err)
		}
		// send on the new read and reset the line stores
		proc.output <- newRead
	} else {
		// grab four lines and create a new FASTQread struct from them - perform some format checks and trim low quality bases
		for line := range proc.input {
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
				proc.output <- newRead
				l1, l2, l3, l4 = nil, nil, nil, nil
			}
		}
	}
}

// FastqChecker is a process to quality check FASTQ reads
type FastqChecker struct {
	info   *Info
	input  chan *seqio.FASTQread
	output chan *seqio.FASTQread
}

// NewFastqChecker is the constructor
func NewFastqChecker(info *Info) *FastqChecker {
	return &FastqChecker{info: info, output: make(chan *seqio.FASTQread, BUFFERSIZE)}
}

// Connect is the method to join the input of this process with the output of FastqHandler
func (proc *FastqChecker) Connect(previous *FastqHandler) {
	proc.input = previous.output
}

// Run is the method to run this process, which satisfies the pipeline interface
// TODO: I've removed the QC bits for now
func (proc *FastqChecker) Run() {
	log.Printf("now streaming reads...")
	// count the number of reads and their lengths as we go
	rawCount, lengthTotal := 0, 0
	for read := range proc.input {
		rawCount++
		// tally the length so we can report the mean
		lengthTotal += len(read.Seq)
		// send the read onwards for mapping
		proc.output <- read
	}
	// check we have received reads & print stats
	if rawCount == 0 {
		misc.ErrorCheck(errors.New("no fastq reads received"))
	}
	log.Printf("\tnumber of reads received from input: %d\n", rawCount)
	meanRL := float64(lengthTotal) / float64(rawCount)
	log.Printf("\tmean read length: %.0f\n", meanRL)
	close(proc.output)
}

// ReadMapper is a pipeline process to query the LSH database, map reads and project alignments onto graphs
type ReadMapper struct {
	info      *Info
	input     chan *seqio.FASTQread
	output    chan *graph.GrootGraph
	readStats [3]int // corresponds to num. reads, total num. mapped, num. multimapped
}

// NewDbQuerier is the constructor
func NewDbQuerier(info *Info) *ReadMapper {
	return &ReadMapper{info: info, output: make(chan *graph.GrootGraph), readStats: [3]int{0, 0, 0}}
}

// Connect is the method to join the input of this process with the output of FastqChecker
func (proc *ReadMapper) Connect(previous *FastqChecker) {
	proc.input = previous.output
}

// CollectReadStats is a method to return the number of reads processed, how many mapped and the number of multimaps
func (proc *ReadMapper) CollectReadStats() [3]int {
	return proc.readStats
}

// Run is the method to run this process, which satisfies the pipeline interface
func (proc *ReadMapper) Run() {
	defer close(proc.output)
	// if requested, set up a bloom filter to prevent unique k-mers being included in sketches
	var bf *minhash.BloomFilter
	if proc.info.Sketch.BloomFilter {
		bf = minhash.NewDefaultBloomFilter()
	}
	var wg sync.WaitGroup
	collectionChan := make(chan *seqio.FASTQread)
	for read := range proc.input {
		// increment the read counter
		proc.readStats[0]++

		// map each read within a go routine
		wg.Add(1)
		go func(r *seqio.FASTQread) {
			mapped := false

			// get sketch for read
			readSketch, err := r.RunMinHash(proc.info.Index.KmerSize, proc.info.Index.SketchSize, proc.info.Index.KMVsketch, bf)
			misc.ErrorCheck(err)
			// query the LSH forest
			for _, result := range proc.info.Db.Query(readSketch) {
				mapped = true

				// convert the stringified db match for this mapping to the constituent parts (graph, node, offset)
				alignment, err := proc.info.Db.GetKey(result)
				misc.ErrorCheck(err)

				// attach the mapping info to the read
				r.Alignments = append(r.Alignments, alignment)

				// project the sketch of this read onto the graph and increment the k-mer count for each segment in the projection's subpaths
				// this also updates the segment coverage information, using a bit vector to indicate when a base is covered
				misc.ErrorCheck(proc.info.Store[alignment.GraphID].IncrementSubPath(alignment.SubPath, alignment.OffSet, len(r.Seq), proc.info.Index.KmerSize))

			}
			if mapped == true {
				collectionChan <- r
			}
			wg.Done()
		}(read)
	}

	// close the channel once all the queries are done
	go func() {
		wg.Wait()
		close(collectionChan)
	}()

	// collect the mapped reads
	for mappedRead := range collectionChan {

		// increment the mapped read counter
		proc.readStats[1]++

		// check for multimaps and increment the counter
		if len(mappedRead.Alignments) > 1 {
			proc.readStats[2]++
		}
	}

	// log some stuff
	if proc.readStats[0] == 0 {
		misc.ErrorCheck(fmt.Errorf("no reads passed quality-based trimming"))
	} else {
		log.Printf("\tnumber of reads sketched: %d\n", proc.readStats[0])

	}
	if proc.readStats[1] == 0 {
		misc.ErrorCheck(fmt.Errorf("no reads could be mapped to the reference graphs"))
	} else {
		log.Printf("\ttotal number of mapped reads: %d\n", proc.readStats[1])
		log.Printf("\t\tuniquely mapped: %d\n", (proc.readStats[1] - proc.readStats[2]))
		log.Printf("\t\tmultimapped: %d\n", proc.readStats[2])
	}
	// send on the graphs now that the mapping is done
	for _, g := range proc.info.Store {
		proc.info.Sketch.TotalKmers += g.KmerTotal
		proc.output <- g
	}
	log.Printf("\tnumber of k-mers projected onto graphs: %.0f\n", proc.info.Sketch.TotalKmers)
}

// GraphPruner is a pipeline process to prune the graphs post mapping
type GraphPruner struct {
	info   *Info
	input  chan *graph.GrootGraph
	output []string
}

// NewGraphPruner is the constructor
func NewGraphPruner(info *Info) *GraphPruner {
	return &GraphPruner{info: info}
}

// Connect is the method to join the input of this process with the output of ReadMapper
func (proc *GraphPruner) Connect(previous *ReadMapper) {
	proc.input = previous.output
}

// CollectOutput is a method to return what paths are left post-pruning
func (proc *GraphPruner) CollectOutput() []string {
	return proc.output
}

// Run is the method to run this process, which satisfies the pipeline interface
func (proc *GraphPruner) Run() {
	graphChan := make(chan *graph.GrootGraph)
	var wg sync.WaitGroup
	counter := 0
	for g := range proc.input {
		wg.Add(1)
		counter++

		go func(graph *graph.GrootGraph) {
			defer wg.Done()
			// check for alignments and prune the graph
			keepGraph := graph.Prune(float64(proc.info.Sketch.MinKmerCoverage), proc.info.Sketch.MinBaseCoverage)

			// check we have some graph
			if keepGraph != false {
				graphChan <- graph
			}
		}(g)
	}
	go func() {
		wg.Wait()
		close(graphChan)
	}()

	// count and print some stuff
	graphCounter := 0
	keptPaths := []string{}
	log.Print("processing graphs...")
	for g := range graphChan {
		// write the graph
		g.GrootVersion = proc.info.Version
		fileName := fmt.Sprintf("%v/groot-graph-%d.gfa", proc.info.Sketch.GraphDir, g.GraphID)
		_, err := g.SaveGraphAsGFA(fileName)
		misc.ErrorCheck(err)
		graphCounter++
		log.Printf("\tgraph %d has %d remaining paths after weighting and pruning", g.GraphID, len(g.Paths))
		for _, path := range g.Paths {
			log.Printf("\t- [%v]", string(path))
			keptPaths = append(keptPaths, string(path))
		}
	}
	log.Printf("\ttotal number of graphs pruned: %d\n", counter)
	if graphCounter == 0 {
		log.Print("\tno graphs remaining after pruning")
	} else {
		log.Printf("writing graphs to \"./%v/\"...", proc.info.Sketch.GraphDir)
		log.Printf("\ttotal number of graphs written to disk: %d\n", graphCounter)
		log.Printf("\ttotal number of possible haplotypes found: %d\n", len(keptPaths))
		log.Printf("updating index with sketching info from this run...\n")
		misc.ErrorCheck(proc.info.Dump(proc.info.Index.IndexDir + "/index.info"))
		log.Printf("\tsaved index info to \"%v/index.info\"", proc.info.Index.IndexDir)
	}
	proc.output = keptPaths
}