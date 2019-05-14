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
	"github.com/will-rowe/baby-groot/src/graph"
	"github.com/will-rowe/baby-groot/src/misc"
	"github.com/will-rowe/baby-groot/src/pipeline"
	"github.com/will-rowe/baby-groot/src/version"
)

// the command line arguments
var (
	graphDirectory  *string                                                              // directory containing the weighted variation graphs
	indexDirectory  *string                                                              // directory containing the index files
	haploDir        *string                                                              // directory to write haplotype files to
	graphList       []string                                                             // the collected GFA files
	minIterations   *int                                                                 // minimum iterations for EM
	maxIterations   *int                                                                 // maximum iterations for EM
	cutOff          *float64                                                             // abundance cutoff for haplotypes
	probability     *float64                                                             // the probability threshold for a path to be reported
	defaultHaploDir = "./groot-haplotype-" + string(time.Now().Format("20060102150405")) // a default haploDir
)

// haplotypeCmd is used by cobra
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

// init the command line arguments
func init() {
	graphDirectory = haplotypeCmd.Flags().StringP("graphDirectory", "g", "", "directory containing the weighted variation graphs - required")
	indexDirectory = haplotypeCmd.Flags().StringP("indexDirectory", "i", "", "directory containing the index files - required")
	haploDir = haplotypeCmd.PersistentFlags().StringP("haploDir", "o", defaultHaploDir, "directory to write haplotype files to")
	minIterations = haplotypeCmd.PersistentFlags().IntP("minIterations", "m", 50, "minimum iterations for EM")
	maxIterations = haplotypeCmd.Flags().IntP("maxIterations", "n", 10000, "maximum iterations for EM")
	cutOff = haplotypeCmd.Flags().Float64P("cutOff", "c", 0.05, "abundance cutoff for calling haplotypes")
	haplotypeCmd.MarkFlagRequired("graphDirectory")
	haplotypeCmd.MarkFlagRequired("indexDirectory")
	RootCmd.AddCommand(haplotypeCmd)
}

// runHaplotype is the main function for the haplotype sub-command
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
	log.Printf("\tmin. iterations for EM: %d", *minIterations)
	log.Printf("\tmax. iterations for EM: %d", *maxIterations)
	log.Printf("\tabundance cut off reporting haplotypes: %0.2f", *cutOff)
	log.Printf("\tprocessors: %d", *proc)
	log.Print("loading index information...")
	info := new(pipeline.Info)
	misc.ErrorCheck(info.Load(*indexDirectory + "/index.info"))
	log.Printf("\tk-mer size: %d\n", info.Index.KmerSize)
	log.Printf("\tsketch size: %d\n", info.Index.SketchSize)
	log.Printf("\tJaccard similarity theshold: %0.2f\n", info.Index.JSthresh)
	log.Printf("\twindow size used in indexing: %d\n", info.Index.WindowSize)
	log.Print("loading the graphs...")
	log.Printf("\tnumber of weighted GFAs for haplotyping: %d", len(graphList))
	log.Print("predicting the best paths through the graphs...")

	// add the haplotype information to the existing groot runtime information
	hc := &pipeline.HaploCmd{
		MinIterations: *minIterations,
		MaxIterations: *maxIterations,
		HaploDir:      *haploDir,
	}
	info.Haplotype = hc

	// create a graphStore
	graphStore := make(graph.Store)
	info.Store = graphStore

	// create the pipeline
	log.Printf("initialising haplotype pipeline...")
	haplotypePipeline := pipeline.NewPipeline()

	// initialise processes
	log.Printf("\tinitialising the processes")
	gfaReader := pipeline.NewGFAreader(info)
	emPathFinder := pipeline.NewEMpathFinder(info)
	haploParser := pipeline.NewHaplotypeParser(info)

	// connect the pipeline processes
	log.Printf("\tconnecting data streams")
	gfaReader.Connect(graphList)
	emPathFinder.Connect(gfaReader)
	haploParser.Connect(emPathFinder)

	// submit each process to the pipeline and run it
	haplotypePipeline.AddProcesses(gfaReader, emPathFinder, haploParser)
	log.Printf("\tnumber of processes added to the haplotype pipeline: %d\n", haplotypePipeline.GetNumProcesses())
	haplotypePipeline.Run()
	log.Println("finished")
}

// haplotypeParamCheck is a function to check user supplied parameters
func haplotypeParamCheck() error {
	misc.ErrorCheck(misc.CheckDir(*indexDir))
	indexFiles := [3]string{"/index.graph", "/index.info", "/index.sketches"}
	for _, indexFile := range indexFiles {
		file := *indexDir + indexFile
		misc.ErrorCheck(misc.CheckFile(file))
	}
	info := new(pipeline.Info)
	misc.ErrorCheck(info.Load(*indexDirectory + "/index.info"))
	if info.Version != version.VERSION {
		return fmt.Errorf("the groot index was created with a different version of groot (you are currently using version %v)", version.VERSION)
	}
	misc.ErrorCheck(misc.CheckDir(*graphDirectory))
	graphs, err := filepath.Glob(*graphDirectory + "/groot-graph-*.gfa")
	if err != nil {
		return fmt.Errorf("can't find any graphs in the supplied graph directory")
	}
	for _, graph := range graphs {
		misc.ErrorCheck(misc.CheckFile(graph))
		graphList = append(graphList, graph)
	}
	// setup the haploDir
	if _, err := os.Stat(*haploDir); os.IsNotExist(err) {
		if err := os.MkdirAll(*haploDir, 0700); err != nil {
			return fmt.Errorf("can't create specified output directory")
		}
	}
	// check the probability
	if *cutOff > 1.0 || *cutOff < 0.0 {
		return fmt.Errorf("cutOff must be between 0.0 and 1.0")
	}
	// set number of processors to use
	if *proc <= 0 || *proc > runtime.NumCPU() {
		*proc = runtime.NumCPU()
	}
	runtime.GOMAXPROCS(*proc)
	return nil
}
