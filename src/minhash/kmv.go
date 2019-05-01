package minhash

import (
	"fmt"
	"math"

	"github.com/will-rowe/ntHash"
)

// KMVsketch is the K-Minimum Values MinHash sketch of a set
type KMVsketch struct {
	kmerSize   int
	sketchSize int
	sketch     []uint64
}

// NewKMVsketch is the constructor for a KMVsketch
func NewKMVsketch(k, s int) *KMVsketch {
	// init the sketch with maximum values
	sketch := make([]uint64, s)
	for i := range sketch {
		sketch[i] = math.MaxUint64
	}
	return &KMVsketch{
		kmerSize:   k,
		sketchSize: s,
		sketch:     sketch,
	}
}

// Add is a method to decompose a read to kmers, hash them and add any minimums to the sketch
func (KMVsketch *KMVsketch) Add(sequence []byte) error {
	if len(sequence) < KMVsketch.kmerSize {
		return fmt.Errorf("sequence length (%d) is short than k-mer length (%d)", len(sequence), KMVsketch.kmerSize)
	}
	// initiate the rolling ntHash
	hasher, err := ntHash.New(&sequence, KMVsketch.kmerSize)
	if err != nil {
		return err
	}
	// get hashed kmers from sequence and evaluate
	for baseHash := range hasher.Hash(canonical) {
		// for each k-mer base hash value, derive a new value for each sketch slot
		for i := 0; i < KMVsketch.sketchSize; i++ {
			hv := baseHash + (uint64(i) * baseHash)
			// evaluate and add to the current sketch slot if it is a minimum
			if hv < KMVsketch.sketch[i] {
				KMVsketch.sketch[i] = hv
			}
		}
	}
	return nil
}

// GetSketch is a method to return the sketch held by a MinHash KMV sketch object
func (KMVsketch *KMVsketch) GetSketch() []uint64 {
	return KMVsketch.sketch
}

// GetSimilarity is a method to estimate the Jaccard similarity between sets
// mismatched sketch lengths are permitted
func (KMVsketch *KMVsketch) GetSimilarity(mh2 flavour) float64 {
	intersect := 0.0
	sketch1 := KMVsketch.GetSketch()
	sketch2 := mh2.GetSketch()
	sharedLength := len(sketch1)
	if sharedLength > len(sketch2) {
		sharedLength = len(sketch2)
	}
	for i := 0; i < sharedLength; i++ {
		if sketch1[i] == sketch2[i] {
			intersect++
		}
	}
	return (intersect / float64(sharedLength))
}
