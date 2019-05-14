package pipeline

import (
	"os"
	"testing"
)

func TestHaplotyping(t *testing.T) {

	// load the files from the previous tests
	testParameters := new(Info)
	if err := testParameters.Load("test-data/tmp/index.info"); err != nil {
		t.Fatal(err)
	}
	haplotypingPipeline := NewPipeline()
	gfaReader := NewGFAreader(testParameters)
	emPathFinder := NewEMpathFinder(testParameters)
	haploParser := NewHaplotypeParser(testParameters)
	gfaReader.Connect(gfaList)
	emPathFinder.Connect(gfaReader)
	haploParser.Connect(emPathFinder)
	haplotypingPipeline.AddProcesses(gfaReader, emPathFinder, haploParser)
	if haplotypingPipeline.GetNumProcesses() != 3 {
		t.Fatal("wrong number of processes in pipeline")
	}
	haplotypingPipeline.Run()

	foundPaths := haploParser.CollectOutput()
	correctPath := false
	for _, path := range foundPaths {
		if path == "argannot~~~(Bla)OXA-90~~~EU547443:1-825" {
			correctPath = true
		}
	}
	if correctPath != true {
		t.Fatal("haplotyping did not identify correct allele in graph")
	}

	// remove the tmp files from all tests
	if err := os.Remove("test-data/tmp/index.info"); err != nil {
		t.Fatal("indexing did not create info file: ", err)
	}
	if err := os.Remove("test-data/tmp/index.graph"); err != nil {
		t.Fatal("indexing did not create graph file: ", err)
	}
	if err := os.Remove("test-data/tmp/index.sketches"); err != nil {
		t.Fatal("indexing did not create sketch file: ", err)
	}
	if err := os.Remove("test-data/tmp/groot-graph-0.gfa"); err != nil {
		t.Fatal("sketching did not create graph file: ", err)
	}
	if err := os.Remove("test-data/tmp/groot-graph-0-haplotype.gfa.fna"); err != nil {
		t.Fatal("haplotyping did not create fasta file: ", err)
	}
	if err := os.RemoveAll("test-data/tmp"); err != nil {
		t.Fatal("tests could not remove tmp directory")
	}

}
