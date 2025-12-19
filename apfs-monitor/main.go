package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const (
	// Default thresholds
	defaultWarningThresholdGB = 30.0
	defaultCriticalThresholdGB = 20.0
	defaultCheckInterval = 5 * time.Minute
)

type APFSContainer struct {
	Name          string
	TotalSpace    float64 // in GB
	FreeSpace     float64 // in GB
	UsedSpace     float64 // in GB
	PercentUsed   float64
	Volumes       []string
}

type Config struct {
	warningThreshold  float64
	criticalThreshold float64
	checkInterval     time.Duration
	logFile           string
	sendNotifications bool
	daemonMode        bool
	containerFilter   string // Specific container to monitor (e.g., "disk3"), empty = auto-detect boot container
}

func main() {
	config := parseFlags()

	if config.daemonMode {
		log.Printf("Starting APFS monitor daemon (checking every %v)", config.checkInterval)
		runDaemon(config)
	} else {
		// One-time check
		checkAndReport(config)
	}
}

func parseFlags() Config {
	config := Config{}

	flag.Float64Var(&config.warningThreshold, "warning", defaultWarningThresholdGB,
		"Warning threshold in GB of free space")
	flag.Float64Var(&config.criticalThreshold, "critical", defaultCriticalThresholdGB,
		"Critical threshold in GB of free space")
	flag.DurationVar(&config.checkInterval, "interval", defaultCheckInterval,
		"Check interval in daemon mode (e.g., 5m, 1h)")
	flag.StringVar(&config.logFile, "log", "",
		"Log file path (default: stdout)")
	flag.BoolVar(&config.sendNotifications, "notify", true,
		"Send macOS notifications on threshold breach")
	flag.BoolVar(&config.daemonMode, "daemon", false,
		"Run as daemon (continuous monitoring)")
	flag.StringVar(&config.containerFilter, "container", "",
		"Specific APFS container to monitor (e.g., disk3). If empty, auto-detects boot container")

	flag.Parse()

	if config.logFile != "" {
		f, err := os.OpenFile(config.logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			log.Fatal(err)
		}
		log.SetOutput(f)
	}

	return config
}

func runDaemon(config Config) {
	ticker := time.NewTicker(config.checkInterval)
	defer ticker.Stop()

	// Run immediately on startup
	checkAndReport(config)

	// Then run on interval
	for range ticker.C {
		checkAndReport(config)
	}
}

func checkAndReport(config Config) {
	// Determine which container to monitor
	var targetContainer string
	if config.containerFilter != "" {
		// User specified a specific container
		targetContainer = config.containerFilter
		log.Printf("Monitoring specified container: %s", targetContainer)
	} else {
		// Auto-detect boot container
		bootContainer, err := getBootContainer()
		if err != nil {
			log.Printf("Error detecting boot container: %v (will monitor all containers)", err)
			targetContainer = "" // Empty means monitor all
		} else {
			targetContainer = bootContainer
			log.Printf("Auto-detected boot container: %s", targetContainer)
		}
	}

	containers, err := getAPFSContainers()
	if err != nil {
		log.Printf("Error getting APFS containers: %v", err)
		return
	}

	for _, container := range containers {
		// Skip containers we're not monitoring
		if targetContainer != "" && container.Name != targetContainer {
			continue
		}

		level := checkThresholds(container, config)

		switch level {
		case "CRITICAL":
			msg := fmt.Sprintf("CRITICAL: APFS container %s has only %.2f GB free (%.1f%% used)",
				container.Name, container.FreeSpace, container.PercentUsed)
			log.Println(msg)
			if config.sendNotifications {
				sendNotification("APFS Space Critical", msg)
			}
		case "WARNING":
			msg := fmt.Sprintf("WARNING: APFS container %s has %.2f GB free (%.1f%% used)",
				container.Name, container.FreeSpace, container.PercentUsed)
			log.Println(msg)
			if config.sendNotifications {
				sendNotification("APFS Space Warning", msg)
			}
		case "OK":
			log.Printf("OK: APFS container %s has %.2f GB free (%.1f%% used)",
				container.Name, container.FreeSpace, container.PercentUsed)
		}
	}
}

func checkThresholds(container APFSContainer, config Config) string {
	if container.FreeSpace <= config.criticalThreshold {
		return "CRITICAL"
	}
	if container.FreeSpace <= config.warningThreshold {
		return "WARNING"
	}
	return "OK"
}

func getAPFSContainers() ([]APFSContainer, error) {
	cmd := exec.Command("diskutil", "apfs", "list")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to run diskutil: %w", err)
	}

	return parseAPFSList(string(output))
}

func parseAPFSList(output string) ([]APFSContainer, error) {
	var containers []APFSContainer
	var currentContainer *APFSContainer

	scanner := bufio.NewScanner(strings.NewReader(output))

	// Regex patterns
	containerPattern := regexp.MustCompile(`Container (disk\d+)`)
	capacityPattern := regexp.MustCompile(`Capacity In Use By Volumes:\s+([\d]+)\s+B`)
	freePattern := regexp.MustCompile(`Capacity Not Allocated:\s+([\d]+)\s+B`)
	volumePattern := regexp.MustCompile(`\+->\s+Volume (\S+)`)

	for scanner.Scan() {
		line := scanner.Text()

		// New container
		if matches := containerPattern.FindStringSubmatch(line); matches != nil {
			if currentContainer != nil {
				containers = append(containers, *currentContainer)
			}
			currentContainer = &APFSContainer{
				Name: matches[1],
			}
		}

		if currentContainer == nil {
			continue
		}

		// Parse capacity
		if matches := capacityPattern.FindStringSubmatch(line); matches != nil {
			bytes, _ := strconv.ParseFloat(matches[1], 64)
			currentContainer.UsedSpace = bytes / (1024 * 1024 * 1024) // Convert to GB
		}

		// Parse free space
		if matches := freePattern.FindStringSubmatch(line); matches != nil {
			bytes, _ := strconv.ParseFloat(matches[1], 64)
			currentContainer.FreeSpace = bytes / (1024 * 1024 * 1024) // Convert to GB
		}

		// Parse volumes
		if matches := volumePattern.FindStringSubmatch(line); matches != nil {
			currentContainer.Volumes = append(currentContainer.Volumes, matches[1])
		}
	}

	// Add last container
	if currentContainer != nil {
		containers = append(containers, *currentContainer)
	}

	// Calculate total space and percentage
	for i := range containers {
		containers[i].TotalSpace = containers[i].UsedSpace + containers[i].FreeSpace
		if containers[i].TotalSpace > 0 {
			containers[i].PercentUsed = (containers[i].UsedSpace / containers[i].TotalSpace) * 100
		}
	}

	return containers, nil
}

func getBootContainer() (string, error) {
	cmd := exec.Command("diskutil", "info", "/")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get boot volume info: %w", err)
	}

	// Parse "APFS Container: disk3" or "Part of Whole: disk3"
	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	containerPattern := regexp.MustCompile(`APFS Container:\s+(disk\d+)`)
	partOfWholePattern := regexp.MustCompile(`Part of Whole:\s+(disk\d+)`)

	for scanner.Scan() {
		line := scanner.Text()

		// Try APFS Container first (most specific)
		if matches := containerPattern.FindStringSubmatch(line); matches != nil {
			return matches[1], nil
		}

		// Fallback to Part of Whole
		if matches := partOfWholePattern.FindStringSubmatch(line); matches != nil {
			return matches[1], nil
		}
	}

	return "", fmt.Errorf("could not determine boot container from diskutil output")
}

func sendNotification(title, message string) {
	script := fmt.Sprintf(`display alert "%s" message "%s"`,
		title, message)
	cmd := exec.Command("osascript", "-e", script)
	if err := cmd.Run(); err != nil {
		log.Printf("Failed to send notification: %v", err)
	}
}
