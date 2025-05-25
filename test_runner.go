package main

import (
	"fmt"
	"os"
	"os/exec"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		return
	}

	command := os.Args[1]

	switch command {
	case "unit":
		runUnitTests()
	case "integration":
		runIntegrationTests()
	case "benchmark":
		runBenchmarkTests()
	case "all":
		runAllTests()
	case "coverage":
		runCoverageTests()
	case "race":
		runRaceTests()
	default:
		fmt.Printf("Unknown command: %s\n", command)
		printUsage()
	}
}

func printUsage() {
	fmt.Println("Usage: go run test_runner.go <command>")
	fmt.Println("Commands:")
	fmt.Println("  unit        - Run unit tests only")
	fmt.Println("  integration - Run integration tests only")
	fmt.Println("  benchmark   - Run benchmark tests")
	fmt.Println("  all         - Run all tests")
	fmt.Println("  coverage    - Run tests with coverage report")
	fmt.Println("  race        - Run tests with race detection")
}

func runUnitTests() {
	fmt.Println("Running unit tests...")
	cmd := exec.Command("go", "test", "-v", "./internal/server", "-run", "^Test[^I]")
	runCommand(cmd)
}

func runIntegrationTests() {
	fmt.Println("Running integration tests...")
	cmd := exec.Command("go", "test", "-v", "./internal/server", "-run", "TestIntegration")
	runCommand(cmd)
}

func runBenchmarkTests() {
	fmt.Println("Running benchmark tests...")
	cmd := exec.Command("go", "test", "-v", "./internal/server", "-bench", ".", "-benchmem")
	runCommand(cmd)
}

func runAllTests() {
	fmt.Println("Running all tests...")
	cmd := exec.Command("go", "test", "-v", "./...")
	runCommand(cmd)
}

func runCoverageTests() {
	fmt.Println("Running tests with coverage...")
	cmd := exec.Command("go", "test", "-v", "./internal/server", "-cover", "-coverprofile=coverage.out")
	runCommand(cmd)

	// Generate HTML coverage report
	fmt.Println("Generating HTML coverage report...")
	htmlCmd := exec.Command("go", "tool", "cover", "-html=coverage.out", "-o", "coverage.html")
	runCommand(htmlCmd)
	fmt.Println("Coverage report generated: coverage.html")
}

func runRaceTests() {
	fmt.Println("Running tests with race detection...")
	cmd := exec.Command("go", "test", "-v", "./internal/server", "-race")
	runCommand(cmd)
}

func runCommand(cmd *exec.Cmd) {
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err := cmd.Run()
	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			fmt.Printf("Command failed with exit code: %d\n", exitError.ExitCode())
		} else {
			fmt.Printf("Command failed: %v\n", err)
		}
	}
}
