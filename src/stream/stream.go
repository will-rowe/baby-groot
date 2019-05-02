// Package stream contains a streaming pipeline implementation based on the Gopher Academy article by S. Lampa - Patterns for composable concurrent pipelines in Go (https://blog.gopheracademy.com/advent-2015/composable-pipelines-improvements/)
package stream

import (
	"io/ioutil"

	"github.com/segmentio/objconv/msgpack"
	"github.com/will-rowe/baby-groot/src/graph"
	"github.com/will-rowe/baby-groot/src/lshForest"
)

// BUFFERSIZE is the size of the buffer used by the pipeline channels
const BUFFERSIZE int = 128

// process is the interface used by pipeline
type process interface {
	Run()
}

// Pipeline is the base type, which takes any types that satisfy the process interface
type Pipeline struct {
	Processes []process
}

// NewPipeline is the pipeline constructor
func NewPipeline() *Pipeline {
	return &Pipeline{}
}

// AddProcess is a method to add a single process to the pipeline
func (pipeline *Pipeline) AddProcess(proc process) {
	pipeline.Processes = append(pipeline.Processes, proc)
}

// AddProcesses is a method to add multiple processes to the pipeline
func (pipeline *Pipeline) AddProcesses(procs ...process) {
	for _, proc := range procs {
		pipeline.AddProcess(proc)
	}
}

// Run is a method that starts the pipeline
func (pipeline *Pipeline) Run() {
	// each pipeline process is run in a Go routines, except the last process which is run in the foreground
	for i, process := range pipeline.Processes {
		if i < len(pipeline.Processes)-1 {
			go process.Run()
		} else {
			process.Run()
		}
	}
}

// PipelineInfo stores the runtime information
type PipelineInfo struct {
	Version         string
	Ksize           int
	SigSize         int
	KMVsketch       bool
	JSthresh        float64
	WindowSize      int
	BloomFilter     bool
	MinKmerCoverage int
	MinBaseCoverage float64
	Db              lshForest.GROOTindex
	GraphStore      graph.GraphStore
	GraphDir        string
}

// Dump is a method to dump the PipelineInfo to file
func (PipelineInfo *PipelineInfo) Dump(path string) error {
	b, err := msgpack.Marshal(PipelineInfo)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(path, b, 0644)
}

// Load is a method to load PipelineInfo from file
func (PipelineInfo *PipelineInfo) Load(path string) error {
	b, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}
	return msgpack.Unmarshal(b, PipelineInfo)
}
