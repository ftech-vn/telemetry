package updater

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"

	"telemetry/internal/config"
)

const (
	repo             = "ftech-vn/telemetry"
	apiURL           = "https://api.github.com/repos/" + repo + "/releases/latest"
	fileName         = "telemetry"
	updateNoticeFile = "/tmp/telemetry_update_notice"
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
		// If we are up to date, remove the notice file if it exists.
		os.Remove(updateNoticeFile)
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

	// Create the notification file for MOTD scripts
	noticeMessage := fmt.Sprintf("Telemetry has been updated to v%s. Please restart the service to apply the changes.", latestVersion)
	if err := os.WriteFile(updateNoticeFile, []byte(noticeMessage), 0644); err != nil {
		log.Printf("❌ Failed to write update notification file: %v", err)
	}

	log.Println("✅ Update successful! Restarting service to apply the changes...")
	// Send SIGTERM to self — the service manager (systemd/launchd) will restart
	// the process, which will then load the newly downloaded binary.
	p, err := os.FindProcess(os.Getpid())
	if err == nil {
		p.Signal(syscall.SIGTERM)
	}
}

func downloadAndReplace(url string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	currentExecutable, err := os.Executable()
	if err != nil {
		return err
	}

	// Create temp file in the same directory as the binary so that os.Rename
	// is guaranteed to be an atomic same-filesystem move. Using os.TempDir()
	// (e.g. /tmp) would fail with "invalid cross-device link" when the binary
	// lives on a different filesystem such as /usr/local/bin.
	tmpFile, err := os.CreateTemp(filepath.Dir(currentExecutable), fileName)
	if err != nil {
		return err
	}
	defer func() {
		tmpFile.Close()
		// Clean up temp file on failure; on success Rename moves it away.
		os.Remove(tmpFile.Name())
	}()

	if _, err := io.Copy(tmpFile, resp.Body); err != nil {
		return err
	}

	if err := os.Chmod(tmpFile.Name(), 0755); err != nil {
		return err
	}

	if err := os.Rename(tmpFile.Name(), currentExecutable); err != nil {
		return err
	}

	return nil
}
