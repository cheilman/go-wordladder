package main

import (
	"bufio"
	"fmt"
	"os"
	"unicode"
)

type WordNode struct {
	word      string    // the word itself
	forestTag int       // what forest the word lives in
	neighbors []*string // list of one-character neighbors
}

const wordFile = "/usr/share/dict/words"

var wordGraph = make(map[string]*WordNode)
var curForest = 1 // valid forest tags are positive integers

func main() {
	//
	// Load all words into graph
	//
	fmt.Printf("Loading words from %v.\n", wordFile)

	var f, err = os.Open(wordFile)
	if err != nil {
		panic(err)
	}

	var scanner = bufio.NewScanner(f)
	for scanner.Scan() {
		var word = scanner.Text()
		if isValidWord(&word) {
			// Add to the graph
			wordGraph[word] = &WordNode{word, 0, nil}
		}
	}

	//
	// Start assigning forests and neighbors
	//
	fmt.Printf("Starting to assign forests and analyze neighbors.  %v words in graph.\n", len(wordGraph))
	for _, v := range wordGraph {
		if v.forestTag <= 0 {
			// It's unassigned so far, need to figure out where it belongs.
			//fmt.Printf("Starting with word '%v'\n", v.word)
			//var explored = exploreForest(&v)
			exploreForest(v)
			//fmt.Printf("-->  Forest %v: %v nodes\n", curForest, explored)

			// Move along to the next forest
			curForest++
		} else {
			// Assigned, ignore it
		}
	}

	//
	// Run some tests
	//
	fmt.Printf("Found %v forest(s).\n", curForest-1)

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

func areTwoWordsConnected(s1 string, s2 string) bool {
	if wordGraph[s1] == nil || wordGraph[s2] == nil {
		return false
	}

	return wordGraph[s1].forestTag == wordGraph[s2].forestTag
}

func shortestPath(s1 string, s2 string) []string {
	//fmt.Printf("--------->  checking %v -> %v\n", s1, s2)

	if !areTwoWordsConnected(s1, s2) {
		//fmt.Printf("--------->  no path.\n")
		return nil
	}

	// We actually search backwards (s2 -> s1), so we don't have to reverse the string at the end

	var visited = make(map[string]bool)
	var target *WNPathQueueNode = nil

	var q = WNPathQueue{}
	q.push(&WNPathQueueNode{wn: wordGraph[s2], parent: nil})

	for {
		var node = q.pop()

		if node == nil {
			return nil
		} else {
			//fmt.Printf("--------->  Looking at '%v'\n", node.wn.word)

			if node.wn.word == s1 {
				target = node
				//fmt.Printf("--------->  Found target!  Breaking out.  Node: %+v\n", target)
				break
			}

			// check neighbors that haven't been visited
			for _, neighborWord := range node.wn.neighbors {

				if !visited[*neighborWord] {
					visited[*neighborWord] = true

					var neighborNode = wordGraph[*neighborWord]

					// Add nodes with the parent set
					q.push(&WNPathQueueNode{wn: neighborNode, parent: node})
				}
			}
		}
	}

	if target == nil {
		return nil
	}

	// Build the path back up
	var retval = []string{}

	var cur = target
	for {
		if cur == nil {
			break
		}

		retval = append(retval, cur.wn.word)
		cur = cur.parent
	}

	return retval
}

func isValidWord(s *string) bool {
	// Limit ourselves on length
	var l = len(*s)
	if l < 3 || l > 5 {
		return false
	}

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

// Returns number of nodes explored
func exploreForest(startWord *WordNode) int {
	var retval = 0

	var q = WNQueue{}
	q.push(startWord)

	for {
		var node = q.pop()

		if node == nil {
			break
		} else {
			// Filter out visited people
			if node.forestTag > 0 {
				continue
			}

			retval++
			//fmt.Printf("----->  Looking at node '%v' (%v)\n", node.word, node.forestTag)

			// Tag the forest
			node.forestTag = curForest

			// Figure out the neighbors
			var neighbors = loadNeighbors(node)
			node.neighbors = make([]*string, len(neighbors))
			copy(node.neighbors, neighbors)

			// Search neighbors
			for _, neigh := range neighbors {
				q.push(wordGraph[*neigh])
			}
		}
	}

	return retval
}
func loadNeighbors(node *WordNode) []*string {
	var retval = []*string{}

	for _, v := range wordGraph {
		var d = distance(node.word, v.word)

		if d == 1 {
			retval = append(retval, &v.word)
		}
	}

	return retval
}

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
