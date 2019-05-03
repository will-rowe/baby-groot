package pipeline

import (
	"testing"
)

// define some dummy components to run a test pipeline
type ComponentA struct {
	input  []int
	output chan int
}

func NewComponentA(i []int) *ComponentA {
	return &ComponentA{input: i, output: make(chan int)}
}

func (ComponentA *ComponentA) Run() {
	defer close(ComponentA.output)
	for _, input := range ComponentA.input {
		ComponentA.output <- input
	}
}

type ComponentB struct {
	input    chan int
	addition int
	results  []int
}

func NewComponentB(i int) *ComponentB {
	return &ComponentB{addition: i}
}

func (ComponentB *ComponentB) Connect(previous *ComponentA) {
	ComponentB.input = previous.output
}

func (ComponentB *ComponentB) Run() {
	results := []int{}
	for input := range ComponentB.input {
		results = append(results, (input + ComponentB.addition))
	}
	ComponentB.results = results

}

// tests
func TestPipeline(t *testing.T) {
	inputValues := []int{1, 2, 3, 4}
	expectedOutput := []int{11, 12, 13, 14}
	// create the processes
	a := NewComponentA(inputValues)
	b := NewComponentB(10)
	// create the pipeline
	newPipeline := NewPipeline()
	// add the processes and connect them
	newPipeline.AddProcesses(a, b)
	b.Connect(a)
	if len(newPipeline.processes) != 2 {
		t.Fatal("did not add correct number of processes to pipeline")
	}
	// run the pipeline
	newPipeline.Run()
	// once the pipeline is done, there should be results in the final component
	if len(expectedOutput) != len(b.results) {
		t.Fatal("pipeline did not produce expected output")
	}
	for i, val := range b.results {
		if val != expectedOutput[i] {
			t.Fatal("pipeline did not produce expected output")
		}
	}
}
