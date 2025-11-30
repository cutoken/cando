<p align="center">
  <img src="docs/images/cando_logo.png" alt="CanDo Logo" width="200">
</p>

# CanDo

The coding agent that actually gets shit done. Autonomous coding - writes, tests, debugs, and ships features. Supports Z.AI and OpenRouter (Claude, GPT-4, 100+ models).

## Install

### Linux / macOS

```bash
curl -fsSL https://raw.githubusercontent.com/cutoken/cando/main/install.sh | bash
```

### Windows (PowerShell)

```powershell
irm https://raw.githubusercontent.com/cutoken/cando/main/install.ps1 | iex
```

### Manual Download

Grab the binary for your platform from [GitHub Releases](https://github.com/cutoken/cando/releases/latest).

## Usage

```bash
cando
```

Opens http://localhost:3737 in your browser. Configure your AI provider and start coding.

```bash
cando --sandbox /path/to/project   # Use specific workspace
cando --port 8080                  # Custom port
```

## CLI / CI-CD

Run without the web UI:

```bash
cando -p "fix the failing tests in src/"
```

## What Can CanDo Build?

![Doom game built with CanDo](docs/images/doom_game.png)

*A fully functional Doom-style game built in under 5 minutes*

![CanDo Web UI](docs/images/cando-ui.png)

| | |
|:---:|:---:|
| ![Pacman Game](docs/images/pacman.png) | ![Milkyway Animation](docs/images/milkyway-animation.png) |
| ![Moon Phases](docs/images/moon-phases-animation.png) | ![Sculpting Tool](docs/images/basic-sculpting-tool.png) |

## Community

- Discord: https://discord.gg/fzWbCf9CA
- Issues: https://github.com/cutoken/cando/issues
- Discussions: https://github.com/cutoken/cando/discussions

## Documentation

- [Installation Guide](INSTALL.md) - Detailed install instructions
- [Build Guide](BUILD.md) - Building from source, releases
- [Contributing](CONTRIBUTING.md) - Development setup, project structure

## License

[GNU Affero General Public License v3.0](LICENSE)
