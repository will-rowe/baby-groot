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
	"path/filepath"
	"runtime"
	"sync"
	"time"

	"github.com/pkg/profile"
	"github.com/spf13/cobra"
	"github.com/will-rowe/baby-groot/src/graph"
	"github.com/will-rowe/baby-groot/src/markov"
	"github.com/will-rowe/baby-groot/src/misc"
	"github.com/will-rowe/baby-groot/src/pipeline"
	"github.com/will-rowe/baby-groot/src/version"
	"github.com/will-rowe/gfa"
)

// the command line arguments
var (
	graphDirectory  *string                                                              // directory containing the weighted variation graphs
	indexDirectory  *string                                                              // directory containing the index files
	haploDir        *string                                                              // directory to write haplotype files to
	graphList       []string                                                             // the collected GFA files
	bootstraps      *int                                                                 // the number of times to traverse the graph when finding markov paths
	scaling         *float64                                                             // the scaling factor to re-weight graph segments once they have been included in a path
	probability     *float64                                                             // the probability threshold for a path to be reported
	defaultHaploDir = "./groot-haplotype-" + string(time.Now().Format("20060102150405")) // a default haploDir
)

// the haplotype command
var haplotypeCmd = &cobra.Command{
	Use:   "haplotype",
	Short: "Call haplotypes based on approximately weighted variation graphs",
	Long:  `Call haplotypes based on approximately weighted variation graphs`,
	Run: func(cmd *cobra.Command, args []string) {
		runHaplotype()
	},
	PreRunE: func(cmd *cobra.Command, args []string) error {
		return misc.CheckRequiredFlags(cmd.Flags())
	},
}

/*
  A function to initialise the command line arguments
*/
func init() {
	graphDirectory = haplotypeCmd.Flags().StringP("graphDirectory", "g", "", "directory containing the weighted variation graphs - required")
	indexDirectory = haplotypeCmd.Flags().StringP("indexDirectory", "i", "", "directory containing the index files - required")
	haploDir = haplotypeCmd.PersistentFlags().StringP("haploDir", "o", defaultHaploDir, "directory to write haplotype files to")
	bootstraps = haplotypeCmd.Flags().IntP("bootstraps", "b", 100, "number of times to traverse the graph when finding markov paths")
	scaling = haplotypeCmd.Flags().Float64P("scaling", "s", 0.001, "scaling factor to re-weight graph segments once they have been included in a path")
	probability = haplotypeCmd.Flags().Float64P("probability", "z", 0.97, "probability threshold for a path to be reported")
	haplotypeCmd.MarkFlagRequired("graphDirectory")
	haplotypeCmd.MarkFlagRequired("indexDirectory")
	RootCmd.AddCommand(haplotypeCmd)
}

/*
  A function to check user supplied parameters
*/
func haplotypeParamCheck() error {
	// check the index directory and has files
	if *indexDirectory == "" {
		misc.ErrorCheck(errors.New("need to specify the directory where the index files are"))
	}
	if _, err := os.Stat(*indexDirectory); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("index directory does not exist: %v", *indexDirectory)
		}
		return fmt.Errorf("can't access an index directory (check permissions): %v", indexDirectory)
	}
	indexFiles := [3]string{"/index.graph", "/index.info", "/index.sketches"}
	for _, indexFile := range indexFiles {
		if _, err := os.Stat(*indexDirectory + indexFile); err != nil {
			if os.IsNotExist(err) {
				return fmt.Errorf("index file does not exist: %v", indexFile)
			}
			return fmt.Errorf("can't access an index file (check permissions): %v", indexFile)
		}
	}
	info := new(pipeline.Info)
	misc.ErrorCheck(info.Load(*indexDirectory + "/index.info"))
	if info.Version != version.VERSION {
		return fmt.Errorf("the groot index was created with a different version of groot (you are currently using version %v)", version.VERSION)
	}
	// check the supplied graphs
	if *graphDirectory == "" {
		misc.ErrorCheck(errors.New("need to specify the directory where the graphs are"))
	}
	if _, err := os.Stat(*graphDirectory); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("graph directory does not exist: %v", *indexDirectory)
		}
		return fmt.Errorf("can't access the graph directory (check permissions): %v", indexDirectory)
	}
	graphs, err := filepath.Glob(*graphDirectory + "/groot-graph-*.gfa")
	if err != nil {
		return fmt.Errorf("can't find any graphs in the supplied graph directory")
	}
	for _, graph := range graphs {
		if _, err := os.Stat(graph); err != nil {
			if os.IsNotExist(err) {
				return fmt.Errorf("graph file does not exist: %v", graph)
			}
			return fmt.Errorf("can't access graph file (check permissions): %v", graph)
		}
		graphList = append(graphList, graph)
	}
	// setup the haploDir
	if _, err := os.Stat(*haploDir); os.IsNotExist(err) {
		if err := os.MkdirAll(*haploDir, 0700); err != nil {
			return fmt.Errorf("can't create specified output directory")
		}
	}
	// check the scaling factor
	if *scaling > 1.0 || *scaling < 0.0 {
		return fmt.Errorf("scaling factor must be between 0.0 and 1.0")
	}
	// check the probability
	if *probability > 1.0 || *probability < 0.0 {
		return fmt.Errorf("probability must be between 0.0 and 1.0")
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
func runHaplotype() {
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
	log.Printf("starting the haplotype subcommand")
	// check the supplied files and then log some stuff
	log.Printf("checking parameters...")
	misc.ErrorCheck(haplotypeParamCheck())
	log.Printf("\tgraph bootstraps: %d", *bootstraps)
	if *scaling == 0.0 {
		log.Printf("\tscaling factor for node re-weighting: deactivated")
	} else {
		log.Printf("\tscaling factor for node re-weighting: %0.4f", *scaling)
	}
	log.Printf("\tprobability threshold for reporting markov paths: %0.2f", *probability)
	log.Printf("\tprocessors: %d", *proc)
	log.Print("loading index information...")
	info := new(pipeline.Info)
	misc.ErrorCheck(info.Load(*indexDirectory + "/index.info"))
	log.Printf("\tk-mer size: %d\n", info.KmerSize)
	log.Printf("\tsketch size: %d\n", info.SketchSize)
	log.Printf("\tJaccard similarity theshold: %0.2f\n", info.JSthresh)
	log.Printf("\twindow size used in indexing: %d\n", info.WindowSize)
	log.Print("loading the graphs...")
	log.Printf("\tnumber of weighted GFAs for haplotyping: %d", len(graphList))
	///////////////////////////////////////////////////////////////////////////////////////
	// process each graph in a go routine
	var wg sync.WaitGroup
	graphChan := make(chan *graph.GrootGraph)
	for i, gfaFile := range graphList {
		// load the GFA outside of the go-routine to prevent 'too many open files' error on OSX
		gfaObj, err := graph.LoadGFA(gfaFile)
		misc.ErrorCheck(err)
		wg.Add(1)
		go func(gfaID int, g *gfa.GFA) {
			defer wg.Done()
			// convert to GROOT graph
			grootGraph, err := graph.CreateGrootGraph(g, gfaID)
			if err != nil {
				log.Fatal(err)
			}
			graphChan <- grootGraph
		}(i, gfaObj)
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
	log.Printf("\tnumber of groot graphs built from GFAs: %d", len(graphStore))
	///////////////////////////////////////////////////////////////////////////////////////
	log.Printf("running MCMC on graphs...")
	// set up markov chain and get the pointer
	chainOrder := 7
	chain := markov.NewChain(chainOrder)
	for _, graph := range graphStore {
		err := graph.BuildMarkovChain(chain)
		misc.ErrorCheck(err)
	}

	///////////////////////////////////////////////////////////////////////////////////////
	log.Printf("finding best paths through graphs...")
	// get the best markov paths through the graph
	graphChan2 := make(chan *graph.GrootGraph)
	var wg2 sync.WaitGroup
	go func() {
		wg2.Wait()
		close(graphChan2)
	}()

	// process each graph in a go routine
	for _, receivedGraph := range graphStore {
		wg2.Add(1)
		go func(g *graph.GrootGraph) {
			defer wg2.Done()
			err := g.FindMarkovPaths(chain, *bootstraps, *scaling)
			misc.ErrorCheck(err)

			err = g.ProcessMarkovPaths(*probability)
			misc.ErrorCheck(err)

			paths, err := g.GetMarkovPaths()
			misc.ErrorCheck(err)
			log.Printf("\tgraph %d has %d markov paths passing thresholds", g.GraphID, len(paths))
			for _, path := range paths {
				log.Printf("\t- [%v]", path)
			}

			// replace the old paths with the new MCMC derived paths
			err = g.PathReplace()
			misc.ErrorCheck(err)

			// send the graph
			graphChan2 <- g
		}(receivedGraph)

	}

	// collect the graphs, print output and save them
	log.Printf("saving graphs to \"./%v/\"...", *haploDir)
	counter := 0
	for graph := range graphChan2 {

		// write the graph
		graph.GrootVersion = version.VERSION
		fileName := fmt.Sprintf("%v/groot-graph-%d.gfa", *haploDir, graph.GraphID)
		graphWritten, err := graph.SaveGraphAsGFA(fileName)
		misc.ErrorCheck(err)
		counter += graphWritten

		// write the sequences
		seqs, err := graph.Graph2Seqs()
		misc.ErrorCheck(err)
		fileName += ".fna"
		fh, err := os.Create(fileName)
		defer fh.Close()
		misc.ErrorCheck(err)
		for id, seq := range seqs {
			fmt.Fprintf(fh, ">%v\n%v\n", string(graph.Paths[id]), string(seq))
		}

	}
	log.Printf("\tnumber of graphs written to disk: %d\n", counter)
	log.Println("finished")
}
