//go:build !windows

package monitor

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"syscall"

	"github.com/shirou/gopsutil/v3/disk"
)

type DiskMonitor struct {
	threshold    float64
	excludedDirs []string
}

func NewDiskMonitor(threshold float64, excludedDirs []string) *DiskMonitor {
	return &DiskMonitor{
		threshold:    threshold,
		excludedDirs: excludedDirs,
	}
}

func (m *DiskMonitor) Check() []Alert {
	usage, err := disk.Usage("/")
	if err != nil {
		log.Printf("Error getting usage for /: %v", err)
		return nil
	}

	if usage.UsedPercent < m.threshold {
		return nil
	}

	// Calculate top-level directory sizes efficiently with a single walk
	dirSizes := make(map[string]int64)
	rootInfo, err := os.Lstat("/")
	if err == nil {
		if stat, ok := rootInfo.Sys().(*syscall.Stat_t); ok {
			rootDev := uint64(stat.Dev)
			
			// Get immediate children of / to identify top-level dirs
			entries, _ := os.ReadDir("/")
			topLevelDirs := make(map[string]bool)
			for _, entry := range entries {
				if entry.IsDir() {
					name := entry.Name()
					path := filepath.Join("/", name)
					
					// Check if this directory is in the excluded list
					excluded := false
					for _, ex := range m.excludedDirs {
						if path == ex || name == ex {
							excluded = true
							break
						}
					}
					
					if !excluded && name != "proc" && name != "sys" && name != "dev" && name != "run" && name != "snap" {
						topLevelDirs[name] = true
					}
				}
			}

			// Single walk to aggregate sizes
			filepath.Walk("/", func(path string, info os.FileInfo, err error) error {
				if err != nil {
					return nil
				}

				// Stay on the same device
				if stat, ok := info.Sys().(*syscall.Stat_t); ok {
					if uint64(stat.Dev) != rootDev {
						if info.IsDir() {
							return filepath.SkipDir
						}
						return nil
					}
				}

				if !info.IsDir() {
					// Identify which top-level dir this belongs to
					rel, err := filepath.Rel("/", path)
					if err == nil && rel != "." {
						parts := strings.Split(filepath.ToSlash(rel), "/")
						if len(parts) > 0 {
							first := parts[0]
							if topLevelDirs[first] {
								dirSizes[first] += info.Size()
							}
						}
					}
				}
				return nil
			})
		}
	}

	type dirEntry struct {
		name string
		size int64
	}
	var sortedDirs []dirEntry
	for name, size := range dirSizes {
		// Only include if >= 0.05 GB (rounds to 0.1 GB or more)
		if float64(size)/1024/1024/1024 >= 0.05 {
			sortedDirs = append(sortedDirs, dirEntry{name, size})
		}
	}

	// Sort by size descending
	sort.Slice(sortedDirs, func(i, j int) bool {
		return sortedDirs[i].size > sortedDirs[j].size
	})

	var details string
	for _, entry := range sortedDirs {
		details += fmt.Sprintf("\n- /%s: %.1f GB", entry.name, float64(entry.size)/1024/1024/1024)
	}

	// Dynamic critical threshold
	criticalThreshold := m.threshold + 15.0
	if criticalThreshold > 95.0 {
		criticalThreshold = 95.0
	}
	
	severity := "warning"
	if usage.UsedPercent >= criticalThreshold {
		severity = "critical"
	}

	message := fmt.Sprintf("Disk / is %.1f%% used (%.1f GB / %.1f GB), exceeding threshold of %.1f%%. Details:%s",
		usage.UsedPercent, 
		float64(usage.Used)/1024/1024/1024, 
		float64(usage.Total)/1024/1024/1024, 
		m.threshold, 
		details)

	return []Alert{
		{
			Type:     "disk",
			Message:  message,
			Value:    usage.UsedPercent,
			Severity: severity,
		},
	}
}
