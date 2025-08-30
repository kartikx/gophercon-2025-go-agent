package main

import (
	// "bufio"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"bufio"
)

func readFromCli() (string, error) {
	// fmt.Println("Coder Agent: Sleeping for 5 seconds")

	// time.Sleep(5 * time.Second)

	fmt.Printf("> ")
	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	return strings.TrimSpace(input), err
}

func writeToCli(message string) error {
	fmt.Println(message)
	return nil
}

// Coder-specific tools
var CoderTools = []ToolDefinition{
	ReadFileDefinition,
	WriteFileDefinition,
	ListFilesDefinition,
	InvokeDocumentationAgentDefinition,
}



// ReadFile tool for reading file contents
type ReadFileInput struct {
	Path string `json:"path" jsonschema_description:"The path of the file." jsonschema_default:"."`
}

var ReadFileInputSchema = GenerateSchema[ReadFileInput]()

var ReadFileDefinition = ToolDefinition{
	Name:        "read_file",
	Description: "Read the contents of a file. Use this when you want to see what is inside a file.",
	InputSchema: ReadFileInputSchema,
	Function:    ReadFile,
}

func ReadFile(input json.RawMessage) (string, error) {
	readFileInput := ReadFileInput{}

	err := json.Unmarshal(input, &readFileInput)
	if err != nil {
		return "", err
	}

	content, err := os.ReadFile(readFileInput.Path)
	if err != nil {
		return "", err
	}

	return string(content), nil
}

// WriteFile tool for writing content to files
type WriteFileInput struct {
	Path    string `json:"path" jsonschema_description:"The path of the file to write to"`
	Content string `json:"content" jsonschema_description:"The content to write to the file"`
}

var WriteFileInputSchema = GenerateSchema[WriteFileInput]()

var WriteFileDefinition = ToolDefinition{
	Name:        "write_file",
	Description: "Write content to a file. Use this when you need to create or modify files. The file will be created if it doesn't exist, or overwritten if it does.",
	InputSchema: WriteFileInputSchema,
	Function:    WriteFile,
}

func WriteFile(input json.RawMessage) (string, error) {
	writeFileInput := WriteFileInput{}

	err := json.Unmarshal(input, &writeFileInput)
	if err != nil {
		return "", err
	}

	err = os.WriteFile(writeFileInput.Path, []byte(writeFileInput.Content), 0644)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("Successfully wrote %d bytes to %s", len(writeFileInput.Content), writeFileInput.Path), nil
}

// ListFiles tool for listing directory contents (equivalent to ls -la)
type ListFilesInput struct {
	Path string `json:"path" jsonschema_description:"The directory path to list files from. Defaults to current directory if not specified." jsonschema_default:"."`
}

var ListFilesInputSchema = GenerateSchema[ListFilesInput]()

var ListFilesDefinition = ToolDefinition{
	Name:        "list_files",
	Description: "List all files and directories in a specified path (equivalent to ls -la). Use this to explore the file system structure.",
	InputSchema: ListFilesInputSchema,
	Function:    ListFiles,
}

func ListFiles(input json.RawMessage) (string, error) {
	listFilesInput := ListFilesInput{}

	err := json.Unmarshal(input, &listFilesInput)
	if err != nil {
		return "", err
	}

	// If no path specified, use current directory
	if listFilesInput.Path == "" {
		listFilesInput.Path = "."
	}

	// Read directory contents
	entries, err := os.ReadDir(listFilesInput.Path)
	if err != nil {
		return "", err
	}

	var result strings.Builder
	result.WriteString(fmt.Sprintf("Directory listing for: %s\n", listFilesInput.Path))
	result.WriteString("Permissions | Size | Modified | Name\n")
	result.WriteString("-----------|------|----------|-----\n")

	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil {
			continue // Skip entries we can't get info for
		}

		// Format permissions (similar to ls -la)
		mode := info.Mode()
		perms := formatPermissions(mode)
		
		// Format size
		size := formatSize(info.Size())
		
		// Format modification time
		modTime := info.ModTime().Format("Jan 02 15:04")
		
		// Format name (with @ for symlinks, / for directories)
		name := entry.Name()
		if entry.IsDir() {
			name += "/"
		} else if mode&os.ModeSymlink != 0 {
			name += "@"
		}

		result.WriteString(fmt.Sprintf("%s | %s | %s | %s\n", perms, size, modTime, name))
	}

	return result.String(), nil
}

// Helper function to format file permissions like ls -la
func formatPermissions(mode os.FileMode) string {
	var perms strings.Builder
	
	// File type
	switch {
	case mode.IsDir():
		perms.WriteRune('d')
	case mode&os.ModeSymlink != 0:
		perms.WriteRune('l')
	default:
		perms.WriteRune('-')
	}
	
	// Owner permissions
	if mode&0400 != 0 {
		perms.WriteRune('r')
	} else {
		perms.WriteRune('-')
	}
	if mode&0200 != 0 {
		perms.WriteRune('w')
	} else {
		perms.WriteRune('-')
	}
	if mode&0100 != 0 {
		perms.WriteRune('x')
	} else {
		perms.WriteRune('-')
	}
	
	// Group permissions
	if mode&0040 != 0 {
		perms.WriteRune('r')
	} else {
		perms.WriteRune('-')
	}
	if mode&0020 != 0 {
		perms.WriteRune('w')
	} else {
		perms.WriteRune('-')
	}
	if mode&0010 != 0 {
		perms.WriteRune('x')
	} else {
		perms.WriteRune('-')
	}
	
	// Other permissions
	if mode&0004 != 0 {
		perms.WriteRune('r')
	} else {
		perms.WriteRune('-')
	}
	if mode&0002 != 0 {
		perms.WriteRune('w')
	} else {
		perms.WriteRune('-')
	}
	if mode&0001 != 0 {
		perms.WriteRune('x')
	} else {
		perms.WriteRune('-')
	}
	
	return perms.String()
}

// Helper function to format file size
func formatSize(size int64) string {
	const unit = 1024
	if size < unit {
		return fmt.Sprintf("%d", size)
	}
	div, exp := int64(unit), 0
	for n := size / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f%c", float64(size)/float64(div), "KMGTPE"[exp])
}

// ExecuteCommand tool for running shell commands
type ExecuteCommandInput struct {
	Command string `json:"command" jsonschema_description:"The command to execute"`
}

var ExecuteCommandInputSchema = GenerateSchema[ExecuteCommandInput]()

var ExecuteCommandDefinition = ToolDefinition{
	Name:        "execute_command",
	Description: "Execute a shell command and return the output. Use this when you need to run terminal commands.",
	InputSchema: ExecuteCommandInputSchema,
	Function:    ExecuteCommand,
}

func ExecuteCommand(input json.RawMessage) (string, error) {
	readFileInput := ExecuteCommandInput{}

	err := json.Unmarshal(input, &readFileInput)
	if err != nil {
		return "", err
	}
	
	// Split the command into command and arguments
	parts := strings.Fields(readFileInput.Command)
	if len(parts) == 0 {
		return "", nil
	}
	
	cmd := exec.Command(parts[0], parts[1:]...)
	
	// Capture both stdout and stderr
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Sprintf("Command: %s\nOutput:\n%s", readFileInput.Command, string(output)), err
	}
	
	return fmt.Sprintf("Command: %s\nOutput:\n%s", readFileInput.Command, string(output)), nil
}


// Invoke documentation agent.
type InvokeDocumentationAgentInput struct {
	Query string `json:"query" jsonschema_description:"The query to search for in the documentation"`
}

var InvokeDocumentationAgentInputSchema = GenerateSchema[InvokeDocumentationAgentInput]()

var InvokeDocumentationAgentDefinition = ToolDefinition{
	Name:        "invoke_documentation_agent",
	Description: "Invoke the documentation agent to search for information. Use this when you need to find documentation for a specific package or function.",
	InputSchema: InvokeDocumentationAgentInputSchema,
	Function:    InvokeDocumentationAgent,
}

func InvokeDocumentationAgent(input json.RawMessage) (string, error) {
	invokeDocumentationAgentInput := InvokeDocumentationAgentInput{}

	err := json.Unmarshal(input, &invokeDocumentationAgentInput)
	if err != nil {
		return "", err
	}

	reqBody, err := json.Marshal(map[string]string{
		"query": invokeDocumentationAgentInput.Query,
	})
	if err != nil {
		return "", err
	}

	fmt.Println("Invoking documentation agent with query: ", invokeDocumentationAgentInput.Query)

	// Get doc agent URL from environment variable
	docAgentURL := os.Getenv("DOC_AGENT_URL")
	if docAgentURL == "" {
		docAgentURL = "http://localhost:8081" // default fallback
	}

	resp, err := http.Post(docAgentURL, "application/json", strings.NewReader(string(reqBody)))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("documentation agent returned status %d", resp.StatusCode)
	}

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}


	return string(respBytes), nil
}