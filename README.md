# Skool-Loom-Downloader

A Go utility to automatically scrape and download Loom videos from Skool.com classrooms.

## Disclaimer

> [!CAUTION]
> **This tool is provided for educational and legitimate purposes only.**

Use this tool only to download content you have the right to access. Please respect copyright laws and terms of service:
- Only download videos you have permission to save
- Do not bypass paywalls or access unauthorized content
- Respect the terms of service of both Skool.com and Loom.com
- The developers accept no liability for misuse

## Features

- Scrapes Loom video links from Skool.com classroom pages
- Authentication via email/password or cookies
- Supports JSON and Netscape cookies.txt formats
- Downloads videos using yt-dlp with proper authentication
- Configurable page loading wait time
- Toggleable headless mode for debugging

## Installation

### Option 1: Download Pre-built Binaries (Recommended)

1. Install [yt-dlp](https://github.com/yt-dlp/yt-dlp#installation)
2. Download the latest release from the [Releases page](https://github.com/fx64b/skool-loom-dl/releases)
3. Choose the appropriate binary for your platform:
   - **Linux (x64)**: `skool-loom-dl-linux-amd64`
   - **Linux (ARM64)**: `skool-loom-dl-linux-arm64`
   - **Windows (x64)**: `skool-loom-dl-windows-amd64.exe`
   - **Windows (ARM64)**: `skool-loom-dl-windows-arm64.exe`
   - **macOS (Intel)**: `skool-loom-dl-darwin-amd64`
   - **macOS (Apple Silicon)**: `skool-loom-dl-darwin-arm64`
4. Make it executable (Linux/macOS): `chmod +x skool-loom-dl-*`

### Option 2: Build from Source

#### Prerequisites

1. Install [Go](https://golang.org/doc/install) (1.18 or newer)
2. Install [yt-dlp](https://github.com/yt-dlp/yt-dlp#installation)

#### Building the Tool

```bash
# Clone the repository
git clone https://github.com/fx64b/skool-loom-dl
cd skool-loom-dl

# Build the executable
go build
```

## Usage

### Basic Usage

```bash
# Recommended: Using email/password for authentication
./skool-loom-dl -url="https://skool.com/yourschool/classroom/your-classroom" -email="your@email.com" -password="yourpassword"

# Alternative: Using cookies for authentication
./skool-loom-dl -url="https://skool.com/yourschool/classroom/your-classroom" -cookies="cookies.json"
```

### Important Options

```
-url        URL of the skool.com classroom page (required)
-email      Email for Skool login (recommended auth method)
-password   Password for Skool login (used with email)
-cookies    Path to cookies file (alternative to email/password)
-output     Directory to save videos (default: "downloads")
-wait       Page load wait time in seconds (default: 2)
-headless   Run browser headless (default: true, set false for debugging)
```

### Authentication Methods

**Email/Password (Recommended)**
```bash
./skool-loom-dl -url="https://skool.com/yourschool/classroom/path" -email="your@email.com" -password="yourpassword"
```

**Cookies (Alternative)**
```bash
./skool-loom-dl -url="https://skool.com/yourschool/classroom/path" -cookies="cookies.json"
```

> **Note:** Email/password authentication is more reliable as it handles session management automatically. Cookie-based authentication may fail if cookies expire or are invalid.

## Getting Cookies (if needed)

If you choose to use cookies instead of email/password:

1. Install a browser extension like "Cookie-Editor" (Chrome) or "Cookie Quick Manager" (Firefox)
2. Log in to your Skool.com account
3. Export cookies as JSON or Netscape format
4. Save the file and use it with the `-cookies` parameter

## Troubleshooting

- **No videos found**: Verify your authentication and classroom URL
- **Authentication fails**: Use email/password instead of cookies
- **Page loads incomplete**: Increase wait time with `-wait=5` or higher
- **Download errors**: Update yt-dlp (`pip install -U yt-dlp`)
- **Login issues**: Try `-headless=false` to see the browser and debug
- **Specific video errors**: Check if the video is still available on Loom

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

### Creating Releases (Maintainers)

To create a new release with cross-platform binaries:

1. Go to the [Actions tab](https://github.com/fx64b/skool-loom-dl/actions)
2. Select "Build and Release" workflow
3. Click "Run workflow"
4. Enter the desired version (e.g., `v1.0.0`)
5. Click "Run workflow"

The workflow will automatically:
- Build binaries for all supported platforms
- Create a new GitHub release
- Attach all binaries to the release

## License

This project is licensed under the MIT License - see the LICENSE file for details.
