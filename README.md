# 🧠 BBRF Client - Bug Bounty Recon Framework CLI

This is the **command-line client** for securely interacting with the [BBRF Server](https://github.com/honoki/bbrf-server)-style backend over **HTTPS and JWT authentication**.

---

## 🚀 Features

- 🔐 Secure JWT-authenticated API interaction
- 🌐 Add, list, and manage subdomains, scope, ASNs, IPs
- 📦 Add data via:
  - inline
  - stdin (`-`)
  - files (`@filename.txt`)
- 🗃️ Supports company-based separation of data

---

Perfect — here's a revised **📦 Installation** section that includes both methods: cloning and building manually **and** installing via `go install`:

---

### 📦 Installation

You can install `bbrf` in one of two ways:

#### ✅ Option 1: Install with `go install` (Recommended)

Make sure you have Go 1.17+ installed, then run:

```bash
go install github.com/Hadiasemi/bbrf@latest
```

This will install the `bbrf` binary to your `$GOPATH/bin` or `$HOME/go/bin`.
Make sure that directory is in your `PATH`:

```bash
export PATH="$PATH:$(go env GOPATH)/bin"
```

---

#### 🛠 Option 2: Manual Build

```bash
git clone https://github.com/Hadiasemi/bbrf.git
cd bbrf
go build -o bbrf bbrf.go
```

This builds the `bbrf` binary in the current directory.


## 🔧 Configuration

After the first login, config will be saved to:

```
~/.bbrf/config.json
```

This includes your token and API server URL.

---

## 🔐 Login

```bash
./bbrf login
```

You’ll be prompted for:

* API Server URL (e.g., `https://localhost:8443`)
* Username
* Password

---

## 📚 Usage

```bash
./bbrf <company> <command> [args...]
```

### Commands

| Command                  | Description               |
| ------------------------ | ------------------------- |
| `add-company`            | Register a new company    |
| `domain add`             | Add domain(s)             |
| `domains`                | List domains              |
| `count`                  | Count domains             |
| `show <query> [count]`   | Search domains            |
| `inscope "<domain>..."`  | Add in-scope domains      |
| `outscope "<domain>..."` | Add out-of-scope domains  |
| `scope in`               | Show in-scope domains     |
| `scope out`              | Show out-of-scope domains |
| `ip add <ip...>`         | Add IP addresses          |
| `ip list`                | List IP addresses         |
| `asn add <AS12345...>`   | Add ASN(s)                |
| `asn list`               | List ASN(s)               |

💡 You can pipe input via `-` or read from file using `@filename.txt`.

```bash
cat domains.txt | ./bbrf Tesla domain add -
./bbrf Tesla asn add @asns.txt
```

---

## 🧱 System Overview (ASCII)

```
                ┌────────────┐        HTTPS + JWT        ┌────────────┐
                │  bbrf CLI  │ ────────────────────────▶ │  BBRF API  │
                └────┬───────┘                            └────┬───────┘
                     │                                         │
                     ▼                                         ▼
          ~/.bbrf/config.json                    PostgreSQL (subdomains_db)
       (Token + API URL stored)                    + TLS Certs (auto)
```

---

## 🗄️ Database Overview (ASCII)

```
      ┌─────────────┐
      │  companies  │
      └────┬────────┘
           │
    ┌──────┴──────┐
    ▼             ▼
subdomains     scope_domains
    │             │
    ▼             ▼
   asns          ips
```

---

## 💡 Example Session

```bash
./bbrf login
./bbrf Tesla add-company
./bbrf Tesla domain add tesla.com
./bbrf Tesla inscope "*.tesla.com"
./bbrf Tesla asn add AS12345
./bbrf Tesla domains
```

---

## ⚠️ TLS Note

This client uses `InsecureSkipVerify: true` to allow self-signed certs. Use with caution in production environments.

---

## ✅ Requirements

* Go 1.18+
* BBRF Server running with HTTPS

---


## 🧠 Credits

* Inspired by [honoki/bbrf-server](https://github.com/honoki/bbrf-server)
* Extended by \[HadiAsemi]

---

## 📜 License

MIT License


