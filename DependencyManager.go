package main

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
		// TODO: Check for all Dependencies, except HytaleJar (VineFlower and Hytale Downloader are to be checked in "dependencies/")
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
