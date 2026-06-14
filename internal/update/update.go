// Package update provides self-update functionality via GitHub Releases.
package update

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

const (
	repoOwner = "archaeondlg"
	repoName  = "aiusage"
	binaryName = "aiusage"
)

// SelfUpdate downloads the latest release and replaces the current executable.
func SelfUpdate() (string, error) {
	release, err := fetchLatestRelease()
	if err != nil {
		return "", fmt.Errorf("fetch release: %w", err)
	}

	asset, err := findAsset(release)
	if err != nil {
		return "", fmt.Errorf("find asset: %w", err)
	}

	fmt.Fprintf(os.Stderr, "→ Downloading %s (%s)...\n", release.Tag, formatBytes(asset.Size))
	tmp, err := downloadAsset(asset)
	if err != nil {
		return "", fmt.Errorf("download: %w", err)
	}
	defer os.Remove(tmp)

	exe, err := extractBinary(tmp)
	if err != nil {
		return "", fmt.Errorf("extract: %w", err)
	}

	// Replace current executable.
	current, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("find current exe: %w", err)
	}

	// On Windows: rename old, write new.
	bak := current + ".old"
	os.Remove(bak)
	if err := os.Rename(current, bak); err != nil {
		return "", fmt.Errorf("backup old: %w", err)
	}
	if err := os.Rename(exe, current); err != nil {
		// Try to restore backup.
		os.Rename(bak, current)
		return "", fmt.Errorf("install new: %w", err)
	}
	os.Remove(bak)
	os.Chmod(current, 0755)

	return release.Tag, nil
}

type githubRelease struct {
	Tag    string        `json:"tag_name"`
	Assets []githubAsset `json:"assets"`
}

type githubAsset struct {
	Name             string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
	Size             int64  `json:"size"`
}

func fetchLatestRelease() (*githubRelease, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/latest", repoOwner, repoName)
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	var release githubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, err
	}
	if release.Tag == "" {
		return nil, fmt.Errorf("no release found")
	}
	return &release, nil
}

// findAsset finds the archive matching current OS/arch.
func findAsset(release *githubRelease) (*githubAsset, error) {
	goos := runtime.GOOS
	goarch := runtime.GOARCH

	// Map Go arch names to common variant names.
	archVariants := []string{goarch}
	if goarch == "amd64" {
		archVariants = append(archVariants, "x86_64")
	}

	for _, asset := range release.Assets {
		name := strings.ToLower(asset.Name)
		// Must contain the OS and arch, and must be an archive.
		if !strings.Contains(name, goos) {
			continue
		}
		archMatch := false
		for _, arch := range archVariants {
			if strings.Contains(name, arch) {
				archMatch = true
				break
			}
		}
		if !archMatch {
			continue
		}
		if strings.HasSuffix(name, ".zip") || strings.HasSuffix(name, ".tar.gz") {
			return &asset, nil
		}
	}
	return nil, fmt.Errorf("no asset for %s/%s", goos, goarch)
}

func downloadAsset(asset *githubAsset) (string, error) {
	resp, err := http.Get(asset.BrowserDownloadURL)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	tmp, err := os.CreateTemp("", "aiusage-update-*")
	if err != nil {
		return "", err
	}
	defer tmp.Close()

	if _, err := io.Copy(tmp, resp.Body); err != nil {
		os.Remove(tmp.Name())
		return "", err
	}
	return tmp.Name(), nil
}

// extractBinary extracts the binary from a zip or tar.gz archive.
func extractBinary(archive string) (string, error) {
	if strings.HasSuffix(archive, ".zip") || strings.HasSuffix(archive, ".tmp") {
		return extractZip(archive)
	}
	return extractTarGz(archive)
}

func extractZip(path string) (string, error) {
	r, err := zip.OpenReader(path)
	if err != nil {
		return "", err
	}
	defer r.Close()

	for _, f := range r.File {
		name := filepath.Base(f.Name)
		if name == binaryName || name == binaryName+".exe" {
			rc, err := f.Open()
			if err != nil {
				return "", err
			}
			defer rc.Close()

			tmp, err := os.CreateTemp("", "aiusage-exe-*")
			if err != nil {
				return "", err
			}
			defer tmp.Close()

			if _, err := io.Copy(tmp, rc); err != nil {
				os.Remove(tmp.Name())
				return "", err
			}
			return tmp.Name(), nil
		}
	}
	return "", fmt.Errorf("%s not found in zip", binaryName)
}

func extractTarGz(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	gz, err := gzip.NewReader(f)
	if err != nil {
		return "", err
	}
	defer gz.Close()

	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", err
		}
		name := filepath.Base(hdr.Name)
		if name == binaryName || name == binaryName+".exe" {
			tmp, err := os.CreateTemp("", "aiusage-exe-*")
			if err != nil {
				return "", err
			}
			defer tmp.Close()

			if _, err := io.Copy(tmp, tr); err != nil {
				os.Remove(tmp.Name())
				return "", err
			}
			return tmp.Name(), nil
		}
	}
	return "", fmt.Errorf("%s not found in tar.gz", binaryName)
}

func formatBytes(n int64) string {
	if n < 1024 {
		return fmt.Sprintf("%dB", n)
	}
	if n < 1024*1024 {
		return fmt.Sprintf("%.1fKB", float64(n)/1024)
	}
	return fmt.Sprintf("%.1fMB", float64(n)/(1024*1024))
}
