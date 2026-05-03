package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	DependenciesDir = "dependencies"
	CodeDir         = "code"
	credentialsFile = "credentials.json"
)

type qdrantCreds struct {
	Endpoint string `json:"endpoint"`
	Key      string `json:"key"`
}

type credentials struct {
	Qdrant qdrantCreds `json:"qdrant"`
}

// exeDir returns the directory containing the running binary, following symlinks.
func exeDir() string {
	exe, err := os.Executable()
	if err != nil {
		return "."
	}
	if resolved, err := filepath.EvalSymlinks(exe); err == nil {
		exe = resolved
	}
	return filepath.Dir(exe)
}

func loadQdrantCreds() (string, string) {
	// Look next to the binary so the path is stable regardless of launch directory.
	path := filepath.Join(exeDir(), credentialsFile)
	data, err := os.ReadFile(path)
	if err != nil {
		return "", ""
	}
	var creds credentials
	if err := json.Unmarshal(data, &creds); err != nil {
		return "", ""
	}
	return creds.Qdrant.Endpoint, creds.Qdrant.Key
}

func main() {
	wantsRAG := askYesNo("Do you want to setup the RAG server? (requires Docker)")
	wantsPrerelease := askYesNo("Do you want to use the pre-release channel?")

	var qdrantURL, qdrantKey string
	if wantsRAG {
		url, key := loadQdrantCreds()
		if url != "" && key != "" {
			if askYesNo("Qdrant credentials detected — use Qdrant cloud instead of local ChromaDB?") {
				qdrantURL, qdrantKey = url, key
			}
		} else {
			fmt.Printf("No Qdrant credentials found at %s — deploying with local ChromaDB.\n",
				filepath.Join(exeDir(), credentialsFile))
		}
	}

	status := NewDependencyStatus()
	DownloadMissingDependencies(status, wantsPrerelease)
	decompile()
	pruneToHypixel()
	strip()
	addJavadoc(wantsPrerelease)
	zipSourceCode()
	if status.HasAllDependencies() && wantsRAG {
		deployRAG(qdrantURL, qdrantKey)
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
