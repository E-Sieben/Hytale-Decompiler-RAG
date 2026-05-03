package main

import (
	"archive/zip"
	"bufio"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"
)

// decompile the HytaleServer.jar using VineFlower, filtering to com/hypixel/ only
func decompile() {
	jars, err := filepath.Glob(filepath.Join(DependenciesDir, "vineflower-*.jar"))
	if err != nil || len(jars) == 0 {
		panic("VineFlower jar not found in " + DependenciesDir)
	}

	if err := os.MkdirAll(CodeDir, 0755); err != nil {
		panic(fmt.Sprintf("Failed to create code directory: %v", err))
	}

	filteredJar := filepath.Join(DependenciesDir, "HytaleServer-hypixel.jar")
	fmt.Println("Filtering jar to com/hypixel/ classes...")
	if err := createFilteredJar(
		filepath.Join(DependenciesDir, "HytaleServer.jar"),
		filteredJar,
		"com/hypixel/",
	); err != nil {
		panic(fmt.Sprintf("Failed to filter jar: %v", err))
	}
	defer os.Remove(filteredJar)

	logFile, err := os.Create("decompile.log")
	if err != nil {
		panic(fmt.Sprintf("Failed to create decompile.log: %v", err))
	}
	defer logFile.Close()

	cmd := exec.Command("java",
		"-Xmx6g",
		"-XX:+UseG1GC",
		"-jar", jars[0],
		filteredJar,
		CodeDir+string(os.PathSeparator),
	)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		panic(fmt.Sprintf("Failed to create stdout pipe: %v", err))
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		panic(fmt.Sprintf("Failed to create stderr pipe: %v", err))
	}

	if err := cmd.Start(); err != nil {
		panic(fmt.Sprintf("Failed to start VineFlower: %v", err))
	}

	var mu sync.Mutex
	relay := func(r io.Reader) {
		scanner := bufio.NewScanner(r)
		for scanner.Scan() {
			line := scanner.Text()
			mu.Lock()
			fmt.Fprintln(logFile, line)
			mu.Unlock()
			if isProgressLine(line) {
				fmt.Println(line)
			}
		}
	}

	var wg sync.WaitGroup
	wg.Add(2)
	go func() { defer wg.Done(); relay(stdout) }()
	go func() { defer wg.Done(); relay(stderr) }()
	wg.Wait()

	if err := cmd.Wait(); err != nil {
		panic("Unable to decompile HytaleServer.jar — check decompile.log for more information")
	}
}

func isProgressLine(line string) bool {
	lower := strings.ToLower(line)
	return strings.Contains(lower, "loading") ||
		strings.Contains(lower, "symboliz") ||
		strings.Contains(lower, "decompil")
}

// pruneToHypixel deletes everything in CodeDir except com/hypixel/hytale/,
// preserving the full package hierarchy that IDEs expect.
func pruneToHypixel() {
	entries, err := os.ReadDir(CodeDir)
	if err != nil {
		panic(fmt.Sprintf("Failed to read code directory: %v", err))
	}
	for _, e := range entries {
		if e.Name() == "com" {
			continue
		}
		if err := os.RemoveAll(filepath.Join(CodeDir, e.Name())); err != nil {
			panic(fmt.Sprintf("Failed to remove %s: %v", e.Name(), err))
		}
	}
}

var (
	reFFLine       = regexp.MustCompile(`(?m)^\s*//\s*\$(?:FF|VF):.*\r?\n`)
	reFFBlock      = regexp.MustCompile(`(?s)/\*\s*\$(?:FF|VF):.*?\*/[ \t]*\r?\n?`)
	reCompiledFrom = regexp.MustCompile(`(?m)^\s*/\*\s*compiled from:.*?\*/[ \t]*\r?\n`)
	reRenamedFrom  = regexp.MustCompile(`(?m)/\*\s*renamed from:.*?\*/\s*`)
)

// strip removes common decompilation artifacts and warnings from .java files
func strip() {
	err := filepath.Walk(CodeDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() || !strings.HasSuffix(path, ".java") {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		content := string(data)
		content = reFFLine.ReplaceAllString(content, "")
		content = reFFBlock.ReplaceAllString(content, "")
		content = reCompiledFrom.ReplaceAllString(content, "")
		content = reRenamedFrom.ReplaceAllString(content, "")
		return os.WriteFile(path, []byte(content), info.Mode())
	})
	if err != nil {
		panic(fmt.Sprintf("Failed to strip decompiler artifacts: %v", err))
	}
}

// addJavadoc fetches Javadoc from the Hytale docs server and inserts it into decompiled source files.
// It scrapes overview-tree.html to build a precise set of documented classes, then processes
// only those files in parallel.
func addJavadoc(wantsPrerelease bool) {
	baseURL := "https://release.server.docs.hytale.com"
	if wantsPrerelease {
		baseURL = "https://prerelease.server.docs.hytale.com"
	}

	overviewClient := &http.Client{Timeout: 30 * time.Second}
	pageClient := &http.Client{Timeout: 10 * time.Second}

	fmt.Println("Fetching documented class list from allclasses-index...")
	documented := fetchDocumentedClasses(baseURL, overviewClient)
	if len(documented) == 0 {
		fmt.Println("Warning: could not fetch class list from overview-tree; skipping Javadoc.")
		return
	}
	fmt.Printf("Found %d documented classes.\n", len(documented))

	const maxConcurrent = 8
	sem := make(chan struct{}, maxConcurrent)
	var wg sync.WaitGroup

	_ = filepath.Walk(CodeDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() || !strings.HasSuffix(path, ".java") {
			return nil
		}
		rel, _ := filepath.Rel(CodeDir, path)
		classPath := filepath.ToSlash(strings.TrimSuffix(rel, ".java"))
		if !documented[classPath] {
			return nil
		}
		wg.Add(1)
		sem <- struct{}{}
		go func() {
			defer wg.Done()
			defer func() { <-sem }()
			if applyErr := applyJavadoc(path, baseURL, pageClient); applyErr != nil {
				fmt.Printf("Warning: skipping Javadoc for %s: %v\n", path, applyErr)
			}
		}()
		return nil
	})
	wg.Wait()
}

// fetchDocumentedClasses scrapes allclasses-index.html and returns the set of documented class paths
// (e.g. "com/hypixel/hytale/server/SomeClass") so undocumented files can be skipped entirely.
func fetchDocumentedClasses(baseURL string, client *http.Client) map[string]bool {
	classes := map[string]bool{}
	resp, err := client.Get(baseURL + "/allclasses-index.html")
	if err != nil || resp.StatusCode != http.StatusOK {
		if resp != nil {
			resp.Body.Close()
		}
		return classes
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return classes
	}

	reHref := regexp.MustCompile(`href="([^"#]*\.html)"`)
	for _, m := range reHref.FindAllSubmatch(body, -1) {
		// Normalize: strip any leading ./ or / so we always store "com/hypixel/hytale/..."
		raw := strings.TrimLeft(string(m[1]), "./")
		if strings.HasPrefix(raw, "com/hypixel/hytale/") {
			classes[strings.TrimSuffix(raw, ".html")] = true
		}
	}
	return classes
}

var reClassDecl = regexp.MustCompile(`(?m)^(public\s+(?:(?:abstract|final|sealed)\s+)*(?:class|interface|enum|record)\s+\w+)`)

func applyJavadoc(javaFilePath, baseURL string, client *http.Client) error {
	rel, err := filepath.Rel(CodeDir, javaFilePath)
	if err != nil {
		return err
	}
	urlPath := filepath.ToSlash(strings.TrimSuffix(rel, ".java"))
	docURL := baseURL + "/" + urlPath + ".html"

	var resp *http.Response
	for attempt := range 2 {
		var reqErr error
		resp, reqErr = client.Get(docURL)
		if reqErr == nil {
			break
		}
		if attempt == 1 {
			return reqErr
		}
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	description := extractClassDescription(string(body))
	if description == "" {
		return nil
	}

	data, err := os.ReadFile(javaFilePath)
	if err != nil {
		return err
	}
	source := string(data)
	loc := reClassDecl.FindStringIndex(source)
	if loc == nil {
		return nil
	}

	javadoc := "/**\n * " + strings.ReplaceAll(description, "\n", "\n * ") + "\n */\n"
	return os.WriteFile(javaFilePath, []byte(source[:loc[0]]+javadoc+source[loc[0]:]), 0644)
}

// extractClassDescription pulls the first block description out of a Javadoc HTML page
var (
	reHTMLTag   = regexp.MustCompile(`<[^>]+>`)
	reBlockDesc = regexp.MustCompile(`(?s)<div[^>]+class="[^"]*block[^"]*"[^>]*>(.*?)</div>`)
	reSection   = regexp.MustCompile(`(?s)<section[^>]+class="[^"]*class-description[^"]*"[^>]*>(.*?)</section>`)
	reEntities  = strings.NewReplacer(
		"&amp;", "&", "&lt;", "<", "&gt;", ">", "&quot;", `"`, "&#39;", "'", "&nbsp;", " ",
	)
)

func extractClassDescription(html string) string {
	var sectionHTML string
	if m := reSection.FindStringSubmatch(html); len(m) > 1 {
		sectionHTML = m[1]
	} else {
		sectionHTML = html
	}

	m := reBlockDesc.FindStringSubmatch(sectionHTML)
	if len(m) < 2 {
		return ""
	}
	text := reHTMLTag.ReplaceAllString(m[1], "")
	text = reEntities.Replace(text)
	text = strings.Join(strings.Fields(text), " ")
	return strings.TrimSpace(text)
}

func zipSourceCode() {
	const dest = "HytaleServer-source.zip"
	fmt.Println("Zipping source code...")

	out, err := os.Create(dest)
	if err != nil {
		panic(fmt.Sprintf("Failed to create %s: %v", dest, err))
	}
	defer out.Close()

	w := zip.NewWriter(out)
	defer w.Close()

	err = filepath.Walk(CodeDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return err
		}
		rel, err := filepath.Rel(CodeDir, path)
		if err != nil {
			return err
		}
		fw, err := w.Create(filepath.ToSlash(rel))
		if err != nil {
			return err
		}
		f, err := os.Open(path)
		if err != nil {
			return err
		}
		defer f.Close()
		_, err = io.Copy(fw, f)
		return err
	})
	if err != nil {
		panic(fmt.Sprintf("Failed to zip source code: %v", err))
	}
	fmt.Printf("Source code written to %s\n", dest)
}

// createFilteredJar copies only entries whose name starts with prefix into a new jar.
func createFilteredJar(srcPath, dstPath, prefix string) error {
	r, err := zip.OpenReader(srcPath)
	if err != nil {
		return err
	}
	defer r.Close()

	out, err := os.Create(dstPath)
	if err != nil {
		return err
	}
	defer out.Close()

	w := zip.NewWriter(out)
	defer w.Close()

	for _, f := range r.File {
		if !strings.HasPrefix(f.Name, prefix) {
			continue
		}
		fw, err := w.CreateHeader(&f.FileHeader)
		if err != nil {
			return err
		}
		rc, err := f.Open()
		if err != nil {
			return err
		}
		_, copyErr := io.Copy(fw, rc)
		rc.Close()
		if copyErr != nil {
			return copyErr
		}
	}
	return nil
}
