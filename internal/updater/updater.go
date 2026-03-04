package updater

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"runtime"
	"strings"

	"telemetry/internal/config"
)

const (
	repo     = "ftech-vn/telemetry"
	apiURL   = "https://api.github.com/repos/" + repo + "/releases/latest"
	fileName = "telemetry"
)

func CheckForUpdates(currentVersion string, cfg *config.Config) {
	if !cfg.AutoUpdate {
		return
	}

	log.Println("🔄 Checking for updates...")

	resp, err := http.Get(apiURL)
	if err != nil {
		log.Printf("❌ Failed to check for updates: %v", err)
		return
	}
	defer resp.Body.Close()

	var release struct {
		TagName string `json:"tag_name"`
		Assets  []struct {
			Name        string `json:"name"`
			DownloadURL string `json:"browser_download_url"`
		} `json:"assets"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		log.Printf("❌ Failed to parse release information: %v", err)
		return
	}

	latestVersion := strings.TrimPrefix(release.TagName, "v")
	if latestVersion == currentVersion {
		log.Println("✅ You are running the latest version.")
		return
	}

	log.Printf("🚀 New version available: %s (current: %s)", latestVersion, currentVersion)

	assetName := fmt.Sprintf("%s-%s-%s", fileName, runtime.GOOS, runtime.GOARCH)
	var downloadURL string
	for _, asset := range release.Assets {
		if asset.Name == assetName {
			downloadURL = asset.DownloadURL
			break
		}
	}

	if downloadURL == "" {
		log.Printf("❌ Could not find an update asset for your system: %s", assetName)
		return
	}

	if err := downloadAndReplace(downloadURL); err != nil {
		log.Printf("❌ Failed to update binary: %v", err)
		return
	}

	log.Println("✅ Update successful! Please restart the service to apply the changes.")
}

func downloadAndReplace(url string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	tmpFile, err := os.CreateTemp("", fileName)
	if err != nil {
		return err
	}
	defer tmpFile.Close()

	if _, err := io.Copy(tmpFile, resp.Body); err != nil {
		return err
	}

	if err := os.Chmod(tmpFile.Name(), 0755); err != nil {
		return err
	}

	currentExecutable, err := os.Executable()
	if err != nil {
		return err
	}

	if err := os.Rename(tmpFile.Name(), currentExecutable); err != nil {
		return err
	}

	return nil
}
