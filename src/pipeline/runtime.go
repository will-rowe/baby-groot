package pipeline

import (
	"fmt"
	"io/ioutil"

	"github.com/segmentio/objconv/msgpack"
	"github.com/will-rowe/baby-groot/src/graph"
	"github.com/will-rowe/baby-groot/src/lshforest"
)

// Info stores the runtime information
type Info struct {
	NumProc   int
	Version   string
	Profiling bool
	Index     IndexCmd
	Sketch    SketchCmd
	Haplotype HaploCmd
	Store     graph.Store
	db        lshforest.IndexWrapper
}

// IndexCmd stores the runtime info for the index command
type IndexCmd struct {
	KmerSize   int
	SketchSize int
	KMVsketch  bool
	JSthresh   float64
	WindowSize int
	IndexDir   string
}

// SketchCmd stores the runtime info for the sketch command
type SketchCmd struct {
	Fasta           bool
	BloomFilter     bool
	MinKmerCoverage int
	MinBaseCoverage float64
	TotalKmers      float64
}

// HaploCmd stores the runtime info for the haplotype command
type HaploCmd struct {
	Cutoff        float64
	MinIterations int
	MaxIterations int
	TotalKmers    int
	HaploDir      string
}

// AttachDB is a method to attach a LSH Forest index to the runtime
func (Info *Info) AttachDB(db *lshforest.IndexWrapper) {
	Info.db = *db
}

// GetDBinfo is a method to return the number of hash functions and buckets used by the current index
func (Info *Info) GetDBinfo() (int32, int32) {
	return Info.db.Settings()
}

// SaveDB is a method to write an LSH Forest index to disk
func (Info *Info) SaveDB(filePath string) error {
	return Info.db.Dump(filePath)
}

// Dump is a method to dump the pipeline info to file
func (Info *Info) Dump(path string) error {
	b, err := msgpack.Marshal(Info)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(path, b, 0644)
}

// Load is a method to load Info from file
func (Info *Info) Load(path string) error {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}
	return Info.LoadFromBytes(data)
}

// LoadFromBytes is a method to load Info from bytes
func (Info *Info) LoadFromBytes(data []byte) error {
	if len(data) == 0 {
		return fmt.Errorf("groot graph store appears empty")
	}
	return msgpack.Unmarshal(data, Info)
}
