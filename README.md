# 🧠 BBRF Client - Bug Bounty Reconnaissance Framework CLI

A powerful command-line client for securely interacting with BBRF Server backends over **HTTPS with JWT authentication**. Manage your bug bounty reconnaissance data with ease.

---

## 🚀 Features

- 🔐 **Secure Authentication**: JWT-based authentication with HTTPS
- 🏢 **Multi-Company Support**: Organize data by company/program
- 🌐 **Domain Management**: Add, remove, and query domains
- 🎯 **Scope Management**: Track in-scope and out-of-scope targets
- 🌍 **Network Intelligence**: Manage IP addresses and ASNs
- 📊 **Search & Analytics**: Query and count your reconnaissance data
- 📦 **Flexible Input**: Support for direct input, stdin, and file imports
- 🔒 **TLS Support**: Works with self-signed certificates

---

## 📦 Installation

### Option 1: Install with Go (Recommended)

```bash
go install github.com/Hadiasemi/bbrf@latest
```

Make sure `$GOPATH/bin` is in your `PATH`:

```bash
export PATH="$PATH:$(go env GOPATH)/bin"
```

### Option 2: Build from Source

```bash
git clone https://github.com/Hadiasemi/bbrf.git
cd bbrf
go build -o bbrf main.go
```

---

## 🔧 Initial Setup

### 1. Login to BBRF Server

```bash
bbrf login
```

You'll be prompted for:
- **API Server URL** (e.g., `https://localhost:8443`)
- **Username**
- **Password**

Configuration is saved to `~/.bbrf/config.json`

### 2. Create Your First Company

```bash
bbrf company add -c mycompany
# or
bbrf -c mycompany company add
```

---

## 📚 Usage

The CLI follows this pattern:
```bash
bbrf [-c <name>] <command> [subcommand] [args...]
```

**Note**: Most commands require a company context. Use `--company` flag or set it as an argument.

---

## 🏢 Company Management

### List All Companies
```bash
bbrf companies
```

### Add New Company
```bash
bbrf -c tesla company add
```

---

## 🌐 Domain Operations

### Add Domains
```bash
# Direct input
bbrf company -c tesla domain add tesla.com shop.tesla.com

# From stdin
cat domains.txt | bbrf company -c tesla domain add -

# From file
bbrf company -c tesla domain add @domains.txt
```

### Remove Domains
```bash
bbrf company -c tesla domain remove old.tesla.com
bbrf company -c tesla domain remove @old_domains.txt
```

### List All Domains
```bash
bbrf company -c tesla domains
```

### Count Domains
```bash
bbrf company -c tesla count
```

### Search Domains
```bash
# Show matching domains
bbrf company -c tesla show "*.tesla.com"

# Get count only
bbrf company -c tesla show "api.*" count
```

---

## 🎯 Scope Management

### Add In-Scope Domains
```bash
bbrf company -t tesla scope inscope tesla.com "*.tesla.com"
bbrf company -t tesla scope inscope @inscope_domains.txt
```

### Add Out-of-Scope Domains
```bash
bbrf company -c tesla scope outscope "*.internal.tesla.com"
echo "dev.tesla.com" | bbrf company -c tesla scope outscope -
```

### Remove from Scope
```bash
bbrf company -c tesla scope remove-inscope old.tesla.com
bbrf company -c tesla scope remove-outscope temp.tesla.com
```

### View Scope
```bash
# Show in-scope domains
bbrf company -c tesla scope show in

# Show out-of-scope domains
bbrf company -c tesla scope show out
```

---

## 🌍 IP Address Management

### Add IP Addresses
```bash
bbrf company -c tesla ip add 1.2.3.4 5.6.7.8
cat ips.txt | bbrf company -c tesla ip add -
```

### Remove IP Addresses
```bash
bbrf company -c tesla ip remove 1.2.3.4
bbrf company -c tesla ip remove @old_ips.txt
```

### List IP Addresses
```bash
bbrf company -c tesla ip list
```

---

## 🔢 ASN Management

### Add ASNs
```bash
bbrf company -c tesla asn add AS13335 AS15169
echo "AS8075" | bbrf company -c tesla asn add -
```

### Remove ASNs
```bash
bbrf company -c tesla asn remove AS13335
bbrf company -c tesla asn remove @old_asns.txt
```

### List ASNs
```bash
bbrf company -c tesla asn list
```

---

## 📝 Input Methods

The CLI supports three input methods for most commands:

### 1. Direct Arguments
```bash
bbrf company -c tesla domain add tesla.com shop.tesla.com
```

### 2. Standard Input (stdin)
```bash
cat domains.txt | bbrf company -c tesla domain add -
subfinder -d tesla.com | bbrf company -c tesla domain add -
```

### 3. File Input
```bash
bbrf company -c tesla domain add @domains.txt
bbrf company -c tesla scope inscope @inscope_list.txt
```

---

## 🔄 Common Workflows

### Initial Reconnaissance Setup
```bash
# Login and setup
bbrf login
bbrf company -c tesla company add

# Add root domains
bbrf company -c tesla domain add tesla.com

# Set scope
bbrf company -c tesla scope inscope "*.tesla.com"
bbrf company -c tesla scope outscope "*.internal.tesla.com"
```

### Subdomain Enumeration Integration
```bash
# Using subfinder
subfinder -d tesla.com | bbrf company -c tesla domain add -

# Using amass
amass enum -d tesla.com | bbrf company -c tesla domain add -

# From multiple tools
cat subdomains_*.txt | sort -u | bbrf company -c tesla domain add -
```

### Network Intelligence
```bash
# Add discovered IPs
nmap -sL tesla.com/24 | grep -oE '\b([0-9]{1,3}\.){3}[0-9]{1,3}\b' | bbrf company -c tesla ip add -

# Add ASNs
whois tesla.com | grep -i "OriginAS" | bbrf --company tesla asn add -
```

---

## 📊 Command Reference

| Category | Command | Description |
|----------|---------|-------------|
| **Auth** | `login` | Authenticate with BBRF server |
| **Company** | `companies` | List all companies |
| | `company add` | Create new company |
| **Domains** | `domain add [items...]` | Add domains |
| | `domain remove [items...]` | Remove domains |
| | `domain list` | List all domains |
| | `domain count` | Count total domains |
| | `show <query> [count]` | Search domains |
| **Scope** | `scope inscope [domains...]` | Add in-scope domains |
| | `scope outscope [domains...]` | Add out-of-scope domains |
| | `scope remove-inscope [domains...]` | Remove from in-scope |
| | `scope remove-outscope [domains...]` | Remove from out-of-scope |
| | `scope show <in\|out>` | Display scope domains |
| **Network** | `ip add [ips...]` | Add IP addresses |
| | `ip remove [ips...]` | Remove IP addresses |
| | `ip list` | List IP addresses |
| | `ip count` | Count IP addresses |
| | `asn add [asns...]` | Add ASNs |
| | `asn remove [asns...]` | Remove ASNs |
| | `asn list` | List ASNs |
| | `asn count` | Count ASNs |

---

## 🔧 Configuration

### Config File Location
```
~/.bbrf/config.json
```

### Config Structure
```json
{
  "token": "your-jwt-token",
  "api": "https://your-bbrf-server:8443"
}
```

---

## ⚠️ Security Considerations

- **TLS Verification**: The client uses `InsecureSkipVerify: true` for self-signed certificates
- **Token Storage**: JWT tokens are stored in `~/.bbrf/config.json` with `0600` permissions
- **HTTPS Only**: All API communication is encrypted over HTTPS
- **Production Use**: Consider proper certificate management for production environments

---

**Certificate Errors**
- The client accepts self-signed certificates by default
- Ensure your BBRF server is running with HTTPS

**File Not Found**
```bash
# Use @ prefix for files
bbrf --company tesla domain add @domains.txt
```

---

## 🏗️ System Architecture

```
┌─────────────┐    HTTPS/JWT    ┌─────────────┐
│  BBRF CLI   │ ←──────────────→ │ BBRF Server │
└─────────────┘                 └─────────────┘
       │                               │
       ▼                               ▼
~/.bbrf/config.json            PostgreSQL Database
```

---

## 📈 Performance Tips

- **Batch Operations**: Use file input for large datasets
- **Piping**: Chain tools together with pipes for efficiency
- **Parallel Processing**: Run multiple instances for different companies

---


## 🙏 Credits

- Inspired by [honoki/bbrf-client](https://github.com/honoki/bbrf-client)
- Built with ❤️ for the bug bounty community
- Developed by [HadiAsemi](https://github.com/Hadiasemi)

---

---

*Happy Bug Hunting! 🐛🔍*
