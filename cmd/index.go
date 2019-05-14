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
	"path/filepath"
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
	kmerSize      *int                                                             // size of k-mer
	sketchSize    *int                                                             // size of MinHash sketch
	kmvSketch     *bool                                                            // if true, MinHash uses KMV algorithm, if false, MinHash uses Bottom-K algorithm
	windowSize    *int                                                             // length of query reads (used during alignment subcommand), needed as window length should ~= read length
	jsThresh      *float64                                                         // minimum Jaccard similarity for LSH forest query
	msaDir        *string                                                          // directory containing the input MSA files
	msaList       []string                                                         // the collected MSA files
	outDir        *string                                                          // directory to save index files and log to
	defaultOutDir = "./groot-index-" + string(time.Now().Format("20060102150405")) // a default dir to store the index files
)

// the index command (used by cobra)
var indexCmd = &cobra.Command{
	Use:   "index",
	Short: "Convert a set of clustered reference sequences to variation graphs and then index them",
	Long:  `Convert a set of clustered reference sequences to variation graphs and then index them`,
	Run: func(cmd *cobra.Command, args []string) {
		runIndex()
	},
	PreRunE: func(cmd *cobra.Command, args []string) error {
		return misc.CheckRequiredFlags(cmd.Flags())
	},
}

// a function to initialise the command line arguments
func init() {
	kmerSize = indexCmd.Flags().IntP("kmerSize", "k", 21, "size of k-mer")
	sketchSize = indexCmd.Flags().IntP("sketchSize", "s", 42, "size of MinHash sketch")
	kmvSketch = indexCmd.Flags().Bool("kmvSketch", false, "if set, MinHash uses KMV algorithm, otherwise MinHash uses Bottom-K algorithm")
	windowSize = indexCmd.Flags().IntP("windowSize", "w", 100, "size of window to sketch graph traversals with")
	jsThresh = indexCmd.Flags().Float64P("jsThresh", "j", 0.99, "minimum Jaccard similarity for a seed to be recorded")
	msaDir = indexCmd.Flags().StringP("msaDir", "i", "", "directory containing the clustered references (MSA files) - required")
	outDir = indexCmd.PersistentFlags().StringP("outDir", "o", defaultOutDir, "directory to save index files to")
	indexCmd.MarkFlagRequired("msaDir")
	RootCmd.AddCommand(indexCmd)
}

// runIndex is the main function for the index sub-command
func runIndex() {

	// set up profiling
	if *profiling == true {
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

	// start the index  sub command
	log.Printf("i am groot (version %s)", version.VERSION)
	log.Printf("starting the index subcommand")

	// check the supplied files and then log some stuff
	log.Printf("checking parameters...")
	misc.ErrorCheck(indexParamCheck())
	log.Printf("\tprocessors: %d", *proc)
	log.Printf("\tk-mer size: %d", *kmerSize)
	log.Printf("\tsketch size: %d", *sketchSize)
	if *kmvSketch {
		log.Printf("\tMinHash algorithm: K-Minimum Values")
	} else {
		log.Printf("\tMinHash algorithm: Bottom-K")
	}
	log.Printf("\tgraph window size: %d", *windowSize)
	log.Printf("\tnumber of MSA files found: %d", len(msaList))

	// record the runtime information for the index sub command
	info := &pipeline.Info{
		Version: version.VERSION,
	}
	ic := &pipeline.IndexCmd{
		KmerSize:   *kmerSize,
		SketchSize: *sketchSize,
		KMVsketch:  *kmvSketch,
		JSthresh:   *jsThresh,
		WindowSize: *windowSize,
		IndexDir:   *outDir,
	}
	info.Index = ic

	misc.ErrorCheck(info.Dump(*outDir + "/index.info"))

	// create the pipeline
	log.Printf("initialising indexing pipeline...")
	indexingPipeline := pipeline.NewPipeline()

	// initialise processes
	log.Printf("\tinitialising the processes")
	msaConverter := pipeline.NewMSAconverter(info)
	graphSketcher := pipeline.NewGraphSketcher(info)
	sketchIndexer := pipeline.NewSketchIndexer(info)

	// connect the pipeline processes
	log.Printf("\tconnecting data streams")
	msaConverter.Connect(msaList)
	graphSketcher.Connect(msaConverter)
	sketchIndexer.Connect(graphSketcher)

	// submit each process to the pipeline and run it
	indexingPipeline.AddProcesses(msaConverter, graphSketcher, sketchIndexer)
	log.Printf("\tnumber of processes added to the indexing pipeline: %d\n", indexingPipeline.GetNumProcesses())
	log.Print("creating graphs, sketching traversals and indexing...")
	indexingPipeline.Run()
	log.Printf("saved index files to \"%v\"...", *outDir)
	log.Println("finished")
}

// indexParamCheck is a function to check user supplied parameters
func indexParamCheck() error {
	log.Printf("\tdirectory containing MSA files: %v", *msaDir)
	misc.ErrorCheck(misc.CheckDir(*msaDir))
	// check the we have received some MSA files
	msas, err := filepath.Glob(*msaDir + "/*.msa")
	if err != nil {
		return fmt.Errorf("can't find any MSAs in the supplied directory")
	}
	log.Printf("\tnumber of MSA files: %d", len(msas))
	for _, msa := range msas {
		misc.ErrorCheck(misc.CheckFile(msa))
		msaList = append(msaList, msa)
	}
	// TODO: check the supplied arguments to make sure they don't conflict with each other eg:
	if *kmerSize > *windowSize {
		return fmt.Errorf("supplied k-mer size greater than read length")
	}
	// setup the outDir
	if _, err := os.Stat(*outDir); os.IsNotExist(err) {
		if err := os.MkdirAll(*outDir, 0700); err != nil {
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
