// Package lshforest is the indexing scheme used for GROOT
package lshforest

import (
	"fmt"
	"io/ioutil"
	"sort"
	"sync"

	"github.com/will-rowe/baby-groot/src/misc"
	"github.com/will-rowe/baby-groot/src/seqio"
	"gopkg.in/vmihailenco/msgpack.v2"
)

// GROOTindex is the interface for the LSH index TODO: I will use this to try some more indexes at a later date
type GROOTindex interface {
	Add(*seqio.Key) error
	Index()
	Dump(string) error
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

// KeyLookupMap relates the stringified seqio.Key to the original, allowing LSHforest search results to easily be related to graph locations
type KeyLookupMap map[string]*seqio.Key

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
		// if the current bucket in the sketch isn't in the current bucket in the LSH forest, add it
		if _, ok := LSHforest.InitHashTables[i][stringifiedSketch[i]]; !ok {
			LSHforest.InitHashTables[i][stringifiedSketch[i]] = make(keys, 1)
			LSHforest.InitHashTables[i][stringifiedSketch[i]][0] = key.StringifiedKey
		} else {
			// if it is, append the current key (graph location) to this hashed sketch bucket
			LSHforest.InitHashTables[i][stringifiedSketch[i]] = append(LSHforest.InitHashTables[i][stringifiedSketch[i]], key.StringifiedKey)
		}
	}
	// delete the sketch from the key (to save some space)
	key.Sketch = make([]uint64, 0, 0)
	return nil
}

// Index transfers the contents of each initialHashTable to the hashTable arrays so they can be sorted and searched
func (LSHforest *LSHforest) Index() {
	// iterate over the empty indexed hash tables
	for i := range LSHforest.hashTables {
		// transfer contents from the corresponding bucket in the initial hash table
		for stringifiedSketch, keys := range LSHforest.InitHashTables[i] {
			LSHforest.hashTables[i] = append(LSHforest.hashTables[i], bucket{stringifiedSketch, keys})
		}
		// sort the new hashtable and store it in the corresponding slot in the indexed hash tables
		sort.Sort(LSHforest.hashTables[i])
		// clear the initial hashtable that has just been processed
		LSHforest.InitHashTables[i] = make(initialHashTable)
	}
}

// Dump an LSH index to disk
func (LSHforest *LSHforest) Dump(path string) error {
	if len(LSHforest.hashTables[0]) != 0 {
		return fmt.Errorf("cannot dump the LSH Forest after running the indexing method")
	}
	b, err := msgpack.Marshal(LSHforest)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(path, b, 0644)
}

// Load an LSH index from disk
func (LSHforest *LSHforest) Load(path string) error {
	b, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}
	return msgpack.Unmarshal(b, LSHforest)
}

// Query is the exported method for querying and returning similar sketches from the LSH forest
func (LSHforest *LSHforest) Query(sketch []uint64) []string {
	result := make([]string, 0)
	// more info on done chans for explicit cancellation in concurrent pipelines: https://blog.golang.org/pipelines
	done := make(chan struct{})
	defer close(done)
	// collect query results and aggregate in a single array to send back
	for key := range LSHforest.runQuery(sketch, done) {
		result = append(result, key)
	}
	return result
}

// GetKey will return the seqio.Key for the stringified version
func (LSHforest *LSHforest) GetKey(key string) (*seqio.Key, error) {
	LSHforest.mapLock.Lock()
	defer LSHforest.mapLock.Unlock()
	if returnKey, ok := LSHforest.KeyLookup[key]; ok {
		return returnKey, nil
	}
	return nil, fmt.Errorf("key not found in LSH Forest: %v", key)
}

// NewLSHforest is the constructor function
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
	// create the KeyLookup map to project sketches on to the graphs
	kl := make(KeyLookupMap)
	// return the address of the new LSH forest
	return &LSHforest{
		K:              k,
		L:              l,
		sketchSize:     sketchSize,
		InitHashTables: iht,
		hashTables:     ht,
		KeyLookup:      kl,
	}
}

// addKey will add a seqio.Key to the lookup map, storing it under a stringified version of the key
func (LSHforest *LSHforest) addKey(key *seqio.Key) error {
	LSHforest.mapLock.Lock()
	defer LSHforest.mapLock.Unlock()
	//if _, ok := LSHforest.KeyLookup[key.StringifiedKey]; ok {
	//	return fmt.Errorf("key already in LSH Forest: %v", key.StringifiedKey)
	//}
	LSHforest.KeyLookup[key.StringifiedKey] = key
	return nil
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
		prefixSize := misc.HASH_SIZE * LSHforest.K
		// run concurrent hashtable queries
		keyChan := make(chan string)
		var wg sync.WaitGroup
		wg.Add(LSHforest.L)
		for i := 0; i < LSHforest.L; i++ {
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
