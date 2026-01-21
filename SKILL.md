---
name: browserwing-executor
description: Control browser automation through HTTP API. Supports page navigation, element interaction (click, type, select), data extraction, semantic tree analysis, screenshot, JavaScript execution, and batch operations.
---

# BrowserWing Executor API

## Overview

BrowserWing Executor provides comprehensive browser automation capabilities through HTTP APIs. You can control browser navigation, interact with page elements, extract data, and analyze page structure.

**API Base URL:** `http://localhost:8080/api/v1/executor`, `host=localhost:8080`

## Core Capabilities

- **Page Navigation:** Navigate to URLs, go back/forward, reload
- **Element Interaction:** Click, type, select, hover on page elements
- **Data Extraction:** Extract text, attributes, values from elements
- **Semantic Analysis:** Get semantic tree to understand page structure
- **Advanced Operations:** Screenshot, JavaScript execution, keyboard input
- **Batch Processing:** Execute multiple operations in sequence

## API Endpoints

### 1. Discover Available Commands

**IMPORTANT:** Always call this endpoint first to see all available commands and their parameters.

```bash
curl -X GET 'http://{host}/api/v1/executor/help'
```

**Response:** Returns complete list of all commands with parameters, examples, and usage guidelines.

**Query specific command:**
```bash
curl -X GET 'http://{host}/api/v1/executor/help?command=extract'
```

### 2. Get Semantic Tree

**CRITICAL:** Always call this after navigation to understand page structure and get element indices.

```bash
curl -X GET 'http://{host}/api/v1/executor/semantic-tree'
```

**Response Example:**
```json
{
  "success": true,
  "tree_text": "Clickable Element [1]: Login Button\nInput Element [1]: Email\nInput Element [2]: Password"
}
```

**Use Cases:**
- Understand what interactive elements are on the page
- Get element indices for reliable identification
- See element labels and roles

### 3. Common Operations

#### Navigate to URL
```bash
curl -X POST 'http://{host}/api/v1/executor/navigate' \
  -H 'Content-Type: application/json' \
  -d '{"url": "https://example.com"}'
```

#### Click Element
```bash
curl -X POST 'http://{host}/api/v1/executor/click' \
  -H 'Content-Type: application/json' \
  -d '{"identifier": "[1]"}'
```
**Identifier formats:** `[1]`, `#button-id`, `.class-name`, `Login` (text), `Clickable Element [1]`

#### Type Text
```bash
curl -X POST 'http://{host}/api/v1/executor/type' \
  -H 'Content-Type: application/json' \
  -d '{"identifier": "Input Element [1]", "text": "user@example.com"}'
```

#### Extract Data
```bash
curl -X POST 'http://{host}/api/v1/executor/extract' \
  -H 'Content-Type: application/json' \
  -d '{
    "selector": ".product-item",
    "fields": ["text", "href"],
    "multiple": true
  }'
```

#### Wait for Element
```bash
curl -X POST 'http://{host}/api/v1/executor/wait' \
  -H 'Content-Type: application/json' \
  -d '{"identifier": ".loading", "state": "hidden", "timeout": 10}'
```

#### Batch Operations
```bash
curl -X POST 'http://{host}/api/v1/executor/batch' \
  -H 'Content-Type: application/json' \
  -d '{
    "operations": [
      {"type": "navigate", "params": {"url": "https://example.com"}, "stop_on_error": true},
      {"type": "click", "params": {"identifier": "[1]"}, "stop_on_error": true},
      {"type": "type", "params": {"identifier": "[1]", "text": "query"}, "stop_on_error": true}
    ]
  }'
```

## Instructions

**Step-by-step workflow:**

1. **Discover commands:** Call `GET /help` to see all available operations and their parameters (do this first if unsure).

2. **Navigate:** Use `POST /navigate` to open the target webpage.

3. **Analyze page:** Call `GET /semantic-tree` to understand page structure and get element indices.

4. **Interact:** Use element indices (like `[1]`, `Input Element [1]`) or CSS selectors to:
   - Click elements: `POST /click`
   - Input text: `POST /type`
   - Select options: `POST /select`
   - Wait for elements: `POST /wait`

5. **Extract data:** Use `POST /extract` to get information from the page.

6. **Present results:** Format and show extracted data to the user.

## Complete Example

**User Request:** "Search for 'laptop' on example.com and get the first 5 results"

**Your Actions:**

1. Navigate to search page:
```bash
curl -X POST 'http://{host}/api/v1/executor/navigate' \
  -H 'Content-Type: application/json' \
  -d '{"url": "https://example.com/search"}'
```

2. Get page structure to find search input:
```bash
curl -X GET 'http://{host}/api/v1/executor/semantic-tree'
```
Response shows: `Input Element [1]: Search Box`

3. Type search query:
```bash
curl -X POST 'http://{host}/api/v1/executor/type' \
  -H 'Content-Type: application/json' \
  -d '{"identifier": "Input Element [1]", "text": "laptop"}'
```

4. Press Enter to submit:
```bash
curl -X POST 'http://{host}/api/v1/executor/press-key' \
  -H 'Content-Type: application/json' \
  -d '{"key": "Enter"}'
```

5. Wait for results to load:
```bash
curl -X POST 'http://{host}/api/v1/executor/wait' \
  -H 'Content-Type: application/json' \
  -d '{"identifier": ".search-results", "state": "visible", "timeout": 10}'
```

6. Extract search results:
```bash
curl -X POST 'http://{host}/api/v1/executor/extract' \
  -H 'Content-Type: application/json' \
  -d '{
    "selector": ".result-item",
    "fields": ["text", "href"],
    "multiple": true
  }'
```

7. Present the extracted data:
```
Found 15 results for 'laptop':
1. Gaming Laptop - $1299 (https://...)
2. Business Laptop - $899 (https://...)
...
```

## Key Commands Reference

### Navigation
- `POST /navigate` - Navigate to URL
- `POST /go-back` - Go back in history
- `POST /go-forward` - Go forward in history
- `POST /reload` - Reload current page

### Element Interaction
- `POST /click` - Click element (supports: CSS selector, semantic index `[1]`, text content)
- `POST /type` - Type text into input (supports: `Input Element [1]`, CSS selector)
- `POST /select` - Select dropdown option
- `POST /hover` - Hover over element
- `POST /wait` - Wait for element state (visible, hidden, enabled)
- `POST /press-key` - Press keyboard key (Enter, Tab, Ctrl+S, etc.)

### Data Extraction
- `POST /extract` - Extract data from elements (supports multiple elements, custom fields)
- `POST /get-text` - Get element text content
- `POST /get-value` - Get input element value
- `GET /page-info` - Get page URL and title
- `GET /page-text` - Get all page text
- `GET /page-content` - Get full HTML

### Page Analysis
- `GET /semantic-tree` - Get semantic tree (⭐ **ALWAYS call after navigation**)
- `GET /clickable-elements` - Get all clickable elements
- `GET /input-elements` - Get all input elements

### Advanced
- `POST /screenshot` - Take page screenshot (base64 encoded)
- `POST /evaluate` - Execute JavaScript code
- `POST /batch` - Execute multiple operations in sequence
- `POST /scroll-to-bottom` - Scroll to page bottom
- `POST /resize` - Resize browser window

## Element Identification

You can identify elements using:

1. **Semantic Index (Recommended):** `[1]`, `[2]`, `Clickable Element [1]`, `Input Element [2]`
   - Most reliable method
   - Get indices from `/semantic-tree` endpoint
   - Example: `"identifier": "[1]"` or `"identifier": "Input Element [1]"`

2. **CSS Selector:** `#id`, `.class`, `button[type="submit"]`
   - Standard CSS selectors
   - Example: `"identifier": "#login-button"`

3. **Text Content:** `Login`, `Sign Up`, `Submit`
   - Searches buttons and links with matching text
   - Example: `"identifier": "Login"`

4. **XPath:** `//button[@id='login']`
   - XPath expressions
   - Example: `"identifier": "//button[@id='login']"`

5. **ARIA Label:** Elements with `aria-label` attribute
   - Automatically searched

## Guidelines

**Before starting:**
- Call `GET /help` if you're unsure about available commands or their parameters
- Ensure browser is started (if not, it will auto-start on first operation)

**During automation:**
- **Always call `/semantic-tree` after navigation** to get page structure
- **Prefer semantic indices** (like `[1]`) over CSS selectors for reliability
- **Use `/wait`** for dynamic content that loads asynchronously
- **Check element states** before interaction (visible, enabled)
- **Use `/batch`** for multiple sequential operations to improve efficiency

**Error handling:**
- If operation fails, check element identifier and try different format
- For timeout errors, increase timeout value
- If element not found, call `/semantic-tree` again to refresh page structure
- Explain errors clearly to user with suggested solutions

**Data extraction:**
- Use `fields` parameter to specify what to extract: `["text", "href", "src"]`
- Set `multiple: true` to extract from multiple elements
- Format extracted data in a readable way for user

## Complete Workflow Example

**Scenario:** User wants to login to a website

```
User: "Please log in to example.com with username 'john' and password 'secret123'"
```

**Your Actions:**

**Step 1:** Navigate to login page
```bash
POST http://{host}/api/v1/executor/navigate
{"url": "https://example.com/login"}
```

**Step 2:** Get page structure
```bash
GET http://{host}/api/v1/executor/semantic-tree
```
Response:
```
Input Element [1]: Username
Input Element [2]: Password
Clickable Element [1]: Login Button
```

**Step 3:** Enter username
```bash
POST http://{host}/api/v1/executor/type
{"identifier": "Input Element [1]", "text": "john"}
```

**Step 4:** Enter password
```bash
POST http://{host}/api/v1/executor/type
{"identifier": "Input Element [2]", "text": "secret123"}
```

**Step 5:** Click login button
```bash
POST http://{host}/api/v1/executor/click
{"identifier": "Clickable Element [1]"}
```

**Step 6:** Wait for login success (optional)
```bash
POST http://{host}/api/v1/executor/wait
{"identifier": ".welcome-message", "state": "visible", "timeout": 10}
```

**Step 7:** Inform user
```
"Successfully logged in to example.com!"
```

## Batch Operation Example

**Scenario:** Fill out a form with multiple fields

Instead of making 5 separate API calls, use one batch operation:

```bash
curl -X POST 'http://{host}/api/v1/executor/batch' \
  -H 'Content-Type: application/json' \
  -d '{
    "operations": [
      {
        "type": "navigate",
        "params": {"url": "https://example.com/form"},
        "stop_on_error": true
      },
      {
        "type": "type",
        "params": {"identifier": "#name", "text": "John Doe"},
        "stop_on_error": true
      },
      {
        "type": "type",
        "params": {"identifier": "#email", "text": "john@example.com"},
        "stop_on_error": true
      },
      {
        "type": "select",
        "params": {"identifier": "#country", "value": "United States"},
        "stop_on_error": true
      },
      {
        "type": "click",
        "params": {"identifier": "#submit"},
        "stop_on_error": true
      }
    ]
  }'
```

## Best Practices

1. **Discovery first:** If unsure, call `/help` or `/help?command=<name>` to learn about commands
2. **Structure first:** Always call `/semantic-tree` after navigation to understand the page
3. **Use semantic indices:** They're more reliable than CSS selectors (elements might have dynamic classes)
4. **Wait for dynamic content:** Use `/wait` before interacting with elements that load asynchronously
5. **Batch when possible:** Use `/batch` for multiple sequential operations
6. **Handle errors gracefully:** Provide clear explanations and suggestions when operations fail
7. **Verify results:** After operations, check if desired outcome was achieved

## Common Scenarios

### Form Filling
1. Navigate to form page
2. Get semantic tree to find input elements
3. Use `/type` for each field: `Input Element [1]`, `Input Element [2]`, etc.
4. Use `/select` for dropdowns
5. Click submit button

### Data Scraping
1. Navigate to target page
2. Wait for content to load with `/wait`
3. Use `/extract` with CSS selector and `multiple: true`
4. Specify fields to extract: `["text", "href", "src"]`

### Search Operations
1. Navigate to search page
2. Get semantic tree to locate search input
3. Type search query into input
4. Press Enter or click search button
5. Wait for results
6. Extract results data

### Login Automation
1. Navigate to login page
2. Get semantic tree
3. Type username: `Input Element [1]`
4. Type password: `Input Element [2]`
5. Click login button: `Clickable Element [1]`
6. Wait for success indicator

## Important Notes

- Browser must be running (it will auto-start on first operation if needed)
- Operations are executed on the **currently active browser tab**
- Semantic tree updates after each navigation and click operation
- All timeouts are in seconds
- Use `wait_visible: true` (default) for reliable element interaction
- Replace `{host}` with actual API host address
- Authentication required: use `X-BrowserWing-Key` header or JWT token

## Troubleshooting

**Element not found:**
- Call `/semantic-tree` to see available elements
- Try different identifier format (semantic index, CSS selector, text)
- Check if page has finished loading

**Timeout errors:**
- Increase timeout value in request
- Check if element actually appears on page
- Use `/wait` with appropriate state before interaction

**Extraction returns empty:**
- Verify CSS selector matches target elements
- Check if content has loaded (use `/wait` first)
- Try different extraction fields or type

## Quick Reference

```bash
# Discover commands
GET http://{host}/api/v1/executor/help

# Navigate
POST http://{host}/api/v1/executor/navigate {"url": "..."}

# Get page structure
GET http://{host}/api/v1/executor/semantic-tree

# Click element
POST http://{host}/api/v1/executor/click {"identifier": "[1]"}

# Type text
POST http://{host}/api/v1/executor/type {"identifier": "[1]", "text": "..."}

# Extract data
POST http://{host}/api/v1/executor/extract {"selector": "...", "fields": [...], "multiple": true}
```

## Response Format

All operations return:
```json
{
  "success": true,
  "message": "Operation description",
  "timestamp": "2026-01-15T10:30:00Z",
  "data": {
    // Operation-specific data
  }
}
```

**Error response:**
```json
{
  "error": "error.operationFailed",
  "detail": "Detailed error message"
}
```
