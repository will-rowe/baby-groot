// Package lshforest is the indexing scheme used for GROOT
package lshforest

import (
	"fmt"
	"io/ioutil"
	"sort"
	"sync"

	"github.com/segmentio/objconv/msgpack"
	"github.com/will-rowe/baby-groot/src/misc"
	"github.com/will-rowe/baby-groot/src/seqio"
)

// GROOTindex is the interface for the LSH index TODO: I will use this to try some more indexes at a later date
type GROOTindex interface {
	Add(*seqio.Key) error
	Index()
	Load(string) error
	Query([]uint64) []string
	GetKey(string) (*seqio.Key, error)
}

// LSHforest is the index definition
type LSHforest struct {
	K              int
	L              int
	sketchSize     int
	InitHashTables []initialHashTable
	hashTables     []hashTable // these sorted slices are equivalent to the prefix trees used in the LSH Forest paper
	KeyLookup      *keyLookup
	SketchCounter  int
}

// the initial hash table uses the stringified sketch as a key - the values are the corresponding keys
type initialHashTable map[string][]string

// a bucket is a single hash table that is stored in the hashTables - it contains part of the stringified sketch and the corresponding keys
type bucket struct {
	stringifiedSketch string
	keys              []string
}

// this is populated during indexing -- it is a slice of buckets and can be sorted
type hashTable []bucket

// methods to satisfy the sort interface
func (h hashTable) Len() int           { return len(h) }
func (h hashTable) Swap(i, j int)      { h[i], h[j] = h[j], h[i] }
func (h hashTable) Less(i, j int) bool { return h[i].stringifiedSketch < h[j].stringifiedSketch }

// KeyLookupMap relates the stringified seqio.Key to the original, allowing LSHforest search results to easily be related to graph locations
type keyLookup struct {
	Mappy  map[string]*seqio.Key
	access sync.RWMutex
}

// Settings will print the number of hash functions and number of buckets set by the LSH forest
func (LSHforest *LSHforest) Settings() (K, L int) {
	return LSHforest.K, LSHforest.L
}

// Add a minhash sketch to the LSH Forest
func (LSHforest *LSHforest) Add(key *seqio.Key) error {
	if len(key.Sketch) != LSHforest.sketchSize {
		return fmt.Errorf("cannot add sketch: wrong size for index")
	}
	// add the key and its stringified version to the lookup map
	if err := LSHforest.addKey(key); err != nil {
		return err
	}
	// split the sketch into the right number of buckets and then hash each one
	stringifiedSketch := make([]string, LSHforest.L)
	for i := 0; i < LSHforest.L; i++ {
		stringifiedSketch[i] = misc.Stringify(key.Sketch[i*LSHforest.K : (i+1)*LSHforest.K])
	}
	// iterate over each bucket in the LSH forest
	for i := 0; i < len(LSHforest.InitHashTables); i++ {
		// if the current partition of the sketch isn't in the current bucket in the LSH forest, add it
		if _, ok := LSHforest.InitHashTables[i][stringifiedSketch[i]]; !ok {
			LSHforest.InitHashTables[i][stringifiedSketch[i]] = make([]string, 1)
			LSHforest.InitHashTables[i][stringifiedSketch[i]][0] = key.StringifiedKey
		} else {
			// if it is, append the current key (graph location) to this hashed sketch bucket
			LSHforest.InitHashTables[i][stringifiedSketch[i]] = append(LSHforest.InitHashTables[i][stringifiedSketch[i]], key.StringifiedKey)
		}
	}
	// delete the sketch from the key (to save some space)
	key.Sketch = make([]uint64, 0, 0)
	LSHforest.SketchCounter++
	return nil
}

/*
// Load is a method to populate an LSH Forest instance using a byte slice from msgPack
func (LSHforest *LSHforest) Load(data []byte) error {
	if len(data) == 0 {
		return fmt.Errorf("no data received for LSH Forest Load()")
	}
	// unpack the data
	err := msgpack.Unmarshal(data, LSHforest)
	if err != nil {
		return err
	}
	if len(LSHforest.InitHashTables) < 1 {
		return fmt.Errorf("could not load the LSH Forest")
	}
	// index the lshForest
	LSHforest.Index()
	if len(LSHforest.hashTables) < 1 {
		return fmt.Errorf("could not index the LSH Forest")
	}
	return nil
}
*/

// Dump is a method to dump the LSHforest to file
func (LSHforest *LSHforest) Dump(path string) error {
	b, err := msgpack.Marshal(LSHforest)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(path, b, 0644)
}

// Load is a method to load LSHforest from file
func (LSHforest *LSHforest) Load(path string) error {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}
	if len(data) == 0 {
		return fmt.Errorf("no data received for LSH Forest Load()")
	}
	err = msgpack.Unmarshal(data, LSHforest)
	if err != nil {
		return err
	}
	LSHforest.sketchSize = LSHforest.K * LSHforest.L
	if LSHforest.sketchSize < 1 {
		return fmt.Errorf("loaded LSH Forest has no bins")
	}
	if len(LSHforest.InitHashTables) < 1 {
		return fmt.Errorf("LSH Forest is corrupted")
	}

	// index the lshForest
	LSHforest.Index()
	if len(LSHforest.hashTables) < 1 {
		return fmt.Errorf("could not index the LSH Forest")
	}
	return nil
}

// Index transfers the contents of each initialHashTable to the hashTable arrays so they can be sorted and searched
func (LSHforest *LSHforest) Index() {

	// make sure the LSH Forest is empty
	LSHforest.hashTables = make([]hashTable, LSHforest.L)
	for i := range LSHforest.hashTables {
		LSHforest.hashTables[i] = make(hashTable, 0)
	}

	// iterate over the empty hash tables
	for i := range LSHforest.hashTables {

		// transfer contents from the corresponding bucket in the initial hash table
		for stringifiedSketch, keys := range LSHforest.InitHashTables[i] {
			LSHforest.hashTables[i] = append(LSHforest.hashTables[i], bucket{stringifiedSketch, keys})
		}

		// sort the new hashtable and store it in the corresponding slot in the indexed hash tables
		sort.Sort(LSHforest.hashTables[i])

		// clear the initial hashtable that has just been processed (have commented this out, as this was preventing re-use of the index)
		//LSHforest.InitHashTables[i] = make(initialHashTable)
	}
}

// Query is the exported method for querying and returning similar sketches from the LSH forest
func (LSHforest *LSHforest) Query(sketch []uint64) []string {
	// TODO: this should be an error, not a panic
	if len(LSHforest.hashTables[0]) == 0 {
		panic("LSH Forest has not been indexed")
	}
	result := make([]string, 0)
	// more info on done chans for explicit cancellation in concurrent pipelines: https://blog.golang.org/pipelines
	done := make(chan struct{})
	// collect query results and aggregate in a single array to send back
	for key := range LSHforest.runQuery(sketch, done) {
		result = append(result, key)
	}
	close(done)
	return result
}

// GetKey will return the seqio.Key for the stringified version
func (LSHforest *LSHforest) GetKey(key string) (*seqio.Key, error) {
	LSHforest.KeyLookup.access.Lock()
	defer LSHforest.KeyLookup.access.Unlock()
	if returnKey, ok := LSHforest.KeyLookup.Mappy[key]; ok {
		return returnKey, nil
	}
	return nil, fmt.Errorf("key not found in LSH Forest: %v", key)
}

// addKey will add a seqio.Key to the lookup map, storing it under a stringified version of the key
func (LSHforest *LSHforest) addKey(key *seqio.Key) error {
	if key.StringifiedKey == "" {
		return fmt.Errorf("encountered key that contains no lookup value")
	}
	LSHforest.KeyLookup.access.Lock()
	if _, contains := LSHforest.KeyLookup.Mappy[key.StringifiedKey]; contains {
		return fmt.Errorf("duplicated key encountered: %v", key.StringifiedKey)
	}
	LSHforest.KeyLookup.Mappy[key.StringifiedKey] = key
	LSHforest.KeyLookup.access.Unlock()
	return nil
}

// NewKeyLookup is the constructor
func NewKeyLookup() *keyLookup {
	return &keyLookup{
		Mappy: make(map[string]*seqio.Key),
	}
}

// NewLSHforest is the constructor
func NewLSHforest(sketchSize int, jsThresh float64) *LSHforest {
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

	// return the address of the new LSH forest
	return &LSHforest{
		K:              k,
		L:              l,
		sketchSize:     sketchSize,
		InitHashTables: iht,
		hashTables:     ht,
		KeyLookup:      NewKeyLookup(),
	}
}

// runQuery does the actual work
func (LSHforest *LSHforest) runQuery(sketch []uint64, done <-chan struct{}) <-chan string {
	queryResultChan := make(chan string)
	go func() {
		defer close(queryResultChan)
		//  convert the query sketch from []uint64 to a string
		stringifiedSketch := make([]string, LSHforest.L)
		for i := 0; i < LSHforest.L; i++ {
			stringifiedSketch[i] = misc.Stringify(sketch[i*LSHforest.K : (i+1)*LSHforest.K])
		}
		// don't send back multiple copies of the same key
		seens := make(map[string]bool)
		// compress internal nodes using a prefix
		prefixSize := misc.HASH_SIZE * (LSHforest.K - 1)
		// run concurrent hashtable queries
		keyChan := make(chan string)
		var wg sync.WaitGroup
		wg.Add(LSHforest.L)
		for i := 0; i < LSHforest.L; i++ {
			go func(bucket hashTable, queryChunk string) {
				defer wg.Done()
				// sort.Search uses binary search to find and return the smallest index i in [0, n) at which f(i) is true
				index := sort.Search(len(bucket), func(x int) bool { return bucket[x].stringifiedSketch[:prefixSize] >= queryChunk[:prefixSize] })
				// k is the index returned by the search
				if index < len(bucket) && bucket[index].stringifiedSketch[:prefixSize] == queryChunk[:prefixSize] {
					for j := index; j < len(bucket) && bucket[j].stringifiedSketch[:prefixSize] == queryChunk[:prefixSize]; j++ {
						if bucket[j].stringifiedSketch == queryChunk {
							// if the query matches the bucket, send the keys as search results
							for _, key := range bucket[j].keys {
								select {
								case keyChan <- key:
								case <-done:
									return
								}
							}
						}
					}
				}
			}(LSHforest.hashTables[i], stringifiedSketch[i])
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
