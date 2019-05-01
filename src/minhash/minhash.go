// Package minhash contains implementations of bottom-k and kmv MinHash algorithms. These implementations use the ntHash rolling hash function.
package minhash

// canonical sets the ntHash hashing function to use the canonical form of each k-mer (i.e. it inspects k-mer and its complement, returning the lowest hash value)
const canonical = true

// MinHash is an interface to group the different flavours of MinHash implemented here
type MinHash interface {
	flavour
	GetSimilarity(flavour) float64
}

type flavour interface {
	Add([]byte) error
	GetSketch() []uint64
}
