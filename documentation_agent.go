package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"io"
	"time"
	"strings"
	"golang.org/x/net/html"
	"github.com/PuerkitoBio/goquery"
)

// Documentation-specific tools
var DocTools = []ToolDefinition{
	SearchGoDocumentationDefinition,
}

// SearchGoDocumentation tool for searching Go documentation
type SearchGoDocumentationInput struct {
	PackageName string `json:"package_name" jsonschema_description:"The name of the package to search for"`
}

var SearchGoDocumentationInputSchema = GenerateSchema[SearchGoDocumentationInput]()

var SearchGoDocumentationDefinition = ToolDefinition{
	Name:        "search_go_documentation",
	Description: "Search Go documentation for information. Use this when you need to find Go language features, standard library functions, or Go-specific information. Call this function with the name of the package you want to search for.",
	InputSchema: SearchGoDocumentationInputSchema,
	Function:    SearchGoDocumentation,
}

// SearchGoDocumentation fetches documentation text from pkg.go.dev for a given package
func SearchGoDocumentation(input json.RawMessage) (string, error) {
	searchInput := SearchGoDocumentationInput{}

	err := json.Unmarshal(input, &searchInput)
	if err != nil {
		return "", err
	}

    url := fmt.Sprintf("https://pkg.go.dev/%s?tab=doc", searchInput.PackageName)
    resp, err := http.Get(url)
    if err != nil {
        return "", fmt.Errorf("failed to fetch package docs: %v", err)
    }
    defer resp.Body.Close()

    if resp.StatusCode != 200 {
        return "", fmt.Errorf("failed to fetch package docs: status %d", resp.StatusCode)
    }

    doc, err := goquery.NewDocumentFromReader(resp.Body)
    if err != nil {
        return "", fmt.Errorf("failed to parse HTML: %v", err)
    }

    // Select the documentation section
    docText := ""
    docSelection := doc.Find(".Documentation-overview")
    if docSelection.Length() == 0 {
        return "", fmt.Errorf("documentation section not found")
    }

    docSelection.Each(func(i int, s *goquery.Selection) {
        docText += s.Text() + "\n"
    })

    return docText, nil
}

// GetPackageContents takes a package name from pkg.go.dev and returns its contents
func GetPackageContents(packageName string) string {
	// Construct the URL for the package
	url := fmt.Sprintf("https://pkg.go.dev/%s", packageName)
	
	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: 30 * time.Second,
	}
	
	// Make HTTP request
	resp, err := client.Get(url)
	if err != nil {
		return fmt.Sprintf("Error fetching package: %v", err)
	}
	defer resp.Body.Close()
	
	// Check if request was successful
	if resp.StatusCode != http.StatusOK {
		return fmt.Sprintf("HTTP error: %s", resp.Status)
	}
	
	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Sprintf("Error reading response: %v", err)
	}
	
	// Parse HTML and extract package information
	content := extractPackageInfo(string(body), packageName)
	return content
}

// extractPackageInfo parses HTML and extracts relevant package information
func extractPackageInfo(htmlContent, packageName string) string {
	doc, err := html.Parse(strings.NewReader(htmlContent))
	if err != nil {
		return fmt.Sprintf("Error parsing HTML: %v", err)
	}
	
	var content strings.Builder
	content.WriteString(fmt.Sprintf("Package: %s\n\n", packageName))
	
	// Extract package description from meta description
	description := extractMetaDescription(doc)
	if description != "" {
		content.WriteString(fmt.Sprintf("Description: %s\n\n", description))
	}
	
	// Extract overview content
	overview := extractTextByClass(doc, "Documentation-overview")
	if overview != "" {
		content.WriteString(fmt.Sprintf("Overview:\n%s\n\n", overview))
	}
	
	// Extract import path from canonical link
	importPath := extractCanonicalLink(doc)
	if importPath != "" {
		content.WriteString(fmt.Sprintf("Import Path: %s\n\n", importPath))
	}
	
	// Extract function/type information
	functions := extractFunctions(doc)
	if functions != "" {
		content.WriteString(fmt.Sprintf("Functions and Types:\n%s\n", functions))
	}
	
	result := content.String()
	if strings.TrimSpace(result) == fmt.Sprintf("Package: %s", packageName) {
		return fmt.Sprintf("Package: %s\n\nNo detailed information found. The package may not exist or may be private.", packageName)
	}
	
	return result
}

// extractMetaDescription extracts the package description from meta description tag
func extractMetaDescription(n *html.Node) string {
	var traverse func(*html.Node)
	var description string
	
	traverse = func(node *html.Node) {
		if node.Type == html.ElementNode && node.Data == "meta" {
			var name, content string
			for _, attr := range node.Attr {
				if attr.Key == "name" && attr.Val == "Description" {
					name = attr.Val
				}
				if attr.Key == "content" {
					content = attr.Val
				}
			}
			if name == "Description" && content != "" {
				description = content
				return
			}
		}
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			traverse(child)
		}
	}
	
	traverse(n)
	return description
}

// extractCanonicalLink extracts the canonical import path
func extractCanonicalLink(n *html.Node) string {
	var traverse func(*html.Node)
	var canonical string
	
	traverse = func(node *html.Node) {
		if node.Type == html.ElementNode && node.Data == "link" {
			for _, attr := range node.Attr {
				if attr.Key == "rel" && attr.Val == "canonical" {
					for _, attr2 := range node.Attr {
						if attr2.Key == "href" {
							canonical = attr2.Val
							return
						}
					}
				}
			}
		}
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			traverse(child)
		}
	}
	
	traverse(n)
	// Extract just the package path from the full URL
	if canonical != "" {
		if strings.HasPrefix(canonical, "https://pkg.go.dev/") {
			return strings.TrimPrefix(canonical, "https://pkg.go.dev/")
		}
		return canonical
	}
	return ""
}

// extractTextByClass finds elements with specific CSS classes and extracts their text
func extractTextByClass(n *html.Node, className string) string {
	var result strings.Builder
	
	var traverse func(*html.Node)
	traverse = func(node *html.Node) {
		if node.Type == html.ElementNode {
			for _, attr := range node.Attr {
				if attr.Key == "class" && strings.Contains(attr.Val, className) {
					// Extract text from this element and its children
					extractText(node, &result)
					return
				}
			}
		}
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			traverse(child)
		}
	}
	
	traverse(n)
	return strings.TrimSpace(result.String())
}

// extractText recursively extracts text from HTML nodes
func extractText(n *html.Node, result *strings.Builder) {
	if n.Type == html.TextNode {
		text := strings.TrimSpace(n.Data)
		if text != "" {
			if result.Len() > 0 {
				result.WriteString(" ")
			}
			result.WriteString(text)
		}
	}
	for child := n.FirstChild; child != nil; child = child.NextSibling {
		extractText(child, result)
	}
}

// extractFunctions extracts function and type information from the package
func extractFunctions(n *html.Node) string {
	var result strings.Builder
	
	// First try to extract from the index section
	indexFunctions := extractIndexFunctions(n)
	if indexFunctions != "" {
		result.WriteString("Functions and Types:\n")
		result.WriteString(indexFunctions)
		return result.String()
	}
	
	// Fallback to the old method
	var traverse func(*html.Node)
	traverse = func(node *html.Node) {
		if node.Type == html.ElementNode && node.Data == "div" {
			for _, attr := range node.Attr {
				if attr.Key == "class" && (strings.Contains(attr.Val, "Documentation-function") || 
					strings.Contains(attr.Val, "Documentation-type")) {
					// Extract function/type name and signature
					name := extractFunctionName(node)
					if name != "" {
						result.WriteString(fmt.Sprintf("- %s\n", name))
					}
					break
				}
			}
		}
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			traverse(child)
		}
	}
	
	traverse(n)
	return result.String()
}

// extractIndexFunctions extracts functions from the Documentation-index section
func extractIndexFunctions(n *html.Node) string {
	var result strings.Builder
	
	var traverse func(*html.Node)
	traverse = func(node *html.Node) {
		if node.Type == html.ElementNode && node.Data == "section" {
			for _, attr := range node.Attr {
				if attr.Key == "class" && strings.Contains(attr.Val, "Documentation-index") {
					// Found the index section, now extract functions and types
					extractIndexItems(node, &result)
					return
				}
			}
		}
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			traverse(child)
		}
	}
	
	traverse(n)
	return result.String()
}

// extractIndexItems extracts individual function and type items from the index
func extractIndexItems(n *html.Node, result *strings.Builder) {
	var traverse func(*html.Node)
	traverse = func(node *html.Node) {
		if node.Type == html.ElementNode && node.Data == "li" {
			for _, attr := range node.Attr {
				if attr.Key == "class" && (strings.Contains(attr.Val, "Documentation-indexFunction") || 
					strings.Contains(attr.Val, "Documentation-indexType")) {
					// Extract the function/type information
					text := extractTextFromNode(node)
					if text != "" {
						result.WriteString(fmt.Sprintf("- %s\n", text))
					}
					break
				}
			}
		}
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			traverse(child)
		}
	}
	
	traverse(n)
}

// extractTextFromNode extracts clean text from a node
func extractTextFromNode(n *html.Node) string {
	var result strings.Builder
	extractText(n, &result)
	return strings.TrimSpace(result.String())
}

// extractFunctionName extracts the name of a function or type
func extractFunctionName(n *html.Node) string {
	var result strings.Builder
	
	var traverse func(*html.Node)
	traverse = func(node *html.Node) {
		if node.Type == html.ElementNode && node.Data == "h3" {
			extractText(node, &result)
			return
		}
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			traverse(child)
		}
	}
	
	traverse(n)
	return strings.TrimSpace(result.String())
}
