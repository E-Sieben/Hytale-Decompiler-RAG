package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

const (
	DependenciesDir = "dependencies"
	CodeDir         = "code"
)

func main() {
	wantsRAG := askYesNo("Do you want to setup the RAG server? (requires Docker)")
	wantsPrerelease := askYesNo("Do you want to use the pre-release channel?")

	status := NewDependencyStatus()
	DownloadMissingDependencies(status, wantsPrerelease)
	decompile()
	pruneToHypixel()
	strip()
	addJavadoc(wantsPrerelease)
	zipSourceCode()
	if status.HasAllDependencies() && wantsRAG {
		deployRAG()
	}
}

func askYesNo(prompt string) bool {
	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Printf("%s [y/n]: ", prompt)
		line, _ := reader.ReadString('\n')
		switch strings.TrimSpace(strings.ToLower(line)) {
		case "y", "yes":
			return true
		case "n", "no":
			return false
		}
		fmt.Println("Please enter 'y' or 'n'.")
	}
}
