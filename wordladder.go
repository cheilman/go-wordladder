package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"unicode"
)

type WordNode struct {
	Word      string    // the word itself
	ForestTag int       // what forest the word lives in
	Neighbors []*string // list of one-character neighbors
}

const wordFile = "/usr/share/dict/words"
const forestGraphFile = "wordForest.json"

var wordGraph = make(map[int]map[string]*WordNode)
var curForest = 1 // valid forest tags are positive integers

func main() {
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
				// Add to the appropriate subgraph (by length)
				var l = len(word)

				_, present := wordGraph[l]
				if !present {
					// Create new map of the right length
					wordGraph[l] = make(map[string]*WordNode)
				}
				wordGraph[l][word] = &WordNode{word, 0, nil}
			}
		}

		//
		// Start assigning forests and neighbors
		//
		fmt.Printf("Assigning forests and analyzing neighbors.  %v distinct word length(s) in graph.\n", len(wordGraph))
		for l, subgraph := range wordGraph {
			fmt.Printf("Looking at words of size %v, there are %v words\n", l, len(subgraph))
			for _, v := range subgraph {
				if v.ForestTag <= 0 {
					// It's unassigned so far, need to figure out where it belongs.
					exploreForest(v)

					// Move along to the next forest
					curForest++
				} else {
					// Assigned, ignore it
				}
			}
		}

		fmt.Printf("Found %v forest(s).\n", curForest-1)

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
		fmt.Printf("Loaded pre-processed forest graph.  %v distinct word lengths in graph.\n", len(wordGraph))
		for l, subgraph := range wordGraph {
			fmt.Printf("There are %v words of size %v.\n", len(subgraph), l)
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
		fmt.Printf("%v -> %v: %v\n", s1, s2, areTwoWordsConnected(s1, s2))
		fmt.Printf("%v -> %v: %v\n", s2, s1, areTwoWordsConnected(s2, s1))
	}

	fmt.Printf("\n")

	for _, p := range pairs {
		var s1, s2 = p[0], p[1]
		fmt.Printf("%v -> %v: %v\n", s1, s2, shortestPath(s1, s2))
		fmt.Printf("%v -> %v: %v\n", s2, s1, shortestPath(s2, s1))
	}

}

// Does a path exist between two strings?  O(1) check by looking at matching forest
// tags (the work was done in pre-processing).
func areTwoWordsConnected(s1 string, s2 string) bool {
	// Length check
	var l = len(s1)
	if l != len(s2) {
		return false
	}

	// Valid words check
	var subgraph = wordGraph[l]
	if subgraph[s1] == nil || subgraph[s2] == nil {
		return false
	}

	return subgraph[s1].ForestTag == subgraph[s2].ForestTag
}

// Return a shortest path from s1 to s2.  Nil if no path exists.
// Could be optimized with a priority queue and some hamming distance calculations (maybe that's A*?)
func shortestPath(s1 string, s2 string) []string {
	if !areTwoWordsConnected(s1, s2) {
		// No path exists
		return nil
	}

	var subgraph = wordGraph[len(s1)]

	// We actually search backwards (s2 -> s1), so we don't have to reverse the string
	// at the end (since the path is built by following parent links up from the end)

	var visited = make(map[string]bool)
	var target *WNPathQueueNode = nil

	var q = WNPathQueue{}
	q.push(&WNPathQueueNode{wn: subgraph[s2], parent: nil})

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

					var neighborNode = subgraph[*neighborWord]

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

// Finds neighbors for a node, explores them (and finds neighbors for those nodes), and assigns all
// connected nodes the same forest tag.
// Returns number of nodes explored
func exploreForest(startWord *WordNode) int {
	var retval = 0
	var subgraph = wordGraph[len(startWord.Word)]

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
			//fmt.Printf("----->  Looking at node '%v' (%v)\n", node.Word, node.ForestTag)

			// Tag the forest
			node.ForestTag = curForest

			// Figure out the neighbors
			var neighbors = loadNeighbors(node)
			node.Neighbors = make([]*string, len(neighbors))
			copy(node.Neighbors, neighbors)

			// Search Neighbors
			for _, neigh := range neighbors {
				q.push(subgraph[*neigh])
			}
		}
	}

	return retval
}

// Figure out the neighbors of a node by filtering the word list, rather than by generation of all possible words.
// Should be faster depending on length of word and size of dictionary.
func loadNeighbors(node *WordNode) []*string {
	var retval = []*string{}
	var subgraph = wordGraph[len(node.Word)]

	for _, v := range subgraph {
		var d = distance(node.Word, v.Word)

		if d == 1 {
			retval = append(retval, &v.Word)
		}
	}

	return retval
}

// How many changes are needed to go from one word to another?
// We only use this to find neighbors, could be optimized to break out after more than one difference.
func distance(s1 string, s2 string) int {
	if len(s1) != len(s2) {
		return 999999
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
