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
	"fmt"
	"log"
	"os"
	"runtime"
	"time"

	"github.com/pkg/profile"
	"github.com/spf13/cobra"
	"github.com/will-rowe/baby-groot/src/misc"
	"github.com/will-rowe/baby-groot/src/pipeline"
	"github.com/will-rowe/baby-groot/src/version"
)

// the command line arguments
var (
	indexDir        *string                                                           // directory containing the index files
	fastq           *[]string                                                         // list of FASTQ files to align
	fasta           *bool                                                             // flag to treat input as fasta sequences
	bloomFilter     *bool                                                             // flag to use a bloom filter in order to prevent unique k-mers being used during sketching
	minKmerCoverage *int                                                              // the minimum k-mer coverage per base of a segment
	minBaseCoverage *float64                                                          // percentage of the segment bases that had reads align
	graphDir        *string                                                           // directory to save gfa graphs to
	defaultGraphDir = "./groot-graphs-" + string(time.Now().Format("20060102150405")) // a default graphDir
)

// sketchCmd is used by cobra
var sketchCmd = &cobra.Command{
	Use:   "sketch",
	Short: "Sketch sequences, align to references and weight variation graphs",
	Long:  `Sketch sequences, align to references and weight variation graphs`,
	Run: func(cmd *cobra.Command, args []string) {
		runSketch()
	},
	PreRunE: func(cmd *cobra.Command, args []string) error {
		return misc.CheckRequiredFlags(cmd.Flags())
	},
}

// init the command line arguments
func init() {
	indexDir = sketchCmd.Flags().StringP("indexDir", "i", "", "directory containing the index files - required")
	fastq = sketchCmd.Flags().StringSliceP("fastq", "f", []string{}, "FASTQ file(s) to align")
	fasta = sketchCmd.Flags().Bool("fasta", false, "if set, the input will be treated as fasta sequence(s) (experimental feature)")
	bloomFilter = sketchCmd.Flags().Bool("bloomFilter", false, "if set, a bloom filter will be used to stop unique k-mers being added to sketches")
	minKmerCoverage = sketchCmd.Flags().IntP("minKmerCov", "k", 1, "minimum k-mer coverage per base of a segment")
	minBaseCoverage = sketchCmd.Flags().Float64P("minBaseCov", "c", 0.1, "percentage of the graph segment bases that must have reads align")
	graphDir = sketchCmd.PersistentFlags().StringP("graphDir", "o", defaultGraphDir, "directory to save variation graphs to")
	sketchCmd.MarkFlagRequired("indexDir")
	RootCmd.AddCommand(sketchCmd)

}

// runAlign is the main function for the sketch sub-command
func runSketch() {

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

	// start the sketch sub command
	log.Printf("i am groot (version %s)", version.VERSION)
	log.Printf("starting the sketch subcommand")

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
	if *fasta {
		log.Print("\tinput file format: fasta")
	}
	log.Print("loading the index...")
	info := new(pipeline.Info)
	misc.ErrorCheck(info.Load(*indexDir + "/groot.index"))
	if info.Version != version.VERSION {
		misc.ErrorCheck(fmt.Errorf("the groot index was created with a different version of groot (you are currently using version %v)", version.VERSION))
	}
	log.Printf("\tk-mer size: %d\n", info.Index.KmerSize)
	log.Printf("\tsketch size: %d\n", info.Index.SketchSize)
	log.Printf("\tJaccard similarity theshold: %0.2f\n", info.Index.JSthresh)
	log.Printf("\twindow size used in indexing: %d\n", info.Index.WindowSize)
	log.Printf("\tnumber of variation graphs: %d\n", len(info.Store))
	numHF, numBucks := info.Db.Settings()
	log.Printf("\tnumber of hash functions per index bucket: %d\n", numHF)
	log.Printf("\tnumber of buckets: %d\n", numBucks)

	// add the sketch information to the existing groot runtime information
	info.Sketch = &pipeline.SketchCmd{
		Fasta:           *fasta,
		BloomFilter:     *bloomFilter,
		MinKmerCoverage: *minKmerCoverage,
		MinBaseCoverage: *minBaseCoverage,
	}

	// create the pipeline
	log.Printf("initialising alignment pipeline...")
	alignmentPipeline := pipeline.NewPipeline()

	// initialise processes
	log.Printf("\tinitialising the processes")
	dataStream := pipeline.NewDataStreamer(info)
	fastqHandler := pipeline.NewFastqHandler(info)
	fastqChecker := pipeline.NewFastqChecker(info)
	readMapper := pipeline.NewDbQuerier(info)
	graphPruner := pipeline.NewGraphPruner(info, false)

	// connect the pipeline processes
	log.Printf("\tconnecting data streams")
	dataStream.Connect(*fastq)
	fastqHandler.Connect(dataStream)
	fastqChecker.Connect(fastqHandler)
	readMapper.Connect(fastqChecker)
	graphPruner.Connect(readMapper)

	// submit each process to the pipeline and run it
	alignmentPipeline.AddProcesses(dataStream, fastqHandler, fastqChecker, readMapper, graphPruner)
	log.Printf("\tnumber of processes added to the alignment pipeline: %d\n", alignmentPipeline.GetNumProcesses())
	alignmentPipeline.Run()
	if len(info.Store) != 0 {
		log.Printf("saving graphs...\n")
		for graphID, g := range info.Store {
			fileName := fmt.Sprintf("%v/groot-graph-%d.gfa", *graphDir, graphID)
			_, err := g.SaveGraphAsGFA(fileName)
			misc.ErrorCheck(err)
		}
		log.Printf("updating index with sketching info from this run...\n")
		misc.ErrorCheck(info.Dump(info.Index.IndexDir + "/groot.index"))
		log.Printf("\tsaved index info to \"%v/groot.index\"", info.Index.IndexDir)
	}
	log.Println("finished")
}

// alignParamCheck is a function to check user supplied parameters
func alignParamCheck() error {
	// check the supplied FASTQ file(s)
	if len(*fastq) == 0 {
		misc.ErrorCheck(misc.CheckSTDIN())
		log.Printf("\tinput file: using STDIN")
	} else {
		for _, fastqFile := range *fastq {
			misc.ErrorCheck(misc.CheckFile(fastqFile))
			misc.ErrorCheck(misc.CheckExt(fastqFile, []string{"fastq", "fq", "fasta", "fna", "fa"}))
		}
	}
	// check the index directory and files
	misc.ErrorCheck(misc.CheckDir(*indexDir))
	misc.ErrorCheck(misc.CheckFile(*indexDir + "/groot.index"))
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
