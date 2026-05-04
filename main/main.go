package main

import (
	"bufio"
	"encoding/json"
	"flag"
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

func saveQdrantCreds(endpoint, key string) error {
	creds := credentials{Qdrant: qdrantCreds{Endpoint: endpoint, Key: key}}
	data, err := json.MarshalIndent(creds, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(exeDir(), credentialsFile), data, 0600)
}

func promptString(prompt string) string {
	reader := bufio.NewReader(os.Stdin)
	fmt.Print(prompt)
	line, _ := reader.ReadString('\n')
	return strings.TrimSpace(line)
}

func main() {
	// Flags for non-interactive / scripted use.
	// Paired --foo / --no-foo flags; if neither is set the program falls back to
	// the interactive prompt for that option.
	flagRAG := flag.Bool("rag", false, "Set up the RAG server (requires Docker)")
	flagNoRAG := flag.Bool("no-rag", false, "Skip RAG server setup")
	flagPrerelease := flag.Bool("prerelease", false, "Use the pre-release channel")
	flagNoPrerelease := flag.Bool("no-prerelease", false, "Use the stable release channel")
	flagQdrant := flag.Bool("qdrant", false, "Use Qdrant cloud instead of local ChromaDB")
	flagNoQdrant := flag.Bool("no-qdrant", false, "Use local ChromaDB (skip Qdrant)")
	flagQdrantEndpoint := flag.String("qdrant-endpoint", "", "Qdrant cloud endpoint URL (implies --qdrant)")
	flagQdrantKey := flag.String("qdrant-key", "", "Qdrant API key (implies --qdrant)")
	flagRedownload := flag.Bool("redownload-jar", false, "Re-download HytaleServer.jar even if it already exists")
	flagNoRedownload := flag.Bool("no-redownload-jar", false, "Skip re-downloading HytaleServer.jar if it already exists")
	flag.Parse()

	// Track which flags were explicitly supplied so we know when to fall back to prompts.
	set := map[string]bool{}
	flag.Visit(func(f *flag.Flag) { set[f.Name] = true })

	// Providing both endpoint + key implies --qdrant.
	if set["qdrant-endpoint"] || set["qdrant-key"] {
		*flagQdrant = true
		set["qdrant"] = true
	}

	// Validate mutually exclusive pairs.
	if *flagRAG && *flagNoRAG {
		fmt.Fprintln(os.Stderr, "Error: --rag and --no-rag are mutually exclusive")
		os.Exit(1)
	}
	if *flagPrerelease && *flagNoPrerelease {
		fmt.Fprintln(os.Stderr, "Error: --prerelease and --no-prerelease are mutually exclusive")
		os.Exit(1)
	}
	if *flagQdrant && *flagNoQdrant {
		fmt.Fprintln(os.Stderr, "Error: --qdrant and --no-qdrant are mutually exclusive")
		os.Exit(1)
	}
	if *flagRedownload && *flagNoRedownload {
		fmt.Fprintln(os.Stderr, "Error: --redownload-jar and --no-redownload-jar are mutually exclusive")
		os.Exit(1)
	}

	// ── RAG ────────────────────────────────────────────────────────────────────
	var wantsRAG bool
	switch {
	case set["rag"]:
		wantsRAG = true
	case set["no-rag"]:
		wantsRAG = false
	default:
		wantsRAG = askYesNo("Do you want to setup the RAG server? (requires Docker)")
	}

	// ── Pre-release ────────────────────────────────────────────────────────────
	var wantsPrerelease bool
	switch {
	case set["prerelease"]:
		wantsPrerelease = true
	case set["no-prerelease"]:
		wantsPrerelease = false
	default:
		wantsPrerelease = askYesNo("Do you want to use the pre-release channel?")
	}

	// ── Qdrant ────────────────────────────────────────────────────────────────
	var qdrantURL, qdrantKey string
	if wantsRAG {
		switch {
		case *flagNoQdrant:
			// Explicitly opted out — stay with local ChromaDB, no prompts.

		case set["qdrant-endpoint"] && set["qdrant-key"]:
			// Full credentials supplied via flags — use them and persist if new.
			qdrantURL, qdrantKey = *flagQdrantEndpoint, *flagQdrantKey
			if existingURL, _ := loadQdrantCreds(); existingURL == "" {
				if err := saveQdrantCreds(qdrantURL, qdrantKey); err != nil {
					fmt.Fprintf(os.Stderr, "Warning: could not save %s: %v\n", credentialsFile, err)
				} else {
					fmt.Printf("Qdrant credentials saved to %s\n", filepath.Join(exeDir(), credentialsFile))
				}
			}

		case set["qdrant"]:
			// --qdrant flag set but no inline credentials — load from file or prompt.
			url, key := loadQdrantCreds()
			if url != "" && key != "" {
				qdrantURL, qdrantKey = url, key
			} else {
				fmt.Println("No Qdrant credentials file found. Please enter your Qdrant cloud details:")
				qdrantURL = promptString("  Endpoint URL: ")
				qdrantKey = promptString("  API key:      ")
				if err := saveQdrantCreds(qdrantURL, qdrantKey); err != nil {
					fmt.Fprintf(os.Stderr, "Warning: could not save %s: %v\n", credentialsFile, err)
				} else {
					fmt.Printf("Credentials saved to %s\n", filepath.Join(exeDir(), credentialsFile))
				}
			}

		default:
			// Fully interactive: ask whether to use Qdrant, then handle credentials.
			if askYesNo("Do you want to use Qdrant cloud instead of local ChromaDB?") {
				url, key := loadQdrantCreds()
				if url != "" && key != "" {
					fmt.Println("Qdrant credentials loaded from credentials.json.")
					qdrantURL, qdrantKey = url, key
				} else {
					fmt.Println("No credentials file found. Please enter your Qdrant cloud details:")
					qdrantURL = promptString("  Endpoint URL: ")
					qdrantKey = promptString("  API key:      ")
					if err := saveQdrantCreds(qdrantURL, qdrantKey); err != nil {
						fmt.Fprintf(os.Stderr, "Warning: could not save %s: %v\n", credentialsFile, err)
					} else {
						fmt.Printf("Credentials saved to %s\n", filepath.Join(exeDir(), credentialsFile))
					}
				}
			}
		}
	}

	// Build tri-state for JAR redownload: nil = ask, true/false = forced.
	// We must not pass the flag pointer directly — both flags are true when present.
	var redownloadJar *bool
	switch {
	case set["redownload-jar"]:
		v := true
		redownloadJar = &v
	case set["no-redownload-jar"]:
		v := false
		redownloadJar = &v
	}

	status := NewDependencyStatus()
	DownloadMissingDependencies(status, wantsPrerelease, redownloadJar)
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
