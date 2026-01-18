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
</p>

<p align="center">
  English · <a href="./README.zh-CN.md">简体中文</a> · <a href="./README.ja.md">日本語</a> · <a href="./README.es.md">Español</a> · <a href="./README.pt.md">Português</a>
</p>

<p align="center"><a href="https://browserwing.com">browserwing.com</a></p>


https://github.com/user-attachments/assets/7018126f-01c8-468f-a30d-3ca36f769876


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

## Quick Start

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

### 3. Direct AI Agent Interface

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
