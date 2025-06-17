package main

import (
	"bufio"
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/user"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

type Config struct {
	Token string `json:"token"`
	API   string `json:"api"`
}

var (
	configPath     = ""
	config         Config
	insecureClient = &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}
	company string
)

func main() {
	initConfigPath()
	loadConfig()

	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}

// Command definitions
var rootCmd = &cobra.Command{
	Use:   "bbrf",
	Short: "BBRF CLI - Bug Bounty Reconnaissance Framework",
	Long:  `A command-line interface for managing bug bounty reconnaissance data.`,
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&company, "company", "c", "", "Company name (required for most commands)")

	// Add all commands
	rootCmd.AddCommand(
		&cobra.Command{
			Use:   "login",
			Short: "Login to BBRF server and save token",
			Run:   func(cmd *cobra.Command, args []string) { doLogin() },
		},
		&cobra.Command{
			Use:   "companies",
			Short: "List all companies",
			Run:   func(cmd *cobra.Command, args []string) { call("GET", "/api/company/list", "") },
		},
		createCompanyCommands(),
	)
}

func createCompanyCommands() *cobra.Command {
	companyCmd := &cobra.Command{
		Use:   "company",
		Short: "Company operations",
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			if company == "" && len(args) > 0 {
				company = args[0]
			}
			if company == "" {
				log.Fatal("Company name required. Use --company flag or provide as argument.")
			}
		},
	}

	// Simple commands
	companyCmd.AddCommand(
		&cobra.Command{
			Use:   "add",
			Short: "Add a new company",
			Run: func(cmd *cobra.Command, args []string) {
				call("POST", "/api/company", fmt.Sprintf(`{"company":"%s"}`, company))
			},
		},
		&cobra.Command{
			Use:   "domains",
			Short: "List all domains",
			Run: func(cmd *cobra.Command, args []string) {
				call("GET", "/api/domains?company="+company, "")
			},
		},
		&cobra.Command{
			Use:   "count",
			Short: "Count domains",
			Run: func(cmd *cobra.Command, args []string) {
				call("GET", "/api/domains/count?company="+company, "")
			},
		},
		&cobra.Command{
			Use:   "show <query> [count]",
			Short: "Show matching domains",
			Args:  cobra.MinimumNArgs(1),
			Run: func(cmd *cobra.Command, args []string) {
				query := args[0]
				countFlag := "false"
				if len(args) > 1 && args[1] == "count" {
					countFlag = "true"
				}
				call("GET", fmt.Sprintf("/api/domains/show?company=%s&q=%s&count=%s", company, query, countFlag), "")
			},
		},
	)

	// Complex commands with subcommands
	companyCmd.AddCommand(
		createCRUDCommand("domain", "domains", map[string]string{
			"add":    "/api/domains/add",
			"remove": "/api/domains/remove",
		}),
		createCRUDCommand("ip", "ips", map[string]string{
			"add":    "/api/ip",
			"remove": "/api/ip/remove",
			"list":   "/api/ip/list",
		}),
		createCRUDCommand("asn", "asns", map[string]string{
			"add":    "/api/asn/add",
			"remove": "/api/asn/remove",
			"list":   "/api/asn/list",
		}),
		createScopeCommand(),
	)

	return companyCmd
}

// Generic CRUD command creator
func createCRUDCommand(name, dataKey string, endpoints map[string]string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   name,
		Short: fmt.Sprintf("%s operations", strings.Title(name)),
	}

	for action, endpoint := range endpoints {
		action, endpoint := action, endpoint // capture loop vars

		if action == "list" {
			cmd.AddCommand(&cobra.Command{
				Use:   action,
				Short: fmt.Sprintf("List %s", name+"s"),
				Run: func(cmd *cobra.Command, args []string) {
					call("GET", endpoint+"?company="+company, "")
				},
			})
		} else {
			cmd.AddCommand(&cobra.Command{
				Use:   fmt.Sprintf("%s [items...]", action),
				Short: fmt.Sprintf("%s %s", strings.Title(action), name+"s"),
				Long: fmt.Sprintf(`%s %s. Supports:
- Direct: %s %s item1 item2
- Stdin: echo 'item' | bbrf company %s %s -
- File: bbrf company %s %s @file.txt`, strings.Title(action), name+"s", name, action, name, action, name, action),
				Run: func(cmd *cobra.Command, args []string) {
					handleInputAndPost(endpoint, company, dataKey, args)
				},
			})
		}
	}

	return cmd
}

// Scope command with special handling
func createScopeCommand() *cobra.Command {
	scopeCmd := &cobra.Command{
		Use:   "scope",
		Short: "Scope management",
	}

	scopeActions := map[string]struct {
		endpoint string
		short    string
	}{
		"inscope":         {"/api/scope/in", "Add in-scope domains"},
		"outscope":        {"/api/scope/out", "Add out-of-scope domains"},
		"remove-inscope":  {"/api/scope/remove", "Remove in-scope domains"},
		"remove-outscope": {"/api/scope/remove", "Remove out-of-scope domains"},
	}

	// Add input commands
	for action, config := range scopeActions {
		action, config := action, config // capture loop vars
		scopeCmd.AddCommand(&cobra.Command{
			Use:   fmt.Sprintf("%s [domains...]", action),
			Short: config.short,
			Run: func(cmd *cobra.Command, args []string) {
				handleInputAndPost(config.endpoint, company, "domains", args)
			},
		})
	}

	// Add show command
	scopeCmd.AddCommand(&cobra.Command{
		Use:   "show <in|out>",
		Short: "Show scope domains",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			scopeType := args[0]
			if scopeType != "in" && scopeType != "out" {
				log.Fatal("Scope type must be 'in' or 'out'")
			}
			call("GET", fmt.Sprintf("/api/scope/show?company=%s&type=%s", company, scopeType), "")
		},
	})

	return scopeCmd
}

// Utility functions
func initConfigPath() {
	usr, err := user.Current()
	if err != nil {
		log.Fatal(err)
	}
	configPath = filepath.Join(usr.HomeDir, ".bbrf", "config.json")
	os.MkdirAll(filepath.Dir(configPath), 0700)
}

func loadConfig() {
	if data, err := os.ReadFile(configPath); err == nil {
		json.Unmarshal(data, &config)
	}
}

func doLogin() {
	reader := bufio.NewReader(os.Stdin)

	fmt.Print("API Server URL (e.g., https://localhost:8443): ")
	api, _ := reader.ReadString('\n')
	fmt.Print("Username: ")
	username, _ := reader.ReadString('\n')
	fmt.Print("Password: ")
	password, _ := reader.ReadString('\n')

	api = strings.TrimSpace(strings.ReplaceAll(api, " ", ""))
	username = strings.TrimSpace(username)
	password = strings.TrimSpace(password)

	body := fmt.Sprintf(`{"username":"%s","password":"%s"}`, username, password)
	resp, err := insecureClient.Post(api+"/login", "application/json", bytes.NewBuffer([]byte(body)))
	if err != nil {
		log.Fatalf("Login failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		respData, _ := io.ReadAll(resp.Body)
		log.Fatalf("Login error: %s", respData)
	}

	respData, _ := io.ReadAll(resp.Body)
	var result map[string]string
	json.Unmarshal(respData, &result)

	config = Config{Token: result["token"], API: api}
	data, _ := json.Marshal(config)
	os.WriteFile(configPath, data, 0600)
	fmt.Println("Login successful and token saved.")
}

func handleInputAndPost(path, company, key string, args []string) {
	if len(args) < 1 {
		log.Fatal("No input provided")
	}

	var value string
	switch {
	case args[0] == "-":
		// From stdin
		input, _ := io.ReadAll(os.Stdin)
		value = string(input)
	case strings.HasPrefix(args[0], "@") || strings.HasSuffix(args[0], ".txt"):
		// From file
		filePath := strings.TrimPrefix(args[0], "@")
		content, err := os.ReadFile(filePath)
		if err != nil {
			log.Fatalf("Failed to read file: %v", err)
		}
		value = string(content)
	default:
		// Direct arguments
		value = strings.Join(args, " ")
	}

	body := map[string]string{"company": company, key: value}
	jsonBody, _ := json.Marshal(body)
	call("POST", path, string(jsonBody))
}

func call(method, path, body string) {
	url := config.API + path
	var req *http.Request
	var err error

	if method == "GET" {
		req, err = http.NewRequest("GET", url, nil)
	} else {
		req, err = http.NewRequest(method, url, bytes.NewBuffer([]byte(body)))
		req.Header.Set("Content-Type", "application/json")
	}

	if err != nil {
		log.Fatalf("Failed to create request: %v", err)
	}

	req.Header.Set("Authorization", "Bearer "+config.Token)
	resp, err := insecureClient.Do(req)
	if err != nil {
		log.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	respData, _ := io.ReadAll(resp.Body)

	// Special handling for company list
	if strings.HasSuffix(path, "/api/company/list") {
		var companies []string
		if err := json.Unmarshal(respData, &companies); err != nil {
			log.Fatalf("Failed to parse company list: %v", err)
		}
		for _, c := range companies {
			fmt.Println(c)
		}
	} else {
		fmt.Println(string(respData))
	}
}
