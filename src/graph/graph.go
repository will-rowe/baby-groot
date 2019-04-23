// this package is used to process and convert between MSA, GFA graphs and GROOT graphs
package graph

import (
	"bytes"
	"fmt"
	"sort"
	"strconv"
	"sync"

	"github.com/will-rowe/bg/src/bitvector"
	"github.com/will-rowe/bg/src/seqio"
	"github.com/will-rowe/gfa"
)

/*
   GrootGraph is the variation graph implementation used by GROOT
*/
type GrootGraph struct {
	GrootVersion string
	GraphID      int
	SortedNodes  []*GrootGraphNode
	NodeLookup   map[uint64]int
	Paths        map[int][]byte // lookup to relate PathIDs in each node to a path name
	Lengths      map[int]int    // lengths of sequences held in graph (lookup key corresponds to key in Paths)
	Pruned       bool           // switch to let us know if the graph has been pruned, so will contain nil nodes in the sorted list
}

// CreateGrootGraph is a GrootGraph constructor that takes a GFA instance and stores the info as a graph and then runs a topological sort
func CreateGrootGraph(gfaInstance *gfa.GFA, id int) (*GrootGraph, error) {
	// construct an empty graph
	newGraph := &GrootGraph{
		GraphID:    id,
		NodeLookup: make(map[uint64]int),
		Paths:      make(map[int][]byte),
		Lengths:    make(map[int]int),
	}
	// collect all the segments from the GFA instance and create the nodes
	segments, err := gfaInstance.GetSegments()
	if err != nil {
		return nil, err
	}
	for nodeIterator, segment := range segments {
		// check the segment name can be stored as an int

		// TODO: will fix the handling of segmentIDs between GFA and GROOT -- need to use uint64

		segID, err := strconv.Atoi(string(segment.Name))
		if err != nil {
			return nil, fmt.Errorf("could not convert segment name from GFA into an int for groot graph: %v", segment.Name)
		}
		// convert all bases to upperCase and check for non-ACTGN chars
		seq := seqio.Sequence{Seq: segment.Sequence}
		if err := seq.BaseCheck(); err != nil {
			return nil, err
		}
		newNode := &GrootGraphNode{
			SegmentID: uint64(segID),
			Sequence:  seq.Seq,
			Coverage:  bitvector.NewBitVector(len(seq.Seq)),
		}
		// store the new node in the graph and record it's location in the silce by using the NodeLookup map
		newGraph.SortedNodes = append(newGraph.SortedNodes, newNode)
		newGraph.NodeLookup[uint64(segID)] = nodeIterator
	}
	// collect all the links from the GFA instance and add edges to the nodes
	links, err := gfaInstance.GetLinks()
	if err != nil {
		return nil, err
	}
	for _, link := range links {
		// get the from and to segment IDs
		fromSegID, err := strconv.Atoi(string(link.From))
		if err != nil {
			return nil, fmt.Errorf("could not convert segment name from GFA into an int for groot graph: %v", link.From)
		}
		toSegID, err := strconv.Atoi(string(link.To))
		if err != nil {
			return nil, fmt.Errorf("could not convert segment name from GFA into an int for groot graph: %v", link.To)
		}
		// add the outEdges
		nodeLocator := newGraph.NodeLookup[uint64(fromSegID)]
		newGraph.SortedNodes[nodeLocator].OutEdges = append(newGraph.SortedNodes[nodeLocator].OutEdges, uint64(toSegID))
	}
	// collect all the paths from the GFA instance and add pathIDs to each node
	paths, err := gfaInstance.GetPaths()
	if err != nil {
		return nil, err
	}
	for pathIterator, path := range paths {
		// add the path name to the lookup
		newGraph.Paths[pathIterator] = path.PathName
		for _, seg := range path.SegNames {
			// strip the plus
			seg = bytes.TrimSuffix(seg, []byte("+"))
			segID, err := strconv.Atoi(string(seg))
			if err != nil {
				return nil, fmt.Errorf("could not convert segment name from GFA into an int for groot graph: %v", string(seg))
			}
			nodeLocator := newGraph.NodeLookup[uint64(segID)]
			newGraph.SortedNodes[nodeLocator].PathIDs = append(newGraph.SortedNodes[nodeLocator].PathIDs, pathIterator)
		}
	}
	// return without toposort if only one node present (graph with single sequence)
	if len(newGraph.SortedNodes) > 1 {
		err = newGraph.topoSort()
	}
	// get and store the lengths of each sequence held in the graph
	// this function also adds Position to each node of the graph
	for pathID, _ := range newGraph.Paths {
		newGraph.Lengths[pathID] = len(newGraph.Graph2Seq(pathID))
	}
	// return the new GrootGraph
	return newGraph, err
}

// topoSort runs a topological sort on the GrootGraph
func (graph *GrootGraph) topoSort() error {
	// copy all of the graph nodes into a map (so we can keep track of what we have processed)
	nodeMap := make(map[uint64]*GrootGraphNode)
	toposortStart := []uint64{}
	seenPaths := make(map[int]int)
	for _, node := range graph.SortedNodes {
		if len(seenPaths) == len(graph.Paths) {
			break
		}
		// if this node is from a sequence we have not seen yet, mark the node as a starting node for toposort
		for _, path := range node.PathIDs {
			if _, ok := seenPaths[path]; !ok {
				toposortStart = append(toposortStart, node.SegmentID)
			}
		}
		// check for duplicate nodes
		if _, ok := nodeMap[node.SegmentID]; ok {
			return fmt.Errorf("graph contains duplicate nodes (identical segment IDs)")
		}
		// add the node to a map
		nodeMap[node.SegmentID] = node
	}
	// clear the SortedNodes and NodeLookup
	graph.SortedNodes = []*GrootGraphNode{}
	graph.NodeLookup = make(map[uint64]int)
	// run the topological sort  - try starting from each node that was in the first slot of the nodeholder (start of the MSA)
	seen := make(map[uint64]struct{})
	for len(nodeMap) > 1 {
		for _, start := range toposortStart {
			if _, ok := nodeMap[start]; !ok {
				continue
			}
			graph.traverse(nodeMap[start], nodeMap, seen)
		}
	}
	// check all traversals have been taken
	if len(nodeMap) > 0 {
		return fmt.Errorf("topological sort failed - too many nodes remaining in the pre-sort list")
	}
	return nil
}

//  traverse is a helper method to topologically sort a graph
func (graph *GrootGraph) traverse(node *GrootGraphNode, nodeMap map[uint64]*GrootGraphNode, seen map[uint64]struct{}) {
	// skip if we are already handling the current node
	if _, ok := seen[node.SegmentID]; ok {
		return
	}
	// make sure node is still in the graph
	if _, ok := nodeMap[node.SegmentID]; ok {
		// record that we are processing this node
		seen[node.SegmentID] = struct{}{}
		// sort the output nodes for this node and then traverse them in reverse order
		sort.Sort(sort.Reverse(node.OutEdges))
		for i, j := 0, len(node.OutEdges); i < j; i++ {
			// check if the outedges have been traversed
			if _, ok := nodeMap[node.OutEdges[i]]; !ok {
				continue
			}
			graph.traverse(nodeMap[node.OutEdges[i]], nodeMap, seen)
		}
		// delete the node from the temporary holders
		delete(nodeMap, node.SegmentID)
		delete(seen, node.SegmentID)
		// update the sorted node slice and the lookup
		graph.SortedNodes = append([]*GrootGraphNode{node}, graph.SortedNodes...)
		graph.NodeLookup[node.SegmentID] = len(nodeMap)
	}
}

// Graph2Seq is a method to convert a variation graph to linear reference sequences
func (GrootGraph *GrootGraph) Graph2Seq(pathID int) []byte {
	newSequence := []byte{}
	for _, node := range GrootGraph.SortedNodes {
		// if the graph has been pruned, some nodes will have been set to nil
		if node == nil {
			continue
		}
		for _, id := range node.PathIDs {
			if id == pathID {
				// build the sequence
				newSequence = append(newSequence, node.Sequence...)
			}
		}
	}
	return newSequence
}

// WindowGraph is a method to slide a window over each path through the graph, sketching the paths and getting window information
func (GrootGraph *GrootGraph) WindowGraph(windowSize, kSize, sketchSize int) chan *seqio.Key {
	// this method returns a channel, which receives windows as they are made
	windowChan := make(chan *seqio.Key)
	var wg sync.WaitGroup
	wg.Add(len(GrootGraph.Paths))

	// window each path
	for pathID, _ := range GrootGraph.Paths {
		go func(pathID int) {
			defer wg.Done()
			// get the length of the linear reference for this path
			pathLength := GrootGraph.Lengths[pathID]

			// for each base in the linear reference sequence, get the segmentID and offset of its location in the graph
			segs := make([]uint64, pathLength, pathLength)
			offSets := make([]int, pathLength, pathLength)
			iterator := 0
			for _, node := range GrootGraph.SortedNodes {
				for _, id := range node.PathIDs {
					if id == pathID {
						for offset := 0; offset < len(node.Sequence); offset++ {
							segs[iterator] = node.SegmentID
							offSets[iterator] = offset
							iterator++
						}
					}
				}
			}

			// get the sequence for this path
			sequence := GrootGraph.Graph2Seq(pathID)

			// window the sequence
			numWindows := pathLength - windowSize + 1
			for i := 0; i < numWindows; i++ {
				// sketch the window
				windowSeq := seqio.Sequence{Seq: sequence[i : i+windowSize]}
				sketch, err := windowSeq.RunMinHash(kSize, sketchSize)
				if err != nil {
					panic(err)
				}
				// get the subpath (make a copy and remove duplicates)
				subpath := segs[i : i+windowSize]
				sp := make([]uint64, 1, len(subpath))
				for x, y := range subpath {
					if x == 0 {
						sp[0] = uint64(y)
					} else {
						if uint64(y) != sp[len(sp)-1] {
							sp = append(sp, uint64(y))
						}
					}
				}

				// populate the window struct
				newWindow := &seqio.Key{
					GraphID: GrootGraph.GraphID,
					Node:    segs[i],
					OffSet:  offSets[i],
					SubPath: sp,
					Ref:     pathID,
					Sketch:  sketch,
				}

				// send this window
				windowChan <- newWindow
			}
		}(pathID)
	}
	go func() {
		wg.Wait()
		close(windowChan)
	}()
	return windowChan
}

// IncrementSubPath is a method to adjust the weight of segments within a subpath through the graph
////////////
// BABY GROOT
// given a subpath through a graph, the offset in the first segment, the end point in the final segment
// combined with the window size and number of k-mers used for sketching the graph
// increment the weight of each segment contained in that subpath by their share of the k-mer coverage for the window
///////////
func (graph *GrootGraph) IncrementSubPath(subPath []uint64, offSet int, windowSize int, kmerSize int) error {
	// check the subpath contains segments
	if len(subPath) < 1 {
		return fmt.Errorf("subpath encountered that does not include any segments...")
	}

	// get the total number of k-mers this subpath is based on
	numKmers := float64(windowSize - kmerSize + 1)

	// if the subPath is only one segment, then it is straightforward to increment
	if len(subPath) == 1 {
		// lookup the node in the graph
		nodeLookup, ok := graph.NodeLookup[subPath[0]]
		if !ok {
			return fmt.Errorf("could not perform nodelookup to increment subpath weight")
		}

		// add the node coverage
		graph.SortedNodes[nodeLookup].AddCoverage(offSet, windowSize)

		// give this segment all the k-mers for this sketch
		if err := graph.SortedNodes[nodeLookup].IncrementWeight(numKmers); err != nil {
			return err
		}

		return nil
	}

	// otherwise, there are multiple segments in the path and we now work out the proportion of k-mer coverage belonging to each segment
	totalBases := 0

	// iterate over the segments in the subpath
	for i := 0; i < len(subPath); i++ {
		// lookup the node in the graph
		nodeLookup, ok := graph.NodeLookup[subPath[i]]
		if !ok {
			return fmt.Errorf("could not perform nodelookup to increment subpath weight")
		}
		// get the node and it's length
		node := graph.SortedNodes[nodeLookup]
		nodeLength := len(node.Sequence)

		// calculate the increment based on the segment length and any offset for the sketch that has been projected onto the graph
		increment := 0.0

		switch i {
		// if this is the first node. There may be an offset so only apply increment for k-mers for the covered portion of the segment
		case 0:
			// add the node coverage
			node.AddCoverage(offSet, windowSize)

			// mark that we have checked these bases
			totalBases += (nodeLength - offSet)

			// calculate it's share of the k-mers
			increment = (float64(totalBases) / float64(windowSize)) * numKmers

			// increment the current segment with it's share of the k-mers
			if err := node.IncrementWeight(increment); err != nil {
				return err
			}
			continue

		// if this is the final node, check where the path finishes within this segment
		case (len(subPath) - 1):
			// determine how much of the final node is covered
			coveredPortion := windowSize - totalBases

			// add the node coverage
			node.AddCoverage(0, coveredPortion)

			// calculate it's share of the k-merst
			increment = (float64(coveredPortion) / float64(windowSize)) * numKmers

			// increment the current segment with it's share of the k-mers
			if err := node.IncrementWeight(increment); err != nil {
				return err
			}

			// mark that we have checked these bases
			totalBases += coveredPortion
			continue

			// the default is that the whole segment is covered, so just use the ratio of segment length to windowSize
		default:
			// add the node coverage
			node.AddCoverage(0, nodeLength)

			// calculate it's share of the k-mers
			increment = (float64(nodeLength) / float64(windowSize)) * numKmers

			// increment the current segment with it's share of the k-mers
			if err := node.IncrementWeight(increment); err != nil {
				return err
			}

			// mark that we have checked these bases
			totalBases += nodeLength
			continue
		}
	}
	// check we have covered enough bases from the subpath segments to match the window size
	if totalBases != windowSize {
		return fmt.Errorf("could not get enough bases from the subpath segments")
	}

	return nil
}

// Prune is a method to remove paths and segments from the graph if they have insufficient coverage
// returns false if pruning would result in no paths through the graph remaining
func (graph *GrootGraph) Prune(minKmerCoverage, minBaseCoverage float64) bool {
	removePathId := make(map[int]struct{})
	removeNode := make(map[uint64]struct{})

	// first pass through the graph
	for _, node := range graph.SortedNodes {
		// check to see if the k-mer count or base coverage for this node are below the supplied threshold
		baseCoverage := float64(node.Coverage.PopCount()) / float64(len(node.Sequence))
		nodeCoverage := node.Weight / float64(len(node.Sequence))
		if nodeCoverage < minKmerCoverage || baseCoverage < minBaseCoverage {
			// add the segmentID and the contained pathIDs to the removal lis
			for _, id := range node.PathIDs {
				removePathId[id] = struct{}{}
				removeNode[node.SegmentID] = struct{}{}
			}
		}
	}
	// if all the paths need removing, just exit now!
	if len(removePathId) == len(graph.Paths) {
		return false
	}
	// if it doesn't need pruning, return true
	if len(removeNode) == 0 {
		return true
	}
	// second pass through the graph to prune all the marked nodes and paths
	// TODO: I'm just creating a new slice at the moment and copying nodes which aren't mark
	// TODO: shall I try popping elements out of the original slices instead -- is that better for time?
	for i, node := range graph.SortedNodes {
		// remove marked paths
		updatedPathIDs := make([]int, 0, len(node.PathIDs))
		for _, id := range node.PathIDs {
			if _, marked := removePathId[id]; !marked {
				updatedPathIDs = append(updatedPathIDs, id)
			}
		}
		node.PathIDs = updatedPathIDs
		// delete any marked nodes
		if _, marked := removeNode[node.SegmentID]; marked {
			// TODO: I've set the node to nil in the sorted node array - in order to keep the NodeLookup in order. But this isn't pretty and now requires you to check for nil when using the SortedNodes array
			graph.SortedNodes[i] = nil
			delete(graph.NodeLookup, node.SegmentID)
		}
		// remove any edges referencing deleted nodes
		updatedEdges := make([]uint64, 0, len(node.OutEdges))
		for _, edge := range node.OutEdges {
			if _, marked := removeNode[edge]; !marked {
				updatedEdges = append(updatedEdges, edge)
			}
		}
		node.OutEdges = updatedEdges
	}
	// remove the pruned paths and their lengths from the graph metadata
	for id := range removePathId {
		if _, path := graph.Paths[id]; path {
			delete(graph.Paths, id)
			delete(graph.Lengths, id)
		}
	}
	graph.Pruned = true
	return true
}
