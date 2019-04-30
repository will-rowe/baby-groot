package markov

import (
	"strings"
	"testing"
)

// test CreateChain
func TestCreateChain(t *testing.T) {
	//Create a chain of order 2
	chain := NewChain(2)

	//Feed in training data
	chain.Add(strings.Split("I want a cheese burger", " "), []float64{1.0, 1.0, 3.0, 4.0, 5.0})
	chain.Add(strings.Split("I want a chilled sprite", " "), []float64{1.0, 98.0, 100.0, 100.0, 100.0})
	chain.Add(strings.Split("I want to go to the movies", " "), []float64{1.0, 1.0, 1.0, 1.0, 1.0, 1.0, 1.0})

	//Get transition probability of a sequence
	prob, err := chain.TransitionProbability("a", []string{"I", "want"})
	if err != nil {
		t.Fatal(err)
	}
	t.Log(prob)
	if prob != 0.9805825242718447 {
		t.Fatalf("incorrect probability returned: %f", prob)
	}
}
