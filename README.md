# GoAT Launcher

GoAT Launcher is a lightweight, third-party desktop launcher designed for Old School RuneScape. It serves as an alternative client interface to manage game sessions, authentication, and updates seamlessly across multiple operating systems.

The core purpose of this launcher is simplicity and ease of installation, targeting **minimal native 
dependencies and avoiding the use of platform features like Chromium and embedded browsers.**

---

## ⚠️ Disclaimer

**GoAT Launcher is an independent, open-source, third-party project and is NOT affiliated, associated, authorized, endorsed by, or in any way officially connected with Jagex Ltd, Old School RuneScape, RuneScape, or any of their subsidiaries or affiliates.** All official product and company names are trademarks™ or registered® trademarks of their respective holders. Use of them does not imply any affiliation with or endorsement by them. Use this launcher at your own risk.

---

## Features

* **Cross-Platform by Design:** Runs on Windows, macOS, and Linux using a single codebase.
* **Native UI Performance:** Powered by a lightweight graphics library for Go that uses direct rendering rather than Chromium/WebView.
* **Minimal System Footprint:** Uses significantly fewer system resources and memory compared to both the official client and Bolt.
* **Secure Authentication:** Uses your computer's Keychain by default for credential storage, interfacing with the Jagex OAuth endpoints via stable Go libraries.

## Technical Philosophy

Unlike other solutions that rely on heavy embedded browser frameworks, GoAT Launcher is as minimal as it gets.

* Standard browser interactions and OS-level hooks for authentication.
* Actively maintained, secure implementations of the necessary OAuth2 and associated libraries.
* A unified, lightweight UI stack designed to compile seamlessly across modern desktop operating systems with minimal system toolchain prerequisites.

## Installation

### Prerequisites

To run pre-compiled versions of GoAT Launcher, your system just needs standard graphics drivers with OpenGL support.

If you are building from source, you will need:

* [Go](https://go.dev/dl/) (version 1.26 or higher)
* A working C compiler (GCC or Clang) and your platform's standard graphics development headers. See [Fyne's Quick Start](https://docs.fyne.io/started/quick/)

### Building from Source

To compile the desktop client, clone the repository and run:

```bash
git clone https://github.com/ccatss/goat-launcher.git
cd goat-launcher
go build -ldflags="-s -w" -o goat-launcher main.go

```

*Note: For detailed, OS-specific setup instructions regarding graphics compilation headers (e.g., X11/Mesa on Linux), please refer to the [Fyne Compiling Guide](https://docs.fyne.io/started/quick/).*

### Download Binaries

Pre-compiled, ready-to-run binaries for Windows, Linux, and macOS are available on the [Releases](https://www.google.com/search?q=https://github.com/yourusername/goat-launcher/releases) page. These binaries are bundled with everything they need to run out of the box.

Note that it doesn't appear that Github Actions support Intel-based Mac anymore. While I could add a custom runner, I don't believe this is worth the effort.

### Attaching the `jagex:` url handler

#### Linux

Add the following file into `~/.local/share/applications/goat-launcher.desktop`:

```ini
[Desktop Entry]
Name=GoAT Launcher
Comment=Third-Party Jagex Launcher Client
Exec=goat-launcher login %u
Icon=goat-launcher
Terminal=false
Type=Application
Categories=Game;Network;
MimeType=x-scheme-handler/jagex;
Keywords=jagex;runescape;osrs;launcher;
StartupNotify=true
```

Update the databases:

```bash
update-desktop-database ~/.local/share/applications/
xdg-mime default goat-launcher.desktop x-scheme-handler/jagex
```

## License

Distributed under the ISC License. See `LICENSE` for more information.