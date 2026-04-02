package gemini_cmd

import (
		"bufio"
		"encoding/json"
		"fmt"
		"os"
		"strings"

		"telemetry/internal/config"
		"telemetry/internal/gemini"
	)

type Response struct {
		Success  bool   `json:"success"`
		Response string `json:"response,omitempty"`
		Error    string `json:"error,omitempty"`
}

func Execute(args []string) {
		var prompt string

		// Check if input is from pipe/stdin or command line args
		stat, _ := os.Stdin.Stat()
		if (stat.Mode() & os.ModeCharDevice) == 0 {
					// Input is from pipe - read from stdin
					scanner := bufio.NewScanner(os.Stdin)
					var lines []string
					for scanner.Scan() {
									lines = append(lines, scanner.Text())
								}
					prompt = strings.Join(lines, "\n")
				} else if len(args) >= 1 {
					// Input is from command line args
					prompt = strings.Join(args, " ")
				} else {
					output := Response{Success: false, Error: "Usage: telemetry gemini <prompt> OR echo 'prompt' | telemetry gemini"}
					jsonOut, _ := json.Marshal(output)
					fmt.Println(string(jsonOut))
					os.Exit(1)
				}

		prompt = strings.TrimSpace(prompt)
		if prompt == "" {
					output := Response{Success: false, Error: "Empty prompt provided"}
					jsonOut, _ := json.Marshal(output)
					fmt.Println(string(jsonOut))
					os.Exit(1)
				}

		cfg, err := config.Load()
		if err != nil {
					output := Response{Success: false, Error: fmt.Sprintf("Failed to load config: %v", err)}
					jsonOut, _ := json.Marshal(output)
					fmt.Println(string(jsonOut))
					os.Exit(1)
				}

		if cfg.GeminiAPIKey == "" {
					output := Response{Success: false, Error: "gemini_api_key not configured in ~/.telemetry/config.yaml"}
					jsonOut, _ := json.Marshal(output)
					fmt.Println(string(jsonOut))
					os.Exit(1)
				}

		client, err := gemini.NewClient(cfg.GeminiAPIKey)
		if err != nil {
					output := Response{Success: false, Error: fmt.Sprintf("Failed to create Gemini client: %v", err)}
					jsonOut, _ := json.Marshal(output)
					fmt.Println(string(jsonOut))
					os.Exit(1)
				}
		defer client.Close()

		// Use a chat session for context-preserving conversations
		chat := client.StartChat()
		response, err := chat.Send(prompt)

		// Always output JSON for CLI compatibility (SSH calls parse this)
		var output Response
		if err != nil {
					output = Response{Success: false, Error: err.Error()}
				} else {
					output = Response{Success: true, Response: response}
				}
		jsonOut, _ := json.Marshal(output)
		fmt.Println(string(jsonOut))

		if !output.Success {
					os.Exit(1)
				}
}
