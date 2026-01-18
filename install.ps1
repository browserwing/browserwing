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

function Write-Debug-Custom {
    param($Message)
    Write-Host "[DEBUG] $Message" -ForegroundColor Cyan
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
    
    # Try GitHub API first
    try {
        $release = Invoke-RestMethod -Uri "https://api.github.com/repos/$REPO/releases/latest" -TimeoutSec 5 -ErrorAction Stop
        $version = $release.tag_name
        Write-Info "Latest version: $version"
        return $version
    }
    catch {
        Write-Warning-Custom "GitHub API failed, trying Gitee..."
    }
    
    # Try Gitee API
    try {
        $release = Invoke-RestMethod -Uri "https://gitee.com/api/v5/repos/browserwing/browserwing/releases/latest" -TimeoutSec 5 -ErrorAction Stop
        $version = $release.tag_name
        Write-Info "Latest version: $version"
        return $version
    }
    catch {
        Write-Warning-Custom "Gitee API failed, using default version: v0.0.1"
        return "v0.0.1"
    }
}

# Test download speed and select fastest mirror
function Select-Mirror {
    param($Version, $Arch)
    
    Write-Info "Testing download mirrors..."
    
    $archiveName = "browserwing-windows-$Arch.zip"
    
    # Test GitHub
    $githubUrl = "https://github.com/$REPO/releases/download/$Version/$archiveName"
    try {
        $githubTime = Measure-Command { 
            Invoke-WebRequest -Uri $githubUrl -Method Head -TimeoutSec 5 -ErrorAction Stop | Out-Null
        }
        $githubMs = $githubTime.TotalMilliseconds
        Write-Debug-Custom "GitHub response time: $githubMs ms"
    }
    catch {
        $githubMs = 999999
        Write-Debug-Custom "GitHub not reachable"
    }
    
    # Test Gitee
    $giteeUrl = "https://gitee.com/browserwing/browserwing/releases/download/$Version/$archiveName"
    try {
        $giteeTime = Measure-Command { 
            Invoke-WebRequest -Uri $giteeUrl -Method Head -TimeoutSec 5 -ErrorAction Stop | Out-Null
        }
        $giteeMs = $giteeTime.TotalMilliseconds
        Write-Debug-Custom "Gitee response time: $giteeMs ms"
    }
    catch {
        $giteeMs = 999999
        Write-Debug-Custom "Gitee not reachable"
    }
    
    # Select fastest mirror
    if ($githubMs -lt $giteeMs) {
        $mirror = "github"
        $mirrorUrl = "https://github.com/$REPO/releases/download"
        Write-Info "Using GitHub mirror (faster)"
    }
    else {
        $mirror = "gitee"
        $mirrorUrl = "https://gitee.com/browserwing/browserwing/releases/download"
        Write-Info "Using Gitee mirror (faster)"
    }
    
    return @{
        Name = $mirror
        Url = $mirrorUrl
    }
}

# Download and install binary
function Install-BrowserWing {
    param($Version, $Arch, $Mirror)
    
    Write-Info "Downloading BrowserWing..."
    
    $archiveName = "browserwing-windows-$Arch.zip"
    $binaryName = "browserwing-windows-$Arch.exe"
    $downloadUrl = "$($Mirror.Url)/$Version/$archiveName"
    
    Write-Info "Download URL: $downloadUrl"
    
    # Create temp directory
    $tempDir = New-Item -ItemType Directory -Path "$env:TEMP\browserwing-install-$(Get-Random)"
    $archivePath = Join-Path $tempDir $archiveName
    
    # Download with retry
    $maxRetries = 3
    $retryCount = 0
    $downloadSuccess = $false
    
    while ($retryCount -lt $maxRetries -and -not $downloadSuccess) {
        try {
            Invoke-WebRequest -Uri $downloadUrl -OutFile $archivePath -TimeoutSec 60
            Write-Info "Download completed"
            $downloadSuccess = $true
        }
        catch {
            $retryCount++
            if ($retryCount -lt $maxRetries) {
                Write-Warning-Custom "Download failed, retrying ($retryCount/$maxRetries)..."
                
                # Switch mirror on failure
                if ($Mirror.Name -eq "github") {
                    $Mirror.Name = "gitee"
                    $Mirror.Url = "https://gitee.com/browserwing/browserwing/releases/download"
                    Write-Info "Switching to Gitee mirror..."
                }
                else {
                    $Mirror.Name = "github"
                    $Mirror.Url = "https://github.com/$REPO/releases/download"
                    Write-Info "Switching to GitHub mirror..."
                }
                
                $downloadUrl = "$($Mirror.Url)/$Version/$archiveName"
                Start-Sleep -Seconds 2
            }
            else {
                Write-Error-Custom "Failed to download after $maxRetries attempts: $_"
                Remove-Item -Path $tempDir -Recurse -Force -ErrorAction SilentlyContinue
                exit 1
            }
        }
    }
    
    # Extract archive
    Write-Info "Extracting archive..."
    try {
        Expand-Archive -Path $archivePath -DestinationPath $tempDir -Force
        
        # Verify binary exists
        $extractedBinary = Join-Path $tempDir $binaryName
        if (-not (Test-Path $extractedBinary)) {
            Write-Error-Custom "Binary not found after extraction: $binaryName"
            Remove-Item -Path $tempDir -Recurse -Force -ErrorAction SilentlyContinue
            exit 1
        }
        
        Write-Info "Extraction completed"
        
        # Install
        Write-Info "Installing BrowserWing..."
        New-Item -ItemType Directory -Path $INSTALL_DIR -Force | Out-Null
        
        $binaryPath = Join-Path $INSTALL_DIR "browserwing.exe"
        Copy-Item -Path $extractedBinary -Destination $binaryPath -Force
        
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
    Write-Info "BrowserWing installed successfully!"
    Write-Host ""
    Write-Host "Installation location: $BinaryPath"
    Write-Host ""
    Write-Host "Quick start:"
    Write-Host "  1. Run: browserwing --port 8080"
    Write-Host "  2. Open: http://localhost:8080"
    Write-Host ""
    Write-Host "Documentation: https://github.com/$REPO"
    Write-Host "中文文档: https://gitee.com/browserwing/browserwing"
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
    $mirror = Select-Mirror -Version $version -Arch $arch
    $binaryPath = Install-BrowserWing -Version $version -Arch $arch -Mirror $mirror
    Show-Success -BinaryPath $binaryPath
}

# Run main function
Main
