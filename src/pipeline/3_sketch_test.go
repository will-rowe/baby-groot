package pipeline

import (
	"testing"

	"github.com/will-rowe/baby-groot/src/graph"
	"github.com/will-rowe/baby-groot/src/lshforest"
)

func TestSketching(t *testing.T) {
	// load the files from the previous tests
	testParameters := new(Info)
	if err := testParameters.Load("test-data/tmp/index.info"); err != nil {
		t.Fatal(err)
	}
	graphStore := make(graph.Store)
	if err := graphStore.Load("test-data/tmp/index.graph"); err != nil {
		t.Fatal(err)
	}
	testParameters.Store = graphStore
	if len(graphStore) != 1 {
		t.Fatal("incorrect number of graphs loaded")
	}
	database := lshforest.NewLSHforest(testParameters.Index.SketchSize, testParameters.Index.JSthresh)
	if err := database.Load("test-data/tmp/index.sketches"); err != nil {
		t.Fatal(err)
	}
	database.Index()
	testParameters.Db = database
	// run the pipeline
	sketchingPipeline := NewPipeline()
	dataStream := NewDataStreamer(testParameters)
	fastqHandler := NewFastqHandler(testParameters)
	fastqChecker := NewFastqChecker(testParameters)
	readMapper := NewDbQuerier(testParameters)
	graphPruner := NewGraphPruner(testParameters)
	dataStream.Connect(fastq)
	fastqHandler.Connect(dataStream)
	fastqChecker.Connect(fastqHandler)
	readMapper.Connect(fastqChecker)
	graphPruner.Connect(readMapper)
	sketchingPipeline.AddProcesses(dataStream, fastqHandler, fastqChecker, readMapper, graphPruner)
	if sketchingPipeline.GetNumProcesses() != 5 {
		t.Fatal("wrong number of processes in pipeline")
	}
	sketchingPipeline.Run()
	// check that the right number of reads mapped
	readStats := readMapper.CollectReadStats()
	t.Logf("total number of test reads = %d", readStats[0])
	t.Logf("number which mapped = %d", readStats[1])

	// check that we got the right allele in the approximately weighted graph
	foundPaths := graphPruner.CollectOutput()
	correctPath := false
	for _, path := range foundPaths {
		if path == "argannot~~~(Bla)OXA-90~~~EU547443:1-825" {
			correctPath = true
		}
	}
	if correctPath != true {
		t.Fatal("sketching did not identify correct allele in graph")
	}
}
