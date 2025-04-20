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

func main() {
	// Define command line flags
	skoolURL := flag.String("url", "", "URL of the skool.com classroom to scrape (required)")
	cookiesFile := flag.String("cookies", "", "Path to cookies file (JSON or TXT) for authentication")
	email := flag.String("email", "", "Email for Skool login (alternative to cookies)")
	password := flag.String("password", "", "Password for Skool login (required with email)")
	outputDir := flag.String("output", "downloads", "Directory to save downloaded videos")
	waitTime := flag.Int("wait", 2, "Time to wait for page to load in seconds")
	headless := flag.Bool("headless", true, "Run in headless mode (no browser UI)")
	flag.Parse()

	// Validate required flags
	if *skoolURL == "" {
		fmt.Println("Usage: skool-loom-dl -url=https://yourschool.skool.com/classroom/path [-cookies=cookies.json | -email=user@example.com -password=pass]")
		os.Exit(1)
	}

	// Check authentication method
	usingEmail := *email != "" && *password != ""
	usingCookies := *cookiesFile != ""

	if !usingEmail && !usingCookies {
		fmt.Println("Error: You must provide either cookies file or email+password for authentication")
		os.Exit(1)
	}

	// Create output directory if it doesn't exist
	if err := os.MkdirAll(*outputDir, 0755); err != nil {
		log.Fatalf("Error creating output directory: %v", err)
	}

	fmt.Println("🔍 Scraping Loom videos from:", *skoolURL)

	var loomURLs []string
	var err error

	if usingEmail {
		loomURLs, err = scrapeWithLogin(*skoolURL, *email, *password, *waitTime, *headless)
	} else {
		loomURLs, err = scrapeWithCookies(*skoolURL, *cookiesFile, *waitTime, *headless)
	}

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
		if err := downloadWithYtDlp(url, *cookiesFile, *outputDir); err != nil {
			fmt.Printf("❌ Error: %v\n", err)
		}
	}

	fmt.Println("\n✅ Download process completed!")
}

// New function for email/password based login and scraping
func scrapeWithLogin(pageURL, email, password string, waitTime int, headless bool) ([]string, error) {
	// Setup browser
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", headless),
		chromedp.Flag("disable-gpu", true),
		chromedp.Flag("no-sandbox", true),
		chromedp.Flag("window-size", "1920,1080"),
		chromedp.UserAgent("Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36"),
	)

	allocCtx, cancel := chromedp.NewExecAllocator(context.Background(), opts...)
	defer cancel()

	ctx, cancel := chromedp.NewContext(allocCtx, chromedp.WithLogf(log.Printf))
	defer cancel()

	ctx, cancel = context.WithTimeout(ctx, 180*time.Second)
	defer cancel()

	// Variables to track state
	var currentURL string
	var html string
	var loginSuccess bool

	fmt.Println("🔑 Attempting login with email and password...")

	// First, navigate to the main Skool site
	err := chromedp.Run(ctx, chromedp.Tasks{
		chromedp.Navigate("https://www.skool.com/"),
		chromedp.Sleep(3 * time.Second),
		chromedp.Location(&currentURL),
	})

	if err != nil {
		return nil, fmt.Errorf("failed to navigate to Skool: %v", err)
	}

	fmt.Println("📍 Landed on:", currentURL)

	// Look for login button and click it
	err = chromedp.Run(ctx, chromedp.Tasks{
		chromedp.Click(`//button[@type="submit" and .//span[contains(text(), "Log") or contains(text(), "Log In") or contains(text(), "Login")]]`, chromedp.BySearch),
		chromedp.Sleep(3 * time.Second),
		chromedp.Location(&currentURL),
	})

	if err != nil {
		fmt.Println("⚠️ Couldn't find login button, trying direct navigation to login page...")
		// If we can't find a login button, try direct navigation to the login page
		err = chromedp.Run(ctx, chromedp.Tasks{
			chromedp.Navigate("https://www.skool.com/login"),
			chromedp.Sleep(3 * time.Second),
			chromedp.Location(&currentURL),
		})

		if err != nil {
			return nil, fmt.Errorf("couldn't access login page: %v", err)
		}
	}

	fmt.Println("📍 Login page:", currentURL)

	err = chromedp.Run(ctx, chromedp.Tasks{
		// Wait for the email input field to be visible
		chromedp.WaitVisible(`//input[@type="email" or @name="email" or contains(@placeholder, "email")]`, chromedp.BySearch),

		// Enter email
		chromedp.SendKeys(`//input[@type="email" or @name="email" or contains(@placeholder, "email")]`, email, chromedp.BySearch),

		// Wait for the password input field
		chromedp.WaitVisible(`//input[@type="password" or @name="password" or contains(@placeholder, "password")]`, chromedp.BySearch),

		// Enter password
		chromedp.SendKeys(`//input[@type="password" or @name="password" or contains(@placeholder, "password")]`, password, chromedp.BySearch),

		// Click submit/login button
		chromedp.Click(`//button[@type="submit" and .//span[contains(text(), "Log") or contains(text(), "Log In") or contains(text(), "Login")]]`, chromedp.BySearch),

		// Wait for navigation to complete
		chromedp.Sleep(5 * time.Second),

		// Check current URL
		chromedp.Location(&currentURL),

		// Determine if login was successful by checking if we're no longer on the login page
		chromedp.Evaluate(`!window.location.href.includes('/login') && !document.body.textContent.includes('Incorrect password') && !document.body.textContent.includes('No account found for this email.')`, &loginSuccess),
	})

	if err != nil {
		return nil, fmt.Errorf("login process failed: %v", err)
	}

	if !loginSuccess {
		return nil, fmt.Errorf("login failed: invalid credentials or captcha required")
	}

	fmt.Println("✅ Login successful! Redirected to:", currentURL)

	fmt.Println("🏫 Navigating to classroom:", pageURL)
	err = chromedp.Run(ctx, chromedp.Tasks{
		chromedp.Navigate(pageURL),
		chromedp.Sleep(time.Duration(waitTime) * time.Second),
		chromedp.Location(&currentURL),
	})

	if err != nil {
		return nil, fmt.Errorf("failed to navigate to classroom: %v", err)
	}

	fmt.Println("📍 Landed on:", currentURL)

	// Check if we're on the right page and scroll to load content
	if strings.Contains(currentURL, "/about") {
		return nil, fmt.Errorf("authentication succeeded but redirected to public page, check URL permissions")
	}

	// NOTE: not 100% sure if this is needed, but it seems to help with loading
	err = chromedp.Run(ctx, chromedp.Tasks{
		// Scroll to load lazy content
		chromedp.Evaluate(`
			function scrollDown() {
				window.scrollTo(0, document.body.scrollHeight/3);
				setTimeout(() => {
					window.scrollTo(0, document.body.scrollHeight*2/3);
					setTimeout(() => {
						window.scrollTo(0, document.body.scrollHeight);
					}, 1000);
				}, 1000);
			}
			scrollDown();
		`, nil),

		chromedp.Sleep(5 * time.Second),

		chromedp.OuterHTML("html", &html),
	})

	if err != nil {
		return nil, err
	}

	// Find Loom URLs
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

	if len(result) == 0 {
		fmt.Println("⚠️ No videos found. Contents of the page:")
		fmt.Println(html[:1000] + "...")
	}

	return result, nil
}

func scrapeWithCookies(pageURL, cookiesFile string, waitTime int, headless bool) ([]string, error) {
	// Setup browser with debugging options
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", headless),
		chromedp.Flag("disable-gpu", true),
		chromedp.Flag("no-sandbox", true),
		chromedp.Flag("window-size", "1920,1080"),
		chromedp.UserAgent("Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36"),
	)

	allocCtx, cancel := chromedp.NewExecAllocator(context.Background(), opts...)
	defer cancel()

	ctx, cancel := chromedp.NewContext(allocCtx, chromedp.WithLogf(log.Printf))
	defer cancel()

	ctx, cancel = context.WithTimeout(ctx, 180*time.Second)
	defer cancel()

	// Load and set cookies
	cookies, err := parseCookiesFile(cookiesFile)
	if err != nil {
		return nil, fmt.Errorf("error parsing cookies: %v", err)
	}

	// Debug cookie values before setting
	fmt.Println("🍪 Setting these cookies:")
	for _, c := range cookies {
		fmt.Printf("  Domain: %s, Name: %s, Path: %s, Secure: %v, HttpOnly: %v, SameSite: %v\n",
			c.Domain, c.Name, c.Path, c.Secure, c.HTTPOnly, c.SameSite)

		// Special handling for Skool auth token
		if c.Name == "auth_token" && strings.Contains(c.Domain, "skool") {
			fmt.Printf("  🔑 Auth token found: %s...\n", c.Value[:20])
		}
	}

	if err := chromedp.Run(ctx, network.Enable()); err != nil {
		return nil, err
	}

	if err := chromedp.Run(ctx, network.SetCookies(cookies)); err != nil {
		return nil, fmt.Errorf("error setting cookies: %v", err)
	}

	var currentURL string
	var html string

	// Add auth headers and try multi-step navigation approach
	err = chromedp.Run(ctx, chromedp.Tasks{
		network.SetExtraHTTPHeaders(network.Headers{
			"Referer":         "https://www.skool.com/",
			"Accept":          "text/html,application/xhtml+xml,application/xml",
			"Accept-Language": "en-US,en;q=0.9",
			"Connection":      "keep-alive",
		}),

		// Navigate to the main site first
		chromedp.Navigate("https://www.skool.com/"),
		chromedp.Sleep(5 * time.Second),

		// Get the current location to see if we're logged in
		chromedp.Location(&currentURL),
		chromedp.ActionFunc(func(ctx context.Context) error {
			fmt.Printf("🌐 Initial navigation landed on: %s\n", currentURL)
			return nil
		}),

		chromedp.Navigate(pageURL),
		chromedp.Sleep(time.Duration(waitTime) * time.Second),

		// Check where we landed
		chromedp.Location(&currentURL),
		chromedp.ActionFunc(func(ctx context.Context) error {
			fmt.Printf("🌐 After classroom navigation, landed on: %s\n", currentURL)
			return nil
		}),

		// Execute JavaScript to scroll through the page to load lazy content
		chromedp.Evaluate(`
			function scrollDown() {
				window.scrollTo(0, document.body.scrollHeight/3);
				setTimeout(() => {
					window.scrollTo(0, document.body.scrollHeight*2/3);
					setTimeout(() => {
						window.scrollTo(0, document.body.scrollHeight);
					}, 1000);
				}, 1000);
			}
			scrollDown();
		`, nil),

		chromedp.Sleep(3 * time.Second),

		chromedp.OuterHTML("html", &html),
	})

	if err != nil {
		return nil, err
	}

	// Check if we're on the about page (indicating auth failed)
	if strings.Contains(currentURL, "/about") || strings.Contains(html, "Sign up") || strings.Contains(html, "Log in") {
		fmt.Println("⚠️ WARNING: Authentication appears to have failed - landed on public page")
		fmt.Println("⚠️ Make sure your cookies are valid and the auth_token cookie is present")
	}

	// Find Loom URLs
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

	return result, nil
}

// parseCookiesFile detects and parses both JSON and TXT cookie formats
func parseCookiesFile(filePath string) ([]*network.CookieParam, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	// Check if this is a JSON file by extension
	isJSON := strings.HasSuffix(strings.ToLower(filePath), ".json")

	// If we're not sure by extension, check content
	if !isJSON && !strings.HasSuffix(strings.ToLower(filePath), ".txt") {
		trimmed := strings.TrimSpace(string(content))
		isJSON = strings.HasPrefix(trimmed, "[") && strings.HasSuffix(trimmed, "]")
	}

	if isJSON {
		return parseJSONCookies(content)
	}
	return parseNetscapeCookies(content)
}

// parseJSONCookies parses cookies in JSON format
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

// parseNetscapeCookies parses cookies in Netscape cookies.txt format
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
				expiry, err := parseExpiry(expiryStr)
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

func parseExpiry(s string) (int64, error) {
	expiry, err := parseInt64(s)
	if err != nil {
		return 0, err
	}
	return expiry, nil
}

func parseInt64(s string) (int64, error) {
	var result int64
	_, err := fmt.Sscanf(s, "%d", &result)
	return result, err
}

func downloadWithYtDlp(videoURL, cookiesFile, outputDir string) error {
	// Create temporary cookies.txt file if input is JSON
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

	args := []string{
		"--cookies", tmpCookiesFile,
		"-o", filepath.Join(outputDir, "%(title)s.%(ext)s"),
		"--no-warnings",
		videoURL,
	}

	cmd := exec.Command("yt-dlp", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

// convertJSONToNetscapeCookies converts JSON cookies to Netscape format for yt-dlp
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
