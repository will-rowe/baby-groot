// this package is used to process and convert between MSA, GFA graphs and GROOT graphs
package graph

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strconv"
	"time"

	"github.com/will-rowe/gfa"
	"gopkg.in/vmihailenco/msgpack.v2"
)

/*
  A struct to store multiple graphs
*/
type GraphStore map[int]*GrootGraph

// Dump is a method to save a GrootGraph to file
func (graphStore *GraphStore) Dump(path string) error {
	b, err := msgpack.Marshal(graphStore)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(path, b, 0644)
}

// Load is a method to load a GrootGraph from file
func (graphStore *GraphStore) Load(path string) error {
	b, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}
	return msgpack.Unmarshal(b, graphStore)
}

// SaveGraphAsGFA is a method to convert and save a GrootGraph in GFA format
func (graph *GrootGraph) SaveGraphAsGFA(fileName string) (int, error) {
	// a flag to prevent dumping graphs which had no reads map
	graphUsed := false
	t := time.Now()
	stamp := fmt.Sprintf("variation graph created by groot at: %v", t.Format("Mon Jan _2 15:04:05 2006"))
	msg := []byte("this graph is approximately weighted using k-mer frequencies from projected read sketches")
	// create a GFA instance
	newGFA := gfa.NewGFA()
	_ = newGFA.AddVersion(1)
	newGFA.AddComment([]byte(stamp))
	newGFA.AddComment(msg)
	// transfer all the GrootGraphNode content to the GFA instance
	for _, node := range graph.SortedNodes {
		// BABY GROOT - some nodes will be nil after pruning, ignore these
		if node == nil {
			continue
		}
		// record if this graph has had reads map
		if (graphUsed == false) && (node.KmerFreq > 0) {
			graphUsed = true
		}
		segID := strconv.FormatUint(node.SegmentID, 10)
		// create the segment
		seg, err := gfa.NewSegment([]byte(segID), []byte(node.Sequence))
		if err != nil {
			return 0, err
		}
		// the k-mer count corresponds to the node weight, which is its share of the k-mers from the projected sketches
		kmerCount := fmt.Sprintf("KC:i:%d", int((node.KmerFreq)))
		ofs, err := gfa.NewOptionalFields([]byte(kmerCount))
		if err != nil {
			return 0, err
		}
		seg.AddOptionalFields(ofs)
		seg.Add(newGFA)
		// create the links
		for _, outEdge := range node.OutEdges {
			toSeg := strconv.FormatUint(outEdge, 10)
			link, err := gfa.NewLink([]byte(segID), []byte("+"), []byte(toSeg), []byte("+"), []byte("0M"))
			if err != nil {
				return 0, err
			}
			link.Add(newGFA)
		}
	}
	// don't save the graph if no reads aligned
	if graphUsed == false {
		return 0, nil
	}
	// create the paths
	for pathID, pathName := range graph.Paths {
		segments, overlaps := [][]byte{}, [][]byte{}
		for _, node := range graph.SortedNodes {
			// BABY GROOT - some nodes will be nil after pruning, ignore these
			if node == nil {
				continue
			}
			for _, id := range node.PathIDs {
				if id == pathID {
					segment := strconv.FormatUint(node.SegmentID, 10) + "+"
					overlap := strconv.Itoa(len(node.Sequence)) + "M"
					segments = append(segments, []byte(segment))
					overlaps = append(overlaps, []byte(overlap))
					break
				}
			}
		}
		// add the path
		path, err := gfa.NewPath(pathName, segments, overlaps)
		if err != nil {
			return 0, err
		}
		path.Add(newGFA)
	}
	// create a gfaWriter and write the GFA instance
	outfile, err := os.Create(fileName)
	if err != nil {
		return 0, err
	}
	defer outfile.Close()
	writer, err := gfa.NewWriter(outfile, newGFA)
	if err != nil {
		return 0, err
	}
	err = newGFA.WriteGFAContent(writer)
	return 1, nil
}

// LoadGFA reads a GFA file into a GFA struct
func LoadGFA(fileName string) (*gfa.GFA, error) {
	// load the GFA file
	fh, err := os.Open(fileName)
	reader, err := gfa.NewReader(fh)
	if err != nil {
		return nil, fmt.Errorf("can't read gfa file: %v", err)
	}
	// collect the GFA instance
	myGFA := reader.CollectGFA()
	// read the file
	for {
		line, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("error reading line in gfa file: %v", err)
		}
		if err := line.Add(myGFA); err != nil {
			return nil, fmt.Errorf("error adding line to GFA instance: %v", err)
		}
	}
	return myGFA, nil
}
