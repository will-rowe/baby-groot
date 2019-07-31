package cmd

import (
	"fmt"
	"log"
	"os"
	"runtime"
	"time"

	"github.com/will-rowe/baby-groot/src/lshforest"

	"github.com/pkg/profile"
	"github.com/spf13/cobra"
	"github.com/will-rowe/baby-groot/src/misc"
	"github.com/will-rowe/baby-groot/src/pipeline"
	"github.com/will-rowe/baby-groot/src/version"
)

// MINION_MULTIPLIER is a temporary way of setting the number of minions used for mapping reads
// it multiplies the number of available processors, yielding the number of concurrent processes for mapping reads
// at some stage, I want to benchmark how many minions can be mapping reads whilst keeping RAM usage reasonable
// there is also the bottleneck of accessing the LSH Forest (and to some extent the graphs)
const MINION_MULTIPLIER = 1

// the command line arguments
var (
	fastq           *[]string                                                         // list of FASTQ files to align
	fasta           *bool                                                             // flag to treat input as fasta sequences
	bloomFilter     *bool                                                             // flag to use a bloom filter in order to prevent unique k-mers being used during sketching
	minKmerCoverage *float64                                                          // the minimum k-mer coverage per base of a segment
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
	fastq = sketchCmd.Flags().StringSliceP("fastq", "f", []string{}, "FASTQ file(s) to align")
	fasta = sketchCmd.Flags().Bool("fasta", false, "if set, the input will be treated as fasta sequence(s) (experimental feature)")
	bloomFilter = sketchCmd.Flags().Bool("bloomFilter", false, "if set, a bloom filter will be used to stop unique k-mers being added to sketches")
	minKmerCoverage = sketchCmd.Flags().Float64P("minKmerCov", "c", 1.0, "minimum k-mer coverage per segment base")
	graphDir = sketchCmd.PersistentFlags().StringP("graphDir", "g", defaultGraphDir, "directory to save variation graphs to")
	RootCmd.AddCommand(sketchCmd)
}

// runAlign is the main function for the sketch sub-command
func runSketch() {

	// set up profiling
	if *profiling {
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

	// start the sketch sub command
	start := time.Now()
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
	log.Printf("\tminimum k-mer coverage: %d", *minKmerCoverage)
	log.Printf("\tprocessors: %d", *proc)
	for _, file := range *fastq {
		log.Printf("\tinput file: %v", file)
	}
	if *fasta {
		log.Print("\tinput file format: fasta")
	}
	log.Print("loading the index information...")
	info := new(pipeline.Info)
	misc.ErrorCheck(info.Load(*indexDir + "/groot.gg"))
	if info.Version != version.VERSION {
		misc.ErrorCheck(fmt.Errorf("the groot index was created with a different version of groot (you are currently using version %v)", version.VERSION))
	}
	log.Printf("\tk-mer size: %d\n", info.KmerSize)
	log.Printf("\tsketch size: %d\n", info.SketchSize)
	log.Printf("\tJaccard similarity theshold: %0.2f\n", info.JSthresh)
	log.Printf("\twindow size used in indexing: %d\n", info.WindowSize)
	log.Print("loading the graphs...")
	log.Printf("\tnumber of variation graphs: %d\n", len(info.Store))
	log.Print("loading the LSH Forest...")
	lshf := lshforest.NewLSHforest(info.SketchSize, info.JSthresh)
	misc.ErrorCheck(lshf.Load(*indexDir + "/groot.lshf"))
	info.AttachDB(lshf)
	numHF, numBucks := info.GetDBinfo()
	log.Printf("\tnumber of LSH Forest buckets: %d\n", numBucks)
	log.Printf("\tnumber of hash functions per bucket: %d\n", numHF)

	if *profiling {
		log.Printf("\tloaded lshf file -> current memory usage %v", misc.PrintMemUsage())
		runtime.GC()
	}

	// add the sketch information to the existing groot runtime information
	info.NumProc = *proc * MINION_MULTIPLIER
	info.Profiling = *profiling
	info.Sketch = pipeline.SketchCmd{
		Fasta:           *fasta,
		BloomFilter:     *bloomFilter,
		MinKmerCoverage: *minKmerCoverage,
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

	// once the sketching pipeline is finished, process the graph store and write the graphs to disk
	if len(info.Store) != 0 {
		log.Printf("saving graphs...\n")
		stats := readMapper.CollectReadStats()
		for graphID, g := range info.Store {
			fileName := fmt.Sprintf("%v/groot-graph-%d.gfa", *graphDir, graphID)
			_, err := g.SaveGraphAsGFA(fileName, stats[3])
			misc.ErrorCheck(err)
		}
	}
	log.Printf("finished in %s", time.Since(start))
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
	misc.ErrorCheck(misc.CheckFile(*indexDir + "/groot.gg"))
	misc.ErrorCheck(misc.CheckFile(*indexDir + "/groot.lshf"))

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
