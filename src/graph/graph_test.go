package graph

import (
	"log"
	"os"
	"path/filepath"
	"testing"

	"github.com/will-rowe/bg/src/markov"
	"github.com/will-rowe/gfa"
)

var (
	inputFile  = "./test.gfa"
	inputFile2 = "./test.msa"
	windowSize = 150
	kSize      = 7
	sigSize    = 128
	blaB10     = []byte("ATGAAAGGATTAAAAGGGCTATTGGTTCTGGCTTTAGGCTTTACAGGACTACAGGTTTTTGGGCAACAGAACCCTGATATTAAAATTGAAAAATTAAAAGATAATTTATACGTCTATACAACCTATAATACCTTCAAAGGAACTAAATATGCGGCTAATGCGGTATATATGGTAACCGATAAAGGAGTAGTGGTTATAGACTCTCCATGGGGAGAAGATAAATTTAAAAGTTTTACAGACGAGATTTATAAAAAGCACGGAAAGAAAGTTATCATGAACATTGCAACCCACTCTCATGATGATAGAGCCGGAGGTCTTGAATATTTTGGTAAACTAGGTGCAAAAACTTATTCTACTAAAATGACAGATTCTATTTTAGCAAAAGAGAATAAGCCAAGAGCAAAGTACACTTTTGATAATAATAAATCTTTTAAAGTAGGAAAGACTGAGTTTCAGGTTTATTATCCGGGAAAAGGTCATACAGCAGATAATGTGGTTGTGTGGTTTCCTAAAGACAAAGTATTAGTAGGAGGCTGCATTGTAAAAAGTGGTGATTCGAAAGACCTTGGGTTTATTGGGGAAGCTTATGTAAACGACTGGACACAGTCCATACACAACATTCAGCAGAAATTTCCCTATGTTCAGTATGTCGTTGCAGGTCATGACGACTGGAAAGATCAAACATCAATACAACATACACTGGATTTAATCAGTGAATATCAACAAAAACAAAAGGCTTCAAATTAA")
)

func loadMSA() *gfa.GFA {
	// load the MSA
	msa, _ := gfa.ReadMSA(inputFile2)
	// convert the MSA to a GFA instance
	myGFA, err := gfa.MSA2GFA(msa)
	if err != nil {
		log.Fatal(err)
	}
	return myGFA
}

// test CreateGrootGraph
func TestCreateGrootGraph(t *testing.T) {
	myGFA, err := LoadGFA(inputFile)
	if err != nil {
		t.Fatal(err)
	}
	_, err = CreateGrootGraph(myGFA, 1)
	if err != nil {
		t.Fatal(err)
	}
}

// test Graph2Seq
func TestGraph2Seqs(t *testing.T) {
	t.Log("replace")
}

// test WindowGraph
func TestWindowGraph(t *testing.T) {
	myGFA := loadMSA()
	grootGraph, err := CreateGrootGraph(myGFA, 1)
	if err != nil {
		t.Fatal(err)
	}
	counter := 0
	for window := range grootGraph.WindowGraph(windowSize, kSize, sigSize) {
		//t.Log(window)
		_ = window
		counter++
	}
	t.Log("number of windows with unique signatures: ", counter)
}

/*
// test ChainSegments
func TestChainSegments(t *testing.T) {
	myGFA := loadMSA()
	grootGraph, err := CreateGrootGraph(myGFA, 1)
	if err != nil {
		t.Fatal(err)
	}
	chains, err := grootGraph.chainSegments(0, 3)
	if err != nil {
		t.Fatal(err)
	}
	counter := 0
	for _, chain := range chains {
		counter++
		t.Log(chain)
	}
	if counter != 2 {
		t.Fatal("two chains should be formed from this node")
	}
	err = grootGraph.BuildMarkovModel(1)
	if err != nil {
		t.Fatal(err)
	}
}
*/

// test FindMarkovPaths
func TestFindMarkovPaths(t *testing.T) {
	myGFA, err := LoadGFA("test2.gfa")
	if err != nil {
		t.Fatal(err)
	}
	grootGraph, err := CreateGrootGraph(myGFA, 1)
	if err != nil {
		t.Fatal(err)
	}
	chainOrder := 7
	chain := markov.NewChain(chainOrder)
	err = grootGraph.BuildMarkovChain(chain)
	if err != nil {
		t.Fatal(err)
	}
	err = grootGraph.FindMarkovPaths(chain)
	if err != nil {
		t.Fatal(err)
	}
	err = grootGraph.ProcessMarkovPaths()
	if err != nil {
		t.Fatal(err)
	}

}

// test GraphStore dump/load
func TestGraphStore(t *testing.T) {
	myGFA, err := LoadGFA(inputFile)
	if err != nil {
		t.Fatal(err)
	}
	grootGraph, err := CreateGrootGraph(myGFA, 1)
	if err != nil {
		t.Fatal(err)
	}
	graphStore := make(GraphStore)
	graphStore[0] = grootGraph
	if err := graphStore.Dump("./test.grootGraph"); err != nil {
		t.Fatal(err)
	}
	if err := graphStore.Load("./test.grootGraph"); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove("./test.grootGraph"); err != nil {
		t.Fatal(err)
	}
}

// test SaveGraphAsGFA to save a gfa
func TestGraphDump(t *testing.T) {
	myGFA, err := LoadGFA(inputFile)
	if err != nil {
		t.Fatal(err)
	}
	grootGraph, err := CreateGrootGraph(myGFA, 1)
	if err != nil {
		t.Fatal(err)
	}
	// add a dummy read so that the graph will write
	grootGraph.SortedNodes[0].IncrementKmerFreq(100.0)
	written, err := grootGraph.SaveGraphAsGFA(".")
	if err != nil {
		t.Fatal(err)
	}
	if written != 1 {
		t.Fatal("graph not written as gfa file")
	}
	files, err := filepath.Glob("*-groot-graph.gfa")
	if err != nil {
		panic(err)
	}
	for _, file := range files {
		if err := os.Remove(file); err != nil {
			t.Fatal(err)
		}
	}

}
