package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
)

// DependencyStatus Checks for Dependencies
type DependencyStatus struct {
	HasDocker           bool
	HasJava             bool
	HasHytaleDownloader bool
	HasHytaleJar        bool
	HasVineFlower       bool
}

// NewDependencyStatus returns a Struct of available Dependencies
func NewDependencyStatus() *DependencyStatus {
	return &DependencyStatus{
		HasDocker:           checkDocker(),
		HasJava:             checkJava(),
		HasHytaleDownloader: checkHytaleDownloader(),
		HasHytaleJar:        checkHytaleJar(),
		HasVineFlower:       checkVineFlower(),
	}
}

// HasAllDependencies Checks if the User has all dependencies available
func (status DependencyStatus) HasAllDependencies() bool {
	return status.HasDocker &&
		status.HasJava &&
		status.HasHytaleDownloader &&
		status.HasHytaleJar &&
		status.HasVineFlower
}

func checkDocker() bool {
	return exec.Command("docker", "--version").Run() == nil
}

func checkJava() bool {
	out, err := exec.Command("java", "-version").CombinedOutput()
	if err != nil {
		return false
	}
	for _, line := range strings.Split(string(out), "\n") {
		if !strings.Contains(line, "version") {
			continue
		}
		for _, field := range strings.Fields(line) {
			field = strings.Trim(field, "\"")
			major := strings.SplitN(field, ".", 2)[0]
			if v, err := strconv.Atoi(major); err == nil && v >= 25 {
				return true
			}
		}
	}
	return false
}

func checkHytaleDownloader() bool {
	_, err := os.Stat(filepath.Join(DependenciesDir, downloaderBinaryName()))
	return err == nil
}

func checkHytaleJar() bool {
	_, err := os.Stat(filepath.Join(DependenciesDir, "HytaleServer.jar"))
	return err == nil
}

func checkVineFlower() bool {
	matches, err := filepath.Glob(filepath.Join(DependenciesDir, "vineflower-*.jar"))
	return err == nil && len(matches) > 0
}

func downloaderBinaryName() string {
	if runtime.GOOS == "windows" {
		return "hytale-downloader-windows-amd64.exe"
	}
	return "hytale-downloader-linux-amd64"
}
