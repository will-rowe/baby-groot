package markov

import "strings"

//Pair is a pair of consecutive states in a sequece
type Pair struct {
	CurrentState  NGram // n = order of the chain
	CurrentWeight float64
	NextState     string // n = 1
}

//NGram is a array of words
type NGram []string

type sparseArray map[int]float64

func (ngram NGram) key() string {
	return strings.Join(ngram, "_")
}

func (s sparseArray) sum() float64 {
	sum := 0.0
	for _, count := range s {
		sum += count
	}
	return sum
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func array(value string, count int) []string {
	arr := make([]string, count)
	for i := range arr {
		arr[i] = value
	}
	return arr
}

func floatArray(value float64, count int) []float64 {
	arr := make([]float64, count)
	for i := range arr {
		arr[i] = value
	}
	return arr
}

//MakePairs generates n-gram pairs of consecutive states in a sequence
func MakePairs(tokens []string, weights []float64, order int) []Pair {
	var pairs []Pair
	for i := 0; i < len(tokens)-order; i++ {
		combinedWeight := 0.0
		for j := i; j < i+order; j++ {
			combinedWeight += weights[j]
		}
		pair := Pair{
			CurrentState:  tokens[i : i+order],
			CurrentWeight: combinedWeight,
			NextState:     tokens[i+order],
		}
		pairs = append(pairs, pair)
	}
	return pairs
}
