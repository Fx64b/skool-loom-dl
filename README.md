# Skool-Loom-Downloader

A Go utility to automatically scrape and download Loom videos from Skool.com classrooms.

## Disclaimer

**This tool is provided for educational and legitimate purposes only.** It is designed to help users download their own videos or content they have explicit permission to download. Please respect copyright laws and terms of service:

- Only download videos you have the right to access and save
- Do not use this tool to bypass paywalls or access unauthorized content
- Respect the terms of service of both Skool.com and Loom.com
- The developers of this tool accept no liability for misuse

## Features

- Automatically scrape Loom video links from Skool.com classroom pages
- Support for both JSON and Netscape cookies.txt formats
- Downloads videos using yt-dlp with proper authentication
- Configurable wait time for page loading
- Simple command-line interface

## Installation

### Prerequisites

1. Install Go (1.16 or newer): https://golang.org/doc/install
2. Install yt-dlp: `pip install yt-dlp`
3. Install required Go dependencies:

```bash
go get github.com/chromedp/chromedp
go get github.com/chromedp/cdproto
```

### Building the Tool

```bash
# Clone the repository (if applicable) or create the file
git clone https://github.com/yourusername/skool-loom-dl
cd skool-loom-dl

# Or simply save the code as skool-loom-dl.go

# Build the executable
go build skool-loom-dl.go
```

## Usage

### Basic Usage

```bash
./skool-loom-dl -url="https://yourschool.skool.com/classroom/your-classroom" -cookies="cookies.json"
```

### Options

```
-url      URL of the skool.com classroom page (required)
-cookies  Path to cookies file (JSON or TXT) for authentication (required)
-output   Directory to save downloaded videos (default: "downloads")
-wait     Time to wait for page to load in seconds (default: 2)
```

### Examples

```bash
# Basic usage with JSON cookies
./skool-loom-dl -url="https://yourschool.skool.com/classroom/path" -cookies="cookies.json"

# Using cookies.txt format
./skool-loom-dl -url="https://yourschool.skool.com/classroom/path" -cookies="cookies.txt"

# Specify output directory
./skool-loom-dl -url="https://yourschool.skool.com/classroom/path" -cookies="cookies.json" -output="my_videos"

# Increase wait time for slow-loading pages
./skool-loom-dl -url="https://yourschool.skool.com/classroom/path" -cookies="cookies.json" -wait=5
```

## Getting Cookies

To use this tool, you'll need to export your browser cookies:

### For Chrome/Chromium
1. Install the "Cookie-Editor" or "Get cookies.txt LOCALLY" extension
2. Log in to your Skool.com account
3. Open the extension and export cookies (as JSON or Netscape format)
4. Save the file and use it with the `-cookies` parameter

### For Firefox
1. Install the "Cookie Quick Manager" extension
2. Log in to your Skool.com account
3. Open the extension, select all cookies for the domain, and export
4. Save the file and use it with the `-cookies` parameter

## Troubleshooting

- **No videos found**: Make sure your cookies are valid and not expired
- **Authentication errors**: Log in to your account again and export fresh cookies
- **Incomplete page loading**: Try increasing the `-wait` parameter
- **Download errors**: Make sure yt-dlp is properly installed and updated

## License

This project is licensed under the MIT License - see the LICENSE file for details.