package main

import (
	"bufio"
	"crypto/tls"
	"flag"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"
)

type Result struct {
	Subdomain  string
	Status     int
	URL        string
	IsLive     bool
	Error      string
}

func main() {
	// Command line flags
	inputFile := flag.String("f", "", "Path to the subdomains file (required)")
	outputFile := flag.String("o", "", "Path to save live subdomains (optional)")
	threads := flag.Int("t", 50, "Number of concurrent threads")
	timeout := flag.Int("timeout", 5, "HTTP request timeout in seconds")
	flag.Parse()

	if *inputFile == "" {
		fmt.Println("Usage: ./subcheck -f subdomains.txt [-o output.txt] [-t 50] [-timeout 5]")
		os.Exit(1)
	}

	// Open the input file
	file, err := os.Open(*inputFile)
	if err != nil {
		fmt.Printf("[-] Error opening file: %v\n", err)
		os.Exit(1)
	}
	defer file.Close()

	// Read and validate subdomains (remove duplicates and invalid entries)
	subdomains := readAndValidateSubdomains(file)
	if len(subdomains) == 0 {
		fmt.Println("[-] No valid subdomains found in the input file")
		os.Exit(1)
	}

	fmt.Printf("[*] Found %d unique subdomains to check\n", len(subdomains))

	// Create a shared HTTP client with connection pooling
	client := &http.Client{
		Timeout: time.Duration(*timeout) * time.Second,
		Transport: &http.Transport{
			MaxIdleConns:        100,
			MaxIdleConnsPerHost: 100,
			IdleConnTimeout:     90 * time.Second,
			TLSClientConfig:     &tls.Config{
				InsecureSkipVerify: false, // Enable proper TLS verification
			},
		},
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	// Channels and WaitGroup for concurrency
	jobs := make(chan string, *threads)
	results := make(chan Result, len(subdomains))
	var wg sync.WaitGroup

	fmt.Printf("[*] Starting liveness check with %d threads...\n", *threads)

	// Start worker goroutines
	for i := 0; i < *threads; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for sub := range jobs {
				result := checkLive(sub, client)
				results <- result
			}
		}()
	}

	// Send subdomains to jobs channel
	for _, sub := range subdomains {
		jobs <- sub
	}
	close(jobs)

	// Wait for all workers to finish
	go func() {
		wg.Wait()
		close(results)
	}()

	// Process results and optionally write to output file
	var outputWriter *bufio.Writer
	if *outputFile != "" {
		outFile, err := os.Create(*outputFile)
		if err != nil {
			fmt.Printf("[-] Error creating output file: %v\n", err)
			os.Exit(1)
		}
		defer outFile.Close()
		outputWriter = bufio.NewWriter(outFile)
		defer outputWriter.Flush()
	}

	liveCount := 0
	for result := range results {
		if result.IsLive {
			liveCount++
			fmt.Printf("[+] Live: %-40s | Status: %d | URL: %s\n", result.Subdomain, result.Status, result.URL)
			if outputWriter != nil {
				outputWriter.WriteString(result.Subdomain + "\n")
			}
		} else {
			fmt.Printf("[-] Dead: %-40s | Error: %s\n", result.Subdomain, result.Error)
		}
	}

	fmt.Printf("\n[*] Scan completed. Found %d live subdomains out of %d\n", liveCount, len(subdomains))
	if *outputFile != "" {
		fmt.Printf("[*] Results saved to: %s\n", *outputFile)
	}
}

func readAndValidateSubdomains(file *os.File) []string {
	subdomainMap := make(map[string]bool)
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		
		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Remove protocol if present (http:// or https://)
		line = strings.TrimPrefix(line, "http://")
		line = strings.TrimPrefix(line, "https://")
		
		// Remove trailing slashes and paths
		if idx := strings.Index(line, "/"); idx != -1 {
			line = line[:idx]
		}

		// Validate subdomain format
		if isValidSubdomain(line) && !subdomainMap[line] {
			subdomainMap[line] = true
		}
	}

	if err := scanner.Err(); err != nil {
		fmt.Printf("[-] Error reading file: %v\n", err)
	}

	// Convert map to slice
	subdomains := make([]string, 0, len(subdomainMap))
	for sub := range subdomainMap {
		subdomains = append(subdomains, sub)
	}

	return subdomains
}

func isValidSubdomain(subdomain string) bool {
	// Basic validation: must contain at least one dot and valid characters
	if !strings.Contains(subdomain, ".") {
		return false
	}

	// Check if it's a valid URL format
	_, err := url.Parse("http://" + subdomain)
	if err != nil {
		return false
	}

	// Check for invalid characters
	for _, char := range subdomain {
		if !((char >= 'a' && char <= 'z') || (char >= '0' && char <= '9') || char == '.' || char == '-') {
			return false
		}
	}

	return true
}

func checkLive(subdomain string, client *http.Client) Result {
	// Try both HTTPS and HTTP
	urls := []string{
		"https://" + subdomain,
		"http://" + subdomain,
	}

	for _, urlStr := range urls {
		req, err := http.NewRequest("GET", urlStr, nil)
		if err != nil {
			continue
		}

		// Set a standard User-Agent
		req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/115.0.0.0 Safari/537.36")

		resp, err := client.Do(req)
		if err == nil {
			resp.Body.Close()
			return Result{
				Subdomain: subdomain,
				Status:    resp.StatusCode,
				URL:       urlStr,
				IsLive:    true,
				Error:     "",
			}
		}

		// If HTTPS fails due to TLS error, try HTTP
		if strings.Contains(err.Error(), "certificate") || strings.Contains(err.Error(), "x509") {
			continue
		}

		// For other errors, return immediately
		return Result{
			Subdomain: subdomain,
			Status:    0,
			URL:       "",
			IsLive:    false,
			Error:     err.Error(),
		}
	}

	return Result{
		Subdomain: subdomain,
		Status:    0,
		URL:       "",
		IsLive:    false,
		Error:     "Both HTTPS and HTTP failed",
	}
}
