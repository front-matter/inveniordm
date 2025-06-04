/*
Copyright © 2025 Front Matter <info@front-matter.io>
*/
package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"

	"github.com/fatih/color"
	"github.com/hashicorp/go-version"
	"github.com/spf13/cobra"
)

type Requirement struct {
	Name        string
	Description string
	CheckFunc   func() (bool, string)
}

var developmentFlag bool

// checkRequirementsCmd represents the checkRequirements command
var checkRequirementsCmd = &cobra.Command{
	Use:   "check-requirements",
	Short: "Check if system meets requirements",
	Long:  `Checks if the system fulfills all pre-requirements for running InvenioRDM.`,
	Run: func(cmd *cobra.Command, args []string) {
		runCheckRequirements()
	},
}

func init() {
	rootCmd.AddCommand(checkRequirementsCmd)
	checkRequirementsCmd.Flags().BoolVarP(&developmentFlag, "development", "d", false, "Check requirements for a local development installation")
}

// runCheckRequirements checks all system requirements
func runCheckRequirements() {
	green := color.New(color.FgGreen).SprintFunc()
	red := color.New(color.FgRed).SprintFunc()
	yellow := color.New(color.FgYellow).SprintFunc()

	fmt.Println(green("Checking requirements..."))

	// Define the requirements to check
	requirements := getRequirements()

	// Add development-specific requirements if flag is set
	if developmentFlag {
		requirements = append(requirements, getDevelopmentRequirements()...)
	}

	// Run all requirement checks
	allPassed := true
	for _, req := range requirements {
		fmt.Printf("Checking %s... ", req.Name)
		passed, msg := req.CheckFunc()

		if passed {
			fmt.Println(green("OK"))
			if msg != "" {
				fmt.Printf("  %s\n", msg)
			}
		} else {
			fmt.Println(red("FAILED"))
			fmt.Printf("  %s\n", msg)
			allPassed = false
		}
	}

	// Display final result
	if allPassed {
		fmt.Println(green("All requirements are fulfilled."))
	} else {
		fmt.Println(red("Requirements not met."))
		fmt.Println(yellow("Please install missing requirements before proceeding."))
		os.Exit(1)
	}
}

// getRequirements returns a list of basic requirements
func getRequirements() []Requirement {
	return []Requirement{
		{
			Name:        "Docker",
			Description: "Docker must be installed and running",
			CheckFunc: func() (bool, string) {
				cmd := exec.Command("docker", "--version")
				output, err := cmd.Output()
				if err != nil {
					return false, "Docker is not installed or not in PATH"
				}

				// Check if Docker daemon is running
				cmd = exec.Command("docker", "info")
				if err := cmd.Run(); err != nil {
					return false, "Docker daemon is not running"
				}

				// Parse version from output like "Docker version 24.0.7, build afdd53b"
				versionStr := strings.TrimSpace(string(output))
				parts := strings.Fields(versionStr)
				var versionNumber string
				if len(parts) >= 3 {
					versionNumber = parts[2] // Extract just the version number
					if strings.Contains(versionNumber, ",") {
						versionNumber = strings.Split(versionNumber, ",")[0]
					}
				} else {
					versionNumber = versionStr
				}

				// Parse version for constraint checking
				dockerVersion, err := version.NewVersion(versionNumber)
				if err != nil {
					return false, fmt.Sprintf("Error checking Docker version.")
				}

				// Make sure Docker version includes Compose command (Docker 19+)
				constraints, err := version.NewConstraint(">= 19.0")
				if err != nil {
					return false, fmt.Sprintf("Error checking Docker version number.")
				}

				if constraints.Check(dockerVersion) {
					return true, fmt.Sprintf("Docker version %s is recent enough.", versionNumber)
				} else {
					return false, fmt.Sprintf("Docker version too old: %s. Minimum required: 19.0", versionNumber)
				}
			},
		},
		{
			Name:        "Free disk space",
			Description: "At least 10GB of free disk space",
			CheckFunc: func() (bool, string) {
				requiredGB := float64(10) // 10GB required

				freeGB, err := getFreeDiskSpace()
				if err != nil {
					return false, fmt.Sprintf("Could not determine free disk space: %v", err)
				}

				if freeGB < requiredGB {
					return false, fmt.Sprintf("Only %.1fGB free disk space available. At least %.0fGB required", freeGB, requiredGB)
				}

				return true, fmt.Sprintf("%.1fGB free disk space available", freeGB)
			},
		},
	}
}

// getFreeDiskSpace returns the amount of free disk space in GB
func getFreeDiskSpace() (float64, error) {
	switch runtime.GOOS {
	case "windows":
		return getFreeDiskSpaceWindows()
	case "darwin", "linux":
		return getFreeDiskSpaceUnix()
	default:
		// Best effort for unsupported OS
		return getFreeDiskSpaceUnix()
	}
}

// getFreeDiskSpaceUnix returns free disk space on Unix/Linux/macOS in GB
func getFreeDiskSpaceUnix() (float64, error) {
	// Get the disk usage for the root filesystem (or current directory)
	cmd := exec.Command("df", "-k", ".")
	output, err := cmd.Output()
	if err != nil {
		return 0, fmt.Errorf("failed to execute df command: %v", err)
	}

	// Parse the output (skip the header line)
	lines := strings.Split(string(output), "\n")
	if len(lines) < 2 {
		return 0, fmt.Errorf("unexpected df output format")
	}

	// Extract fields (varies slightly between BSD and GNU versions of df)
	fields := strings.Fields(lines[1])
	if len(fields) < 4 {
		return 0, fmt.Errorf("could not parse df output")
	}

	// The available space is usually in the 3rd or 4th field, in KB
	// Try to find a numeric field that could be the available space
	var availableKB float64
	for i := 3; i < len(fields); i++ {
		if val, err := strconv.ParseFloat(fields[i], 64); err == nil {
			availableKB = val
			break
		}
	}

	if availableKB == 0 {
		return 0, fmt.Errorf("could not determine available space")
	}

	// Convert KB to GB
	availableGB := availableKB / 1024 / 1024

	return availableGB, nil
}

// getFreeDiskSpaceWindows returns free disk space on Windows in GB
func getFreeDiskSpaceWindows() (float64, error) {
	// PowerShell command to get free space on C: drive
	cmd := exec.Command("powershell", "-Command",
		"(Get-PSDrive C | Select-Object Free).Free")
	output, err := cmd.Output()
	if err != nil {
		return 0, fmt.Errorf("failed to execute PowerShell command: %v", err)
	}

	// Parse the output (in bytes)
	freeBytes, err := strconv.ParseFloat(strings.TrimSpace(string(output)), 64)
	if err != nil {
		return 0, fmt.Errorf("could not parse disk space output: %v", err)
	}

	// Convert bytes to GB
	freeGB := freeBytes / 1024 / 1024 / 1024

	return freeGB, nil
}

// getDevelopmentRequirements returns additional requirements for development mode
func getDevelopmentRequirements() []Requirement {
	return []Requirement{
		{
			Name:        "Python",
			Description: "Python 3.7+ must be installed",
			CheckFunc: func() (bool, string) {
				cmd := exec.Command("python3", "--version")
				output, err := cmd.Output()
				if err != nil {
					return false, "Python 3 is not installed or not in PATH"
				}

				version := strings.TrimSpace(string(output))
				return true, version
			},
		},
		{
			Name:        "pip",
			Description: "pip must be installed",
			CheckFunc: func() (bool, string) {
				cmd := exec.Command("pip3", "--version")
				if err := cmd.Run(); err != nil {
					return false, "pip is not installed or not in PATH"
				}
				return true, ""
			},
		},
		{
			Name:        "npm",
			Description: "npm must be installed",
			CheckFunc: func() (bool, string) {
				cmd := exec.Command("npm", "--version")
				if err := cmd.Run(); err != nil {
					return false, "npm is not installed or not in PATH"
				}
				return true, ""
			},
		},
		{
			Name:        "git",
			Description: "git must be installed",
			CheckFunc: func() (bool, string) {
				cmd := exec.Command("git", "--version")
				if err := cmd.Run(); err != nil {
					return false, "git is not installed or not in PATH"
				}
				return true, ""
			},
		},
	}
}
