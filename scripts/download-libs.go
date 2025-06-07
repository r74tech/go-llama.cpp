//go:build ignore
// +build ignore

package main

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

const (
	repoOwner = "r74tech" // Change this to your GitHub username/org
	repoName  = "go-llama.cpp"
)

func main() {
	// Check if libbinding.a already exists
	if _, err := os.Stat("libbinding.a"); err == nil {
		fmt.Println("libbinding.a already exists, skipping download")
		return
	}

	// Determine platform and architecture
	platform := runtime.GOOS
	arch := runtime.GOARCH

	// Map Go arch names to our release naming
	if arch == "amd64" {
		arch = "amd64"
	} else if arch == "arm64" {
		arch = "arm64"
	}

	// Get the latest release
	releaseTag := os.Getenv("LLAMA_CPP_RELEASE_TAG")
	if releaseTag == "" {
		releaseTag = getLatestRelease()
	}

	// Construct download URL
	fileName := fmt.Sprintf("libbinding-%s-%s.tar.gz", platform, arch)
	downloadURL := fmt.Sprintf("https://github.com/%s/%s/releases/download/%s/%s",
		repoOwner, repoName, releaseTag, fileName)

	fmt.Printf("Downloading pre-built library from %s\n", downloadURL)

	// Download the file
	if err := downloadFile(fileName, downloadURL); err != nil {
		fmt.Printf("Failed to download: %v\n", err)
		os.Exit(1)
	}

	// Extract the tarball
	if err := extractTarGz(fileName, "."); err != nil {
		fmt.Printf("Failed to extract: %v\n", err)
		os.Exit(1)
	}

	// Clean up
	os.Remove(fileName)

	fmt.Println("Successfully downloaded and extracted pre-built library")
}

func getLatestRelease() string {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/latest", repoOwner, repoName)

	resp, err := http.Get(url)
	if err != nil {
		fmt.Printf("Failed to get latest release: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	// Parse JSON response to get tag_name
	// For simplicity, we'll just look for the tag_name field
	body, _ := io.ReadAll(resp.Body)
	bodyStr := string(body)

	tagStart := strings.Index(bodyStr, `"tag_name":"`)
	if tagStart == -1 {
		fmt.Println("Failed to find tag_name in response")
		os.Exit(1)
	}

	tagStart += len(`"tag_name":"`)
	tagEnd := strings.Index(bodyStr[tagStart:], `"`)
	if tagEnd == -1 {
		fmt.Println("Failed to parse tag_name")
		os.Exit(1)
	}

	return bodyStr[tagStart : tagStart+tagEnd]
}

func downloadFile(filepath string, url string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("bad status: %s", resp.Status)
	}

	out, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	return err
}

func extractTarGz(gzipPath, dest string) error {
	file, err := os.Open(gzipPath)
	if err != nil {
		return err
	}
	defer file.Close()

	gzr, err := gzip.NewReader(file)
	if err != nil {
		return err
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)

	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		target := filepath.Join(dest, header.Name)

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0755); err != nil {
				return err
			}
		case tar.TypeReg:
			// Ensure the directory exists
			if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
				return err
			}

			file, err := os.OpenFile(target, os.O_CREATE|os.O_RDWR, os.FileMode(header.Mode))
			if err != nil {
				return err
			}

			if _, err := io.Copy(file, tr); err != nil {
				file.Close()
				return err
			}

			file.Close()
		}
	}

	return nil
}
