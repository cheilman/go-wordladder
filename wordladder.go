package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"unicode"
)

const wordFile = "/usr/share/dict/words"
const forestGraphFile = "wordForest.json"

var wordGraph *WordGraph

func main() {
	wordGraph = NewWordGraph()

	//
	// See if we have a pre-processed forest graph
	//

	// Open a RO file
	decodeFile, err := os.Open(forestGraphFile)
	if err != nil {

		//
		// Load all words into graph
		//
		fmt.Printf("Loading words from %v.\n", wordFile)

		// Open the file
		var f, err = os.Open(wordFile)
		if err != nil {
			panic(err)
		}

		// Read each word into the graph
		var scanner = bufio.NewScanner(f)
		for scanner.Scan() {
			var word = scanner.Text()
			if isValidWord(&word) {
				wordGraph.AddWord(word)
			}
		}

		//
		// Start assigning forests and neighbors
		//
		fmt.Printf("Assigning forests and analyzing neighbors.  There are %v distinct word lengths.\n", wordGraph.GetTotalDistinctWordLengths())

		wordGraph.ExploreForests()

		fmt.Printf("Assigned %v words into %v forests.\n", wordGraph.GetTotalWords(), wordGraph.GetTotalForests())

		//
		// Serialize forest map
		//

		forestFile, err := os.Create(forestGraphFile)
		if err != nil {
			panic(err)
		}

		// Dump it to JSON
		encoder := json.NewEncoder(forestFile)

		// Write to the file
		if err := encoder.Encode(wordGraph); err != nil {
			panic(err)
		}
		forestFile.Close()
	} else {

		//
		// Load the pre-processed graph into memory
		//

		fmt.Printf("Reading pre-processed graph from %v.\n", forestGraphFile)

		defer decodeFile.Close()

		// Create a decoder
		decoder := json.NewDecoder(decodeFile)

		// Decode -- We need to pass a pointer otherwise wordGraph isn't modified
		decoder.Decode(&wordGraph)

		// And let's just make sure it all worked
		fmt.Printf("Loaded pre-processed forest graph.  %v distinct word lengths in graph.\n", wordGraph.GetTotalDistinctWordLengths())

		for _, subgraph := range wordGraph.Graphs {
			fmt.Printf("There are %v words of size %v.\n", subgraph.GetTotalWords(), subgraph.WordLength)
		}
	}

	//
	// Run some tests
	//
	var pairs = [][]string{{"cat", "dog"}, {"ape", "man"}, {"pig", "sty"}, {"pen", "ink"}, {"one", "two"}, {"bat", "cry"},
		{"goat", "fish"}, {"bake", "farm"}, {"lawn", "brat"},
		{"snake", "cards"}, {"plant", "graph"}}

	for _, p := range pairs {
		var s1, s2 = p[0], p[1]
		fmt.Printf("%v -> %v: ", s1, s2)
		fmt.Printf("%v\n", wordGraph.AreTwoWordsConnected(s1, s2))
		fmt.Printf("%v -> %v: ", s2, s1)
		fmt.Printf("%v\n", wordGraph.AreTwoWordsConnected(s2, s1))
	}

	fmt.Printf("\n")

	for _, p := range pairs {
		var s1, s2 = p[0], p[1]
		fmt.Printf("%v -> %v: %v\n", s1, s2, wordGraph.ShortestPath(s1, s2))
		fmt.Printf("%v -> %v: %v\n", s2, s1, wordGraph.ShortestPath(s2, s1))
	}

}

// Will we import this word from the word list into our forest graph?
func isValidWord(s *string) bool {
	for _, c := range *s {
		// Skip words with non-letters
		if !unicode.IsLetter(c) {
			return false
		}

		// Skip words with capitals
		if !unicode.IsLower(c) {
			return false
		}
	}

	return true
}

// Are two words within one change of each other?
func areNeighbors(s1 string, s2 string) bool {
	if len(s1) != len(s2) {
		return false
	}

	var foundOneChange = false

	for i := 0; i < len(s1); i++ {
		if s1[i] != s2[i] {
			if foundOneChange {
				// Already found a change, this is too many
				return false
			}

			foundOneChange = true
		}
	}

	return true
}

// How many changes are needed to go from one word to another?
func distance(s1 string, s2 string) int {
	if len(s1) != len(s2) {
		// Impossible
		return math.MaxInt32
	}

	var retval = 0

	for i := 0; i < len(s1); i++ {
		if s1[i] != s2[i] {
			retval++
		}
	}

	return retval
}

// A queue of word nodes.  For our forest exploration BFS.
type WNQueue struct {
	nodes []*WordNode // nodes in the queue
}

func (q *WNQueue) push(n *WordNode) {
	q.nodes = append(q.nodes, n)
}

func (q *WNQueue) pop() *WordNode {
	if len(q.nodes) > 0 {
		var retval = q.nodes[0]
		q.nodes = q.nodes[1:]
		return retval
	} else {
		return nil
	}
}

// A queue of word nodes with pathing information.  For our shortest path BFS.
type WNPathQueueNode struct {
	wn     *WordNode
	parent *WNPathQueueNode
}

type WNPathQueue struct {
	nodes []*WNPathQueueNode // nodes in the queue
}

func (q *WNPathQueue) push(n *WNPathQueueNode) {
	q.nodes = append(q.nodes, n)
}

func (q *WNPathQueue) pop() *WNPathQueueNode {
	if len(q.nodes) > 0 {
		var retval = q.nodes[0]
		q.nodes = q.nodes[1:]
		return retval
	} else {
		return nil
	}
}

/**
 * A node in the graph (represents a word and its neighbors).
 */
type WordNode struct {
	Word      string    // the word itself
	ForestTag int       // what forest the word lives in
	Neighbors []*string // list of one-character neighbors
}

/**
 * A set of forests of words of all the same length.
 */
type WordGraphOfSameLength struct {
	curForest  int                  // Forest tag counter.  Forest tags are not unique across different word lengths
	WordLength int                  // Length of words in this group
	WordGraph  map[string]*WordNode // Map of words in the graph
}

// Initialize
func NewWordGraphOfSameLength(len int) *WordGraphOfSameLength {
	return &WordGraphOfSameLength{curForest: 1, WordLength: len, WordGraph: make(map[string]*WordNode)}
}

// Add a word to the graph
func (g *WordGraphOfSameLength) AddWord(word string) {
	if len(word) != g.WordLength {
		panic("Trying to add a word of the incorrect length!")
	}

	g.WordGraph[word] = &WordNode{Word: word, ForestTag: 0, Neighbors: nil}
}

func (g *WordGraphOfSameLength) GetTotalWords() int {
	return len(g.WordGraph)
}

func (g *WordGraphOfSameLength) GetTotalForests() int {
	return g.curForest - 1
}

// Figure out the neighbors of a node by filtering the word list, rather than by generation of all possible words.
// Should be faster depending on length of word and size of dictionary.
func (g *WordGraphOfSameLength) figureOutNeighbors(node *WordNode) []*string {
	var retval = []*string{}

	for _, v := range g.WordGraph {
		if areNeighbors(node.Word, v.Word) {
			retval = append(retval, &v.Word)
		}
	}

	return retval
}

// Finds neighbors for a node, explores them (and finds neighbors for those nodes), and assigns all
// connected nodes the same forest tag.
// Returns number of nodes explored
func (g *WordGraphOfSameLength) exploreForest(startWord *WordNode) int {
	var retval = 0

	var q = WNQueue{}
	q.push(startWord)

	for {
		var node = q.pop()

		if node == nil {
			break
		} else {
			// Filter out visited people
			if node.ForestTag > 0 {
				continue
			}

			retval++

			// Tag the forest
			node.ForestTag = g.curForest

			// Figure out the neighbors
			var neighbors = g.figureOutNeighbors(node)
			node.Neighbors = make([]*string, len(neighbors))
			copy(node.Neighbors, neighbors)

			// Search Neighbors
			for _, neigh := range neighbors {
				q.push(g.WordGraph[*neigh])
			}
		}
	}

	return retval
}

// Explore the entire graph, finding all forests and neighbors
func (g *WordGraphOfSameLength) ExploreAllForests() {
	for _, v := range g.WordGraph {
		if v.ForestTag <= 0 {
			// It's unassigned so far, need to figure out where it belongs.
			g.exploreForest(v)

			// Move along to the next forest
			g.curForest++
		} else {
			// Assigned, ignore it
		}
	}
}

// Does a path exist between two strings?  O(1) check by looking at matching forest
// tags (the work was done in pre-processing).
func (g *WordGraphOfSameLength) AreTwoWordsConnected(s1 string, s2 string) bool {
	// Valid words check
	if g.WordGraph[s1] == nil || g.WordGraph[s2] == nil {
		return false
	}

	return g.WordGraph[s1].ForestTag == g.WordGraph[s2].ForestTag
}

// Return a shortest path from s1 to s2.  Nil if no path exists.
// Could be optimized with a priority queue and some hamming distance calculations (maybe that's A*?)
func (g *WordGraphOfSameLength) ShortestPath(s1 string, s2 string) []string {
	if !g.AreTwoWordsConnected(s1, s2) {
		// No path exists
		return nil
	}

	// We actually search backwards (s2 -> s1), so we don't have to reverse the string
	// at the end (since the path is built by following parent links up from the end)

	var visited = make(map[string]bool)
	var target *WNPathQueueNode = nil

	var q = WNPathQueue{}
	q.push(&WNPathQueueNode{wn: g.WordGraph[s2], parent: nil})

	for {
		var node = q.pop()

		if node == nil {
			return nil
		} else {
			// Have we found our target word?
			if node.wn.Word == s1 {
				target = node // Save to follow the path back up
				break
			}

			// check neighbors that haven't been visited
			for _, neighborWord := range node.wn.Neighbors {

				if !visited[*neighborWord] {
					visited[*neighborWord] = true

					var neighborNode = g.WordGraph[*neighborWord]

					// Add nodes with the parent set
					q.push(&WNPathQueueNode{wn: neighborNode, parent: node})
				}
			}
		}
	}

	if target == nil {
		// Didn't find it.  I'm not sure if this can happen, we should be safe from the areTwoWordsConnected() check
		return nil
	}

	// Build the path back up
	var retval = []string{}

	var cur = target
	for {
		if cur == nil {
			break
		}

		retval = append(retval, cur.wn.Word)
		cur = cur.parent
	}

	return retval
}

/**
 * Set of graphs of different length words.
 */
type WordGraph struct {
	Graphs     map[int]*WordGraphOfSameLength // Map of length to graph
	totalWords int
}

// Initialize
func NewWordGraph() *WordGraph {
	return &WordGraph{Graphs: make(map[int]*WordGraphOfSameLength), totalWords: 0}
}

// Add a word to the appropriate subgraph
func (g *WordGraph) AddWord(word string) {
	var l = len(word)

	_, present := g.Graphs[l]
	if !present {
		// Create new map of the right length
		g.Graphs[l] = NewWordGraphOfSameLength(l)
	}
	g.Graphs[l].AddWord(word)
}

func (g *WordGraph) ExploreForests() {
	for _, subgraph := range g.Graphs {
		subgraph.ExploreAllForests()
	}
}

// Does a path exist between two strings?  Figure out what length we're looking at and pass it along
func (g *WordGraph) AreTwoWordsConnected(s1 string, s2 string) bool {
	if len(s1) != len(s2) {
		return false
	}

	return g.Graphs[len(s1)].AreTwoWordsConnected(s1, s2)
}

// Return a shortest path from s1 to s2.  Nil if no path exists.
// Could be optimized with a priority queue and some hamming distance calculations (maybe that's A*?)
func (g *WordGraph) ShortestPath(s1 string, s2 string) []string {
	if len(s1) != len(s2) {
		return nil
	}

	return g.Graphs[len(s1)].ShortestPath(s1, s2)
}

func (g *WordGraph) GetTotalWords() int {
	var retval = 0

	for _, subgraph := range g.Graphs {
		retval += subgraph.GetTotalWords()
	}

	return retval
}

func (g *WordGraph) GetTotalDistinctWordLengths() int {
	return len(g.Graphs)
}

func (g *WordGraph) GetTotalForests() int {
	var retval = 0

	for _, subgraph := range g.Graphs {
		retval += subgraph.GetTotalForests()
	}

	return retval
}
