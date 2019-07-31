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
	kmerSize   *int     // size of k-mer
	sketchSize *int     // size of MinHash sketch
	windowSize *int     // length of query reads (used during alignment subcommand), needed as window length should ~= read length
	jsThresh   *float64 // minimum Jaccard similarity for LSH forest query
	kmvSketch  *bool    // use the K-Minimum Values MinHash algorithm instead of the K-Hash Functions
	msaDir     *string  // directory containing the input MSA files
	msaList    []string // the collected MSA files
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
	windowSize = indexCmd.Flags().IntP("windowSize", "w", 100, "size of window to sketch graph traversals with")
	jsThresh = indexCmd.Flags().Float64P("jsThresh", "j", 0.99, "minimum Jaccard similarity for a seed to be recorded")
	kmvSketch = indexCmd.Flags().Bool("kmv", false, "use the KMV MinHash algorithm instead of KHF")
	msaDir = indexCmd.Flags().StringP("msaDir", "m", "", "directory containing the clustered references (MSA files) - required")
	indexCmd.MarkFlagRequired("msaDir")
	RootCmd.AddCommand(indexCmd)
}

// runIndex is the main function for the index sub-command
func runIndex() {

	// set up profiling
	if *profiling == true {
		defer profile.Start(profile.MemProfile, profile.ProfilePath("./")).Stop()
		//defer profile.Start(profile.ProfilePath("./")).Stop()
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
	start := time.Now()
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
		log.Printf("\tMinHash algorithm: K-Hash Functions")
	}
	log.Printf("\tgraph window size: %d", *windowSize)
	log.Printf("\tminimum Jaccard similarity: %0.2f", *jsThresh)

	// record the runtime information for the index sub command
	info := &pipeline.Info{
		Version: version.VERSION,
	}
	info.Index = pipeline.IndexCmd{
		KmerSize:   *kmerSize,
		SketchSize: *sketchSize,
		KMVsketch:  *kmvSketch,
		JSthresh:   *jsThresh,
		WindowSize: *windowSize,
		IndexDir:   *indexDir,
	}

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
	log.Printf("writing index files in \"%v\"...", *indexDir)
	misc.ErrorCheck(info.SaveDB(*indexDir + "/groot.lshf"))
	misc.ErrorCheck(info.Dump(*indexDir + "/groot.gg"))
	log.Printf("finished in %s", time.Since(start))
}

// indexParamCheck is a function to check user supplied parameters
func indexParamCheck() error {

	// check the supplied directory is accessible etc.
	log.Printf("\tdirectory containing MSA files: %v", *msaDir)
	misc.ErrorCheck(misc.CheckDir(*msaDir))

	// check there are some files with the msa extension TODO: this can be better...
	msas, err := filepath.Glob(*msaDir + "/cluster*.msa")
	if err != nil {
		return fmt.Errorf("no MSA files in the supplied directory (must be named cluster-DD.msa)")
	}

	// check the file accessibility and add to the pile
	for _, msa := range msas {
		misc.ErrorCheck(misc.CheckFile(msa))
		msaList = append(msaList, msa)
	}
	if len(msas) == 0 {
		return fmt.Errorf("no MSA files found that passed the file checks (make sure filenames follow 'clusterXXX.msa' convention)")
	}
	log.Printf("\tnumber of MSA files: %d", len(msas))

	// TODO: check the supplied arguments to make sure they don't conflict with each other eg:
	if *kmerSize > *windowSize {
		return fmt.Errorf("supplied k-mer size greater than read length")
	}
	// setup the indexDir
	if _, err := os.Stat(*indexDir); os.IsNotExist(err) {
		if err := os.MkdirAll(*indexDir, 0700); err != nil {
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
