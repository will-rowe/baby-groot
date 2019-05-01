// Copyright © 2017 Will Rowe <will.rowe@stfc.ac.uk>
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

package cmd

import (
	"errors"
	"fmt"
	"log"
	"os"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/pkg/profile"
	"github.com/spf13/cobra"
	"github.com/will-rowe/baby-groot/src/graph"
	"github.com/will-rowe/baby-groot/src/lshForest"
	"github.com/will-rowe/baby-groot/src/misc"
	"github.com/will-rowe/baby-groot/src/stream"
	"github.com/will-rowe/baby-groot/src/version"
)

// the command line arguments
var (
	indexDir        *string                                                           // directory containing the index files
	fastq           *[]string                                                         // list of FASTQ files to align
	bloomFilter     *bool                                                             // flag to use a bloom filter in order to prevent unique k-mers being used during sketching
	minKmerCoverage *int                                                              // the minimum k-mer coverage per base of a segment
	minBaseCoverage *float64                                                          // percentage of the segment bases that had reads align
	graphDir        *string                                                           // directory to save gfa graphs to
	defaultGraphDir = "./groot-graphs-" + string(time.Now().Format("20060102150405")) // a default graphDir
)

// the align command (used by cobra)
var alignCmd = &cobra.Command{
	Use:   "align",
	Short: "Align a set of FASTQ reads to indexed variation graphs",
	Long:  `Align a set of FASTQ reads to indexed variation graphs`,
	Run: func(cmd *cobra.Command, args []string) {
		runAlign()
	},
	PreRunE: func(cmd *cobra.Command, args []string) error {
		return misc.CheckRequiredFlags(cmd.Flags())
	},
}

/*
  A function to initialise the command line arguments
*/
func init() {
	indexDir = alignCmd.Flags().StringP("indexDir", "i", "", "directory containing the index files - required")
	fastq = alignCmd.Flags().StringSliceP("fastq", "f", []string{}, "FASTQ file(s) to align")
	bloomFilter = alignCmd.Flags().Bool("bloomFilter", false, "if set, a bloom filter will be used to stop unique k-mers being added to sketches")
	minKmerCoverage = alignCmd.Flags().IntP("minKmerCov", "k", 1, "minimum k-mer coverage per base of a segment")
	minBaseCoverage = alignCmd.Flags().Float64P("minBaseCov", "c", 0.1, "percentage of the graph segment bases that must have reads align")
	graphDir = alignCmd.PersistentFlags().StringP("graphDir", "o", defaultGraphDir, "directory to save variation graphs to")
	alignCmd.MarkFlagRequired("indexDir")
	RootCmd.AddCommand(alignCmd)
}

/*
  A function to check user supplied parameters
*/
func alignParamCheck() error {
	// check the supplied FASTQ file(s)
	if len(*fastq) == 0 {
		stat, err := os.Stdin.Stat()
		if err != nil {
			return fmt.Errorf("error with STDIN")
		}
		if (stat.Mode() & os.ModeNamedPipe) == 0 {
			return fmt.Errorf("no STDIN found")
		}
		log.Printf("\tinput file: using STDIN")
	} else {
		for _, fastqFile := range *fastq {
			if _, err := os.Stat(fastqFile); err != nil {
				if os.IsNotExist(err) {
					return fmt.Errorf("FASTQ file does not exist: %v", fastqFile)
				} else {
					return fmt.Errorf("can't access FASTQ file (check permissions): %v", fastqFile)
				}
			}
			splitFilename := strings.Split(fastqFile, ".")
			if splitFilename[len(splitFilename)-1] == "gz" {
				if splitFilename[len(splitFilename)-2] == "fastq" || splitFilename[len(splitFilename)-2] == "fq" {
					continue
				}
			} else {
				if splitFilename[len(splitFilename)-1] == "fastq" || splitFilename[len(splitFilename)-1] == "fq" {
					continue
				}
			}
			return fmt.Errorf("does not look like a FASTQ file: %v", fastqFile)
		}
	}
	// check the index directory and files
	if *indexDir == "" {
		misc.ErrorCheck(errors.New("need to specify the directory where the index files are"))
	}
	if _, err := os.Stat(*indexDir); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("index directory does not exist: %v", *indexDir)
		} else {
			return fmt.Errorf("can't access an index directory (check permissions): %v", indexDir)
		}
	}
	indexFiles := [3]string{"/index.graph", "/index.info", "/index.sketches"}
	for _, indexFile := range indexFiles {
		if _, err := os.Stat(*indexDir + indexFile); err != nil {
			if os.IsNotExist(err) {
				return fmt.Errorf("index file does not exist: %v", indexFile)
			} else {
				return fmt.Errorf("can't access an index file (check permissions): %v", indexFile)
			}
		}
	}
	info := new(misc.IndexInfo)
	misc.ErrorCheck(info.Load(*indexDir + "/index.info"))
	if info.Version != version.VERSION {
		return fmt.Errorf("the groot index was created with a different version of groot (you are currently using version %v)", version.VERSION)
	}
	// setup the graphDir
	if _, err := os.Stat(*graphDir); os.IsNotExist(err) {
		if err := os.MkdirAll(*graphDir, 0700); err != nil {
			return fmt.Errorf("can't create specified output directory")
		}
	}
	// check the thresholds
	if *minBaseCoverage < 0.0 || *minBaseCoverage > 1.0 {
		return fmt.Errorf("minimum base coverage must be between 0.0 and 1.0")
	}
	// set number of processors to use
	if *proc <= 0 || *proc > runtime.NumCPU() {
		*proc = runtime.NumCPU()
	}
	runtime.GOMAXPROCS(*proc)
	return nil
}

/*
  The main function for the align sub-command
*/
func runAlign() {
	// set up profiling
	if *profiling == true {
		//defer profile.Start(profile.MemProfile, profile.ProfilePath("./")).Stop()
		defer profile.Start(profile.ProfilePath("./")).Stop()
	}
	// start logging
	if *logFile != "" {
		logFH := misc.StartLogging(*logFile)
		defer logFH.Close()
		log.SetOutput(logFH)
	} else {
		log.SetOutput(os.Stdout)
	}
	// start sub command
	log.Printf("i am groot (version %s)", version.VERSION)
	log.Printf("starting the align subcommand")
	// check the supplied files and then log some stuff
	log.Printf("checking parameters...")
	misc.ErrorCheck(alignParamCheck())
	if *bloomFilter {
		log.Printf("\tignoring unique k-mers: true")
	} else {
		log.Printf("\tignoring unique k-mers: false")
	}
	log.Printf("\tminimum k-mer frequency: %d", *minKmerCoverage)
	log.Printf("\tminimum base coverage: %0.0f%%", *minBaseCoverage*100)
	log.Printf("\tprocessors: %d", *proc)
	for _, file := range *fastq {
		log.Printf("\tinput file: %v", file)
	}
	log.Print("loading index information...")
	info := new(misc.IndexInfo)
	misc.ErrorCheck(info.Load(*indexDir + "/index.info"))
	log.Printf("\tk-mer size: %d\n", info.Ksize)
	log.Printf("\tsketch size: %d\n", info.SigSize)
	log.Printf("\tJaccard similarity theshold: %0.2f\n", info.JSthresh)
	log.Printf("\twindow size used in indexing: %d\n", info.WindowSize)
	log.Print("loading the groot graphs...")
	graphStore := make(graph.GraphStore)
	misc.ErrorCheck(graphStore.Load(*indexDir + "/index.graph"))
	log.Printf("\tnumber of variation graphs: %d\n", len(graphStore))
	log.Print("loading the MinHash sketches...")
	database := lshForest.NewLSHforest(info.SigSize, info.JSthresh)
	misc.ErrorCheck(database.Load(*indexDir + "/index.sketches"))
	database.Index()
	numHF, numBucks := database.Settings()
	log.Printf("\tnumber of hash functions per bucket: %d\n", numHF)
	log.Printf("\tnumber of buckets: %d\n", numBucks)
	///////////////////////////////////////////////////////////////////////////////////////

	// create the pipeline
	log.Printf("initialising alignment pipeline...")
	pipeline := stream.NewPipeline()

	// initialise processes
	log.Printf("\tinitialising the processes")
	dataStream := stream.NewDataStreamer()
	fastqHandler := stream.NewFastqHandler()
	fastqChecker := stream.NewFastqChecker()
	dbQuerier := stream.NewDbQuerier()

	// add in the process parameters
	dataStream.InputFile = *fastq
	fastqChecker.WindowSize = info.WindowSize
	dbQuerier.Db = database
	dbQuerier.CommandInfo = info
	dbQuerier.GraphStore = graphStore
	dbQuerier.BloomFilter = *bloomFilter

	// arrange pipeline processes
	log.Printf("\tconnecting data streams")
	fastqHandler.Input = dataStream.Output
	fastqChecker.Input = fastqHandler.Output
	dbQuerier.Input = fastqChecker.Output

	// submit each process to the pipeline to be run
	pipeline.AddProcesses(dataStream, fastqHandler, fastqChecker, dbQuerier)
	log.Printf("\tnumber of processes added to the alignment pipeline: %d\n", len(pipeline.Processes))
	pipeline.Run()

	// prune the graphs
	log.Printf("pruning graphs...")
	graphChan := make(chan *graph.GrootGraph)
	var wg sync.WaitGroup
	wg.Add(len(graphStore))
	for _, g := range graphStore {
		go func(graph *graph.GrootGraph) {
			defer wg.Done()
			// check for alignments and prune the graph
			keepGraph := graph.Prune(float64(*minKmerCoverage), *minBaseCoverage)

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

	// save the graph files
	log.Printf("saving graphs to \"./%v/\"...", *graphDir)
	graphCounter := 0
	pathCounter := 0
	for graph := range graphChan {
		graph.GrootVersion = version.VERSION
		fileName := fmt.Sprintf("%v/groot-graph-%d.gfa", *graphDir, graph.GraphID)
		graphWritten, err := graph.SaveGraphAsGFA(fileName)
		misc.ErrorCheck(err)
		graphCounter += graphWritten
		pathCounter += len(graph.Paths)
		log.Printf("\tgraph %d has %d remaining paths after weighting and pruning", graph.GraphID, len(graph.Paths))
		for _, path := range graph.Paths {
			log.Printf("\t- [%v]", string(path))
		}
	}
	if graphCounter == 0 {
		log.Print("\tno graphs remaining after pruning")
	} else {
		log.Printf("\ttotal number of graphs written to disk: %d\n", graphCounter)
		log.Printf("\ttotal number of possible alleles found: %d\n", pathCounter)
	}
	log.Println("finished")
}
