#!/usr/bin/env node

/**
 * BrowserWing npm postinstall script
 * Downloads the appropriate binary for the current platform
 */

const https = require('https');
const http = require('http');
const fs = require('fs');
const path = require('path');
const { execSync } = require('child_process');
const os = require('os');

const REPO = 'browserwing/browserwing';
const VERSION = require('./package.json').version;
const TAG = `v${VERSION}`;

// ANSI color codes
const colors = {
  reset: '\x1b[0m',
  green: '\x1b[32m',
  yellow: '\x1b[33m',
  red: '\x1b[31m',
  cyan: '\x1b[36m'
};

function log(message, color = 'reset') {
  console.log(`${colors[color]}${message}${colors.reset}`);
}

function logInfo(message) {
  log(`[INFO] ${message}`, 'green');
}

function logWarning(message) {
  log(`[WARNING] ${message}`, 'yellow');
}

function logError(message) {
  log(`[ERROR] ${message}`, 'red');
}

function logDebug(message) {
  log(`[DEBUG] ${message}`, 'cyan');
}

// Detect platform and architecture
function detectPlatform() {
  const platform = os.platform();
  const arch = os.arch();

  let platformName;
  let archName;
  let isWindows = false;

  // Map platform
  switch (platform) {
    case 'darwin':
      platformName = 'darwin';
      break;
    case 'linux':
      platformName = 'linux';
      break;
    case 'win32':
      platformName = 'windows';
      isWindows = true;
      break;
    default:
      throw new Error(`Unsupported platform: ${platform}`);
  }

  // Map architecture
  switch (arch) {
    case 'x64':
      archName = 'amd64';
      break;
    case 'arm64':
      archName = 'arm64';
      break;
    default:
      throw new Error(`Unsupported architecture: ${arch}`);
  }

  return { platform: platformName, arch: archName, isWindows };
}

// Test download speed and select fastest mirror
async function selectMirror(archiveName) {
  logInfo('Testing download mirrors...');

  const githubUrl = `https://github.com/${REPO}/releases/download/${TAG}/${archiveName}`;
  const giteeUrl = `https://gitee.com/browserwing/browserwing/releases/download/${TAG}/${archiveName}`;

  // Test GitHub
  const githubTime = await testUrl(githubUrl);
  logDebug(`GitHub response time: ${githubTime}ms`);

  // Test Gitee
  const giteeTime = await testUrl(giteeUrl);
  logDebug(`Gitee response time: ${giteeTime}ms`);

  // Select fastest
  if (githubTime < giteeTime) {
    logInfo('Using GitHub mirror (faster)');
    return {
      name: 'github',
      url: `https://github.com/${REPO}/releases/download`
    };
  } else {
    logInfo('Using Gitee mirror (faster)');
    return {
      name: 'gitee',
      url: 'https://gitee.com/browserwing/browserwing/releases/download'
    };
  }
}

// Test URL response time
function testUrl(url) {
  return new Promise((resolve) => {
    const startTime = Date.now();
    const protocol = url.startsWith('https') ? https : http;
    
    const request = protocol.get(url, { method: 'HEAD', timeout: 5000 }, (res) => {
      const elapsed = Date.now() - startTime;
      request.abort();
      resolve(elapsed);
    });

    request.on('error', () => {
      resolve(999999); // Return large number on error
    });

    request.on('timeout', () => {
      request.abort();
      resolve(999999);
    });
  });
}

// Download file with retry and mirror fallback
async function downloadFile(url, dest, maxRetries = 3) {
  let mirror = await selectMirror(path.basename(dest));
  let currentUrl = `${mirror.url}/${TAG}/${path.basename(dest)}`;
  
  logInfo('Downloading BrowserWing...');
  logInfo(`Download URL: ${currentUrl}`);

  for (let i = 0; i < maxRetries; i++) {
    try {
      await downloadFileOnce(currentUrl, dest);
      logInfo('Download completed');
      return;
    } catch (error) {
      if (i < maxRetries - 1) {
        logWarning(`Download failed, retrying (${i + 1}/${maxRetries})...`);
        
        // Switch mirror
        if (mirror.name === 'github') {
          mirror.name = 'gitee';
          mirror.url = 'https://gitee.com/browserwing/browserwing/releases/download';
          logInfo('Switching to Gitee mirror...');
        } else {
          mirror.name = 'github';
          mirror.url = `https://github.com/${REPO}/releases/download`;
          logInfo('Switching to GitHub mirror...');
        }
        
        currentUrl = `${mirror.url}/${TAG}/${path.basename(dest)}`;
        await new Promise(resolve => setTimeout(resolve, 2000));
      } else {
        throw new Error(`Failed to download after ${maxRetries} attempts: ${error.message}`);
      }
    }
  }
}

// Download file once
function downloadFileOnce(url, dest) {
  return new Promise((resolve, reject) => {
    const protocol = url.startsWith('https') ? https : http;
    const file = fs.createWriteStream(dest);

    const request = protocol.get(url, (response) => {
      if (response.statusCode === 302 || response.statusCode === 301) {
        // Handle redirect
        file.close();
        fs.unlinkSync(dest);
        downloadFileOnce(response.headers.location, dest).then(resolve).catch(reject);
        return;
      }

      if (response.statusCode !== 200) {
        file.close();
        fs.unlinkSync(dest);
        reject(new Error(`HTTP ${response.statusCode}: ${response.statusMessage}`));
        return;
      }

      response.pipe(file);

      file.on('finish', () => {
        file.close();
        resolve();
      });
    });

    request.on('error', (err) => {
      file.close();
      fs.unlinkSync(dest);
      reject(err);
    });

    file.on('error', (err) => {
      file.close();
      fs.unlinkSync(dest);
      reject(err);
    });
  });
}

// Extract archive
function extractArchive(archivePath, destDir, isWindows) {
  logInfo('Extracting archive...');
  
  try {
    if (isWindows) {
      // Windows: use PowerShell to extract zip
      const psCommand = `Expand-Archive -Path "${archivePath}" -DestinationPath "${destDir}" -Force`;
      execSync(`powershell -command "${psCommand}"`, { stdio: 'ignore' });
    } else {
      // Unix: use tar
      execSync(`tar -xzf "${archivePath}" -C "${destDir}"`, { stdio: 'ignore' });
    }
    logInfo('Extraction completed');
  } catch (error) {
    throw new Error(`Failed to extract archive: ${error.message}`);
  }
}

// Main installation function
async function install() {
  console.log('');
  console.log('╔════════════════════════════════════════╗');
  console.log('║   BrowserWing npm Installation        ║');
  console.log('╚════════════════════════════════════════╝');
  console.log('');

  try {
    // Detect platform
    const { platform, arch, isWindows } = detectPlatform();
    logInfo(`Detected platform: ${platform}-${arch}`);

    // Determine archive and binary names
    const archiveExt = isWindows ? '.zip' : '.tar.gz';
    const binaryExt = isWindows ? '.exe' : '';
    const archiveName = `browserwing-${platform}-${arch}${archiveExt}`;
    const binaryName = `browserwing-${platform}-${arch}${binaryExt}`;

    // Download paths
    const binDir = path.join(__dirname, 'bin');
    const archivePath = path.join(binDir, archiveName);
    const binaryPath = path.join(binDir, `browserwing${binaryExt}`);

    // Create bin directory
    if (!fs.existsSync(binDir)) {
      fs.mkdirSync(binDir, { recursive: true });
    }

    // Download archive
    await downloadFile(archiveName, archivePath);

    // Extract archive
    extractArchive(archivePath, binDir, isWindows);

    // Find extracted binary
    const extractedBinary = path.join(binDir, binaryName);
    if (!fs.existsSync(extractedBinary)) {
      throw new Error(`Binary not found after extraction: ${binaryName}`);
    }

    // Move binary to final location
    if (fs.existsSync(binaryPath)) {
      fs.unlinkSync(binaryPath);
    }
    fs.renameSync(extractedBinary, binaryPath);

    // Set executable permissions (Unix only)
    if (!isWindows) {
      fs.chmodSync(binaryPath, 0o755);
    }

    // Cleanup archive
    fs.unlinkSync(archivePath);

    console.log('');
    logInfo('BrowserWing installed successfully!');
    console.log('');
    console.log('Quick start:');
    console.log('  1. Run: browserwing --port 8080');
    console.log('  2. Open: http://localhost:8080');
    console.log('');
    console.log('Documentation: https://github.com/' + REPO);
    console.log('中文文档: https://gitee.com/browserwing/browserwing');
    console.log('');

  } catch (error) {
    logError(error.message);
    console.log('');
    console.log('Manual installation:');
    console.log(`  Visit: https://github.com/${REPO}/releases/tag/${TAG}`);
    console.log('');
    process.exit(1);
  }
}

// Run installation
install();
