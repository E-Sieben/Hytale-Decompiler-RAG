package main

// DownloadMissingDependencies handles the download of the Hytale Server Downloader and VineFlower if not present
func DownloadMissingDependencies(status DependencyStatus) *DependencyStatus {
	if !status.HasJava {
		panic("No Java 25 install was found, please download at https://adoptium.net/temurin/releases")
	}

	// Download the Hytale Server Downloader from https://downloader.hytale.com/hytale-downloader.zip
	if !status.HasHytaleDownloader {
		// TODO: Download, unzip, Choose "hytale-downloader-linux-amd64" or "hytale-downloader-windows-amd64.exe",
		// extract chosen executable to "dependencies/" and lastly remove "hytale-downloader" folder and .zip
		status.HasHytaleDownloader = true
	}

	// Download VineFlower from https://github.com/Vineflower/vineflower/releases/latest
	if !status.HasVineFlower {
		// TODO: List all available downloads in latest and download "-slim" version, if not available download the normal version
		status.HasVineFlower = true
	}

	// TODO: Download HytaleServer.jar via Hytale Downloader and extract from Version Folder to "dependencies/"
	// (example: dependencies/2026.03.26-89796e57b/Server/HytaleServer. jar to dependencies/HytaleServer.jar")

	return &status
}
