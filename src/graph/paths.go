package graph

import (
	"fmt"
	"sort"
	"sync"

	"github.com/will-rowe/baby-groot/src/markov"
	"github.com/will-rowe/baby-groot/src/misc"
)

// grootGraphPath
type grootGraphPath struct {
	name      []byte
	nodes     []uint64
	sequences [][]byte
	weights   []float64
}

// pathMatch is a method to check if two paths are identical
func (grootGraphPath *grootGraphPath) pathMatch(queryPath *grootGraphPath) bool {
	if len(queryPath.nodes) != len(grootGraphPath.nodes) {
		return false
	}
	match := true
	for i, value := range queryPath.nodes {
		if value != grootGraphPath.nodes[i] {
			match = false
			break
		}
	}
	return match
}

// segWeightPair is a struct used to hold nodes so that they can be sorted by their weight
type segWeightPair struct {
	id     uint64  // the segmentID field of a node
	weight float64 // the k-mer frequency field of a node
}

// ByWeight implements sort.Interface for []weights based on the weight field
type ByWeight []segWeightPair

func (a ByWeight) Len() int           { return len(a) }
func (a ByWeight) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ByWeight) Less(i, j int) bool { return a[i].weight > a[j].weight }

// BuildMarkovChain is a method to add the paths from a graph to a markov chain model
func (GrootGraph *GrootGraph) BuildMarkovChain(chain *markov.Chain) error {
	// see if path finding has already been run on this graph
	if len(GrootGraph.mcmcPaths) != 0 {
		return fmt.Errorf("MCMC path finding has already been done on this graph - possible duplicate graph")
	}

	// get all the reference paths
	if len(GrootGraph.originalPaths) == 0 {
		if err := GrootGraph.GetPaths(); err != nil {
			return err
		}
	}

	// add the paths to the markov chain
	for _, path := range GrootGraph.originalPaths {
		// convert sequences from []byte to []string
		seqs := make([]string, len(path.sequences))
		for i, seq := range path.sequences {
			seqs[i] = string(seq)
		}
		chain.Add(seqs, path.weights)
	}

	// TODO: add some checks, possible a mark to note that a graph has been added to the chain
	return nil
}

// FindMarkovPaths is a method to collect probable paths through the graph
func (GrootGraph *GrootGraph) FindMarkovPaths(chain *markov.Chain, bootstraps int, scaling float64) error {

	// get a copy of the startingNodes
	startingNodes, err := GrootGraph.GetStartNodes()
	if err != nil {
		return err
	}

	// setupPathSearch is a function that sets up and returns a start node, an empty path tracker and path memory, and any error
	setupPathSearch := func() (*GrootGraphNode, *grootGraphPath, []string, error) {
		// set up the path tracker
		pathTracker := &grootGraphPath{}

		// take a starting nodeID from the top of the slice
		nodeID := startingNodes[0]

		// add it to the tracker
		pathTracker.nodes = []uint64{nodeID}

		// get the node
		node, err := GrootGraph.GetNode(nodeID)
		if err != nil {
			return nil, nil, nil, err
		}

		// check to make sure this starting node has some edges...
		if len(node.OutEdges) == 0 {
			return nil, nil, nil, fmt.Errorf("graph has an unconnected starting node")
		}

		// create a short term path memory
		pathMemory := make([]string, chain.Order, chain.Order+1)
		for i := 0; i < chain.Order; i++ {
			pathMemory[i] = markov.StartToken
		}
		return node, pathTracker, pathMemory, nil
	}

	// start a pathCount
	pathCount := 0

	// run the path builder inside a go routine
	pathSend := make(chan *grootGraphPath)
	errChannel := make(chan error, 1)
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		wg.Wait()
		close(pathSend)
		close(errChannel)
	}()

	go func() {

		// get all the markov paths from each starting node in the graph
		for {

			// get the starting node, path tracker, memory
			node, pathTracker, pathMemory, err := setupPathSearch()
			if err != nil {
				errChannel <- err
			}

			// start following a path
			for {

				// if there are no more edges in the current path, report the current path and then go back to the start node and try another path
				if len(node.OutEdges) == 0 {
					// send the path by a channel
					pathSend <- pathTracker

					// increment the path counter
					pathCount++

					// start path build again from the start node
					node, pathTracker, pathMemory, err = setupPathSearch()
					if err != nil {
						errChannel <- err
					}
				}

				// if the graph has been traversed X times, stop finding markov paths from this start node
				if pathCount >= bootstraps {
					if len(startingNodes) != 1 {
						pathCount = 0
						break
					}
					// if all start nodes have been used, end the path finding
					wg.Done()
					return
				}

				// add the current node sequence to the end of the pathMemory, and remove the oldest item
				pathMemory = append(pathMemory, string(node.Sequence))
				pathMemory = pathMemory[1:]

				// if the current node has only one edge, get ready to follow it
				if len(node.OutEdges) == 1 {
					node, err = GrootGraph.GetNode(node.OutEdges[0])
					if err != nil {
						errChannel <- err
					}
				} else {

					// otherwise, we need to choose which edge we are going to follow
					segWeightPairs := make([]segWeightPair, len(node.OutEdges))

					// start by getting all the weights
					for i, edgeID := range node.OutEdges {
						edgeNode, err := GrootGraph.GetNode(edgeID)
						if err != nil {
							errChannel <- err
						}
						edgeWeight, err := chain.TransitionProbability(string(edgeNode.Sequence), pathMemory)
						segWeightPairs[i] = segWeightPair{edgeID, edgeWeight}
					}

					// sort the edges by decreasing weight
					sort.Sort(ByWeight(segWeightPairs))

					// generate a random number from a uniform distribution, in the range 0..1
					randomNumber := GrootGraph.rng.Float64Range(0.0, 1.0)

					// now select an edge
					var selectedEdgeID uint64
					selected := false
					total := 0.0
					for _, edge := range segWeightPairs {
						total += edge.weight
						if total > randomNumber {
							selectedEdgeID = edge.id
							selected = true
							pathTracker.weights = append(pathTracker.weights, total)
							break
						}
					}

					// check we selected an edge, if we didn't then end the current path search
					if selected == false {
						pathCount++
						break
						//errChannel <- fmt.Errorf("could not find suitable edge from node %d", node.SegmentID)
					}

					// get ready to follow the selected node
					node, err = GrootGraph.GetNode(selectedEdgeID)
					if err != nil {
						errChannel <- err
					}
				}

				// re-weight the transition to the selected node
				if scaling != 0.0 {
					err := chain.Scale(string(node.Sequence), pathMemory, scaling)
					if err != nil {
						errChannel <- err
					}
				}

				// add the ID of the selected node to the tracker before going on to check its edges
				pathTracker.nodes = append(pathTracker.nodes, node.SegmentID)

			}

			// pop the first start node out of the list
			startingNodes = startingNodes[1:]

			// if there are no more starting nodes, we've finished building paths
			if len(startingNodes) == 0 {
				break
			}
		} // have now finished checking each starting point for possible paths
	}()

	// collect all the paths
	paths := []*grootGraphPath{}
	for path := range pathSend {
		if len(paths) == 0 {
			paths = append(paths, path)
		}
		keep := true
		for _, x := range paths {
			if exists := misc.Uint64SliceEqual(path.nodes, x.nodes); exists {
				keep = false
				break
			}
		}
		if keep {
			pathHolder := ""
			for _, node := range path.nodes {
				pathHolder += fmt.Sprintf("%d+,", node)
			}
			paths = append(paths, path)
		}
	}
	GrootGraph.mcmcPaths = paths

	// check there were no errors
	if err := <-errChannel; err != nil {
		return err
	}

	return nil
}

// ProcessMarkovPaths is a method to
func (GrootGraph *GrootGraph) ProcessMarkovPaths(probCutoff float64) error {
	if len(GrootGraph.mcmcPaths) == 0 {
		return fmt.Errorf("no markov paths to process for this graph")
	}
	keptPaths := []*grootGraphPath{}
	for pathCount, path := range GrootGraph.mcmcPaths {
		// get the combined probability of each graph transition in this path
		combProb := path.weights[0]
		for _, prob := range path.weights {
			combProb *= prob
		}
		// discard any paths below the probability threshold
		if combProb < probCutoff {
			continue
		}
		// check this mcmcPath against the original paths
		for _, originalPath := range GrootGraph.originalPaths {
			// use the name if we have it
			if path.pathMatch(originalPath) {
				path.name = originalPath.name
				break
			}
		}
		// if we don't know the name, give it one using the pathCount
		if path.name == nil {
			name := fmt.Sprintf("groot-graph-%d-unknownPath-%d", GrootGraph.GraphID, pathCount)
			path.name = []byte(name)
		}
		// keep this path
		keptPaths = append(keptPaths, path)
	}
	// replace the original paths with only the ones passing the probability theshold
	GrootGraph.mcmcPaths = keptPaths
	if len(GrootGraph.mcmcPaths) == 0 {
		return fmt.Errorf("no markov paths were kept for this graph")
	}
	return nil
}

// GetMarkovPaths is a method to return the identified paths
func (GrootGraph *GrootGraph) GetMarkovPaths() ([]string, error) {
	if len(GrootGraph.mcmcPaths) == 0 {
		return nil, fmt.Errorf("no markov paths were found for this graph")
	}
	// collect each path as a string, in GFA path format
	paths := make([]string, len(GrootGraph.mcmcPaths))
	for i, path := range GrootGraph.mcmcPaths {
		// path holder
		pathHolder := ""
		for _, node := range path.nodes {
			pathHolder += fmt.Sprintf("%d+,", node)
		}
		prefix := fmt.Sprintf("P\t%v\t", string(path.name))
		paths[i] = prefix + pathHolder
	}
	return paths, nil
}
