package minhash

import (
	"container/heap"
	"fmt"
	"sort"

	"github.com/will-rowe/ntHash"
)

// BottomKsketch is the bottom-k MinHash sketch of a set
type BottomKsketch struct {
	kmerSize    int
	sketchSize  int
	sketch      *intHeap
	BloomFilter *BloomFilter
}

// NewBottomKsketch is the constructor for a BottomKsketch
func NewBottomKsketch(k, s int, bf *BloomFilter) *BottomKsketch {
	return &BottomKsketch{
		kmerSize:    k,
		sketchSize:  s,
		sketch:      &intHeap{},
		BloomFilter: bf,
	}
}

// Add is a method to decompose a read to kmers, hash them and add any minimums to the sketch
func (BottomKsketch *BottomKsketch) Add(sequence []byte) error {
	if len(sequence) < BottomKsketch.kmerSize {
		return fmt.Errorf("sequence length (%d) is short than k-mer length (%d)", len(sequence), BottomKsketch.kmerSize)
	}
	// initiate the rolling ntHash
	hasher, err := ntHash.New(&sequence, BottomKsketch.kmerSize)
	if err != nil {
		return err
	}
	// get hashed kmers from sequence and evaluate
	for hv := range hasher.Hash(canonical) {
		// if there is a bloom filter attached to the minhash object, use it to only add non-unique kmers
		if BottomKsketch.BloomFilter != nil {
			if !BottomKsketch.BloomFilter.Check(hv) {
				BottomKsketch.BloomFilter.Add(hv)
				continue
			}
		}
		// if the sketch isn't full yet, add the hashed k-mer
		if len(*BottomKsketch.sketch) < BottomKsketch.sketchSize {
			heap.Push(BottomKsketch.sketch, hv)
			// otherwise, update the sketch if the new value is smaller than the largest value in the sketch
		} else if hv < (*BottomKsketch.sketch)[0] {
			// replace the largest sketch value with the new value
			(*BottomKsketch.sketch)[0] = hv
			// the heap Fix method re-establishes the heap ordering after the element at index i has changed its value
			heap.Fix(BottomKsketch.sketch, 0)
		}
	}
	return nil
}

// GetSimilarity is a method to estimate the Jaccard similarity between sets
// mismatched sketch lengths are permitted
func (BottomKsketch *BottomKsketch) GetSimilarity(mh2 flavour) float64 {
	intersect := 0.0
	sketch1 := BottomKsketch.GetSketch()
	sketch2 := mh2.GetSketch()
	sharedLength := len(sketch1)
	if sharedLength > len(sketch2) {
		sharedLength = len(sketch2)
	}
	for i := 1; i <= sharedLength; i++ {
		if sketch1[len(sketch1)-i] == sketch2[len(sketch2)-i] {
			intersect++
		}
	}
	return (intersect / float64(sharedLength))
}

// GetSketch is a method to return the sketch held by a MinHash Bottom-k sketch object
func (BottomKsketch *BottomKsketch) GetSketch() []uint64 {
	sketch := make(intHeap, len(*BottomKsketch.sketch))
	copy(sketch, *BottomKsketch.sketch)
	sort.Sort(sketch)
	return sketch
}

// intHeap is a min-heap of uint64s (we're satisfying the heap interface: https://golang.org/pkg/container/heap/)
type intHeap []uint64

// the less method is returning the largest value, so that it is at index position 0 in the heap
func (intHeap intHeap) Less(i, j int) bool { return intHeap[i] > intHeap[j] }
func (intHeap intHeap) Swap(i, j int)      { intHeap[i], intHeap[j] = intHeap[j], intHeap[i] }
func (intHeap intHeap) Len() int           { return len(intHeap) }

// Push is a method to add an element to the heap
func (intHeap *intHeap) Push(x interface{}) {
	// dereference the pointer to modify the slice's length, not just its contents
	*intHeap = append(*intHeap, x.(uint64))
}

// Pop is a method to remove an element from the heap
func (intHeap *intHeap) Pop() interface{} {
	old := *intHeap
	n := len(old)
	x := old[n-1]
	*intHeap = old[0 : n-1]
	return x
}
