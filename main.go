package main

import (
	"fmt"
	"os"

	"telemetry/cmd/gemini_cmd"
	"telemetry/cmd/run"
)

var Version = "dev"

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	command := os.Args[1]

	switch command {
	case "run":
		run.Execute(Version)
	case "gemini":
		gemini_cmd.Execute(os.Args[2:])
	case "version", "-v", "--version":
		fmt.Printf("telemetry %s\n", Version)
	case "help", "-h", "--help":
		printUsage()
	default:
		fmt.Printf("Unknown command: %s\n\n", command)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println(`telemetry - Server monitoring and alerting tool

Usage:
  telemetry <command> [arguments]

Commands:
  run          Start the monitoring daemon
  gemini       Send a prompt to Gemini AI
  version      Show version information
  help         Show this help message

Examples:
  telemetry run                        Start the monitoring service
  telemetry gemini "analyze this"      Send prompt to Gemini
  echo "prompt" | telemetry gemini     Pipe prompt to Gemini

Configuration:
  Config file: ~/.telemetry/config.yaml`)
}
