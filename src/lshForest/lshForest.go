// lshForest is the indexing scheme used for GROOT
package lshForest

import (
	"fmt"
	"io/ioutil"
	"math"
	"sort"
	"sync"

	"github.com/will-rowe/baby-groot/src/misc"
	"github.com/will-rowe/baby-groot/src/seqio"
	"gopkg.in/vmihailenco/msgpack.v2"
)

type GROOTindex interface {
	Add(*seqio.Key) error
	Index()
	Dump(string) error
	Load(string) error
	Query([]uint64) []string
	GetKey(string) (*seqio.Key, error)
}

/*
	The types needed by the LSH forest index
*/
// lshForest is the index definition
type lshForest struct {
	K              int
	L              int
	sketchSize     int
	InitHashTables []initialHashTable
	hashTables     []hashTable
	KeyLookup      KeyLookupMap
	mapLock        sync.RWMutex
}

// keys is a slice containing all the stringified keys for a given sketch
type keys []string

// the initial hash table uses the stringified sketch as a key - the values are the corresponding keys
type initialHashTable map[string]keys

// a bucket is a single hash table that is stored in the hashTables - it contains part of the stringified sketch and the corresponding keys
type bucket struct {
	stringifiedSketch string
	keys              keys
}

// this is populated during indexing -- it is a slice of buckets and can be sorted
type hashTable []bucket

//methods to satisfy the sort interface
func (h hashTable) Len() int           { return len(h) }
func (h hashTable) Swap(i, j int)      { h[i], h[j] = h[j], h[i] }
func (h hashTable) Less(i, j int) bool { return h[i].stringifiedSketch < h[j].stringifiedSketch }

// this map relates the stringified seqio.Key to the original, allowing lshForest search results to easily be related to graph locations
type KeyLookupMap map[string]*seqio.Key

/*
	Methods
*/
// Settings will print the number of hash functions and number of buckets set by the LSH forest
func (lshForest *lshForest) Settings() (K, L int) {
	return lshForest.K, lshForest.L
}

// Add a minhash sketch to the LSH Forest
func (lshForest *lshForest) Add(key *seqio.Key) error {
	if len(key.Sketch) != lshForest.sketchSize {
		return fmt.Errorf("cannot add sketch: wrong size for index")
	}
	// add the key and its stringified version to the lookup map
	if err := lshForest.addKey(key); err != nil {
		return err
	}
	// split the sketch into the right number of buckets and then hash each one
	stringifiedSketch := make([]string, lshForest.L)
	for i := 0; i < lshForest.L; i++ {
		stringifiedSketch[i] = misc.Stringify(key.Sketch[i*lshForest.K : (i+1)*lshForest.K])
	}
	// iterate over each bucket in the LSH forest
	for i := 0; i < len(lshForest.InitHashTables); i++ {
		// if the current bucket in the sketch isn't in the current bucket in the LSH forest, add it
		if _, ok := lshForest.InitHashTables[i][stringifiedSketch[i]]; !ok {
			lshForest.InitHashTables[i][stringifiedSketch[i]] = make(keys, 1)
			lshForest.InitHashTables[i][stringifiedSketch[i]][0] = key.StringifiedKey
		} else {
			// if it is, append the current key (graph location) to this hashed sketch bucket
			lshForest.InitHashTables[i][stringifiedSketch[i]] = append(lshForest.InitHashTables[i][stringifiedSketch[i]], key.StringifiedKey)
		}
	}
	// delete the sketch from the key (to save some space)
	key.Sketch = make([]uint64, 0, 0)
	return nil
}

// Index will transfers the contents of each initialHashTable to the hashTable arrays so they can be sorted and searched
func (lshForest *lshForest) Index() {
	// iterate over the empty indexed hash tables
	for i := range lshForest.hashTables {
		// transfer contents from the corresponding bucket in the initial hash table
		for stringifiedSketch, keys := range lshForest.InitHashTables[i] {
			lshForest.hashTables[i] = append(lshForest.hashTables[i], bucket{stringifiedSketch, keys})
		}
		// sort the new hashtable and store it in the corresponding slot in the indexed hash tables
		sort.Sort(lshForest.hashTables[i])
		// clear the initial hashtable that has just been processed
		lshForest.InitHashTables[i] = make(initialHashTable)
	}
}

// Dump an LSH index to disk
func (lshForest *lshForest) Dump(path string) error {
	if len(lshForest.hashTables[0]) != 0 {
		return fmt.Errorf("cannot dump the LSH Forest after running the indexing method")
	}
	b, err := msgpack.Marshal(lshForest)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(path, b, 0644)
}

// Load an LSH index from disk
func (lshForest *lshForest) Load(path string) error {
	b, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}
	return msgpack.Unmarshal(b, lshForest)
}

// Query is the exported method for querying and returning similar sketches from the LSH forest
func (lshForest *lshForest) Query(sketch []uint64) []string {
	result := make([]string, 0)
	// more info on done chans for explicit cancellation in concurrent pipelines: https://blog.golang.org/pipelines
	done := make(chan struct{})
	defer close(done)
	// collect query results and aggregate in a single array to send back
	for key := range lshForest.runQuery(sketch, done) {
		result = append(result, key)
	}
	return result
}

// GetKey will return the seqio.Key for the stringified version
func (lshForest *lshForest) GetKey(key string) (*seqio.Key, error) {
	lshForest.mapLock.Lock()
	defer lshForest.mapLock.Unlock()
	if returnKey, ok := lshForest.KeyLookup[key]; ok {
		return returnKey, nil
	}
	return nil, fmt.Errorf("key not found in LSH Forest: %v", key)
}

/*
	Functions
*/
// NewlshForest is the constructor function
func NewLSHforest(sketchSize int, jsThresh float64) *lshForest {
	// calculate the optimal number of buckets and hash functions to use, based on the length of MinHash sketch and a Jaccard Similarity theshhold
	k, l, _, _ := optimise(sketchSize, jsThresh)
	// create the initial hash tables
	iht := make([]initialHashTable, l)
	for i := range iht {
		iht[i] = make(initialHashTable)
	}
	// create the hash tables that will be populated once the LSH forest indexing method has been run
	ht := make([]hashTable, l)
	for i := range ht {
		ht[i] = make(hashTable, 0)
	}
	// create the KeyLookup map to project sketches on to the graphs
	kl := make(KeyLookupMap)
	// return the address of the new LSH forest
	return &lshForest{
		K:              k,
		L:              l,
		sketchSize:     sketchSize,
		InitHashTables: iht,
		hashTables:     ht,
		KeyLookup:      kl,
	}
}

// addKey will add a seqio.Key to the lookup map, storing it under a stringified version of the key
func (lshForest *lshForest) addKey(key *seqio.Key) error {
	lshForest.mapLock.Lock()
	defer lshForest.mapLock.Unlock()
	//if _, ok := lshForest.KeyLookup[key.StringifiedKey]; ok {
	//	return fmt.Errorf("key already in LSH Forest: %v", key.StringifiedKey)
	//}
	lshForest.KeyLookup[key.StringifiedKey] = key
	return nil
}

// runQuery does the actual work
func (lshForest *lshForest) runQuery(sketch []uint64, done <-chan struct{}) <-chan string {
	queryResultChan := make(chan string)
	go func() {
		defer close(queryResultChan)
		//  convert the query sketch from []uint64 to a string
		stringifiedSketch := make([]string, lshForest.L)
		for i := 0; i < lshForest.L; i++ {
			stringifiedSketch[i] = misc.Stringify(sketch[i*lshForest.K : (i+1)*lshForest.K])
		}
		// don't send back multiple copies of the same key
		seens := make(map[string]bool)
		// compress internal nodes using a prefix
		prefixSize := misc.HASH_SIZE * lshForest.K
		// run concurrent hashtable queries
		keyChan := make(chan string)
		var wg sync.WaitGroup
		wg.Add(lshForest.L)
		for i := 0; i < lshForest.L; i++ {
			go func(bucket hashTable, queryChunk string) {
				defer wg.Done()
				// sort.Search uses binary search to find and return the smallest index i in [0, n) at which f(i) is true
				index := sort.Search(len(bucket), func(x int) bool { return bucket[x].stringifiedSketch[:prefixSize] >= queryChunk })
				// k is the index returned by the search
				if index < len(bucket) && bucket[index].stringifiedSketch[:prefixSize] == queryChunk {
					for j := index; j < len(bucket) && bucket[j].stringifiedSketch[:prefixSize] == queryChunk; j++ {
						// copies key values from this hashtable to the keyChan until all values from bucket[j] copied or done is closed
						for _, key := range bucket[j].keys {
							select {
							case keyChan <- key:
							case <-done:
								return
							}
						}
					}
				}
			}(lshForest.hashTables[i], stringifiedSketch[i])
		}
		go func() {
			wg.Wait()
			close(keyChan)
		}()
		for key := range keyChan {
			if _, seen := seens[key]; seen {
				continue
			}
			queryResultChan <- key
			seens[key] = true
		}
	}()
	return queryResultChan
}

//  the following funcs are taken from https://github.com/ekzhu/minhash-lsh
// optimise returns the optimal number of hash functions and the optimal number of buckets for Jaccard similarity search, as well as  the false positive and negative probabilities.
func optimise(sketchSize int, jsThresh float64) (int, int, float64, float64) {
	optimumK, optimumL := 0, 0
	fp, fn := 0.0, 0.0
	minError := math.MaxFloat64
	for l := 1; l <= sketchSize; l++ {
		for k := 1; k <= sketchSize; k++ {
			if l*k > sketchSize {
				break
			}
			currFp := probFalsePositive(l, k, jsThresh, 0.01)
			currFn := probFalseNegative(l, k, jsThresh, 0.01)
			currErr := currFn + currFp
			if minError > currErr {
				minError = currErr
				optimumK = k
				optimumL = l
				fp = currFp
				fn = currFn
			}
		}
	}
	return optimumK, optimumL, fp, fn
}

// integral of function f, lower limit a, upper limit l, and precision defined as the quantize step
func integral(f func(float64) float64, a, b, precision float64) float64 {
	var area float64
	for x := a; x < b; x += precision {
		area += f(x+0.5*precision) * precision
	}
	return area
}

// falsePositive is the probability density function for false positive
func falsePositive(l, k int) func(float64) float64 {
	return func(j float64) float64 {
		return 1.0 - math.Pow(1.0-math.Pow(j, float64(k)), float64(l))
	}
}

// falseNegative is the probability density function for false negative
func falseNegative(l, k int) func(float64) float64 {
	return func(j float64) float64 {
		return 1.0 - (1.0 - math.Pow(1.0-math.Pow(j, float64(k)), float64(l)))
	}
}

// probFalseNegative to compute the cummulative probability of false negative given threshold t
func probFalseNegative(l, k int, t, precision float64) float64 {
	return integral(falseNegative(l, k), t, 1.0, precision)
}

// probFalsePositive to compute the cummulative probability of false positive given threshold t
func probFalsePositive(l, k int, t, precision float64) float64 {
	return integral(falsePositive(l, k), 0, t, precision)
}
