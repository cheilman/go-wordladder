package main

import (
	"bufio"
	"fmt"
	"os"
)

type Node struct {
	word      string   // the word itself
	forestTag int      // what forest the word lives in
	neighbors []string // list of one-character neighbors
}

const wordFile = "/usr/share/dict/words"

var wordGraph = make(map[string]Node)
var curForest = 1 // valid forest tags are positive integers

func main() {
	//
	// Load all words into graph
	//
	fmt.Printf("Loading words from %v.\n", wordFile)
	var f, err = os.Open(wordFile)
	check(err)
	var scanner = bufio.NewScanner(f)
	for scanner.Scan() {
		var word = scanner.Text()
		wordGraph[word] = Node{word, 0, nil}
	}

	//
	// Start assigning forests and neighbors
	//
	fmt.Printf("Starting to assign forests and analyze neighbors.  %v words in graph.\n", len(wordGraph))

	//
	// Run some tests
	//
	fmt.Printf("Let's try some ladders.\n")
}

func check(e error) {
	if e != nil {
		panic(e)
	}
}
