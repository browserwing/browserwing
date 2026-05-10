<p align="center">
  <img width="600" alt="BrowserWing" src="https://raw.githubusercontent.com/browserwing/browserwing/main/docs/assets/banner.svg">
</p>

<p align="center">
  <img alt="Go" src="https://img.shields.io/badge/Go-1.21%2B-00ADD8?logo=go&logoColor=white" />
  <img alt="React" src="https://img.shields.io/badge/React-18-61DAFB?logo=react&logoColor=white" />
  <img alt="TypeScript" src="https://img.shields.io/badge/TypeScript-5-3178C6?logo=typescript&logoColor=white" />
  <img alt="Vite" src="https://img.shields.io/badge/Vite-5-646CFF?logo=vite&logoColor=white" />
  <img alt="pnpm" src="https://img.shields.io/badge/pnpm-9-F69220?logo=pnpm&logoColor=white" />
  <img alt="MCP" src="https://img.shields.io/badge/MCP-Model%20Context%20Protocol-7B61FF" />
  <img alt="npm" src="https://img.shields.io/npm/v/browserwing?color=CB3837&logo=npm" />
  <img alt="License" src="https://img.shields.io/github/license/browserwing/browserwing" />
  <img alt="Scripts" src="https://img.shields.io/badge/Built--in%20Scripts-78-brightgreen" />
</p>

<p align="center">
  English · <a href="./README.zh-CN.md">简体中文</a> · <a href="./README.ja.md">日本語</a> · <a href="./README.es.md">Español</a> · <a href="./README.pt.md">Português</a>
</p>

<p align="center"><a href="https://browserwing.com">browserwing.com</a></p>

> **Turn any website into structured data with one command.** Browser automation platform with 78 built-in scripts, CLI for AI agents, MCP & Skills protocol support, and a visual recorder.

https://github.com/user-attachments/assets/e5377892-4b88-433a-8620-43b38a2fb28f

```bash
# Install and try it in 30 seconds
npm install -g browserwing && browserwing --port 8080

# Get data from any supported site
browserwing run github-trending
browserwing run bilibili-hot
browserwing run hackernews-top | jq '.[0:5]'
```

## Highlights

**Native Browser Automation Platform with AI Integration**

- **Complete Browser Control**: 26+ HTTP API endpoints for full-featured browser automation
- **Built-in AI Agent**: Direct conversational interface for browser automation tasks
- **Universal AI Tool Integration**: Native MCP & Skills protocol support - compatible with any AI tool that supports these standards
- **Visual Script Recording**: Record browser actions, edit visually, and replay with precision
- **Flexible Export Options**: Convert recorded scripts to MCP commands or Skills files for AI tool integration
- **Intelligent Data Extraction**: LLM-powered semantic extraction supporting OpenAI, Claude, DeepSeek, and more
- **Session Management**: Robust cookie and storage handling for stable, authenticated browsing sessions

## Requirements

- Google Chrome or Chromium installed and accessible in your environment.

## Screenshots

<img width="600" alt="BrowserWing Homepage" src="https://raw.githubusercontent.com/browserwing/browserwing/main/docs/assets/screenshot_homepage.png">

### Turn Scripts Into Claude Skill

You can now combine any scripts into a SKILL.md.

<img width="600" alt="BrowserWing Skill" src="https://raw.githubusercontent.com/browserwing/browserwing/main/docs/assets/screenshot_skill.png">

## Quick Start

### Let Your AI Agent Install It For You

Simply send the following message to your AI agent (OpenClaw, Cursor, Claude Code, etc.):

> **"Help me install BrowserWing following https://raw.githubusercontent.com/browserwing/browserwing/main/INSTALL.md"**

The agent will read the guide and handle the installation, configuration, Chrome setup, and Skill integration automatically.

---

### Option A — Install via Package Manager (recommended)

**Using npm:**
```bash
npm install -g browserwing
browserwing --port 8080
```

**Using pnpm:**
```bash
pnpm add -g browserwing
browserwing --port 8080
```

The npm package automatically tests GitHub and Gitee mirrors during installation and selects the fastest one.

**⚠️ macOS Users:**  
If you encounter a "killed" error when running, fix it with:
```bash
xattr -d com.apple.quarantine $(which browserwing)
```
See the [macOS Installation Fix Guide](docs/MACOS_INSTALLATION_FIX.md) for details.

**Using Homebrew (macOS/Linux):**
```bash
# Coming soon
brew install browserwing
```

### Option B — One-Line Install Script

**Linux / macOS:**
```bash
curl -fsSL https://raw.githubusercontent.com/browserwing/browserwing/main/install.sh | bash
```

**Windows (PowerShell):**
```powershell
iwr -useb https://raw.githubusercontent.com/browserwing/browserwing/main/install.ps1 | iex
```

The script automatically:
- Detects your OS/architecture
- Tests GitHub and Gitee mirrors, selects the fastest one
- Downloads and extracts the binary
- Adds to PATH

**Then start BrowserWing:**
```bash
browserwing --port 8080
# Open http://localhost:8080 in your browser
```

**Note for users in China:** The installation script automatically uses Gitee mirror if GitHub is slow.

### Option C — Manual Download

Download the prebuilt binary for your OS from [Releases](https://github.com/browserwing/browserwing/releases):

```bash
# Linux/macOS
chmod +x ./browserwing
./browserwing --port 8080

# Windows (PowerShell)
./browserwing.exe --port 8080
```

### Option D — Build from Source

```bash
# Install deps (Go + pnpm required)
make install

# Build integrated binary (frontend embedded)
make build-embedded
./build/browserwing --port 8080

# Or build all targets and packages
make build-all
make package
```

## Quick Integration with AI Tools

**Three Ways to Use BrowserWing:**

### 1. MCP Server Integration

Configure BrowserWing as an MCP server in any MCP-compatible AI tool:

```json
{
  "mcpServers": {
    "browserwing": {
      "type": "http",
      "url": "http://localhost:8080/api/v1/mcp/message"
    }
  }
}
```

Paste this configuration into your AI tool's MCP settings to enable browser automation capabilities.

### 2. Skills File Integration

Download and import the Skills file into any AI tool that supports the Skills protocol:

1. Start BrowserWing
2. Download [SKILL.md](https://raw.githubusercontent.com/browserwing/browserwing/refs/heads/main/SKILL.md) from the repository
3. Import into your AI tool's Skills settings
4. Start automating with natural language commands

**Example:**
```
"Navigate to example.com, search for 'AI tools', and extract the top 5 results"
```

### 3. CLI Mode (Agent-Friendly)

BrowserWing includes a CLI designed for AI agents and shell pipelines. Run scripts and get structured data directly from the terminal — no browser window needed (headless by default).

```bash
# List all available scripts
browserwing ls --format=json

# Run a script and get JSON output
browserwing run bilibili-hot
browserwing run github-trending
browserwing run hackernews-top

# Pass parameters to scripts
browserwing run jd-search --keyword="机械键盘"

# Pipe to other tools
browserwing run sinafinance-rank --format=csv > stocks.csv
browserwing run hackernews-top | jq '.[0:5]'

# Show browser window for debugging
browserwing run zhihu-hot --no-headless
```

**78 built-in scripts** across 10 categories (Bilibili, GitHub, Reddit, Hacker News, YouTube, Steam, BBC, Bloomberg, Reuters, Google Scholar, Binance, Amazon, Weibo, Zhihu, CNKI, Yahoo Finance, etc.) — ready to use out of the box.

**AI Agent Workflow:**
1. `browserwing ls --format=json` — discover available scripts and their parameters
2. `browserwing run <name> --key=value` — execute and get structured JSON data
3. Parse the output for further processing

### 4. Direct AI Agent Interface

Use BrowserWing's built-in AI Agent for immediate browser automation:

1. Open BrowserWing web interface at `http://localhost:8080`
2. Navigate to "AI Agent" section
3. Configure your LLM (OpenAI, Claude, DeepSeek, etc.)
4. Start conversational browser automation

**Export Custom Scripts:**
```bash
# Export your recorded scripts as Skills or MCP commands
curl -X POST 'http://localhost:8080/api/v1/scripts/export/skill' \
  -H 'Content-Type: application/json' \
  -d '{"script_ids": []}' \
  -o MY_CUSTOM_SCRIPTS.md
```

## Why BrowserWing

**Professional Browser Automation with AI Integration**

- **Universal Protocol Support**: Native MCP & Skills implementation works with any compatible AI tool
- **Complete Automation API**: 26+ HTTP endpoints providing comprehensive browser control capabilities
- **Flexible Integration Options**: Use as MCP server, Skills file, or standalone AI Agent
- **Visual Workflow Builder**: Record, edit, and replay browser actions without writing code
- **Token-Efficient Design**: Optimized for LLM usage with fast performance and minimal token consumption
- **Production-Ready**: Stable session management, cookie handling, and error recovery
- **Extensible Architecture**: Convert recorded scripts to reusable MCP commands or Skills files
- **Multi-LLM Support**: Works with OpenAI, Anthropic, DeepSeek, and other providers
- **Enterprise Use Cases**: Data extraction, RPA, testing, monitoring, and agent-driven automation

## Usage Guide

### Getting Started in Three Steps

1. **Choose Integration Method**
   - Copy MCP server configuration for AI tool integration
   - Download Skills file for Skills-compatible AI tools
   - Or use built-in AI Agent for immediate access

2. **Configure Your AI Tool**
   - Import MCP configuration or Skills file into your preferred AI tool
   - Configure LLM settings (API keys, model selection)
   - Verify connection to BrowserWing

3. **Start Automating**
   - Control browser through natural language commands
   - Record custom scripts for repeated tasks
   - Export scripts as MCP commands or Skills for reuse

### Advanced Workflows

**For Browser Automation:**
- Launch and manage multiple browser instances
- Configure profiles, proxies, and browser settings
- Handle cookies and authentication sessions
- Execute complex interaction sequences

**For Script Recording:**
- Capture clicks, inputs, navigation, and waits
- Edit actions visually in the script editor
- Test and debug with step-by-step replay
- Add variables and conditional logic

**For AI Integration:**
- Convert scripts to MCP commands or Skills files
- Integrate with multiple LLM providers
- Use semantic extraction for data parsing
- Build agent-driven automation workflows

### Using CloakBrowser for Advanced Stealth

BrowserWing supports [CloakBrowser](https://github.com/CloakHQ/CloakBrowser) — a stealth Chromium binary with source-level fingerprint patches that passes all major bot detection tests.

**Why CloakBrowser?**
- 49-57 C++ source-level patches (canvas, WebGL, audio, fonts, GPU, WebRTC, etc.)
- Passes Cloudflare Turnstile, FingerprintJS, BrowserScan, reCAPTCHA v3
- Works as a drop-in replacement for standard Chromium

**Setup:**

1. Install CloakBrowser:
```bash
pip install cloakbrowser
python -c "from cloakbrowser import ensure_binary; ensure_binary()"
```

2. Launch CloakBrowser's `cloakserve` CDP server:
```bash
python /path/to/cloakbrowser/bin/cloakserve --port=9222 --headless=false
```

3. Configure BrowserWing to use CloakBrowser's CDP endpoint. Add to your `config.toml`:
```toml
[browser]
ControlURL = "http://localhost:9222"
```

Or set the `BROWSER_CONTROL_URL` environment variable:
```bash
export BROWSER_CONTROL_URL="http://localhost:9222"
browserwing --port 8080
```

4. **Important:** CloakBrowser handles fingerprint spoofing at the binary level, so BrowserWing's JS-level stealth mode will conflict with CloakBrowser's C++ patches. When using CloakBrowser, set `use_stealth = false` in your browser config or the `USE_STEALTH` environment variable to disable BrowserWing's JS stealth injection.

**How it works:**
- `cloakserve` is a CDP multiplexer that spawns one Chrome process per fingerprint seed
- Each Chrome process has compiled-in fingerprint patches (no JS injection needed)
- BrowserWing connects to `cloakserve` via HTTP, resolves the WebSocket URL automatically, and controls the browser via standard CDP
- Cookie persistence and session management work the same as with regular Chrome

**Key benefits:**
- Stronger bot detection evasion than JS-level stealth injection
- Consistent fingerprints across sessions (seed-based)
- Native SOCKS5 proxy and geoip-based timezone/locale detection built-in
- Works with Playwright, Puppeteer, LangChain, browser-use, and BrowserWing

### HTTP API Reference

BrowserWing exposes 26+ RESTful endpoints for programmatic browser control:

**Navigation & Control**
- Navigate to URLs, go back/forward, refresh pages
- Manage browser windows and tabs
- Handle page loading and timeouts

**Element Interaction**
- Click, type, select, and hover actions
- File uploads and form submissions
- Keyboard shortcuts and key presses

**Data Extraction**
- Extract text, HTML, and attributes
- Semantic content analysis with LLM
- Screenshot capture (full page or element)

**Advanced Operations**
- Execute custom JavaScript
- Manage cookies and local storage
- Batch operations for efficiency
- Wait conditions and element visibility

**Complete Documentation**: See `docs/EXECUTOR_HTTP_API.md` for detailed endpoint specifications

### CLI Exit Codes

For AI agent integration, the CLI uses structured exit codes:

| Code | Meaning |
|------|---------|
| 0 | Success |
| 1 | General error |
| 2 | Server connection error |
| 3 | Script not found |
| 4 | Script execution failed |
| 64 | Bad arguments |

Set `BROWSERWING_JSON_ERRORS=1` for machine-readable JSON error output on stderr.

## Star History

[![Star History Chart](https://api.star-history.com/svg?repos=browserwing/browserwing&type=Date)](https://star-history.com/#browserwing/browserwing&Date)

## Contributing

- Issues and PRs are welcome. Please include clear steps to reproduce or a concise rationale.
- For feature ideas, open a discussion with use cases and expected outcomes.

## Community

Discord: [https://discord.gg/BkqcApRj](https://discord.gg/BkqcApRj)
twitter: [https://x.com/chg80333](https://x.com/chg80333)

## Acknowledgements

- Inspired by modern browser automation, agentic workflows, and MCP.

## License

- MIT License. See `LICENSE`.

## Disclaimer

- Do not use for illegal purposes or to violate site terms.
- Intended for personal learning and legitimate automation only.
