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
	"time"

	"github.com/pkg/profile"
	"github.com/spf13/cobra"
	"github.com/will-rowe/bg/src/graph"
	"github.com/will-rowe/bg/src/lshForest"
	"github.com/will-rowe/bg/src/misc"
	"github.com/will-rowe/bg/src/stream"
	"github.com/will-rowe/bg/src/version"
)

// the command line arguments
var (
	trimSwitch      *bool                                                             // enable quality based trimming of reads
	minQual         *int                                                              // minimum base quality (used in quality based trimming)
	minRL           *int                                                              // minimum read length (evaluated post trimming)
	clip            *int                                                              // maximum number of clipped bases allowed during local alignment
	indexDir        *string                                                           // directory containing the index files
	fastq           *[]string                                                         // list of FASTQ files to align
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
	trimSwitch = alignCmd.Flags().Bool("trim", false, "enable quality based trimming of reads (post seeding)")
	minQual = alignCmd.Flags().IntP("minQual", "q", 20, "minimum base quality (used in quality based trimming)")
	minRL = alignCmd.Flags().IntP("minRL", "l", 100, "minimum read length (evaluated post trimming)")
	clip = alignCmd.Flags().IntP("clip", "c", 5, "maximum number of clipped bases allowed during local alignment")
	indexDir = alignCmd.Flags().StringP("indexDir", "i", "", "directory containing the index files - required")
	fastq = alignCmd.Flags().StringSliceP("fastq", "f", []string{}, "FASTQ file(s) to align")
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
	indexFiles := [3]string{"/index.graph", "/index.info", "/index.sigs"}
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
	logFH := misc.StartLogging(*logFile)
	defer logFH.Close()
	log.SetOutput(logFH)
	log.Printf("i am groot (version %s)", version.VERSION)
	log.Printf("starting the align subcommand")
	// check the supplied files and then log some stuff
	log.Printf("checking parameters...")
	misc.ErrorCheck(alignParamCheck())
	log.Printf("\tprocessors: %d", *proc)
	if *trimSwitch {
		log.Printf("\tread trimming: enabled")
		log.Printf("\tminimum base quality: %d", *minQual)
		log.Printf("\tminimum read length: %d", *minRL)
	} else {
		log.Printf("\tread trimming: disabled")
	}
	log.Printf("\tmaximum clipped bases allowed: %d", *clip)
	for _, file := range *fastq {
		log.Printf("\tinput file: %v", file)
	}
	log.Print("loading index information...")
	info := new(misc.IndexInfo)
	misc.ErrorCheck(info.Load(*indexDir + "/index.info"))
	log.Printf("\tk-mer size: %d\n", info.Ksize)
	log.Printf("\tsignature size: %d\n", info.SigSize)
	log.Printf("\tJaccard similarity theshold: %0.2f\n", info.JSthresh)
	log.Printf("\twindow sized used in indexing: %d\n", info.ReadLength)
	log.Print("loading the groot graphs...")
	graphStore := make(graph.GraphStore)
	misc.ErrorCheck(graphStore.Load(*indexDir + "/index.graph"))
	log.Printf("\tnumber of variation graphs: %d\n", len(graphStore))
	log.Print("loading the MinHash signatures...")
	database := lshForest.NewLSHforest(info.SigSize, info.JSthresh)
	misc.ErrorCheck(database.Load(*indexDir + "/index.sigs"))
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
	graphAligner := stream.NewAligner()

	// add in the process parameters
	dataStream.InputFile = *fastq
	fastqChecker.WindowSize = info.ReadLength
	dbQuerier.Db = database
	dbQuerier.CommandInfo = info
	dbQuerier.GraphStore = graphStore
	graphAligner.GraphStore = graphStore
	graphAligner.Ksize = info.Ksize

	// arrange pipeline processes
	log.Printf("\tconnecting data streams")
	fastqHandler.Input = dataStream.Output
	fastqChecker.Input = fastqHandler.Output
	dbQuerier.Input = fastqChecker.Output
	graphAligner.Input = dbQuerier.Output

	// submit each process to the pipeline to be run
	pipeline.AddProcesses(dataStream, fastqHandler, fastqChecker, dbQuerier, graphAligner)
	log.Printf("\tnumber of processes added to the alignment pipeline: %d\n", len(pipeline.Processes))
	pipeline.Run()

	// save the graph files
	log.Printf("saving graphs to \"%v\"...", *graphDir)
	counter := 0

	// TODO: run this in go routines
	for _, graph := range graphStore {

		// TODO: user to set these
		minKmerCoverage := 20  // the minimum k-mer coverage per base of a segment
		minBaseCoverage := 0.0 // percentage of the segment bases that had reads align

		// check for alignments and prune the graph
		keepGraph := graph.Prune(float64(minKmerCoverage), minBaseCoverage)

		// check we have some graph
		if keepGraph == false {
			continue
		}

		// print seqs
		for i, path := range graph.Paths {
			fmt.Println(string(path))
			fmt.Println(string(graph.Graph2Seq(i)))
		}

		// write the graph
		graph.GrootVersion = version.VERSION
		graphWritten, err := graph.SaveGraphAsGFA(*graphDir)
		misc.ErrorCheck(err)
		counter += graphWritten
	}

	log.Printf("\tnumber of graphs written to disk: %d\n", counter)
	log.Println("finished")
}
