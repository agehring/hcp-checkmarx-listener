package terraform

import (
	"archive/zip"
	"fmt"
	"io"
	"path/filepath"
	"strings"
)

// ExtractCheckmarxProjectID extracts Checkmarx project ID from Terraform configuration
func ExtractCheckmarxProjectID(archivePath string) (string, error) {
	// Try multiple extraction methods
	projectID, err := extractFromVariables(archivePath)
	if err == nil && projectID != "" {
		return projectID, nil
	}

	projectID, err = extractFromTags(archivePath)
	if err == nil && projectID != "" {
		return projectID, nil
	}

	projectID, err = extractFromComments(archivePath)
	if err == nil && projectID != "" {
		return projectID, nil
	}

	return "", fmt.Errorf("no Checkmarx project ID found in Terraform configuration")
}

// extractFromVariables looks for checkmarx_project_id variable
func extractFromVariables(archivePath string) (string, error) {
	files, err := readTerraformFiles(archivePath)
	if err != nil {
		return "", err
	}

	for _, content := range files {
		// Look for variable definitions
		if strings.Contains(content, "checkmarx_project_id") {
			// Parse for variable default value or extract from tfvars
			lines := strings.Split(content, "\n")
			for i, line := range lines {
				if strings.Contains(line, "checkmarx_project_id") {
					// Look for default value in next few lines
					for j := i; j < len(lines) && j < i+10; j++ {
						if strings.Contains(lines[j], "default") {
							// Extract value between quotes
							parts := strings.Split(lines[j], "\"")
							if len(parts) >= 2 {
								return strings.TrimSpace(parts[1]), nil
							}
						}
					}
				}
			}
		}
	}

	return "", fmt.Errorf("checkmarx_project_id variable not found")
}

// extractFromTags looks for tags in resources
func extractFromTags(archivePath string) (string, error) {
	files, err := readTerraformFiles(archivePath)
	if err != nil {
		return "", err
	}

	for _, content := range files {
		// Look for tags with checkmarx_project
		if strings.Contains(content, "checkmarx_project") {
			lines := strings.Split(content, "\n")
			for _, line := range lines {
				if strings.Contains(line, "checkmarx_project") && strings.Contains(line, "=") {
					// Extract value between quotes
					parts := strings.Split(line, "\"")
					if len(parts) >= 2 {
						return strings.TrimSpace(parts[1]), nil
					}
				}
			}
		}
	}

	return "", fmt.Errorf("checkmarx_project tag not found")
}

// extractFromComments looks for special comments
func extractFromComments(archivePath string) (string, error) {
	files, err := readTerraformFiles(archivePath)
	if err != nil {
		return "", err
	}

	for _, content := range files {
		// Look for comments like # checkmarx-project: PROJECT_ID
		lines := strings.Split(content, "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "#") && strings.Contains(line, "checkmarx-project:") {
				parts := strings.Split(line, ":")
				if len(parts) >= 2 {
					return strings.TrimSpace(parts[1]), nil
				}
			}
		}
	}

	return "", fmt.Errorf("checkmarx-project comment not found")
}

// readTerraformFiles extracts and reads all .tf files from the archive
func readTerraformFiles(archivePath string) (map[string]string, error) {
	files := make(map[string]string)

	r, err := zip.OpenReader(archivePath)
	if err != nil {
		return nil, err
	}
	defer r.Close()

	for _, f := range r.File {
		if filepath.Ext(f.Name) == ".tf" || filepath.Ext(f.Name) == ".tfvars" {
			rc, err := f.Open()
			if err != nil {
				continue
			}

			content, err := io.ReadAll(rc)
			rc.Close()
			if err != nil {
				continue
			}

			files[f.Name] = string(content)
		}
	}

	return files, nil
}
