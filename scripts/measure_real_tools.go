package main

import (
	"encoding/json"
	"fmt"
	"log"

	"cando/internal/tooling"
)

func main() {
	// Create tool options (minimal for this measurement)
	opts := tooling.Options{
		WorkspaceRoot: "/tmp",
		ShellTimeout:  60,
		PlanPath:      "/tmp/plan.json",
		BinDir:        "/tmp/bin",
		ExternalData:  true,
		ProcessDir:    "/tmp/processes",
	}

	// Get default tools
	tools := tooling.DefaultTools(opts)
	registry := tooling.NewRegistry(tools...)

	// Get tool definitions as they would be sent to LLM
	definitions := registry.Definitions()

	// Marshal to JSON
	data, err := json.Marshal(definitions)
	if err != nil {
		log.Fatalf("Failed to marshal tool definitions: %v", err)
	}

	fmt.Printf("Real tool definitions from cando:\n")
	fmt.Printf("  Count: %d tools\n", len(definitions))
	fmt.Printf("  JSON size: %d bytes (~%.1fk)\n", len(data), float64(len(data))/1000.0)
	fmt.Println()

	// Show breakdown by tool
	fmt.Println("Size breakdown by tool:")
	for _, def := range definitions {
		defData, _ := json.Marshal(def)
		fmt.Printf("  %-20s: %5d bytes\n", def.Function.Name, len(defData))
	}
}
