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
- Authentication via cookies or email/password login
- Support for both JSON and Netscape cookies.txt formats
- Downloads videos using yt-dlp with proper authentication
- Configurable wait time for page loading
- Toggleable headless mode for debugging
- Improved error handling and detailed logs
- Auto-scrolling to capture lazy-loaded content

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
go build -o skool-loom-dl
```

## Usage

### Basic Usage

```bash
# Using cookies for authentication
./skool-loom-dl -url="https://yourschool.skool.com/classroom/your-classroom" -cookies="cookies.json"

# Using email/password for authentication
./skool-loom-dl -url="https://yourschool.skool.com/classroom/your-classroom" -email="your@email.com" -password="yourpassword"
```

### Options

```
-url        URL of the skool.com classroom page (required)
-cookies    Path to cookies file (JSON or TXT) for authentication
-email      Email for Skool login (alternative to cookies)
-password   Password for Skool login (required with email)
-output     Directory to save downloaded videos (default: "downloads")
-wait       Time to wait for page to load in seconds (default: 2)
-headless   Run in headless mode (default: true, set to false for debugging)
```

### Authentication Methods

You must provide one of these authentication methods:
1. A cookies file with `-cookies`
2. Email and password combination with `-email` and `-password`

### Examples

```bash
# Basic usage with JSON cookies
./skool-loom-dl -url="https://yourschool.skool.com/classroom/path" -cookies="cookies.json"

# Using email/password authentication
./skool-loom-dl -url="https://yourschool.skool.com/classroom/path" -email="your@email.com" -password="yourpassword"

# Using cookies.txt format
./skool-loom-dl -url="https://yourschool.skool.com/classroom/path" -cookies="cookies.txt"

# Specify output directory
./skool-loom-dl -url="https://yourschool.skool.com/classroom/path" -cookies="cookies.json" -output="my_videos"

# Increase wait time for slow-loading pages
./skool-loom-dl -url="https://yourschool.skool.com/classroom/path" -cookies="cookies.json" -wait=5

# Disable headless mode for debugging
./skool-loom-dl -url="https://yourschool.skool.com/classroom/path" -cookies="cookies.json" -headless=false
```

## Getting Cookies

You can use either cookie authentication or email/password login. For cookie authentication:

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

## Email/Password Authentication

As an alternative to cookie-based authentication, you can now directly provide your Skool.com login credentials:

```bash
./skool-loom-dl -url="https://yourschool.skool.com/classroom/path" -email="your@email.com" -password="yourpassword"
```

**Note:** Using email/password authentication is generally more reliable as it handles cookie refreshing automatically.

## Troubleshooting

- **No videos found**: Make sure your authentication is valid and the classroom URL is correct
- **Authentication errors**: Try using email/password authentication instead of cookies
- **Incomplete page loading**: Try increasing the `-wait` parameter (e.g., `-wait=5` or `-wait=10`)
- **Download errors**: Make sure yt-dlp is properly installed and updated
- **Debugging**: Use `-headless=false` to see the browser window and what's happening
- **Login issues**: For hard-to-debug problems, try disabling headless mode and increasing wait time

## License

This project is licensed under the MIT License - see the LICENSE file for details.