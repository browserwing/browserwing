# BrowserWing Installation Script for Windows
# Usage: iwr -useb https://raw.githubusercontent.com/browserwing/browserwing/main/install.ps1 | iex

$ErrorActionPreference = "Stop"

$REPO = "browserwing/browserwing"
$INSTALL_DIR = if ($env:BROWSERWING_INSTALL_DIR) { $env:BROWSERWING_INSTALL_DIR } else { "$env:USERPROFILE\.browserwing" }

# Colors for output
function Write-Info {
    param($Message)
    Write-Host "[INFO] $Message" -ForegroundColor Green
}

function Write-Error-Custom {
    param($Message)
    Write-Host "[ERROR] $Message" -ForegroundColor Red
}

function Write-Warning-Custom {
    param($Message)
    Write-Host "[WARNING] $Message" -ForegroundColor Yellow
}

# Detect architecture
function Get-Architecture {
    $arch = $env:PROCESSOR_ARCHITECTURE
    switch ($arch) {
        "AMD64" { return "amd64" }
        "ARM64" { return "arm64" }
        default {
            Write-Error-Custom "Unsupported architecture: $arch"
            exit 1
        }
    }
}

# Get latest release version
function Get-LatestVersion {
    Write-Info "Fetching latest release..."
    
    try {
        $release = Invoke-RestMethod -Uri "https://api.github.com/repos/$REPO/releases/latest"
        $version = $release.tag_name
        Write-Info "Latest version: $version"
        return $version
    }
    catch {
        Write-Error-Custom "Failed to fetch latest version: $_"
        exit 1
    }
}

# Download and install binary
function Install-BrowserWing {
    param($Version, $Arch)
    
    Write-Info "Downloading BrowserWing..."
    
    $binaryName = "browserwing-windows-$Arch.exe"
    $downloadUrl = "https://github.com/$REPO/releases/download/$Version/$binaryName"
    
    Write-Info "Download URL: $downloadUrl"
    
    # Create temp directory
    $tempDir = New-Item -ItemType Directory -Path "$env:TEMP\browserwing-install-$(Get-Random)"
    $downloadPath = Join-Path $tempDir $binaryName
    
    try {
        # Download binary directly
        Invoke-WebRequest -Uri $downloadUrl -OutFile $downloadPath
        
        # Install
        Write-Info "Installing BrowserWing..."
        New-Item -ItemType Directory -Path $INSTALL_DIR -Force | Out-Null
        
        $binaryPath = Join-Path $INSTALL_DIR "browserwing.exe"
        Copy-Item -Path $downloadPath -Destination $binaryPath -Force
        
        Write-Info "Installation complete!"
        
        # Add to PATH if not already there
        $userPath = [Environment]::GetEnvironmentVariable("PATH", "User")
        if ($userPath -notlike "*$INSTALL_DIR*") {
            Write-Info "Adding to PATH..."
            [Environment]::SetEnvironmentVariable("PATH", "$userPath;$INSTALL_DIR", "User")
            Write-Warning-Custom "Please restart your terminal for PATH changes to take effect"
        }
        
        return $binaryPath
    }
    catch {
        Write-Error-Custom "Installation failed: $_"
        exit 1
    }
    finally {
        # Cleanup
        Remove-Item -Path $tempDir -Recurse -Force -ErrorAction SilentlyContinue
    }
}

# Print success message
function Show-Success {
    param($BinaryPath)
    
    Write-Host ""
    Write-Info "BrowserWing installed successfully!" -ForegroundColor Green
    Write-Host ""
    Write-Host "Installation location: $BinaryPath"
    Write-Host ""
    Write-Host "Quick start:"
    Write-Host "  1. Run: browserwing --port 8080"
    Write-Host "  2. Open: http://localhost:8080"
    Write-Host ""
    Write-Host "Documentation: https://github.com/$REPO"
    Write-Host "Report issues: https://github.com/$REPO/issues"
    Write-Host ""
}

# Main installation flow
function Main {
    Write-Host ""
    Write-Host "╔════════════════════════════════════════╗"
    Write-Host "║   BrowserWing Installation Script     ║"
    Write-Host "╚════════════════════════════════════════╝"
    Write-Host ""
    
    $arch = Get-Architecture
    Write-Info "Detected platform: windows-$arch"
    
    $version = Get-LatestVersion
    $binaryPath = Install-BrowserWing -Version $version -Arch $arch
    Show-Success -BinaryPath $binaryPath
}

# Run main function
Main
