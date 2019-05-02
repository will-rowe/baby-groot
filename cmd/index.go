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
	"strings"
	"sync"
	"time"

	"github.com/biogo/biogo/seq/multi"
	"github.com/pkg/profile"
	"github.com/spf13/cobra"
	"github.com/will-rowe/baby-groot/src/graph"
	"github.com/will-rowe/baby-groot/src/lshForest"
	"github.com/will-rowe/baby-groot/src/misc"
	"github.com/will-rowe/baby-groot/src/seqio"
	"github.com/will-rowe/baby-groot/src/stream"
	"github.com/will-rowe/baby-groot/src/version"
	"github.com/will-rowe/gfa"
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

//  a function to check user supplied parameters
func indexParamCheck() error {
	if *msaDir == "" {
		misc.ErrorCheck(fmt.Errorf("no MSA directory specified - run `groot index --help` for more info on the command"))
	}
	if _, err := os.Stat(*msaDir); os.IsNotExist(err) {
		return fmt.Errorf("can't find specified MSA directory")
	}
	// check the we have received some MSA files TODO: could do with a better way of collecting these
	err := filepath.Walk(*msaDir, func(path string, f os.FileInfo, err error) error {
		// ignore dot files
		if f.Name()[0] == 46 {
			return nil
		}
		if len(strings.Split(path, ".msa")) == 2 {
			msaList = append(msaList, path)
		}
		return nil
	})
	misc.ErrorCheck(err)
	if len(msaList) == 0 {
		return fmt.Errorf("no MSA files (.msa) found in the supplied directory")
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

/*
  The main function for the index command
*/
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
	// start sub command
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
	///////////////////////////////////////////////////////////////////////////////////////
	log.Printf("building groot graphs...")
	// process each msa in a go routine
	var wg sync.WaitGroup
	graphChan := make(chan *graph.GrootGraph)
	for i, msaFile := range msaList {
		// load the MSA outside of the go-routine to prevent 'too many open files' error on OSX
		msa, err := gfa.ReadMSA(msaFile)
		misc.ErrorCheck(err)
		wg.Add(1)
		go func(msaID int, msa *multi.Multi) {
			defer wg.Done()
			// convert the MSA to a GFA instance
			newGFA, err := gfa.MSA2GFA(msa)
			misc.ErrorCheck(err)
			// create a GrootGraph
			grootGraph, err := graph.CreateGrootGraph(newGFA, msaID)
			if err != nil {
				log.Fatal(err)
			}
			graphChan <- grootGraph
		}(i, msa)
	}
	go func() {
		wg.Wait()
		close(graphChan)
	}()
	///////////////////////////////////////////////////////////////////////////////////////
	// collect and store the GrootGraphs
	graphStore := make(graph.GraphStore)
	for graph := range graphChan {
		graphStore[graph.GraphID] = graph
	}
	if len(graphStore) == 0 {
		misc.ErrorCheck(fmt.Errorf("could not create any graphs"))
	}
	log.Printf("\tnumber of groot graphs built: %d", len(graphStore))
	///////////////////////////////////////////////////////////////////////////////////////
	log.Printf("windowing graphs and generating MinHash sketches...")
	// process each graph in a go routine
	windowChan := make(chan *seqio.Key)
	for _, grootGraph := range graphStore {
		wg.Add(1)
		go func(grootGraph *graph.GrootGraph) {
			defer wg.Done()
			keyChecker := make(map[string]string)
			// create sketch for each window in the graph
			for window := range grootGraph.WindowGraph(*windowSize, *kmerSize, *sketchSize, *kmvSketch) {

				// there may be multiple copies of the same window key
				// - one graph+node+offset can have several subpaths to window
				// - or windows can be derived from identical regions of the graph that multiple sequences share
				// the keyCheck map will keep track of seen window keys
				window.StringifiedKey = fmt.Sprintf("g%dn%do%d", window.GraphID, window.Node, window.OffSet)
				newSubPath := misc.Stringify(window.SubPath)
				if seenSubPath, ok := keyChecker[window.StringifiedKey]; ok {

					// check if the subpath is identical
					if newSubPath != seenSubPath {
						// if it is different, we have multiple unique traversals from one node, so adjust the new key and send the window
						window.StringifiedKey = fmt.Sprintf("g%dn%do%dp%d", window.GraphID, window.Node, window.OffSet, window.Ref)
					} else {
						// if it is identical, we don't need another sketch from the same subpath
						continue
					}
				} else {
					keyChecker[window.StringifiedKey] = newSubPath
				}

				// send the windows for this graph onto the single collection channel
				windowChan <- window
			}
		}(grootGraph)
	}
	go func() {
		wg.Wait()
		close(windowChan)
	}()
	///////////////////////////////////////////////////////////////////////////////////////
	log.Printf("running LSH forest...\n")
	database := lshForest.NewLSHforest(*sketchSize, *jsThresh)
	var sketchCount int = 0
	for window := range windowChan {
		sketchCount++
		// add the sketch to the lshForest index
		misc.ErrorCheck(database.Add(window))
	}
	numHF, numBucks := database.Settings()
	log.Printf("\tnumber of hash functions per bucket: %d\n", numHF)
	log.Printf("\tnumber of buckets: %d\n", numBucks)
	log.Printf("\tnumber of sketches added: %d\n", sketchCount)
	///////////////////////////////////////////////////////////////////////////////////////
	// record runtime info
	info := &stream.PipelineInfo{Version: version.VERSION, Ksize: *kmerSize, SigSize: *sketchSize, KMVsketch: *kmvSketch, JSthresh: *jsThresh, WindowSize: *windowSize}
	// save the index files
	log.Printf("saving index files to \"%v\"...", *outDir)
	misc.ErrorCheck(info.Dump(*outDir + "/index.info"))
	log.Printf("\tsaved runtime info")
	misc.ErrorCheck(graphStore.Dump(*outDir + "/index.graph"))
	log.Printf("\tsaved groot graphs")
	misc.ErrorCheck(database.Dump(*outDir + "/index.sketches"))
	log.Printf("\tsaved MinHash sketches")
	log.Println("finished")
}
