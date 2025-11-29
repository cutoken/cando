# Cando installer for Windows
# Usage: irm https://raw.githubusercontent.com/cutoken/cando/main/install.ps1 | iex
# Or: Invoke-WebRequest -Uri "https://raw.githubusercontent.com/cutoken/cando/main/install.ps1" -OutFile install.ps1; .\install.ps1

$ErrorActionPreference = "Stop"

# Configuration
$RepoOwner = "cutoken"
$RepoName = "cando"
$BinaryName = "cando.exe"

# Detect architecture
function Get-Architecture {
    $arch = [System.Environment]::GetEnvironmentVariable("PROCESSOR_ARCHITECTURE")
    switch ($arch) {
        "AMD64" { return "amd64" }
        "ARM64" { return "arm64" }
        default {
            Write-Host "Unsupported architecture: $arch" -ForegroundColor Red
            exit 1
        }
    }
}

# Get latest release version
function Get-LatestVersion {
    try {
        $release = Invoke-RestMethod -Uri "https://api.github.com/repos/$RepoOwner/$RepoName/releases/latest"
        return $release.tag_name
    } catch {
        Write-Host "Warning: Could not fetch latest version, using 'latest'" -ForegroundColor Yellow
        return "latest"
    }
}

# Create install directory
function Get-InstallDir {
    $installDir = Join-Path $env:LOCALAPPDATA "Programs\cando"
    if (-not (Test-Path $installDir)) {
        New-Item -ItemType Directory -Path $installDir -Force | Out-Null
    }
    return $installDir
}

# Add to PATH if not already present
function Add-ToPath {
    param([string]$Dir)

    $currentPath = [Environment]::GetEnvironmentVariable("PATH", "User")
    if ($currentPath -notlike "*$Dir*") {
        $newPath = "$Dir;$currentPath"
        [Environment]::SetEnvironmentVariable("PATH", $newPath, "User")
        Write-Host "Added $Dir to PATH" -ForegroundColor Green
        Write-Host "Note: Restart your terminal for PATH changes to take effect" -ForegroundColor Yellow
    } else {
        Write-Host "Directory already in PATH" -ForegroundColor Green
    }
}

# Create Start Menu shortcut
function New-StartMenuShortcut {
    param([string]$ExePath)

    $startMenuDir = Join-Path $env:APPDATA "Microsoft\Windows\Start Menu\Programs"
    $shortcutPath = Join-Path $startMenuDir "Cando.lnk"

    try {
        $shell = New-Object -ComObject WScript.Shell
        $shortcut = $shell.CreateShortcut($shortcutPath)
        $shortcut.TargetPath = $ExePath
        $shortcut.WorkingDirectory = $env:USERPROFILE
        $shortcut.Description = "Cando - AI Coding Assistant"
        $shortcut.Save()
        Write-Host "Created Start Menu shortcut" -ForegroundColor Green
    } catch {
        Write-Host "Warning: Could not create Start Menu shortcut: $_" -ForegroundColor Yellow
    }
}

# Main installation
function Install-Cando {
    Write-Host "Cando Installer for Windows" -ForegroundColor Cyan
    Write-Host "============================" -ForegroundColor Cyan
    Write-Host ""

    $arch = Get-Architecture
    $version = Get-LatestVersion
    $installDir = Get-InstallDir
    $exePath = Join-Path $installDir $BinaryName

    Write-Host "Architecture: windows-$arch" -ForegroundColor Green
    Write-Host "Version: $version" -ForegroundColor Green
    Write-Host "Install directory: $installDir" -ForegroundColor Green
    Write-Host ""

    # Construct download URL
    if ($version -eq "latest") {
        $downloadUrl = "https://github.com/$RepoOwner/$RepoName/releases/latest/download/cando-windows-$arch.exe"
    } else {
        $downloadUrl = "https://github.com/$RepoOwner/$RepoName/releases/download/$version/cando-windows-$arch.exe"
    }

    Write-Host "Downloading from: $downloadUrl" -ForegroundColor Cyan

    # Download binary
    try {
        $tempFile = Join-Path $env:TEMP "cando-download.exe"
        Invoke-WebRequest -Uri $downloadUrl -OutFile $tempFile -UseBasicParsing

        # Move to install directory (replace if exists)
        if (Test-Path $exePath) {
            # Rename old binary first (in case it's running)
            $backupPath = "$exePath.backup"
            if (Test-Path $backupPath) { Remove-Item $backupPath -Force }
            Rename-Item -Path $exePath -NewName "$BinaryName.backup" -Force -ErrorAction SilentlyContinue
        }
        Move-Item -Path $tempFile -Destination $exePath -Force

        Write-Host "Downloaded successfully" -ForegroundColor Green
    } catch {
        Write-Host "Failed to download: $_" -ForegroundColor Red
        exit 1
    }

    # Add to PATH
    Add-ToPath -Dir $installDir

    # Create Start Menu shortcut
    New-StartMenuShortcut -ExePath $exePath

    Write-Host ""
    Write-Host "Installation complete!" -ForegroundColor Green
    Write-Host ""
    Write-Host "To get started:" -ForegroundColor Cyan
    Write-Host "  1. Open a new terminal (for PATH changes)" -ForegroundColor White
    Write-Host "  2. Run: cando" -ForegroundColor White
    Write-Host "  3. Open http://localhost:3737 in your browser" -ForegroundColor White
    Write-Host ""
    Write-Host "Or launch from Start Menu: search for 'Cando'" -ForegroundColor Cyan
    Write-Host ""
    Write-Host "To uninstall: Remove-Item -Recurse '$installDir'" -ForegroundColor Gray
}

# Run installer
Install-Cando
