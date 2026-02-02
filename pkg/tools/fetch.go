package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"chewbacca/internal/util"

	"golang.org/x/net/html"
)

type FetchInput struct {
	URL            string   `json:"url" jsonschema_description:"URL to fetch content from"`
	Description    string   `json:"description,omitempty" jsonschema_description:"Why I'm fetching this URL"`
	Timeout        int      `json:"timeout,omitempty" jsonschema_description:"Timeout in seconds (default: 30)"`
	MaxLength      int      `json:"max_length,omitempty" jsonschema_description:"Maximum content length to return (default: 50000)"`
	ExtractText    bool     `json:"extract_text,omitempty" jsonschema_description:"Extract text content only, removing HTML tags (default: true)"`
	AllowedDomains []string `json:"allowed_domains,omitempty" jsonschema_description:"List of allowed domains"`
	BlockedDomains []string `json:"blocked_domains,omitempty" jsonschema_description:"List of blocked domains"`
}

type FetchResult struct {
	URL         string            `json:"url"`
	StatusCode  int               `json:"status_code"`
	ContentType string            `json:"content_type"`
	Title       string            `json:"title,omitempty"`
	Content     string            `json:"content"`
	Length      int               `json:"length"`
	TimeTaken   string            `json:"time_taken"`
	Headers     map[string]string `json:"headers,omitempty"`
	Error       string            `json:"error,omitempty"`
}

func Fetch(input json.RawMessage) (string, error) {
	fetchInput := FetchInput{
		Timeout:     util.GetTimeoutOrDefault(0, 30),
		MaxLength:   50000,
		ExtractText: true,
	}

	err := json.Unmarshal(input, &fetchInput)
	if err != nil {
		return "", fmt.Errorf("failed to parse input: %w", err)
	}

	if fetchInput.URL == "" {
		return "", fmt.Errorf("url is required")
	}

	parsedURL, err := url.Parse(fetchInput.URL)
	if err != nil {
		return "", fmt.Errorf("invalid URL: %w", err)
	}

	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return "", fmt.Errorf("only http and https URLs are supported")
	}

	if err := checkDomainRestrictions(parsedURL.Host, fetchInput.AllowedDomains, fetchInput.BlockedDomains); err != nil {
		return "", err
	}

	result := fetchURL(fetchInput)

	jsonResult, _ := json.Marshal(result)
	return string(jsonResult), nil
}

func checkDomainRestrictions(host string, allowed, blocked []string) error {
	for _, domain := range blocked {
		if strings.Contains(host, domain) {
			return fmt.Errorf("domain %s is blocked", host)
		}
	}

	if len(allowed) > 0 {
		isAllowed := false
		for _, domain := range allowed {
			if strings.Contains(host, domain) {
				isAllowed = true
				break
			}
		}
		if !isAllowed {
			return fmt.Errorf("domain %s is not in allowed list", host)
		}
	}

	return nil
}

func fetchURL(input FetchInput) *FetchResult {
	result := &FetchResult{
		URL:     input.URL,
		Headers: make(map[string]string),
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(input.Timeout)*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", input.URL, nil)
	if err != nil {
		result.Error = fmt.Sprintf("failed to create request: %v", err)
		return result
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; ChewbaccaBot/1.0)")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.5")

	client := &http.Client{
		Timeout: time.Duration(input.Timeout) * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 10 {
				return fmt.Errorf("too many redirects")
			}
			return nil
		},
	}

	start := time.Now()
	resp, err := client.Do(req)
	elapsed := time.Since(start)
	result.TimeTaken = elapsed.Round(time.Millisecond).String()

	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			result.Error = fmt.Sprintf("request timed out after %d seconds", input.Timeout)
		} else {
			result.Error = fmt.Sprintf("request failed: %v", err)
		}
		return result
	}
	defer resp.Body.Close()

	result.StatusCode = resp.StatusCode
	result.ContentType = resp.Header.Get("Content-Type")

	for _, header := range []string{"Content-Type", "Content-Length", "Last-Modified", "Cache-Control"} {
		if val := resp.Header.Get(header); val != "" {
			result.Headers[header] = val
		}
	}

	limitedReader := io.LimitReader(resp.Body, int64(input.MaxLength))
	body, err := io.ReadAll(limitedReader)
	if err != nil {
		result.Error = fmt.Sprintf("failed to read response: %v", err)
		return result
	}

	content := string(body)

	if input.ExtractText && strings.Contains(result.ContentType, "text/html") {
		result.Title = extractTitle(content)
		content = extractTextFromHTML(content)
		content = cleanText(content)
	}

	if len(content) > input.MaxLength {
		content = content[:input.MaxLength] + "\n\n[Content truncated...]"
	}

	result.Content = content
	result.Length = len(content)

	return result
}

func extractTitle(htmlContent string) string {
	doc, err := html.Parse(strings.NewReader(htmlContent))
	if err != nil {
		return ""
	}

	var title string
	var findTitle func(*html.Node)
	findTitle = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "title" {
			if n.FirstChild != nil {
				title = n.FirstChild.Data
			}
			return
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			findTitle(c)
		}
	}
	findTitle(doc)

	return strings.TrimSpace(title)
}

func extractTextFromHTML(htmlContent string) string {
	doc, err := html.Parse(strings.NewReader(htmlContent))
	if err != nil {
		return htmlContent
	}

	var sb strings.Builder
	var extractText func(*html.Node)

	skipTags := map[string]bool{
		"script": true, "style": true, "noscript": true,
		"header": false, "footer": false, "nav": true,
		"aside": true, "iframe": true, "svg": true,
	}

	blockTags := map[string]bool{
		"p": true, "div": true, "br": true, "h1": true, "h2": true,
		"h3": true, "h4": true, "h5": true, "h6": true, "li": true,
		"tr": true, "blockquote": true, "pre": true, "article": true,
		"section": true, "main": true,
	}

	extractText = func(n *html.Node) {
		if n.Type == html.ElementNode {
			if skipTags[n.Data] {
				return
			}
		}

		if n.Type == html.TextNode {
			text := strings.TrimSpace(n.Data)
			if text != "" {
				sb.WriteString(text)
				sb.WriteString(" ")
			}
		}

		for c := n.FirstChild; c != nil; c = c.NextSibling {
			extractText(c)
		}

		if n.Type == html.ElementNode && blockTags[n.Data] {
			sb.WriteString("\n")
		}
	}

	extractText(doc)
	return sb.String()
}

func cleanText(text string) string {
	spaceRegex := regexp.MustCompile(`[ \t]+`)
	text = spaceRegex.ReplaceAllString(text, " ")

	newlineRegex := regexp.MustCompile(`\n{3,}`)
	text = newlineRegex.ReplaceAllString(text, "\n\n")

	lines := strings.Split(text, "\n")
	var cleaned []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			cleaned = append(cleaned, line)
		}
	}

	return strings.Join(cleaned, "\n")
}

var FetchDefinition = ToolDefinition{
	Name: "Fetch",
	Description: `Fetch the contents of a web page at a given URL.
Extracts text content from HTML pages by default.
Use this to retrieve information from websites, documentation, or APIs.`,
	InputSchema: GenerateSchema[FetchInput](),
	Function:    Fetch,
}
