package terraform

import (
	"archive/tar"
	"bufio"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// CxOneConfig represents the Checkmarx One configuration from variables.tf
type CxOneConfig struct {
	ProjectName string
	Group       string
	Branch      string
}

// DefaultCxOneConfig returns a default configuration
func DefaultCxOneConfig() CxOneConfig {
	return CxOneConfig{
		ProjectName: "hcp-terraform-project",
		Group:       "hcp-tasks",
		Branch:      "main",
	}
}

// extractTarGz extracts a tar.gz file to a temporary directory
func extractTarGz(tarGzPath string) (string, error) {
	// Create temporary directory for extraction
	tempDir, err := os.MkdirTemp("", "terraform-workspace-*")
	if err != nil {
		return "", fmt.Errorf("failed to create temp directory: %w", err)
	}

	// Open the tar.gz file
	file, err := os.Open(tarGzPath)
	if err != nil {
		os.RemoveAll(tempDir)
		return "", fmt.Errorf("failed to open tar.gz file: %w", err)
	}
	defer file.Close()

	// Create gzip reader
	gzReader, err := gzip.NewReader(file)
	if err != nil {
		os.RemoveAll(tempDir)
		return "", fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer gzReader.Close()

	// Create tar reader
	tarReader := tar.NewReader(gzReader)

	// Extract files
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			os.RemoveAll(tempDir)
			return "", fmt.Errorf("failed to read tar header: %w", err)
		}

		targetPath := filepath.Join(tempDir, header.Name)

		// Ensure the target directory exists
		if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
			os.RemoveAll(tempDir)
			return "", fmt.Errorf("failed to create directory: %w", err)
		}

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(targetPath, 0755); err != nil {
				os.RemoveAll(tempDir)
				return "", fmt.Errorf("failed to create directory: %w", err)
			}
		case tar.TypeReg:
			outFile, err := os.Create(targetPath)
			if err != nil {
				os.RemoveAll(tempDir)
				return "", fmt.Errorf("failed to create file: %w", err)
			}

			if _, err := io.Copy(outFile, tarReader); err != nil {
				outFile.Close()
				os.RemoveAll(tempDir)
				return "", fmt.Errorf("failed to copy file: %w", err)
			}
			outFile.Close()
		}
	}

	return tempDir, nil
}

// ParseCxOneConfig parses the variables.tf file to extract CxOne configuration
// Accepts either a directory path or a tar.gz file path
func ParseCxOneConfig(inputPath string) (CxOneConfig, error) {
	config := DefaultCxOneConfig()

	var terraformDir string
	var cleanupRequired bool

	// Check if inputPath is a tar.gz file or directory
	if strings.HasSuffix(inputPath, ".tar.gz") || strings.HasSuffix(inputPath, ".tgz") {
		// Extract tar.gz to temporary directory
		var err error
		terraformDir, err = extractTarGz(inputPath)
		if err != nil {
			return config, fmt.Errorf("failed to extract tar.gz: %w", err)
		}
		cleanupRequired = true
		defer func() {
			if cleanupRequired {
				os.RemoveAll(terraformDir)
			}
		}()
	} else {
		// Assume it's a directory path
		terraformDir = inputPath
	}

	// Look for variables.tf files in the extracted/provided directory
	variablesFiles, err := findVariablesFiles(terraformDir)
	if err != nil {
		return config, fmt.Errorf("failed to find variables files: %w", err)
	}

	if len(variablesFiles) == 0 {
		// No variables files found, return default config
		return config, nil
	}

	// Parse all variables files
	for _, variablesFile := range variablesFiles {
		fileConfig, err := parseVariablesFile(variablesFile)
		if err != nil {
			// Log warning but continue with other files
			continue
		}

		// Merge configuration (last file wins for conflicts)
		if fileConfig.ProjectName != "" {
			config.ProjectName = fileConfig.ProjectName
		}
		if fileConfig.Group != "" {
			config.Group = fileConfig.Group
		}
		if fileConfig.Branch != "" {
			config.Branch = fileConfig.Branch
		}
	}

	return config, nil
}

// findVariablesFiles recursively finds all variables.tf files in a directory
func findVariablesFiles(rootDir string) ([]string, error) {
	var variablesFiles []string

	err := filepath.Walk(rootDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() && (strings.HasSuffix(info.Name(), "variables.tf") || info.Name() == "vars.tf") {
			variablesFiles = append(variablesFiles, path)
		}

		return nil
	})

	return variablesFiles, err
}

// parseVariablesFile parses a single variables.tf file
func parseVariablesFile(variablesFile string) (CxOneConfig, error) {
	config := CxOneConfig{}

	file, err := os.Open(variablesFile)
	if err != nil {
		return config, fmt.Errorf("failed to open variables file %s: %v", variablesFile, err)
	}
	defer file.Close()

	var currentVariable string
	var inVariableBlock bool
	var inDefaultBlock bool

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "//") {
			continue
		}

		// Check for variable declarations
		if strings.HasPrefix(line, "variable ") {
			// Extract variable name
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				varName := strings.Trim(parts[1], `"`)
				if varName == "checkmarx_project_name" || varName == "checkmarx_group" || varName == "checkmarx_branch" {
					currentVariable = varName
					inVariableBlock = true
					inDefaultBlock = false
				} else {
					currentVariable = ""
					inVariableBlock = false
					inDefaultBlock = false
				}
			}
			continue
		}

		// Check for end of variable block
		if inVariableBlock && strings.Contains(line, "}") && !strings.Contains(line, "=") {
			// Simple check: if line only contains }, end the block
			if strings.TrimSpace(line) == "}" {
				inVariableBlock = false
				inDefaultBlock = false
				currentVariable = ""
			}
			continue
		}

		// Check for default block
		if inVariableBlock && strings.Contains(line, "default") && strings.Contains(line, "=") {
			inDefaultBlock = true
			// Check if value is on the same line
			if strings.Contains(line, `"`) {
				value := extractStringValue(line)
				if value != "" {
					assignVariableValue(&config, currentVariable, value)
				}
				inDefaultBlock = false
			}
			continue
		}

		// Extract default value if we're in a default block
		if inDefaultBlock && currentVariable != "" {
			value := extractStringValue(line)
			if value != "" {
				assignVariableValue(&config, currentVariable, value)
				inDefaultBlock = false
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return config, fmt.Errorf("error reading variables file: %v", err)
	}

	return config, nil
}

// assignVariableValue assigns a parsed value to the appropriate config field
func assignVariableValue(config *CxOneConfig, variableName, value string) {
	switch variableName {
	case "checkmarx_project_name":
		config.ProjectName = value
	case "checkmarx_group":
		config.Group = value
	case "checkmarx_branch":
		config.Branch = value
	}
}

// extractStringValue extracts a string value from a Terraform assignment line
func extractStringValue(line string) string {
	// Remove comments
	if idx := strings.Index(line, "#"); idx != -1 {
		line = line[:idx]
	}
	if idx := strings.Index(line, "//"); idx != -1 {
		line = line[:idx]
	}

	// Find the value after = sign
	parts := strings.SplitN(line, "=", 2)
	if len(parts) != 2 {
		return ""
	}

	value := strings.TrimSpace(parts[1])

	// Remove surrounding quotes and trailing comma/semicolon
	value = strings.Trim(value, `"'`)
	value = strings.TrimRight(value, ",;")
	value = strings.TrimSpace(value)

	return value
}

// ParseTerraformVariables parses various Terraform variable formats
func ParseTerraformVariables(terraformDir string) (map[string]interface{}, error) {
	variables := make(map[string]interface{})

	// Check for .tfvars files
	tfvarsFiles := []string{"terraform.tfvars", "terraform.tfvars.json", "vars.tfvars"}
	for _, filename := range tfvarsFiles {
		filePath := filepath.Join(terraformDir, filename)
		if _, err := os.Stat(filePath); err == nil {
			vars, err := parseTfvarsFile(filePath)
			if err == nil {
				for k, v := range vars {
					variables[k] = v
				}
			}
		}
	}

	return variables, nil
}

// parseTfvarsFile parses a .tfvars file for variable assignments
func parseTfvarsFile(filePath string) (map[string]interface{}, error) {
	variables := make(map[string]interface{})

	file, err := os.Open(filePath)
	if err != nil {
		return variables, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Skip comments and empty lines
		if strings.HasPrefix(line, "#") || strings.HasPrefix(line, "//") || line == "" {
			continue
		}

		// Parse variable assignments
		if strings.Contains(line, "=") {
			parts := strings.SplitN(line, "=", 2)
			if len(parts) == 2 {
				key := strings.TrimSpace(parts[0])
				value := strings.TrimSpace(parts[1])
				value = strings.Trim(value, `"'`)
				value = strings.TrimRight(value, ",;")
				variables[key] = value
			}
		}
	}

	return variables, scanner.Err()
}

// ExtractCxOneFromVariables extracts CxOne config from parsed variables
func ExtractCxOneFromVariables(variables map[string]interface{}) CxOneConfig {
	config := DefaultCxOneConfig()

	// Look for cxone variable
	if cxoneVar, exists := variables["cxone"]; exists {
		if cxoneMap, ok := cxoneVar.(map[string]interface{}); ok {
			if projectName, exists := cxoneMap["ProjectName"]; exists {
				if str, ok := projectName.(string); ok {
					config.ProjectName = str
				}
			}
			if group, exists := cxoneMap["Group"]; exists {
				if str, ok := group.(string); ok {
					config.Group = str
				}
			}
			if branch, exists := cxoneMap["Branch"]; exists {
				if str, ok := branch.(string); ok {
					config.Branch = str
				}
			}
		}
	}

	// Also check for individual variables
	if projectName, exists := variables["cxone_project_name"]; exists {
		if str, ok := projectName.(string); ok {
			config.ProjectName = str
		}
	}
	if group, exists := variables["cxone_group"]; exists {
		if str, ok := group.(string); ok {
			config.Group = str
		}
	}
	if branch, exists := variables["cxone_branch"]; exists {
		if str, ok := branch.(string); ok {
			config.Branch = str
		}
	}

	return config
}
