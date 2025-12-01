# Cando BETA installer for Windows
# Usage: irm https://raw.githubusercontent.com/cutoken/cando/main/dev/install-beta.ps1 | iex
# Or specify version: $env:CANDO_BETA_VERSION="v1.0.0-beta.1"; irm ... | iex

$ErrorActionPreference = "Stop"

# Configuration
$RepoOwner = "cutoken"
$RepoName = "cando"
$BinaryName = "cando-beta.exe"

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

# Get latest beta (prerelease) version
function Get-LatestBetaVersion {
    try {
        $releases = Invoke-RestMethod -Uri "https://api.github.com/repos/$RepoOwner/$RepoName/releases"
        foreach ($release in $releases) {
            if ($release.prerelease -eq $true) {
                return $release.tag_name
            }
        }
        Write-Host "No beta releases found" -ForegroundColor Red
        Write-Host "Check: https://github.com/$RepoOwner/$RepoName/releases" -ForegroundColor Yellow
        exit 1
    } catch {
        Write-Host "Failed to fetch releases: $_" -ForegroundColor Red
        exit 1
    }
}

# Create install directory
function Get-InstallDir {
    $installDir = Join-Path $env:LOCALAPPDATA "Programs\cando-beta"
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
    $shortcutPath = Join-Path $startMenuDir "Cando Beta.lnk"

    try {
        $shell = New-Object -ComObject WScript.Shell
        $shortcut = $shell.CreateShortcut($shortcutPath)
        $shortcut.TargetPath = $ExePath
        $shortcut.WorkingDirectory = $env:USERPROFILE
        $shortcut.Description = "Cando Beta - AI Coding Assistant (Beta Version)"
        $shortcut.Save()
        Write-Host "Created Start Menu shortcut: Cando Beta" -ForegroundColor Green
    } catch {
        Write-Host "Warning: Could not create Start Menu shortcut: $_" -ForegroundColor Yellow
    }
}

# Show beta warning
function Show-BetaNotice {
    Write-Host ""
    Write-Host "========================================" -ForegroundColor Blue
    Write-Host "           BETA VERSION                 " -ForegroundColor Blue
    Write-Host "  This is a beta release for testing.   " -ForegroundColor Blue
    Write-Host "  Report issues to the dev team.        " -ForegroundColor Blue
    Write-Host "========================================" -ForegroundColor Blue
    Write-Host ""
}

# Main installation
function Install-CandoBeta {
    Write-Host "Cando BETA Installer for Windows" -ForegroundColor Cyan
    Write-Host "=================================" -ForegroundColor Cyan
    Write-Host ""

    Show-BetaNotice

    $arch = Get-Architecture

    # Check for environment variable override
    $version = $env:CANDO_BETA_VERSION
    if ([string]::IsNullOrEmpty($version)) {
        $version = Get-LatestBetaVersion
    }

    $installDir = Get-InstallDir
    $exePath = Join-Path $installDir $BinaryName

    Write-Host "Architecture: windows-$arch" -ForegroundColor Green
    Write-Host "Beta Version: $version" -ForegroundColor Green
    Write-Host "Install directory: $installDir" -ForegroundColor Green
    Write-Host ""

    # Construct download URL
    $downloadUrl = "https://github.com/$RepoOwner/$RepoName/releases/download/$version/cando-windows-$arch.exe"

    Write-Host "Downloading from: $downloadUrl" -ForegroundColor Cyan

    # Download binary
    try {
        $tempFile = Join-Path $env:TEMP "cando-beta-download.exe"
        Invoke-WebRequest -Uri $downloadUrl -OutFile $tempFile -UseBasicParsing

        # Move to install directory (replace if exists)
        if (Test-Path $exePath) {
            # Rename old binary first (in case it's running)
            $backupPath = "$exePath.backup"
            if (Test-Path $backupPath) { Remove-Item $backupPath -Force }
            Rename-Item -Path $exePath -NewName "cando-beta.exe.backup" -Force -ErrorAction SilentlyContinue
        }
        Move-Item -Path $tempFile -Destination $exePath -Force

        Write-Host "Downloaded successfully" -ForegroundColor Green
    } catch {
        Write-Host "Failed to download: $_" -ForegroundColor Red
        Write-Host ""
        Write-Host "Possible reasons:" -ForegroundColor Yellow
        Write-Host "  1. Beta version $version doesn't exist" -ForegroundColor White
        Write-Host "  2. Network issues" -ForegroundColor White
        Write-Host "  3. GitHub API rate limit" -ForegroundColor White
        Write-Host ""
        Write-Host "Try:" -ForegroundColor Yellow
        Write-Host "  - Check releases at https://github.com/$RepoOwner/$RepoName/releases" -ForegroundColor White
        Write-Host "  - Specify version: `$env:CANDO_BETA_VERSION='v1.0.0-beta.1'" -ForegroundColor White
        exit 1
    }

    # Add to PATH
    Add-ToPath -Dir $installDir

    # Create Start Menu shortcut
    New-StartMenuShortcut -ExePath $exePath

    Write-Host ""
    Write-Host "Beta installation complete!" -ForegroundColor Green
    Write-Host ""
    Write-Host "To get started:" -ForegroundColor Cyan
    Write-Host "  1. Open a new terminal (for PATH changes)" -ForegroundColor White
    Write-Host "  2. Run: cando-beta" -ForegroundColor White
    Write-Host "  3. Open http://localhost:3737 in your browser" -ForegroundColor White
    Write-Host ""
    Write-Host "Or launch from Start Menu: search for 'Cando Beta'" -ForegroundColor Cyan
    Write-Host ""
    Write-Host "Report beta issues at: https://github.com/$RepoOwner/$RepoName/issues" -ForegroundColor Yellow
    Write-Host ""
    Write-Host "To uninstall beta: Remove-Item -Recurse '$installDir'" -ForegroundColor Gray
    Write-Host "To switch to stable: Run the regular installer" -ForegroundColor Gray
}

# Run installer
Install-CandoBeta
