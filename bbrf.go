package main

import (
	"bufio"
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/user"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/charmbracelet/fang"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

type Config struct {
	Token string `json:"token"`
	API   string `json:"api"`
}

type ScopeManager struct {
	InScope  []string
	OutScope []string
	company  string
}

var (
	configPath     = ""
	config         Config
	insecureClient = &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}
	company           string
	enableScopeFilter bool
	allowOutOfScope   bool
	verboseScope      bool

	// Color functions using fatih/color
	title     = color.New(color.FgMagenta, color.Bold).SprintFunc()
	success   = color.New(color.FgGreen, color.Bold).SprintFunc()
	errorC    = color.New(color.FgRed, color.Bold).SprintFunc()
	warning   = color.New(color.FgYellow, color.Bold).SprintFunc()
	info      = color.New(color.FgBlue).SprintFunc()
	prompt    = color.New(color.FgMagenta, color.Bold).SprintFunc()
	data      = color.New(color.FgCyan).SprintFunc()
	count     = color.New(color.FgGreen, color.Bold).SprintFunc()
	domainClr = color.New(color.FgBlue).SprintFunc()
	header    = color.New(color.FgWhite, color.Bold, color.BgBlack).SprintFunc()
)

func main() {
	initConfigPath()
	loadConfig()

	// Use fang.Execute instead of rootCmd.Execute
	if err := fang.Execute(context.TODO(), rootCmd); err != nil {
		fmt.Println(errorC("❌ Error: " + err.Error()))
		os.Exit(1)
	}
}

// Command definitions
var rootCmd = &cobra.Command{
	Use:   "bbrf",
	Short: title("🔍 BBRF CLI - Bug Bounty Reconnaissance Framework"),
	Long: title("🔍 BBRF CLI - Bug Bounty Reconnaissance Framework") + "\n\n" +
		info("A command-line interface for managing bug bounty reconnaissance data!"),
	Example: `  # Login to BBRF server
  bbrf login

  # List all companies
  bbrf companies

  # Add a new company
  bbrf company add -c example

  # List domains for a company
  bbrf company domain list -c example

  # Add domains with scope filtering (default)
  bbrf company domain add example.com sub.example.com -c example

  # Add domains with verbose scope info
  bbrf company domain add @domains.txt -c example --verbose-scope

  # Disable scope filtering
  bbrf company domain add example.com -c example --scope-filter=false

  # Allow out-of-scope domains
  bbrf company domain add example.com -c example --allow-out-of-scope`,
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&company, "company", "c", "", "Company name (required for most commands)")
	rootCmd.PersistentFlags().BoolVar(&enableScopeFilter, "scope-filter", true, "Enable automatic scope filtering")
	rootCmd.PersistentFlags().BoolVar(&allowOutOfScope, "allow-out-of-scope", false, "Allow out-of-scope domains to be added")
	rootCmd.PersistentFlags().BoolVar(&verboseScope, "verbose-scope", false, "Show detailed scope filtering info")

	// Add all commands
	rootCmd.AddCommand(
		&cobra.Command{
			Use:     "login",
			Short:   "🔐 Login to BBRF server and save token",
			Example: "  bbrf login",
			Run:     func(cmd *cobra.Command, args []string) { doLogin() },
		},
		&cobra.Command{
			Use:     "companies",
			Short:   "🏢 List all companies",
			Example: "  bbrf companies",
			Run:     func(cmd *cobra.Command, args []string) { call("GET", "/api/company/list", "") },
		},
		createCompanyCommands(),
	)
}

func createCompanyCommands() *cobra.Command {
	companyCmd := &cobra.Command{
		Use:   "company",
		Short: "🏢 Company operations",
		Example: `  # Add a company
  bbrf company add -c acme

  # List domains
  bbrf company domain list -c acme

  # Add domains with scope filtering
  bbrf company domain add domain1.com domain2.com -c acme

  # Add domains from file with verbose scope info
  bbrf company domain add @domains.txt -c acme --verbose-scope

  # List IPs
  bbrf company ip list -c acme

  # Show matching domains
  bbrf company show "*.example.com" -c acme

  # Manage scope
  bbrf company scope inscope domain.com -c acme`,
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			if company == "" && len(args) > 0 {
				company = args[0]
			}
		},
	}

	// Simple commands
	companyCmd.AddCommand(
		&cobra.Command{
			Use:   "add",
			Short: "➕ Add a new company",
			Example: `  # Add a company using flag
  bbrf company add -c acme

  # Add a company using argument
  bbrf company add -c acme`,
			Run: func(cmd *cobra.Command, args []string) {
				fmt.Println(info("📝 Adding company: " + company))
				call("POST", "/api/company", fmt.Sprintf(`{"company":"%s"}`, company))
			},
		},
		&cobra.Command{
			Use:   "remove",
			Short: "🗑️ Remove a company",
			Example: `  # Remove a company using flag
		  bbrf company remove -c acme

		  # Remove a company using argument
		  bbrf company remove acme`,
			Run: func(cmd *cobra.Command, args []string) {
				fmt.Println(info("🗑️ Removing company: " + company))
				call("POST", "/api/company/remove", fmt.Sprintf(`{"company":"%s"}`, company))
			},
		},
		&cobra.Command{
			Use:   "show <query> [count]",
			Short: "👁️  Show matching domains",
			Example: `  # Show domains matching a pattern
  bbrf company show "*.example.com" -c acme

  # Show domains with count
  bbrf company show "*.example.com" count -c acme`,
			Args: cobra.MinimumNArgs(1),
			Run: func(cmd *cobra.Command, args []string) {
				query := args[0]
				countFlag := "false"
				if len(args) > 1 && args[1] == "count" {
					countFlag = "true"
				}
				fmt.Println(info(fmt.Sprintf("🔍 Searching for domains matching '%s' in %s", query, company)))
				call("GET", fmt.Sprintf("/api/domains/show?company=%s&q=%s&count=%s", company, query, countFlag), "")
			},
		},
	)

	// Complex commands with subcommands
	companyCmd.AddCommand(
		createCRUDCommand("domain", "domains", map[string]string{
			"add":    "/api/domains/add",
			"remove": "/api/domains/remove",
			"list":   "/api/domains",       // Added list endpoint for domains
			"count":  "/api/domains/count", // Added count endpoint for domains
		}),
		createCRUDCommand("ip", "ips", map[string]string{
			"add":    "/api/ip",
			"remove": "/api/ip/remove",
			"list":   "/api/ip/list",
			"count":  "/api/ip/count", // Added count endpoint for ips
		}),
		createCRUDCommand("asn", "asns", map[string]string{
			"add":    "/api/asn/add",
			"remove": "/api/asn/remove",
			"list":   "/api/asn/list",
			"count":  "/api/asn/count", // Added count endpoint for asns
		}),
		createScopeCommand(),
	)

	return companyCmd
}

// Generic CRUD command creator
func createCRUDCommand(name, dataKey string, endpoints map[string]string) *cobra.Command {
	emoji := getEmojiForResource(name)
	cmd := &cobra.Command{
		Use:   name,
		Short: fmt.Sprintf("%s %s operations", emoji, strings.Title(name)),
		Example: fmt.Sprintf(`  # List %s
  bbrf company %s list -c acme

  # Add %s from arguments (with scope filtering for domains)
  bbrf company %s add domain1.com domain2.com -c acme

  # Add %s from file with verbose scope info
  bbrf company %s add @domains.txt -c acme --verbose-scope

  # Add %s from stdin
  cat domains.txt | bbrf company %s add - -c acme

  # Disable scope filtering for domains
  bbrf company %s add domain1.com -c acme --scope-filter=false`,
			name+"s", name, name+"s", name, name+"s", name, name+"s", name, name),
	}

	for action, endpoint := range endpoints {
		action, endpoint := action, endpoint // capture loop vars
		actionEmoji := getEmojiForAction(action)

		if action == "list" {
			cmd.AddCommand(&cobra.Command{
				Use:     action,
				Short:   fmt.Sprintf("%s List %s", actionEmoji, name+"s"),
				Example: fmt.Sprintf("  bbrf company %s list -c acme", name),
				Run: func(cmd *cobra.Command, args []string) {
					// fmt.Println(info(fmt.Sprintf("%s Listing %s for: %s", actionEmoji, name+"s", company)))
					call("GET", endpoint+"?company="+company, "")
				},
			})
		} else if action == "count" {
			cmd.AddCommand(&cobra.Command{
				Use:     action,
				Short:   fmt.Sprintf("🔢 Count %s", name+"s"),
				Example: fmt.Sprintf("  bbrf company %s count -c acme", name),
				Run: func(cmd *cobra.Command, args []string) {
					fmt.Println(info(fmt.Sprintf("📊 Counting %s for: %s", name+"s", company)))
					call("GET", endpoint+"?company="+company, "")
				},
			})
		} else {
			cmd.AddCommand(&cobra.Command{
				Use:   fmt.Sprintf("%s [items...]", action),
				Short: fmt.Sprintf("%s %s %s", actionEmoji, strings.Title(action), name+"s"),
				Long: fmt.Sprintf(`%s %s %s. Supports:
%s Direct: %s %s item1 item2
%s Stdin: echo 'item' | bbrf company %s %s -
%s File: bbrf company %s %s @file.txt

Scope Filtering (for domains):
%s Enabled by default with --scope-filter=true
%s Use --allow-out-of-scope to include out-of-scope domains
%s Use --verbose-scope for detailed filtering info`,
					actionEmoji, strings.Title(action), name+"s",
					info("•"), name, action,
					info("•"), name, action,
					info("•"), name, action,
					info("•"), info("•"), info("•")),
				Example: fmt.Sprintf(`  # %s %s directly
  bbrf company %s %s item1 item2 -c acme

  # %s %s from file  
  bbrf company %s %s @items.txt -c acme

  # %s %s from stdin
  cat items.txt | bbrf company %s %s - -c acme

  # %s %s with verbose scope filtering (domains only)
  bbrf company %s %s @items.txt -c acme --verbose-scope`,
					strings.Title(action), name+"s", name, action,
					strings.Title(action), name+"s", name, action,
					strings.Title(action), name+"s", name, action,
					strings.Title(action), name+"s", name, action),
				Run: func(cmd *cobra.Command, args []string) {
					if enableScopeFilter && name == "domain" {
						fmt.Printf("%s %s %s for: %s (with scope filtering)\n", info(actionEmoji), info(strings.Title(action)), info(name+"s"), info(company))
					} else {
						fmt.Printf("%s %s %s for: %s\n", info(actionEmoji), info(strings.Title(action)), info(name+"s"), info(company))
					}
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
		Short: "🎯 Scope management",
		Example: `  # Add domains to in-scope
  bbrf company scope inscope example.com sub.example.com -c acme

  # Add domains to out-of-scope from file
  bbrf company scope outscope @outscope.txt -c acme

  # Show in-scope domains
  bbrf company scope show in -c acme

  # Show out-of-scope domains  
  bbrf company scope show out -c acme`,
	}

	scopeActions := map[string]struct {
		endpoint string
		short    string
		emoji    string
	}{
		"inscope":         {"/api/scope/in", "Add in-scope domains", "✅"},
		"outscope":        {"/api/scope/out", "Add out-of-scope domains", "❌"},
		"remove-inscope":  {"/api/scope/remove", "Remove in-scope domains", "🗑️"},
		"remove-outscope": {"/api/scope/remove", "Remove out-of-scope domains", "🗑️"},
	}

	// Add input commands
	for action, config := range scopeActions {
		action, config := action, config // capture loop vars
		scopeCmd.AddCommand(&cobra.Command{
			Use:   fmt.Sprintf("%s [domains...]", action),
			Short: fmt.Sprintf("%s %s", config.emoji, config.short),
			Example: fmt.Sprintf(`  # %s directly
  bbrf company scope %s domain1.com domain2.com -c acme

  # %s from file
  bbrf company scope %s @domains.txt -c acme

  # %s from stdin
  cat domains.txt | bbrf company scope %s - -c acme`,
				config.short, action, config.short, action, config.short, action),
			Run: func(cmd *cobra.Command, args []string) {
				if (action == "outscope" || action == "inscope" || action == "remove-outscope" || action == "remove-inscope") &&
					!cmd.Flags().Changed("scope-filter") {
					enableScopeFilter = false
				}

				fmt.Println(info(fmt.Sprintf("%s %s for: %s", config.emoji, config.short, company)))
				handleInputAndPost(config.endpoint, company, "domains", args)
			},
		})
	}

	// Add show command
	scopeCmd.AddCommand(&cobra.Command{
		Use:   "show <in|out>",
		Short: "👁️  Show scope domains",
		Example: `  # Show in-scope domains
  bbrf company scope show in --company=acme

  # Show out-of-scope domains
  bbrf company scope show out -c acme`,
		Args: cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			scopeType := args[0]
			if scopeType != "in" && scopeType != "out" {
				fmt.Println(errorC("❌ Scope type must be 'in' or 'out'"))
				os.Exit(1)
			}
			emoji := "✅"
			if scopeType == "out" {
				emoji = "❌"
			}
			fmt.Println(info(fmt.Sprintf("%s Showing %s-scope domains for: %s", emoji, scopeType, company)))
			call("GET", fmt.Sprintf("/api/scope/show?company=%s&type=%s", company, scopeType), "")
		},
	})

	// Add test command for debugging scope filtering
	scopeCmd.AddCommand(&cobra.Command{
		Use:   "test <domain>",
		Short: "🧪 Test if a domain matches scope rules",
		Example: `  # Test if a domain is in scope
  bbrf company scope test example.com -c acme

  # Test multiple domains
  bbrf company scope test example.com sub.example.com -c acme`,
		Args: cobra.MinimumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println(info(fmt.Sprintf("🧪 Testing scope for company: %s", company)))

			scopeManager := NewScopeManager(company)
			err := scopeManager.LoadScope()
			if err != nil {
				fmt.Println(errorC("❌ Failed to load scope rules: " + err.Error()))
				return
			}

			fmt.Printf("%s Loaded %d in-scope and %d out-of-scope patterns\n",
				info("ℹ️"), len(scopeManager.InScope), len(scopeManager.OutScope))

			if len(scopeManager.InScope) > 0 {
				fmt.Printf("%s In-scope patterns: %v\n", info("📋"), scopeManager.InScope)
			}
			if len(scopeManager.OutScope) > 0 {
				fmt.Printf("%s Out-of-scope patterns: %v\n", info("📋"), scopeManager.OutScope)
			}

			fmt.Println()

			for _, domain := range args {
				shouldAccept, reason := scopeManager.ShouldAcceptDomain(domain)
				if shouldAccept {
					fmt.Printf("%s %s - %s\n", success("✅ ACCEPT:"), domainClr(domain), info(reason))
				} else {
					fmt.Printf("%s %s - %s\n", errorC("❌ REJECT:"), domainClr(domain), warning(reason))
				}
			}
		},
	})

	return scopeCmd
}

// Scope Manager Methods
func NewScopeManager(company string) *ScopeManager {
	return &ScopeManager{
		InScope:  make([]string, 0),
		OutScope: make([]string, 0),
		company:  company,
	}
}

// LoadScope loads scope rules from the server
func (sm *ScopeManager) LoadScope() error {
	// Load in-scope patterns
	inscope, err := sm.fetchScopeFromServer("in")
	if err == nil {
		sm.InScope = inscope
	}

	// Load out-scope patterns
	outscope, err := sm.fetchScopeFromServer("out")
	if err == nil {
		sm.OutScope = outscope
	}

	return nil
}

func (sm *ScopeManager) fetchScopeFromServer(scopeType string) ([]string, error) {
	url := fmt.Sprintf("%s/api/scope/show?company=%s&type=%s", config.API, sm.company, scopeType)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+config.Token)
	resp, err := insecureClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respData, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != 200 {
		return []string{}, nil
	}

	// Handle empty response
	if len(strings.TrimSpace(string(respData))) == 0 {
		return []string{}, nil
	}

	var patterns []string

	// Try to parse as JSON first
	err = json.Unmarshal(respData, &patterns)
	if err != nil {
		// If JSON parsing fails, treat as plain text (split by lines)
		text := strings.TrimSpace(string(respData))
		if text != "" {
			// Split by newlines and filter empty lines
			lines := strings.Split(text, "\n")
			for _, line := range lines {
				line = strings.TrimSpace(line)
				if line != "" {
					patterns = append(patterns, line)
				}
			}
		}
	}

	return patterns, nil
}

// ShouldAcceptDomain determines if a domain should be accepted based on scope rules
func (sm *ScopeManager) ShouldAcceptDomain(domain string) (bool, string) {
	domain = strings.ToLower(strings.TrimSpace(domain))

	// First check if it's explicitly out of scope
	if sm.IsOutOfScope(domain) {
		return false, "domain matches out-of-scope pattern"
	}

	// Then check if it's in scope
	if sm.IsInScope(domain) {
		return true, "domain matches in-scope pattern"
	}

	// If no in-scope patterns are defined, default to accept
	if len(sm.InScope) == 0 {
		return true, "no in-scope patterns defined, accepting by default"
	}

	// Domain doesn't match any in-scope pattern
	return false, "domain does not match any in-scope pattern"
}

func (sm *ScopeManager) IsInScope(domain string) bool {
	for _, pattern := range sm.InScope {
		if sm.matchesPattern(domain, pattern) {
			return true
		}
	}
	return false
}

func (sm *ScopeManager) IsOutOfScope(domain string) bool {
	for _, pattern := range sm.OutScope {
		if sm.matchesPattern(domain, pattern) {
			return true
		}
	}
	return false
}

func (sm *ScopeManager) matchesPattern(domain, pattern string) bool {
	pattern = strings.ToLower(strings.TrimSpace(pattern))
	domain = strings.ToLower(strings.TrimSpace(domain))

	// Exact match
	if pattern == domain {
		return true
	}

	// Wildcard pattern matching
	if strings.Contains(pattern, "*") {
		return sm.matchesWildcard(domain, pattern)
	}

	// Subdomain matching (implicit wildcard)
	if strings.HasSuffix(domain, "."+pattern) {
		return true
	}

	return false
}

func (sm *ScopeManager) matchesWildcard(domain, pattern string) bool {
	// Handle *.example.com patterns
	if strings.HasPrefix(pattern, "*.") {
		suffix := pattern[2:] // Remove "*."
		return domain == suffix || strings.HasSuffix(domain, "."+suffix)
	}

	// Handle other wildcard patterns like sub.*.example.com
	// Convert wildcard pattern to regex
	escapedPattern := regexp.QuoteMeta(pattern)
	// Replace escaped \* with .* for regex matching
	regexPattern := strings.ReplaceAll(escapedPattern, "\\*", ".*")
	// Anchor the pattern
	regexPattern = "^" + regexPattern + "$"

	regex, err := regexp.Compile(regexPattern)
	if err != nil {
		// Fallback to simple glob matching
		matched, _ := filepath.Match(pattern, domain)
		return matched
	}

	return regex.MatchString(domain)
}

// Helper functions for emojis
func getEmojiForResource(resource string) string {
	switch resource {
	case "domain":
		return "🌐"
	case "ip":
		return "🖥️"
	case "asn":
		return "🏢"
	default:
		return "📋"
	}
}

func getEmojiForAction(action string) string {
	switch action {
	case "add":
		return "➕"
	case "remove":
		return "🗑️"
	case "list":
		return "📋"
	default:
		return "🔧"
	}
}

// Utility functions
func initConfigPath() {
	usr, err := user.Current()
	if err != nil {
		fmt.Println(errorC("❌ Failed to get user directory: " + err.Error()))
		os.Exit(1)
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
	fmt.Println(title("🔐 BBRF Login"))
	fmt.Println(info("Please enter your credentials:"))
	fmt.Println()

	reader := bufio.NewReader(os.Stdin)

	fmt.Print(prompt("🌐 API Server URL (e.g., https://localhost:8443): "))
	api, _ := reader.ReadString('\n')
	fmt.Print(prompt("👤 Username: "))
	username, _ := reader.ReadString('\n')
	fmt.Print(prompt("🔑 Password: "))
	password, _ := reader.ReadString('\n')

	api = strings.TrimSpace(strings.ReplaceAll(api, " ", ""))
	username = strings.TrimSpace(username)
	password = strings.TrimSpace(password)

	fmt.Println(info("\n🔄 Authenticating..."))

	body := fmt.Sprintf(`{"username":"%s","password":"%s"}`, username, password)
	resp, err := insecureClient.Post(api+"/login", "application/json", bytes.NewBuffer([]byte(body)))
	if err != nil {
		fmt.Println(errorC("❌ Login failed: " + err.Error()))
		os.Exit(1)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		respData, _ := io.ReadAll(resp.Body)
		fmt.Println(errorC("❌ Login error: " + string(respData)))
		os.Exit(1)
	}

	respData, _ := io.ReadAll(resp.Body)
	var result map[string]string
	json.Unmarshal(respData, &result)

	config = Config{Token: result["token"], API: api}
	data, _ := json.Marshal(config)
	os.WriteFile(configPath, data, 0600)
	fmt.Println(success("✅ Login successful and token saved!"))
}

func handleInputAndPost(path, company, key string, args []string) {
	if len(args) < 1 {
		fmt.Println(errorC("❌ No input provided"))
		os.Exit(1)
	}

	var value string
	switch {
	case args[0] == "-":
		// From stdin
		fmt.Printf("%s Reading from stdin...\n", info("📥"))
		input, _ := io.ReadAll(os.Stdin)
		value = string(input)
	case strings.HasPrefix(args[0], "@") || strings.HasSuffix(args[0], ".txt"):
		// From file
		filePath := strings.TrimPrefix(args[0], "@")
		fmt.Printf("%s Reading from file: %s\n", info("📁"), filePath)
		content, err := os.ReadFile(filePath)
		if err != nil {
			fmt.Printf("%s Failed to read file: %s\n", errorC("❌"), err.Error())
			os.Exit(1)
		}
		value = string(content)
	default:
		// Direct arguments
		fmt.Printf("%s Processing direct arguments...\n", info("📝"))
		value = strings.Join(args, " ")
	}

	// Apply scope filtering for domain operations
	if enableScopeFilter && !allowOutOfScope && key == "domains" {
		// fmt.Printf("%s Applying scope filtering...\n", info("🔍"))
		// originalValue := value
		value = filterDomainsBeforePost(company, value)

		// If all domains were filtered out, don't make the API call
		if strings.TrimSpace(value) == "" {
			fmt.Printf("%s All domains were filtered out. No API call will be made.\n", warning("⚠️"))
			return
		}

		// if originalValue != value {
		// 	fmt.Printf("%s Scope filtering applied successfully\n", info("✅"))
		// }
	} else if key == "domains" {
		fmt.Printf("%s Scope filtering is DISABLED or bypassed\n", warning("⚠️"))
		if !enableScopeFilter {
			fmt.Printf("   - Reason: scope-filter flag is false\n")
		}
		if allowOutOfScope {
			fmt.Printf("   - Reason: allow-out-of-scope flag is true\n")
		}
	}

	body := map[string]string{"company": company, key: value}
	jsonBody, _ := json.Marshal(body)

	call("POST", path, string(jsonBody))
}

func filterDomainsBeforePost(company, domainsInput string) string {
	if verboseScope {
		fmt.Printf("%s Loading scope rules for filtering...\n", info("🔍"))
	}

	scopeManager := NewScopeManager(company)
	err := scopeManager.LoadScope()
	if err != nil {
		if verboseScope {
			fmt.Printf("%s Could not load scope rules, proceeding without filtering\n", warning("⚠️"))
		}
		return domainsInput
	}

	if verboseScope {
		fmt.Printf("%s Loaded %d in-scope and %d out-of-scope patterns\n",
			info("ℹ️"), len(scopeManager.InScope), len(scopeManager.OutScope))
		if len(scopeManager.InScope) > 0 {
			fmt.Printf("%s In-scope patterns: %v\n", info("ℹ️"), scopeManager.InScope)
		}
		if len(scopeManager.OutScope) > 0 {
			fmt.Printf("%s Out-of-scope patterns: %v\n", info("ℹ️"), scopeManager.OutScope)
		}
	}

	// Parse domains from input (handle newlines and spaces)
	allInput := strings.ReplaceAll(domainsInput, "\n", " ")
	domains := strings.Fields(allInput)
	var acceptedDomains []string
	var rejectedCount int

	fmt.Printf("%s Processing %d domains for scope filtering...\n", info("ℹ️"), len(domains))

	for _, domain := range domains {
		domain = strings.TrimSpace(domain)
		if domain == "" {
			continue
		}

		// Extract domain from domain:ip format if present
		cleanDomain := strings.Split(domain, ":")[0]

		shouldAccept, reason := scopeManager.ShouldAcceptDomain(cleanDomain)
		if shouldAccept {
			acceptedDomains = append(acceptedDomains, domain)
			if verboseScope {
				fmt.Printf("%s %s - %s\n", success("✅ ACCEPTED:"), domainClr(domain), reason)
			}
		} else {
			rejectedCount++
			// fmt.Printf("%s %s - %s\n", errorC("❌ REJECTED:"), domainClr(domain), reason)
		}
	}

	// if rejectedCount > 0 {
	// 	fmt.Printf("%s %d domains filtered out due to scope rules\n",
	// 		warning("⚠️"), rejectedCount)
	// }

	if len(acceptedDomains) != len(domains) {
		fmt.Printf("%s %d/%d domains will be added\n",
			info("ℹ️"), len(acceptedDomains), len(domains))
	}

	if len(acceptedDomains) == 0 {
		fmt.Printf("%s No domains passed scope filtering! Nothing will be added.\n", warning("⚠️"))
		return ""
	}

	return strings.Join(acceptedDomains, " ")
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
		fmt.Println(errorC("❌ Failed to create request: " + err.Error()))
		os.Exit(1)
	}

	req.Header.Set("Authorization", "Bearer "+config.Token)
	resp, err := insecureClient.Do(req)
	if err != nil {
		fmt.Println(errorC("❌ Request failed: " + err.Error()))
		os.Exit(1)
	}
	defer resp.Body.Close()

	respData, _ := io.ReadAll(resp.Body)

	// Handle different response types with styling
	if resp.StatusCode >= 400 {
		fmt.Println(errorC("❌ API Error: " + string(respData)))
		return
	}

	// Special handling for company list
	if strings.HasSuffix(path, "/api/company/list") {
		var companies []string
		if err := json.Unmarshal(respData, &companies); err != nil {
			fmt.Println(errorC("❌ Failed to parse company list: " + err.Error()))
			return
		}

		fmt.Println(header(" 🏢 Companies "))
		for i, c := range companies {
			fmt.Printf("%s %s\n",
				warning(fmt.Sprintf("%d.", i+1)),
				domainClr(c))
		}
		fmt.Println(count(fmt.Sprintf("\n📊 Total: %d companies", len(companies))))
		return
	}

	// Try to parse as JSON for better formatting
	var jsonData interface{}
	if err := json.Unmarshal(respData, &jsonData); err == nil {
		// If it's a simple string or number, display it directly
		switch v := jsonData.(type) {
		case string:
			fmt.Println(data(v))
		case float64:
			fmt.Println(count(fmt.Sprintf("📊 Count: %.0f", v)))
		case []interface{}:
			fmt.Println(header(" 📋 Results "))
			for i, item := range v {
				fmt.Printf("%s %s\n",
					warning(fmt.Sprintf("%d.", i+1)),
					domainClr(fmt.Sprintf("%v", item)))
			}
			fmt.Println(count(fmt.Sprintf("\n📊 Total: %d items", len(v))))
		default:
			// Pretty print JSON with basic formatting
			prettyJSON, _ := json.MarshalIndent(jsonData, "", "  ")
			fmt.Println(data(string(prettyJSON)))
		}
	} else {
		// Raw output if not JSON
		fmt.Println(data(string(respData)))
	}
}
