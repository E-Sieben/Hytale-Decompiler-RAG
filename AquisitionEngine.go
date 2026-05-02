package main

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// DownloadMissingDependencies handles the download of the Hytale Server Downloader and VineFlower if not present
func DownloadMissingDependencies(status *DependencyStatus, wantsPrerelease bool) *DependencyStatus {
	if !status.HasJava {
		panic("No Java 25 install was found, please download at https://adoptium.net/temurin/releases")
	}

	if err := os.MkdirAll(DependenciesDir, 0755); err != nil {
		panic(fmt.Sprintf("Failed to create dependencies directory: %v", err))
	}

	if !status.HasHytaleDownloader {
		fmt.Println("Downloading Hytale Server Downloader...")
		if err := downloadHytaleDownloader(); err != nil {
			panic(fmt.Sprintf("Failed to download Hytale Downloader: %v", err))
		}
		status.HasHytaleDownloader = true
		fmt.Println("Hytale Server Downloader ready.")
	}

	if !status.HasVineFlower {
		fmt.Println("Downloading VineFlower decompiler...")
		if err := downloadVineFlower(); err != nil {
			panic(fmt.Sprintf("Failed to download VineFlower: %v", err))
		}
		status.HasVineFlower = true
		fmt.Println("VineFlower ready.")
	}

	shouldDownloadJar := !status.HasHytaleJar
	if status.HasHytaleJar {
		shouldDownloadJar = askYesNo("HytaleServer.jar already exists. Re-download to update? (~1.4 GB)")
	}
	if shouldDownloadJar {
		fmt.Println("Downloading Hytale Server Jar...")
		if err := downloadHytaleJar(wantsPrerelease); err != nil {
			panic(fmt.Sprintf("Failed to download Hytale Server Jar: %v", err))
		}
		status.HasHytaleJar = true
		fmt.Println("HytaleServer.jar ready.")
	}

	return status
}

func downloadHytaleDownloader() error {
	const url = "https://downloader.hytale.com/hytale-downloader.zip"
	zipPath := filepath.Join(DependenciesDir, "hytale-downloader.zip")

	if err := downloadFile(url, zipPath); err != nil {
		return fmt.Errorf("download failed: %w", err)
	}
	defer os.Remove(zipPath)

	extractDir := filepath.Join(DependenciesDir, "hytale-downloader-extract")
	if err := unzip(zipPath, extractDir); err != nil {
		return fmt.Errorf("unzip failed: %w", err)
	}
	defer os.RemoveAll(extractDir)

	binaryName := downloaderBinaryName()
	var srcPath string
	_ = filepath.Walk(extractDir, func(path string, info os.FileInfo, err error) error {
		if err == nil && info.Name() == binaryName {
			srcPath = path
		}
		return nil
	})
	if srcPath == "" {
		return fmt.Errorf("binary %s not found in zip", binaryName)
	}

	destPath := filepath.Join(DependenciesDir, binaryName)
	if err := copyFile(srcPath, destPath); err != nil {
		return fmt.Errorf("failed to copy binary: %w", err)
	}
	if runtime.GOOS != "windows" {
		return os.Chmod(destPath, 0755)
	}
	return nil
}

func downloadVineFlower() error {
	type Asset struct {
		Name               string `json:"name"`
		BrowserDownloadURL string `json:"browser_download_url"`
	}
	type Release struct {
		Assets []Asset `json:"assets"`
	}

	resp, err := http.Get("https://api.github.com/repos/Vineflower/vineflower/releases/latest")
	if err != nil {
		return fmt.Errorf("failed to fetch release info: %w", err)
	}
	defer resp.Body.Close()

	var release Release
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return fmt.Errorf("failed to parse release info: %w", err)
	}

	// Prefer -slim variant, fall back to any jar that isn't a sources jar
	var downloadURL, fileName string
	for _, asset := range release.Assets {
		if strings.HasSuffix(asset.Name, ".jar") && strings.Contains(asset.Name, "slim") {
			downloadURL = asset.BrowserDownloadURL
			fileName = asset.Name
			break
		}
	}
	if downloadURL == "" {
		for _, asset := range release.Assets {
			if strings.HasSuffix(asset.Name, ".jar") && !strings.Contains(asset.Name, "sources") {
				downloadURL = asset.BrowserDownloadURL
				fileName = asset.Name
				break
			}
		}
	}
	if downloadURL == "" {
		return fmt.Errorf("no suitable VineFlower jar found in release")
	}

	return downloadFile(downloadURL, filepath.Join(DependenciesDir, fileName))
}

func downloadHytaleJar(wantsPrerelease bool) error {
	binPath := filepath.Join(DependenciesDir, downloaderBinaryName())
	var args []string
	if wantsPrerelease {
		args = append(args, "-patchline", "pre-release")
	}

	cmd := exec.Command(binPath, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("hytale downloader failed: %w", err)
	}

	// The downloader produces a zip named like "2026.03.26-89796e57b.zip"
	entries, err := os.ReadDir(".")
	if err != nil {
		return fmt.Errorf("failed to read directory: %w", err)
	}
	var versionZip string
	for _, e := range entries {
		if !e.IsDir() && isVersionZip(e.Name()) {
			versionZip = e.Name()
		}
	}
	if versionZip == "" {
		return fmt.Errorf("no version zip found after running hytale downloader")
	}
	defer os.Remove(versionZip)

	extractDir := strings.TrimSuffix(versionZip, ".zip")
	if err := unzip(versionZip, extractDir); err != nil {
		return fmt.Errorf("failed to unzip version archive: %w", err)
	}
	defer os.RemoveAll(extractDir)

	// Walk the extracted archive to find HytaleServer.jar
	var srcJar string
	_ = filepath.Walk(extractDir, func(path string, info os.FileInfo, err error) error {
		if err == nil && !info.IsDir() && info.Name() == "HytaleServer.jar" {
			srcJar = path
		}
		return nil
	})
	if srcJar == "" {
		return fmt.Errorf("HytaleServer.jar not found inside %s", versionZip)
	}

	destJar := filepath.Join(DependenciesDir, "HytaleServer.jar")
	return copyFile(srcJar, destJar)
}

// isVersionZip matches Hytale version archives like "2026.03.26-89796e57b.zip"
func isVersionZip(name string) bool {
	if !strings.HasSuffix(name, ".zip") {
		return false
	}
	parts := strings.SplitN(strings.TrimSuffix(name, ".zip"), ".", 3)
	return len(parts) == 3 && len(parts[0]) == 4 && parts[0] >= "2020"
}

func downloadFile(url, destPath string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d for %s", resp.StatusCode, url)
	}
	f, err := os.Create(destPath)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = io.Copy(f, resp.Body)
	return err
}

func unzip(zipPath, destDir string) error {
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return err
	}
	defer r.Close()

	for _, f := range r.File {
		fpath := filepath.Join(destDir, f.Name)
		if f.FileInfo().IsDir() {
			os.MkdirAll(fpath, f.Mode())
			continue
		}
		if err := os.MkdirAll(filepath.Dir(fpath), 0755); err != nil {
			return err
		}
		rc, err := f.Open()
		if err != nil {
			return err
		}
		out, err := os.Create(fpath)
		if err != nil {
			rc.Close()
			return err
		}
		_, copyErr := io.Copy(out, rc)
		out.Close()
		rc.Close()
		if copyErr != nil {
			return copyErr
		}
	}
	return nil
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return err
	}
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, in)
	return err
}
