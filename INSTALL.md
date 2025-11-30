# Installation Guide

## Linux

**Automatic:**
```bash
curl -fsSL https://raw.githubusercontent.com/cutoken/cando/main/install.sh | bash
```

**Manual:**
```bash
# Download (choose your architecture)
curl -L https://github.com/cutoken/cando/releases/latest/download/cando-linux-amd64-bin -o cando
# or for ARM64:
curl -L https://github.com/cutoken/cando/releases/latest/download/cando-linux-arm64-bin -o cando

# Install
chmod +x cando
mv cando ~/.local/bin/

# Add to PATH if needed
echo 'export PATH="$HOME/.local/bin:$PATH"' >> ~/.bashrc
source ~/.bashrc
```

## macOS

**Automatic:**
```bash
curl -fsSL https://raw.githubusercontent.com/cutoken/cando/main/install.sh | bash
```

**Manual:**
```bash
# Download (choose your architecture)
curl -L https://github.com/cutoken/cando/releases/latest/download/cando-darwin-arm64-bin -o cando  # Apple Silicon
# or for Intel:
curl -L https://github.com/cutoken/cando/releases/latest/download/cando-darwin-amd64-bin -o cando

# Install
chmod +x cando
mv cando ~/.local/bin/

# Add to PATH if needed
echo 'export PATH="$HOME/.local/bin:$PATH"' >> ~/.zshrc
source ~/.zshrc
```

## Windows

**Automatic (PowerShell):**
```powershell
irm https://raw.githubusercontent.com/cutoken/cando/main/install.ps1 | iex
```

Installs to `%LOCALAPPDATA%\Programs\cando`, adds to PATH, creates Start Menu shortcut.

**Manual:**
```powershell
# Download (choose your architecture)
Invoke-WebRequest -Uri "https://github.com/cutoken/cando/releases/latest/download/cando-windows-amd64.exe" -OutFile "cando.exe"
# or for ARM64:
Invoke-WebRequest -Uri "https://github.com/cutoken/cando/releases/latest/download/cando-windows-arm64.exe" -OutFile "cando.exe"

# Move to a folder in PATH
Move-Item cando.exe $HOME\.local\bin\
```

## Verify Installation

```bash
cando --version
```

## Uninstall

**Linux/macOS:**
```bash
rm ~/.local/bin/cando
rm -rf ~/.cando  # Remove config/data
```

**Windows:**
```powershell
Remove-Item $env:LOCALAPPDATA\Programs\cando -Recurse
Remove-Item $HOME\.cando -Recurse  # Remove config/data
```

## Troubleshooting

**Command not found:** Ensure `~/.local/bin` is in your PATH.

**Permission denied:** Run `chmod +x ~/.local/bin/cando`

**Port in use:** Try `cando --port 3738`
