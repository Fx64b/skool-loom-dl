package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/chromedp"
)

const (
	defaultWaitTime  = 2
	defaultOutputDir = "downloads"
	defaultHeadless  = true
	browserTimeout   = 180 * time.Second
	initialWaitTime  = 3 * time.Second
	loginWaitTime    = 3 * time.Second
	skoolBaseURL     = "https://www.skool.com/"
	skoolLoginURL    = "https://www.skool.com/login"
)

// JSONCookie represents a cookie in the JSON format
type JSONCookie struct {
	Host       string `json:"host"`
	Name       string `json:"name"`
	Value      string `json:"value"`
	Path       string `json:"path"`
	Expiry     int64  `json:"expiry"`
	IsSecure   int    `json:"isSecure"`
	IsHttpOnly int    `json:"isHttpOnly"`
	SameSite   int    `json:"sameSite"`
}

// Config holds application configuration
type Config struct {
	SkoolURL    string
	CookiesFile string
	Email       string
	Password    string
	OutputDir   string
	WaitTime    int
	Headless    bool
}

func main() {
	printBanner()
	config := parseFlags()
	validateConfig(config)

	// Create output directory if it doesn't exist
	if err := os.MkdirAll(config.OutputDir, 0755); err != nil {
		log.Fatalf("Error creating output directory: %v", err)
	}

	fmt.Println("🔍 Scraping Loom videos from:", config.SkoolURL)

	// Scrape videos based on auth method
	loomURLs, err := scrapeVideos(config)
	if err != nil {
		log.Fatalf("Error scraping: %v", err)
	}

	if len(loomURLs) == 0 {
		fmt.Println("❌ No Loom videos found. Check authentication and URL.")
		return
	}

	fmt.Printf("✅ Found %d Loom videos\n", len(loomURLs))

	// Download each video
	for i, url := range loomURLs {
		fmt.Printf("\n[%d/%d] 📥 Downloading: %s\n", i+1, len(loomURLs), url)
		if err := downloadWithYtDlp(url, config.CookiesFile, config.OutputDir); err != nil {
			fmt.Printf("❌ Error: %v\n", err)
		}
	}

	fmt.Println("\n✅ Download process completed!")
}

func printBanner() {
	fmt.Println(`
    ╔═══╗╔╗   ╔═══╗
    ║╔═╗║║║   ║╔═╗║
    ║╚══╗║║   ║║ ║║
    ╚══╗║║║   ║║ ║║
    ║╚═╝║║╚═╗ ║╚═╝║
    ╚═══╝╚══╝ ╚═══╝
    Skool Loom Downloader
    `)
}

func parseFlags() Config {
	config := Config{}

	flag.StringVar(&config.SkoolURL, "url", "", "URL of the skool.com classroom to scrape (required)")
	flag.StringVar(&config.CookiesFile, "cookies", "", "Path to cookies file (JSON or TXT) for authentication")
	flag.StringVar(&config.Email, "email", "", "Email for Skool login (alternative to cookies)")
	flag.StringVar(&config.Password, "password", "", "Password for Skool login (required with email)")
	flag.StringVar(&config.OutputDir, "output", defaultOutputDir, "Directory to save downloaded videos")
	flag.IntVar(&config.WaitTime, "wait", defaultWaitTime, "Time to wait for page to load in seconds")
	flag.BoolVar(&config.Headless, "headless", defaultHeadless, "Run in headless mode (no browser UI)")

	flag.Parse()
	return config
}

func validateConfig(config Config) {
	if config.SkoolURL == "" {
		fmt.Println("Usage: skool-loom-dl -url=https://skool.com/yourschool/classroom/path [-cookies=cookies.json | -email=user@example.com -password=pass]")
		os.Exit(1)
	}

	usingEmail := config.Email != "" && config.Password != ""
	usingCookies := config.CookiesFile != ""

	if !usingEmail && !usingCookies {
		fmt.Println("Error: You must provide either cookies file or email+password for authentication")
		os.Exit(1)
	}
}

func scrapeVideos(config Config) ([]string, error) {
	if config.Email != "" && config.Password != "" {
		return scrapeWithLogin(config)
	}
	return scrapeWithCookies(config)
}

func setupBrowser(headless bool) (context.Context, context.CancelFunc) {
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", headless),
		chromedp.Flag("disable-gpu", true),
		chromedp.Flag("no-sandbox", true),
		chromedp.Flag("window-size", "1920,1080"),
		chromedp.UserAgent("Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36"),
	)

	allocCtx, cancel := chromedp.NewExecAllocator(context.Background(), opts...)
	ctx, cancel2 := chromedp.NewContext(allocCtx, chromedp.WithLogf(log.Printf))
	ctx, cancel3 := context.WithTimeout(ctx, browserTimeout)

	// Return a cancel function that calls all three cancel functions
	return ctx, func() {
		cancel3()
		cancel2()
		cancel()
	}
}

func extractLoomURLs(html string) []string {
	shareRegex := regexp.MustCompile(`https?://(?:www\.)?loom\.com/share/[a-zA-Z0-9]+`)
	embedRegex := regexp.MustCompile(`https?://(?:www\.)?loom\.com/embed/([a-zA-Z0-9]+)`)

	matches := shareRegex.FindAllString(html, -1)

	// Convert embed URLs to share URLs
	embedMatches := embedRegex.FindAllStringSubmatch(html, -1)
	for _, match := range embedMatches {
		if len(match) >= 2 {
			shareURL := fmt.Sprintf("https://www.loom.com/share/%s", match[1])
			matches = append(matches, shareURL)
		}
	}

	// Remove duplicates
	uniqueURLs := make(map[string]bool)
	var result []string
	for _, url := range matches {
		if !uniqueURLs[url] {
			uniqueURLs[url] = true
			result = append(result, url)
		}
	}

	return result
}

func scrapeWithLogin(config Config) ([]string, error) {
	ctx, cancel := setupBrowser(config.Headless)
	defer cancel()

	var currentURL string
	var loginSuccess bool

	fmt.Println("🔑 Attempting login with email and password...")

	// Navigate to the main Skool site
	if err := chromedp.Run(ctx, chromedp.Tasks{
		chromedp.Navigate(skoolBaseURL),
		chromedp.Sleep(initialWaitTime),
		chromedp.Location(&currentURL),
	}); err != nil {
		return nil, fmt.Errorf("failed to navigate to Skool: %v", err)
	}

	fmt.Println("📍 Landed on:", currentURL)

	// Try to find and click the login button
	err := chromedp.Run(ctx, chromedp.Tasks{
		chromedp.WaitVisible(`//button[@type="button"]/span[text()="Log In"]`, chromedp.BySearch),
		chromedp.Click(`//button[@type="button"]/span[text()="Log In"]`, chromedp.BySearch),
		chromedp.Sleep(2 * time.Second),
		chromedp.Location(&currentURL),
	})

	// If login button not found, navigate directly to login page
	if err != nil {
		fmt.Println("⚠️ Couldn't find login button, trying direct navigation to login page...")
		if err := chromedp.Run(ctx, chromedp.Tasks{
			chromedp.Navigate(skoolLoginURL),
			chromedp.Sleep(initialWaitTime),
			chromedp.Location(&currentURL),
		}); err != nil {
			return nil, fmt.Errorf("couldn't access login page: %v", err)
		}
	}

	fmt.Println("📍 Login page:", currentURL)

	// Complete the login form
	if err := chromedp.Run(ctx, chromedp.Tasks{
		chromedp.WaitVisible(`//input[@type="email" or @name="email" or contains(@placeholder, "email")]`, chromedp.BySearch),
		chromedp.SendKeys(`//input[@type="email" or @name="email" or contains(@placeholder, "email")]`, config.Email, chromedp.BySearch),

		chromedp.WaitVisible(`//input[@type="password" or @name="password" or contains(@placeholder, "password")]`, chromedp.BySearch),
		chromedp.SendKeys(`//input[@type="password" or @name="password" or contains(@placeholder, "password")]`, config.Password, chromedp.BySearch),

		chromedp.Click(`//button[@type="submit" and .//span[contains(text(), "Log") or contains(text(), "Log In") or contains(text(), "Login")]]`, chromedp.BySearch),

		chromedp.Sleep(loginWaitTime),
		chromedp.Location(&currentURL),
		chromedp.Evaluate(`!window.location.href.includes('/login') && !document.body.textContent.includes('Incorrect password') && !document.body.textContent.includes('No account found for this email.')`, &loginSuccess),
	}); err != nil {
		return nil, fmt.Errorf("login process failed: %v", err)
	}

	if !loginSuccess {
		return nil, fmt.Errorf("login failed: invalid credentials or captcha required")
	}

	fmt.Println("✅ Login successful! Redirected to:", currentURL)
	return navigateAndScrape(ctx, config.SkoolURL, config.WaitTime)
}

func scrapeWithCookies(config Config) ([]string, error) {
	ctx, cancel := setupBrowser(config.Headless)
	defer cancel()

	// Load and set cookies
	cookies, err := parseCookiesFile(config.CookiesFile)
	if err != nil {
		return nil, fmt.Errorf("error parsing cookies: %v", err)
	}

	// Log cookie info
	fmt.Println("🍪 Setting cookies...")
	for _, c := range cookies {
		if c.Name == "auth_token" && strings.Contains(c.Domain, "skool") {
			truncatedValue := c.Value
			if len(truncatedValue) > 20 {
				truncatedValue = truncatedValue[:20] + "..."
			}
			fmt.Printf("  🔑 Auth token found: %s\n", truncatedValue)
		}
	}

	// Enable network and set cookies
	if err := chromedp.Run(ctx, network.Enable()); err != nil {
		return nil, err
	}

	if err := chromedp.Run(ctx, network.SetCookies(cookies)); err != nil {
		return nil, fmt.Errorf("error setting cookies: %v", err)
	}

	var currentURL string
	// Set headers and navigate first to main site, then to target URL
	err = chromedp.Run(ctx, chromedp.Tasks{
		network.SetExtraHTTPHeaders(network.Headers{
			"Referer":         skoolBaseURL,
			"Accept":          "text/html,application/xhtml+xml,application/xml",
			"Accept-Language": "en-US,en;q=0.9",
			"Connection":      "keep-alive",
		}),
		chromedp.Navigate(skoolBaseURL),
		chromedp.Sleep(initialWaitTime),
		chromedp.Location(&currentURL),
	})

	if err != nil {
		return nil, fmt.Errorf("failed to navigate to main site: %v", err)
	}

	fmt.Printf("🌐 Initial navigation landed on: %s\n", currentURL)
	return navigateAndScrape(ctx, config.SkoolURL, config.WaitTime)
}

func navigateAndScrape(ctx context.Context, targetURL string, waitTime int) ([]string, error) {
	var currentURL, html string

	fmt.Println("🏫 Navigating to classroom:", targetURL)
	if err := chromedp.Run(ctx, chromedp.Tasks{
		chromedp.Navigate(targetURL),
		chromedp.Sleep(time.Duration(waitTime) * time.Second),
		chromedp.Location(&currentURL),
	}); err != nil {
		return nil, fmt.Errorf("failed to navigate to classroom: %v", err)
	}

	fmt.Println("📍 Landed on:", currentURL)

	// Check if we're on the right page
	if strings.Contains(currentURL, "/about") {
		return nil, fmt.Errorf("authentication succeeded but redirected to public page, check URL permissions")
	}

	// Get page content
	if err := chromedp.Run(ctx, chromedp.Tasks{
		chromedp.OuterHTML("html", &html),
	}); err != nil {
		return nil, err
	}

	// Extract and return video URLs
	urls := extractLoomURLs(html)
	if len(urls) == 0 {
		fmt.Println("⚠️ No videos found on the page.")
	}

	return urls, nil
}

// Cookie parsing functions
func parseCookiesFile(filePath string) ([]*network.CookieParam, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	// Determine file type based on extension and content
	isJSON := strings.HasSuffix(strings.ToLower(filePath), ".json")
	if !isJSON && !strings.HasSuffix(strings.ToLower(filePath), ".txt") {
		trimmed := strings.TrimSpace(string(content))
		isJSON = strings.HasPrefix(trimmed, "[") && strings.HasSuffix(trimmed, "]")
	}

	if isJSON {
		return parseJSONCookies(content)
	}
	return parseNetscapeCookies(content)
}

func parseJSONCookies(content []byte) ([]*network.CookieParam, error) {
	var jsonCookies []JSONCookie
	if err := json.Unmarshal(content, &jsonCookies); err != nil {
		return nil, fmt.Errorf("error parsing JSON cookies: %v", err)
	}

	var cookies []*network.CookieParam
	for _, c := range jsonCookies {
		// Clean up the host field (remove leading dot if present)
		domain := strings.TrimPrefix(c.Host, ".")

		cookie := &network.CookieParam{
			Domain:   domain,
			Name:     c.Name,
			Value:    c.Value,
			Path:     c.Path,
			Secure:   c.IsSecure == 1,
			HTTPOnly: c.IsHttpOnly == 1,
		}

		// Convert SameSite value
		switch c.SameSite {
		case 1:
			cookie.SameSite = network.CookieSameSiteLax
		case 2:
			cookie.SameSite = network.CookieSameSiteStrict
		case 3:
			cookie.SameSite = network.CookieSameSiteNone
		}

		// Add expiry if present
		if c.Expiry > 0 {
			t := cdp.TimeSinceEpoch(time.Unix(c.Expiry, 0))
			cookie.Expires = &t
		}

		cookies = append(cookies, cookie)
	}

	return cookies, nil
}

func parseNetscapeCookies(content []byte) ([]*network.CookieParam, error) {
	lines := strings.Split(string(content), "\n")
	var cookies []*network.CookieParam

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		fields := strings.Split(line, "\t")
		if len(fields) < 7 {
			continue
		}

		domain := strings.TrimPrefix(fields[0], ".")

		cookie := &network.CookieParam{
			Domain:   domain,
			Path:     fields[2],
			Secure:   fields[3] == "TRUE",
			Name:     fields[5],
			Value:    fields[6],
			HTTPOnly: false,
		}

		// Try to parse expiry if present
		if len(fields) > 4 {
			expiryStr := fields[4]
			if expiryStr != "" && expiryStr != "0" {
				expiry, err := parseInt64(expiryStr)
				if err == nil && expiry > 0 {
					t := cdp.TimeSinceEpoch(time.Unix(expiry, 0))
					cookie.Expires = &t
				}
			}
		}

		cookies = append(cookies, cookie)
	}

	return cookies, nil
}

func parseInt64(s string) (int64, error) {
	var result int64
	_, err := fmt.Sscanf(s, "%d", &result)
	return result, err
}

func downloadWithYtDlp(videoURL, cookiesFile, outputDir string) error {
	args := []string{
		"-o", filepath.Join(outputDir, "%(title)s.%(ext)s"),
		"--no-warnings",
		videoURL,
	}

	// Only add cookies argument if a cookies file is provided
	if cookiesFile != "" {
		tmpCookiesFile := cookiesFile
		isJSON := strings.HasSuffix(strings.ToLower(cookiesFile), ".json")

		if isJSON {
			tmpFile, err := convertJSONToNetscapeCookies(cookiesFile)
			if err != nil {
				return fmt.Errorf("error converting JSON cookies: %v", err)
			}
			defer os.Remove(tmpFile)
			tmpCookiesFile = tmpFile
		}

		// Add cookies argument only when we have a valid file
		args = append([]string{"--cookies", tmpCookiesFile}, args...)
	}

	cmd := exec.Command("yt-dlp", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

func convertJSONToNetscapeCookies(jsonFile string) (string, error) {
	content, err := os.ReadFile(jsonFile)
	if err != nil {
		return "", err
	}

	var jsonCookies []JSONCookie
	if err := json.Unmarshal(content, &jsonCookies); err != nil {
		return "", err
	}

	// Create temporary file
	tmpFile, err := os.CreateTemp("", "cookies-*.txt")
	if err != nil {
		return "", err
	}
	defer tmpFile.Close()

	// Write header
	fmt.Fprintln(tmpFile, "# Netscape HTTP Cookie File")
	fmt.Fprintln(tmpFile, "# This file was generated by skool-loom-dl")

	// Write cookies
	for _, c := range jsonCookies {
		host := c.Host
		if !strings.HasPrefix(host, ".") && strings.Count(host, ".") > 1 {
			host = "." + host
		}

		secure := "FALSE"
		if c.IsSecure == 1 {
			secure = "TRUE"
		}

		// Format: DOMAIN FLAG PATH SECURE EXPIRY NAME VALUE
		fmt.Fprintf(tmpFile, "%s\tTRUE\t%s\t%s\t%d\t%s\t%s\n",
			host, c.Path, secure, c.Expiry, c.Name, c.Value)
	}

	return tmpFile.Name(), nil
}
