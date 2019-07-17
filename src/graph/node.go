package graph

import (
	"fmt"
	"sync"

	"github.com/will-rowe/baby-groot/src/bitvector"
)

// Nodes is a type that implements the sort interface
type Nodes []uint64

func (a Nodes) Len() int           { return len(a) }
func (a Nodes) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a Nodes) Less(i, j int) bool { return a[i] < a[j] }

// GrootGraphNode is a GFA segment (plus the extra info from path, links etc.)
type GrootGraphNode struct {
	sync.RWMutex // lock the node for write access
	SegmentID    uint64
	Sequence     []byte
	OutEdges     Nodes
	PathIDs      []int               // PathIDs are the lookup IDs to the linear reference sequences that use this segment (value corresponds to key in GrootGraph.Paths)
	KmerFreq     float64             // KmerFreq is the number of k-mers belonging to this node
	Coverage     bitvector.BitVector // Coverage is a bit vector that tracks which bases in this GFA segment are covered by mapped reads
}

// IncrementKmerFreq is a method to increment a node's k-mer count
func (GrootGraphNode *GrootGraphNode) IncrementKmerFreq(increment float64) error {
	if increment <= 0.0 {
		return fmt.Errorf("positive increment not received: %f", increment)
	}
	GrootGraphNode.Lock()
	GrootGraphNode.KmerFreq += increment
	GrootGraphNode.Unlock()
	return nil
}

// DecrementKmerFreq is a method to decrement a node's k-mer count
func (GrootGraphNode *GrootGraphNode) DecrementKmerFreq(decrement float64) error {
	if decrement <= 0.0 {
		return fmt.Errorf("positive decrement not received: %f", decrement)
	}
	GrootGraphNode.Lock()
	GrootGraphNode.KmerFreq -= decrement
	if GrootGraphNode.KmerFreq < 0 {
		GrootGraphNode.KmerFreq = 0
	}
	GrootGraphNode.Unlock()
	return nil
}

// AddCoverage is a method to mark a region within a node as covered, given the start position and the number of bases to cover
func (GrootGraphNode *GrootGraphNode) AddCoverage(start, numberOfBases int) {
	// if numberOfBases is greater than the sequence length, just mark to the end of the node sequence
	if numberOfBases >= len(GrootGraphNode.Sequence) {
		numberOfBases = len(GrootGraphNode.Sequence)
	}
	GrootGraphNode.Lock()
	for i := start; i < numberOfBases; i++ {
		GrootGraphNode.Coverage.Add(i)
	}
	GrootGraphNode.Unlock()
}
