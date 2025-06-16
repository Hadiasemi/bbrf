// File: main.go (BBRF CLI)
package main

import (
	"bufio"
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/user"
	"path/filepath"
	"strings"
)

type Config struct {
	Token string `json:"token"`
	API   string `json:"api"`
}

var configPath = ""
var config Config
var insecureClient = &http.Client{
	Transport: &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	},
}

func main() {
	initConfigPath()
	loadConfig()
	if len(os.Args) < 2 {
		printUsage()
		return
	}
	if os.Args[1] == "companies" {
		call("GET", "/api/company/list", "")
		return
	}

	switch os.Args[1] {
	case "login":
		doLogin()
	case "help":
		printUsage()
	default:
		if len(os.Args) < 3 {
			printUsage()
			return
		}
		company := os.Args[1]
		cmd := os.Args[2]
		args := os.Args[3:]
		handleCommand(company, cmd, args)
	}
}

func initConfigPath() {
	usr, err := user.Current()
	if err != nil {
		log.Fatal(err)
	}
	configPath = filepath.Join(usr.HomeDir, ".bbrf", "config.json")
	os.MkdirAll(filepath.Dir(configPath), 0700)
}

func loadConfig() {
	data, err := ioutil.ReadFile(configPath)
	if err == nil {
		_ = json.Unmarshal(data, &config)
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
		respData, _ := ioutil.ReadAll(resp.Body)
		log.Fatalf("Login error: %s", respData)
	}

	respData, _ := ioutil.ReadAll(resp.Body)
	var result map[string]string
	_ = json.Unmarshal(respData, &result)
	config = Config{Token: result["token"], API: api}
	data, _ := json.Marshal(config)
	ioutil.WriteFile(configPath, data, 0600)
	fmt.Println("Login successful and token saved.")
}

func printUsage() {
	fmt.Println("BBRF CLI - Usage:")
	fmt.Println("  bbrf login                                           Login to BBRF server and save token")
	fmt.Println("  bbrf companies                                       List all companies")
	fmt.Println("  bbrf <company> add-company                           Add a new company")

	fmt.Println("  bbrf <company> domain add example.com                Add one or more subdomains")
	fmt.Println("  echo 'x.com' | bbrf <company> domain add -           Add subdomains via stdin")
	fmt.Println("  bbrf <company> domain add @domains.txt               Add subdomains from a file")
	fmt.Println("  bbrf <company> domain remove example.com             Remove one or more subdomains")
	fmt.Println("  bbrf <company> domains                               List all subdomains")
	fmt.Println("  bbrf <company> count                                 Count subdomains")
	fmt.Println("  bbrf <company> show www.example.com [count]          Show matching subdomains (or count only)")

	fmt.Println("  bbrf <company> inscope \"*.a.com *.b.com\"             Add in-scope domains")
	fmt.Println("  bbrf <company> outscope \"blog.a.com test.b.co\"       Add out-of-scope domains")
	fmt.Println("  echo '*.x.com' | bbrf <company> inscope -            Add in-scope domains via stdin")
	fmt.Println("  bbrf <company> inscope @in.txt                       Add in-scope domains from file")
	fmt.Println("  bbrf <company> outscope @out.txt                     Add out-of-scope domains from file")
	fmt.Println("  bbrf <company> scope in                              Show in-scope domains")
	fmt.Println("  bbrf <company> scope out                             Show out-of-scope domains")
	fmt.Println("  bbrf <company> remove-inscope '*.a.com'              Remove in-scope domains")
	fmt.Println("  bbrf <company> remove-outscope 'blog.a.com'          Remove out-of-scope domains")
	fmt.Println("  bbrf <company> ip add 1.2.3.4                        Add IPs manually")
	fmt.Println("  cat ips.txt | bbrf <company> ip add -                Add IPs from stdin")
	fmt.Println("  bbrf <company> ip add @ips.txt                       Add IPs from file")
	fmt.Println("  bbrf <company> ip remove 1.2.3.4                     Remove IPs manually")
	fmt.Println("  bbrf <company> ip list                               List IPs")

	fmt.Println("  bbrf <company> asn add AS1234 AS5678                 Add ASN(s)")
	fmt.Println("  echo 'AS1234' | bbrf <company> asn add -             Add ASN(s) from stdin")
	fmt.Println("  bbrf <company> asn add @asns.txt                     Add ASN(s) from a text file")
	fmt.Println("  bbrf <company> asn remove AS1234                     Remove ASN(s)")
	fmt.Println("  bbrf <company> asn list                              List ASNs")

	fmt.Println("  bbrf <company> help                                  Show this help message")
}
func handleCommand(company, cmd string, args []string) {
	switch cmd {

	case "add-company":
		call("POST", "/api/company", fmt.Sprintf(`{"company":"%s"}`, company))

	case "inscope":
		handleInputAndPost("/api/scope/in", company, "domains", args)

	case "outscope":
		handleInputAndPost("/api/scope/out", company, "domains", args)

	case "domain":
		if len(args) >= 1 && args[0] == "add" {
			handleInputAndPost("/api/domains/add", company, "domains", args[1:])
		} else if len(args) >= 1 && args[0] == "remove" {
			handleInputAndPost("/api/domains/remove", company, "domains", args[1:])
		} else {
			printUsage()
		}

	case "ip":
		if len(args) >= 1 && args[0] == "add" {
			handleInputAndPost("/api/ip", company, "ips", args[1:])
		} else if len(args) >= 1 && args[0] == "remove" {
			handleInputAndPost("/api/ip/remove", company, "ips", args[1:])
		} else if len(args) == 1 && args[0] == "list" {
			call("GET", fmt.Sprintf("/api/ip/list?company=%s", company), "")
		} else {
			printUsage()
		}
	case "asn":
		if len(args) >= 1 && args[0] == "add" {
			handleInputAndPost("/api/asn/add", company, "asns", args[1:])
		} else if len(args) >= 1 && args[0] == "remove" {
			handleInputAndPost("/api/asn/remove", company, "asns", args[1:])
		} else if len(args) == 1 && args[0] == "list" {
			call("GET", "/api/asn/list?company="+company, "")
		} else {
			printUsage()
		}

	case "domains":
		call("GET", "/api/domains?company="+company, "")

	case "count":
		call("GET", "/api/domains/count?company="+company, "")

	case "show":
		if len(args) < 1 {
			printUsage()
			return
		}
		query := args[0]
		countFlag := "false"
		if len(args) > 1 && args[1] == "count" {
			countFlag = "true"
		}
		call("GET", fmt.Sprintf("/api/domains/show?company=%s&q=%s&count=%s", company, query, countFlag), "")

	case "scope":
		if len(args) == 1 && (args[0] == "in" || args[0] == "out") {
			call("GET", fmt.Sprintf("/api/scope/show?company=%s&type=%s", company, args[0]), "")
		} else {
			printUsage()
		}
	case "remove-inscope":
		handleInputAndPost("/api/scope/remove", company, "domains", args)
	case "remove-outscope":
		handleInputAndPost("/api/scope/remove", company, "domains", args)

	default:
		fmt.Println("Unknown command:", cmd)
		printUsage()
	}
}
func handleInputAndPost(path string, company string, key string, args []string) {
	if len(args) < 1 {
		printUsage()
		return
	}

	var value string

	// stdin
	if args[0] == "-" {
		input, _ := io.ReadAll(os.Stdin)
		value = string(input)

		// from file
	} else if strings.HasPrefix(args[0], "@") || strings.HasSuffix(args[0], ".txt") {
		filePath := strings.TrimPrefix(args[0], "@")
		content, err := os.ReadFile(filePath)
		if err != nil {
			log.Fatalf("Failed to read file: %v", err)
		}
		value = string(content)

		// inline
	} else {
		value = strings.Join(args, " ")
	}

	body := map[string]string{
		"company": company,
		key:       value,
	}
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
