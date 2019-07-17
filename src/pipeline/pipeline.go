// Package pipeline contains a streaming pipeline implementation based on the Gopher Academy article by S. Lampa - Patterns for composable concurrent pipelines in Go (https://blog.gopheracademy.com/advent-2015/composable-pipelines-improvements/)
package pipeline

import (
	"io/ioutil"

	"github.com/segmentio/objconv/msgpack"
	"github.com/will-rowe/baby-groot/src/graph"
	"github.com/will-rowe/baby-groot/src/lshforest"
)

// BUFFERSIZE is the size of the buffer used by the pipeline channels
const BUFFERSIZE int = 64

// process is the interface used by pipeline
type process interface {
	Run()
}

// Pipeline is the base type, which takes any types that satisfy the process interface
type Pipeline struct {
	processes []process
}

// NewPipeline is the pipeline constructor
func NewPipeline() *Pipeline {
	return &Pipeline{}
}

// AddProcess is a method to add a single process to the pipeline
func (Pipeline *Pipeline) AddProcess(proc process) {
	// add the process to the pipeline
	Pipeline.processes = append(Pipeline.processes, proc)
}

// AddProcesses is a method to add multiple processes to the pipeline
func (Pipeline *Pipeline) AddProcesses(procs ...process) {
	for _, proc := range procs {
		Pipeline.AddProcess(proc)
	}
}

// Run is a method that starts the pipeline
func (Pipeline *Pipeline) Run() {
	// each pipeline process is run in a Go routines, except the last process which is run in the foreground to control the flow
	for i, process := range Pipeline.processes {
		if i < len(Pipeline.processes)-1 {
			go process.Run()
		} else {
			process.Run()
		}
	}
}

// GetNumProcesses is a method to return the number of processes registered in a pipeline
func (Pipeline *Pipeline) GetNumProcesses() int {
	return len(Pipeline.processes)
}

// Info stores the runtime information
type Info struct {
	Version   string
	Index     *IndexCmd
	Sketch    *SketchCmd
	Haplotype *HaploCmd
	Db        *lshforest.LSHforest
	DbDump    []byte
	Store     graph.Store
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

// Dump is a method to dump the pipeline info to file
func (Info *Info) Dump(path string) error {
	// get the lshForest dump
	b, err := msgpack.Marshal(Info.Db)
	if err != nil {
		return err
	}
	Info.DbDump = b
	Info.Db = nil

	// marshal the data
	b, err = msgpack.Marshal(Info)
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

// LoadFromBytes is a method to load Info from a msgPack strean
func (Info *Info) LoadFromBytes(data []byte) error {
	err := msgpack.Unmarshal(data, Info)
	if err != nil {
		return err
	}
	Info.Db = lshforest.NewLSHforest(Info.Index.SketchSize, Info.Index.JSthresh)
	err = Info.Db.Load(Info.DbDump)
	Info.DbDump = nil
	return err
}
