package minhash

import (
	"testing"
)

var (
	kmerSize   = 7
	sketchSize = 10
	sequence   = []byte("ACTGCGTGCGTGAAACGTGCACGTGACGTG")
	sequence2  = []byte("TGACGCACGCACTTTGCACGTGCACTGCAC")
	hashvalues = []uint64{12345, 54321, 9999999, 98765}
)

func TestBloomfilter(t *testing.T) {
	filter := NewBloomFilter(3)
	for i := 0; i < len(hashvalues); i++ {
		filter.Add(hashvalues[i])
	}
	for i := 0; i < len(hashvalues); i++ {
		if !filter.Check(hashvalues[i]) {
			t.Fatalf("'%d' should be have been marked present", hashvalues[i])
		}
	}
	filter.Reset()
	for i := 0; i < len(hashvalues); i++ {
		if filter.Check(hashvalues[i]) {
			t.Fatalf("'%d' shouldn't be marked as present", hashvalues[i])
		}
	}
}

func TestMinHashConstructors(t *testing.T) {
	mhKMV := NewKMVsketch(kmerSize, sketchSize)
	if len(mhKMV.GetSketch()) != sketchSize || mhKMV.sketchSize != sketchSize || mhKMV.kmerSize != kmerSize {
		t.Fatalf("NewKMVsketch constructor did not initiate MinHash KMV sketch correctly")
	}
	mhBK := NewBottomKsketch(kmerSize, sketchSize, nil)
	if len(mhBK.GetSketch()) != 0 || mhBK.sketchSize != sketchSize || mhBK.kmerSize != kmerSize {
		t.Fatalf("NewBottomKsketch constructor did not initiate MinHash Bottom-k sketch correctly")
	}

}

func TestAdd(t *testing.T) {
	// test KMV
	mhKMV := NewKMVsketch(kmerSize, sketchSize)
	// try adding a sequence that is too short for the given k
	if err := mhKMV.Add(sequence[0:1]); err == nil {
		t.Fatal("should fault as sequences must be >= kmerSize")
	}
	// try adding a sequence that passes the length check
	if err := mhKMV.Add(sequence); err != nil {
		t.Fatal(err)
	}
	// test bottomK
	mhBK := NewBottomKsketch(kmerSize, sketchSize, nil)
	// try adding a sequence that is too short for the given k
	if err := mhBK.Add(sequence[0:1]); err == nil {
		t.Fatal("should fault as sequences must be >= kmerSize")
	}
	// try adding a sequence that passes the length check
	if err := mhBK.Add(sequence); err != nil {
		t.Fatal(err)
	}
	if len(mhBK.GetSketch()) == 0 {
		t.Fatal("bottom-k sketch should now have values")
	}
}

func TestSimilarityEstimates(t *testing.T) {
	// test KMV
	mhKMV1 := NewKMVsketch(kmerSize, sketchSize)
	if err := mhKMV1.Add(sequence); err != nil {
		t.Fatal(err)
	}
	mhKMV2 := NewKMVsketch(kmerSize, sketchSize)
	if err := mhKMV2.Add(sequence2); err != nil {
		t.Fatal(err)
	}
	if js := mhKMV1.GetSimilarity(mhKMV2); js != 0.6 {
		t.Fatalf("incorrect similarity estimate: %f", js)
	}
	// test bottomK
	mhBK1 := NewBottomKsketch(kmerSize, sketchSize, nil)
	if err := mhBK1.Add(sequence); err != nil {
		t.Fatal(err)
	}
	mhBK2 := NewBottomKsketch(kmerSize, sketchSize, nil)
	if err := mhBK2.Add(sequence2); err != nil {
		t.Fatal(err)
	}
	if js := mhBK1.GetSimilarity(mhBK2); js != 0.3 {
		t.Fatalf("incorrect similarity estimate: %f", js)
	}
}

// benchmark KMV
func BenchmarkKMV(b *testing.B) {
	mhKMV1 := NewKMVsketch(kmerSize, sketchSize)
	// run the add method b.N times
	for n := 0; n < b.N; n++ {
		if err := mhKMV1.Add(sequence); err != nil {
			b.Fatal(err)
		}
	}
}

// benchmark Bottom-K
func BenchmarkBottomK(b *testing.B) {
	mhBK1 := NewBottomKsketch(kmerSize, sketchSize, nil)
	// run the add method b.N times
	for n := 0; n < b.N; n++ {
		if err := mhBK1.Add(sequence); err != nil {
			b.Fatal(err)
		}
	}
}
