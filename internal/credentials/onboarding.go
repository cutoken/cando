package credentials

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// Onboard runs the interactive first-time setup wizard
func Onboard(manager *Manager) (*Credentials, error) {
	fmt.Println()
	fmt.Println("═══════════════════════════════════════════════════════════")
	fmt.Println("  Welcome to Cando! Let's get you set up.")
	fmt.Println("═══════════════════════════════════════════════════════════")
	fmt.Println()

	creds := &Credentials{
		Providers: make(map[string]Provider),
	}

	// Step 1: Choose provider
	provider, err := chooseProvider()
	if err != nil {
		return nil, err
	}

	// Step 2: Get API key
	apiKey, err := getAPIKey(provider)
	if err != nil {
		return nil, err
	}

	// Save credentials
	creds.DefaultProvider = provider
	creds.SetProvider(provider, apiKey)

	if err := manager.Save(creds); err != nil {
		return nil, fmt.Errorf("save credentials: %w", err)
	}

	fmt.Println()
	fmt.Println("✓ API key saved securely to:", manager.Path())
	fmt.Println("✓", strings.ToUpper(provider), "set as default provider")
	fmt.Println("✓ Default config will be created at: ~/.cando/config.yaml")
	fmt.Println()
	fmt.Println("Setup complete! Starting Cando...")
	fmt.Println()

	return creds, nil
}

func chooseProvider() (string, error) {
	fmt.Println("Which AI provider would you like to use?")
	fmt.Println()
	fmt.Println("  1) Z.AI        - Fast Chinese models (GLM-4, recommended)")
	fmt.Println("  2) OpenRouter  - Access to Claude, GPT-4, and more")
	fmt.Println()

	choice := promptWithDefault("Choice", "1")

	switch choice {
	case "1", "zai", "z.ai":
		fmt.Println()
		fmt.Println("Great! You'll need a Z.AI API key.")
		fmt.Println("Get one free at: https://z.ai")
		fmt.Println()
		return "zai", nil
	case "2", "openrouter", "or":
		fmt.Println()
		fmt.Println("Great! You'll need an OpenRouter API key.")
		fmt.Println("Get one at: https://openrouter.ai/keys")
		fmt.Println()
		return "openrouter", nil
	default:
		return "", fmt.Errorf("invalid choice: %s", choice)
	}
}

func getAPIKey(provider string) (string, error) {
	for {
		apiKey := prompt(fmt.Sprintf("Enter your %s API key", strings.ToUpper(provider)))
		apiKey = strings.TrimSpace(apiKey)

		if apiKey == "" {
			fmt.Println("❌ API key cannot be empty. Please try again.")
			continue
		}

		// Basic validation
		if !strings.HasPrefix(apiKey, "sk-") && !strings.Contains(apiKey, "glm-") {
			fmt.Println("⚠ Warning: API key doesn't look valid (should start with 'sk-')")
			confirm := promptWithDefault("Continue anyway? [y/n]", "n")
			if !strings.HasPrefix(strings.ToLower(confirm), "y") {
				continue
			}
		}

		return apiKey, nil
	}
}

// SetupMenu shows the credential management menu
func SetupMenu(manager *Manager) error {
	creds, err := manager.Load()
	if err != nil {
		return err
	}

	for {
		fmt.Println()
		fmt.Println("═══════════════════════════════════════════════════════════")
		fmt.Println("  Cando Setup")
		fmt.Println("═══════════════════════════════════════════════════════════")
		fmt.Println()
		fmt.Println("Current Configuration:")
		if creds.DefaultProvider != "" {
			fmt.Println("  Default Provider:", strings.ToUpper(creds.DefaultProvider))
		} else {
			fmt.Println("  Default Provider: (not set)")
		}
		fmt.Println()

		fmt.Println("Configured Providers:")
		if len(creds.Providers) == 0 {
			fmt.Println("  (none)")
		} else {
			for name, p := range creds.Providers {
				status := "✗"
				if p.APIKey != "" {
					status = "✓"
				}
				active := ""
				if name == creds.DefaultProvider {
					active = " (active)"
				}
				fmt.Printf("  %s %s%s\n", status, strings.ToUpper(name), active)
			}
		}
		fmt.Println()

		fmt.Println("Options:")
		fmt.Println("  1) Add/update provider API key")
		fmt.Println("  2) Change default provider")
		fmt.Println("  3) Remove provider")
		fmt.Println("  4) Exit")
		fmt.Println()

		choice := promptWithDefault("Choice", "4")

		switch choice {
		case "1":
			if err := addProvider(creds, manager); err != nil {
				fmt.Println("❌ Error:", err)
			}
		case "2":
			if err := changeDefaultProvider(creds, manager); err != nil {
				fmt.Println("❌ Error:", err)
			}
		case "3":
			if err := removeProvider(creds, manager); err != nil {
				fmt.Println("❌ Error:", err)
			}
		case "4", "exit", "quit", "q":
			return nil
		default:
			fmt.Println("❌ Invalid choice")
		}
	}
}

func addProvider(creds *Credentials, manager *Manager) error {
	fmt.Println()
	fmt.Println("Which provider?")
	fmt.Println("  1) Z.AI")
	fmt.Println("  2) OpenRouter")
	fmt.Println()

	choice := prompt("Choice")

	var provider string
	switch choice {
	case "1", "zai", "z.ai":
		provider = "zai"
	case "2", "openrouter", "or":
		provider = "openrouter"
	default:
		return fmt.Errorf("invalid provider: %s", choice)
	}

	apiKey, err := getAPIKey(provider)
	if err != nil {
		return err
	}

	creds.SetProvider(provider, apiKey)

	// If this is the only provider, make it default
	if creds.DefaultProvider == "" {
		creds.DefaultProvider = provider
	}

	if err := manager.Save(creds); err != nil {
		return err
	}

	fmt.Println()
	fmt.Println("✓ API key saved for", strings.ToUpper(provider))
	return nil
}

func changeDefaultProvider(creds *Credentials, manager *Manager) error {
	providers := creds.ListProviders()
	if len(providers) == 0 {
		return fmt.Errorf("no providers configured. Add one first")
	}

	fmt.Println()
	fmt.Println("Available providers:")
	for i, name := range providers {
		current := ""
		if name == creds.DefaultProvider {
			current = " (current)"
		}
		fmt.Printf("  %d) %s%s\n", i+1, strings.ToUpper(name), current)
	}
	fmt.Println()

	choice := prompt("Choice")
	idx := 0
	fmt.Sscanf(choice, "%d", &idx)
	idx-- // Convert to 0-based

	if idx < 0 || idx >= len(providers) {
		return fmt.Errorf("invalid choice")
	}

	creds.DefaultProvider = providers[idx]

	if err := manager.Save(creds); err != nil {
		return err
	}

	fmt.Println()
	fmt.Println("✓ Default provider set to", strings.ToUpper(creds.DefaultProvider))
	return nil
}

func removeProvider(creds *Credentials, manager *Manager) error {
	providers := creds.ListProviders()
	if len(providers) == 0 {
		return fmt.Errorf("no providers configured")
	}

	fmt.Println()
	fmt.Println("Which provider to remove?")
	for i, name := range providers {
		fmt.Printf("  %d) %s\n", i+1, strings.ToUpper(name))
	}
	fmt.Println()

	choice := prompt("Choice")
	idx := 0
	fmt.Sscanf(choice, "%d", &idx)
	idx-- // Convert to 0-based

	if idx < 0 || idx >= len(providers) {
		return fmt.Errorf("invalid choice")
	}

	providerName := providers[idx]
	confirm := promptWithDefault(fmt.Sprintf("Really remove %s? [y/n]", strings.ToUpper(providerName)), "n")
	if !strings.HasPrefix(strings.ToLower(confirm), "y") {
		fmt.Println("Cancelled")
		return nil
	}

	creds.RemoveProvider(providerName)

	// If we removed the default, pick a new one
	if creds.DefaultProvider == providerName {
		remaining := creds.ListProviders()
		if len(remaining) > 0 {
			creds.DefaultProvider = remaining[0]
		} else {
			creds.DefaultProvider = ""
		}
	}

	if err := manager.Save(creds); err != nil {
		return err
	}

	fmt.Println()
	fmt.Println("✓ Removed", strings.ToUpper(providerName))
	return nil
}

// Helper functions
func prompt(msg string) string {
	fmt.Printf("%s: ", msg)
	reader := bufio.NewReader(os.Stdin)
	line, _ := reader.ReadString('\n')
	return strings.TrimSpace(line)
}

func promptWithDefault(msg, defaultValue string) string {
	fmt.Printf("%s [%s]: ", msg, defaultValue)
	reader := bufio.NewReader(os.Stdin)
	line, _ := reader.ReadString('\n')
	line = strings.TrimSpace(line)
	if line == "" {
		return defaultValue
	}
	return line
}
