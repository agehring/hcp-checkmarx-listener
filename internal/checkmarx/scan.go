package checkmarx

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"sort"
	"strings"
	"time"

	"github.com/Checkmarx-PS/hcp-checkmarx-listener/internal/httpclient"
	"github.com/cxpsemea/Cx1ClientGo"
	log "github.com/sirupsen/logrus"
)

// convertTarGzToZip converts a tar.gz file to a zip file and returns the zip file path
func convertTarGzToZip(tarGzPath string) (string, error) {
	debug := os.Getenv("DEBUG") == "1" || os.Getenv("DEBUG") == "true"
	logger := log.New()
	if debug {
		logger.SetLevel(log.TraceLevel)
		logger.Infof("Converting tar.gz file %s to zip format", tarGzPath)
	}

	// Open the tar.gz file
	file, err := os.Open(tarGzPath)
	if err != nil {
		return "", fmt.Errorf("failed to open tar.gz file: %w", err)
	}
	defer file.Close()

	// Create gzip reader
	gzReader, err := gzip.NewReader(file)
	if err != nil {
		return "", fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer gzReader.Close()

	// Create tar reader
	tarReader := tar.NewReader(gzReader)

	// Create zip file
	zipPath := strings.TrimSuffix(tarGzPath, ".tar.gz") + ".zip"
	zipFile, err := os.Create(zipPath)
	if err != nil {
		return "", fmt.Errorf("failed to create zip file: %w", err)
	}
	defer zipFile.Close()

	// Create zip writer
	zipWriter := zip.NewWriter(zipFile)
	defer zipWriter.Close()

	// Process each file in the tar archive
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", fmt.Errorf("failed to read tar header: %w", err)
		}

		// Skip directories for simplicity - create them as needed
		if header.Typeflag == tar.TypeDir {
			continue
		}

		// Clean the path to avoid issues
		cleanPath := path.Clean(header.Name)
		cleanPath = strings.TrimPrefix(cleanPath, "/")

		if debug {
			logger.Infof("Adding file to zip: %s (size: %d)", cleanPath, header.Size)
		}

		// Create file in zip
		zipFile, err := zipWriter.Create(cleanPath)
		if err != nil {
			return "", fmt.Errorf("failed to create file in zip: %w", err)
		}

		// Copy file content
		_, err = io.Copy(zipFile, tarReader)
		if err != nil {
			return "", fmt.Errorf("failed to copy file content: %w", err)
		}
	}

	if debug {
		logger.Infof("Successfully converted tar.gz to zip: %s", zipPath)
	}

	return zipPath, nil
}

// GetCheckmarxToken authenticates with Checkmarx One and returns a bearer token.
// If apiKey is provided, uses API key authentication. Otherwise, uses OAuth2 client credentials.
func GetCheckmarxToken(apiKey, tenant, clientID, clientSecret, authURL string) (string, error) {
	debug := os.Getenv("DEBUG") == "1" || os.Getenv("DEBUG") == "true"
	if apiKey != "" {
		// Use Cx1ClientGo for API key authentication
		if debug {
			fmt.Printf("[DEBUG] Checkmarx Auth: Using Cx1ClientGo for API Key authentication. AuthURL: %s, Tenant: %s, API Key: %s\n", authURL, tenant, maskToken(apiKey))
		}
		// Import logrus and Cx1ClientGo
		// Create logger
		logger := log.New()
		if debug {
			logger.SetLevel(log.TraceLevel)
		}
		// Create HTTP client (optionally with proxy/TLS config)
		httpClient := httpclient.NewClient()
		// Initialize Cx1ClientGo client
		cx1client, err := Cx1ClientGo.NewAPIKeyClient(httpClient, authURL, authURL, tenant, apiKey, logger)
		if err != nil {
			if debug {
				logger.Errorf("[DEBUG] Cx1ClientGo: Error creating client: %v", err)
			}
			return "", fmt.Errorf("Cx1ClientGo client init failed: %w", err)
		}
		// Set custom User-Agent for SDK requests
		cx1client.SetUserAgent("CxOne-HCP")
		// Verify authentication by making a simple API call
		_, err = cx1client.GetGroups()
		if err != nil {
			if debug {
				logger.Errorf("[DEBUG] Cx1ClientGo: API key authentication failed: %v", err)
			}
			return "", fmt.Errorf("Cx1ClientGo API key authentication failed: %w", err)
		}
		if debug {
			logger.Infof("[DEBUG] Cx1ClientGo: API key authentication succeeded")
		}
		// Return a dummy token (not used, but for interface compatibility)
		return "authenticated", nil
	}
	// OAuth2 client credentials
	if debug {
		fmt.Printf("[DEBUG] Checkmarx Auth: Using OAuth2 authentication. AuthURL: %s, Tenant: %s, ClientID: %s\n", authURL, tenant, maskToken(clientID))
	}
	data := fmt.Sprintf("grant_type=client_credentials&client_id=%s&client_secret=%s", clientID, clientSecret)
	if debug {
		fmt.Printf("[DEBUG] Checkmarx Auth: OAuth2 request body: %s\n", data)
	}
	req, err := http.NewRequest("POST", authURL+"/auth/realms/Checkmarx/protocol/openid-connect/token", bytes.NewBufferString(data))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	if debug {
		fmt.Printf("[DEBUG] Checkmarx Auth: OAuth2 request headers: Content-Type=%s\n", req.Header.Get("Content-Type"))
	}
	resp, err := httpclient.Client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if debug {
		fmt.Printf("[DEBUG] Checkmarx Auth: OAuth2 response status: %d %s\n", resp.StatusCode, resp.Status)
		bodyBytes, _ := io.ReadAll(resp.Body)
		fmt.Printf("[DEBUG] Checkmarx Auth: OAuth2 response body: %s\n", string(bodyBytes))
		// Reset body for decoder
		resp.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("OAuth2 auth failed: status %d", resp.StatusCode)
	}
	var result struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}
	if debug {
		fmt.Printf("[DEBUG] Checkmarx Auth: OAuth2 access_token (masked): %s\n", maskToken(result.AccessToken))
	}
	return result.AccessToken, nil
}

// maskToken returns a masked version of a token for debug logging
func maskToken(token string) string {
	if len(token) <= 8 {
		return "****"
	}
	return token[:4] + "..." + token[len(token)-4:]
}

// updateProjectOrigin updates the origin field of a project via direct API call
// since the SDK's ProjectPatch doesn't include the origin field
func updateProjectOrigin(cx1client *Cx1ClientGo.Cx1Client, projectID, origin, checkmarxBaseURL string) error {
	// Create the patch payload with the origin field
	patchData := map[string]interface{}{
		"origin": origin,
	}

	jsonBody, err := json.Marshal(patchData)
	if err != nil {
		return fmt.Errorf("failed to marshal project patch: %w", err)
	}

	// Make direct API call to patch the project
	url := fmt.Sprintf("%s/api/projects/%s", checkmarxBaseURL, projectID)
	req, err := http.NewRequest("PATCH", url, bytes.NewReader(jsonBody))
	if err != nil {
		return fmt.Errorf("failed to create PATCH request: %w", err)
	}

	req.Header.Set("Accept", "application/json; version=1.0")
	req.Header.Set("Content-Type", "application/json; version=1.0")
	req.Header.Set("Authorization", "Bearer "+cx1client.GetAccessToken())

	resp, err := httpclient.Client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to execute PATCH request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("project origin update failed with status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// GetCheckmarxToken authenticates with Checkmarx One and returns a bearer token.
// If apiKey is provided, uses API key authentication. Otherwise, uses OAuth2 client credentials.

// ScanSubmissionResult holds the result of a scan submission
// including the HCP queue/run id and the Checkmarx scan id.
type ScanSubmissionResult struct {
	HCPTaskResultID string
	CheckmarxScanID string
}

// SubmitIaCScan submits a file to the Checkmarx One API for an IaC (Kics) scan using the provided authenticated client.
// Returns the Checkmarx scan ID or an error.
func SubmitIaCScan(cx1client *Cx1ClientGo.Cx1Client, filePath, hcpTaskResultID, projectName, groupName, branch, checkmarxBaseURL, checkmarxToken string) (*ScanSubmissionResult, error) {
	debug := os.Getenv("DEBUG") == "1" || os.Getenv("DEBUG") == "true"
	logger := log.New()
	if debug {
		logger.SetLevel(log.TraceLevel)
		fmt.Printf("[DEBUG] SubmitIaCScan: Using project=%s, group=%s, branch=%s\n", projectName, groupName, branch)
	}

	// Find or create group
	group, err := cx1client.GetGroupByName(groupName)
	if err != nil {
		// Check for various "not found" error messages from the SDK
		errMsg := err.Error()
		if errMsg == "No matching group found" ||
			errMsg == "no group "+groupName+" found" ||
			strings.Contains(errMsg, "not found") ||
			strings.Contains(errMsg, "does not exist") {
			logger.Infof("No group named %s exists - it will now be created", groupName)
			group, err = cx1client.CreateGroup(groupName)
			if err != nil {
				logger.Errorf("Failed to create group %s: %v", groupName, err)
				return nil, err
			}
			logger.Infof("Created group named '%v' with ID %v", group.Name, group.GroupID)
		} else {
			logger.Errorf("Failed to retrieve group named %s: %v", groupName, err)
			return nil, err
		}
	}

	// Find or create project
	projects, err := cx1client.GetProjectsByName(projectName)
	if err != nil {
		logger.Errorf("Failed to retrieve project named %s: %v", projectName, err)
		return nil, err
	}
	var project Cx1ClientGo.Project
	if len(projects) == 0 {
		logger.Infof("No project named %s found - it will now be created", projectName)
		project, err = cx1client.CreateProject(projectName, []string{group.GroupID}, map[string]string{"CreatedBy": "hcp-checkmarx-listener"})
		if err != nil {
			logger.Errorf("Failed to create project %s: %v", projectName, err)
			return nil, err
		}
		logger.Infof("Created project named '%v' with ID %v", project.Name, project.ProjectID)

		// Update the project origin to "CxOne-HCP"
		projectPatch := Cx1ClientGo.ProjectPatch{}
		projectPatch.Name = &projectName // Keep the same name
		// Set custom origin - unfortunately the SDK doesn't expose origin in ProjectPatch
		// So we'll need to make a direct API call to update the origin
		logger.Infof("Setting project origin to 'CxOne-HCP' for project %s", project.ProjectID)
		err = updateProjectOrigin(cx1client, project.ProjectID, "CxOne-HCP", checkmarxBaseURL)
		if err != nil {
			logger.Warnf("Failed to update project origin: %v", err)
			// Don't fail the scan for this, just continue
		}
	} else {
		project = projects[0]
		logger.Infof("First project matching '%v' is named '%v' with ID %v", projectName, project.Name, project.ProjectID)
	}

	// Validate project ID before proceeding
	if project.ProjectID == "" {
		logger.Errorf("Project ID is empty for project %s", projectName)
		return nil, fmt.Errorf("invalid project ID - project may not exist or user lacks permissions")
	}

	if debug {
		logger.Infof("DEBUG: Using project ID %s for scan submission", project.ProjectID)
	}

	// Trigger scan (assume filePath is a ZIP archive of IaC config)
	scanConfig := Cx1ClientGo.ScanConfiguration{}
	scanConfig.ScanType = "iac"
	scanConfig.Values = map[string]string{"engine": "kics"}

	// Step 1: Convert tar.gz to zip format (Checkmarx One expects ZIP files)
	var zipFilePath string
	if strings.HasSuffix(strings.ToLower(filePath), ".tar.gz") {
		zipFilePath, err = convertTarGzToZip(filePath)
		if err != nil {
			logger.Errorf("Failed to convert tar.gz to zip: %s", err)
			return nil, err
		}
		// Clean up the zip file after we're done
		defer func() {
			if err := os.Remove(zipFilePath); err != nil {
				logger.Warnf("Failed to clean up temporary zip file %s: %s", zipFilePath, err)
			}
		}()
		logger.Infof("Converted tar.gz to zip: %s", zipFilePath)
	} else {
		// Assume it's already a zip file
		zipFilePath = filePath
	}

	// Step 2: Read the ZIP file contents
	fileContents, err := os.ReadFile(zipFilePath)
	if err != nil {
		logger.Errorf("Failed to read ZIP file '%v': %s", zipFilePath, err)
		return nil, err
	}

	// Step 3: Upload the ZIP file using SDK's built-in method
	uploadURL, err := cx1client.UploadBytes(&fileContents)
	if err != nil {
		logger.Errorf("Failed to upload ZIP file: %s", err)
		return nil, err
	}

	// Validate upload URL
	if uploadURL == "" {
		logger.Errorf("Upload URL is empty - file upload may have failed")
		return nil, fmt.Errorf("invalid upload URL - file upload failed")
	}

	if debug {
		logger.Infof("DEBUG: Uploaded ZIP file to URL: %s", uploadURL)
	} else {
		logger.Infof("ZIP file uploaded successfully to Checkmarx One storage")
	}

	// Step 3: Submit scan using the correct API structure for upload type
	scanRequest := map[string]interface{}{
		"project": map[string]interface{}{
			"id": project.ProjectID,
		},
		"type": "upload",
		"handler": map[string]interface{}{
			"uploadurl": uploadURL,
			"branch":    branch, // Use the branch from CxOne config
		},
		"tags": map[string]interface{}{
			"CreatedBy":     "hcp-checkmarx-listener",
			"HCPTaskID":     hcpTaskResultID,
			"WorkspaceName": "terraform-workspace",
			"Branch":        branch, // Also add as tag for reference
		},
		"config": []map[string]interface{}{
			{
				"type": "kics",
				"value": map[string]interface{}{
					"filter":    "",
					"platforms": "",
				},
			},
		},
	}

	scanRequestJSON, err := json.Marshal(scanRequest)
	if err != nil {
		logger.Errorf("Failed to marshal scan request: %s", err)
		return nil, err
	}

	// Get base URL and token for direct API call
	scanURL := fmt.Sprintf("%s/api/scans/", checkmarxBaseURL)

	if debug {
		logger.Infof("DEBUG: Scan request JSON: %s", string(scanRequestJSON))
		logger.Infof("DEBUG: Scan URL: %s", scanURL)
	}

	req, err := http.NewRequest("POST", scanURL, bytes.NewReader(scanRequestJSON))
	if err != nil {
		logger.Errorf("Failed to create scan request: %s", err)
		return nil, err
	}

	req.Header.Set("Accept", "application/json; version=1.0")
	req.Header.Set("Content-Type", "application/json; version=1.0")

	// Ensure we have a fresh access token by making a test API call
	// This forces the SDK to refresh the token if needed
	testGroups, err := cx1client.GetGroups()
	if err != nil {
		logger.Errorf("Failed to refresh access token: %s", err)
		return nil, fmt.Errorf("token refresh failed: %w", err)
	}

	if debug {
		logger.Infof("DEBUG: Token validation successful, found %d groups", len(testGroups))
	}

	// Use the SDK client's access token
	accessToken := cx1client.GetAccessToken()
	if accessToken == "" {
		logger.Errorf("No access token available from SDK client")
		return nil, fmt.Errorf("no access token available")
	}

	if debug {
		logger.Infof("DEBUG: Using access token: %s", maskToken(accessToken))
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := httpclient.Client.Do(req)
	if err != nil {
		logger.Errorf("Failed to submit scan: %s", err)
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)

		// Provide specific error handling for common status codes
		switch resp.StatusCode {
		case http.StatusUnauthorized:
			logger.Errorf("Scan submission failed with 401 Unauthorized - token may be expired or invalid")
			logger.Errorf("Response body: %s", string(body))
			return nil, fmt.Errorf("scan submission failed: authentication error (401) - check API key/credentials")
		case http.StatusForbidden:
			logger.Errorf("Scan submission failed with 403 Forbidden - insufficient permissions")
			logger.Errorf("Response body: %s", string(body))
			return nil, fmt.Errorf("scan submission failed: permission denied (403) - check user permissions")
		case http.StatusBadRequest:
			logger.Errorf("Scan submission failed with 400 Bad Request - invalid request format")
			logger.Errorf("Response body: %s", string(body))
			return nil, fmt.Errorf("scan submission failed: bad request (400) - check scan configuration")
		default:
			logger.Errorf("Scan submission failed with status: %d, body: %s", resp.StatusCode, string(body))
			return nil, fmt.Errorf("scan submission failed with status: %d", resp.StatusCode)
		}
	}

	var scanResponse map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&scanResponse); err != nil {
		logger.Errorf("Failed to decode scan response: %s", err)
		return nil, err
	}

	scanID, ok := scanResponse["id"].(string)
	if !ok {
		logger.Errorf("Invalid scan response - no scan ID found")
		return nil, fmt.Errorf("invalid scan response - no scan ID found")
	}

	logger.Infof("Triggered scan %v, polling status", scanID)

	// Wait for scan completion using SDK method
	err = WaitForScanCompletion(cx1client, scanID, 10*60*time.Second)
	if err != nil {
		logger.Errorf("Scan did not complete successfully: %v", err)
		return nil, err
	}

	// Return scan ID for further processing
	return &ScanSubmissionResult{
		HCPTaskResultID: hcpTaskResultID,
		CheckmarxScanID: scanID,
	}, nil
}

// IaCFinding represents a single finding from a Checkmarx IaC scan result.
type IaCFinding struct {
	ID            string `json:"ID"`
	SimilarityID  string `json:"similarityID"`
	Severity      string `json:"severity"`
	FirstScanID   string `json:"firstScanID"`
	FirstFoundAt  string `json:"firstFoundAt"`
	FoundAt       string `json:"foundAt"`
	Status        string `json:"status"`
	State         string `json:"state"`
	Type          string `json:"type"`
	QueryID       string `json:"queryID"`
	QueryName     string `json:"queryName"`
	Group         string `json:"group"`
	QueryURL      string `json:"queryURL"`
	FileName      string `json:"fileName"`
	Line          int    `json:"line"`
	Platform      string `json:"platform"`
	IssueType     string `json:"issueType"`
	SearchKey     string `json:"searchKey"`
	SearchValue   string `json:"searchValue"`
	ExpectedValue string `json:"expectedValue"`
	ActualValue   string `json:"actualValue"`
	Value         string `json:"value"`
	Description   string `json:"description"`
	Comments      string `json:"comments"`
	Category      string `json:"category"`
}

type IaCResultsResponse struct {
	Results    []IaCFinding `json:"results"`
	TotalCount int          `json:"totalCount"`
}

// FetchIaCFindings fetches all IAC findings for a given scan ID from Checkmarx One using the KICS results API.
// Returns all severity levels (CRITICAL, HIGH, MEDIUM, LOW, INFO).
func FetchIaCFindings(cx1client *Cx1ClientGo.Cx1Client, scanID, checkmarxBaseURL string, debug bool) ([]IaCFinding, error) {
	if debug {
		fmt.Printf("[DEBUG] FetchIaCFindings: Starting with scanID='%s' using KICS API\n", scanID)
	}

	// Get access token from the SDK client
	token := cx1client.GetAccessToken()

	// Build KICS results API URL with parameters to get ALL results
	// Use limit=1000 and pagination to get all results
	var allResults []IaCFinding
	offset := 0
	limit := 1000 // Maximum reasonable limit per request

	for {
		url := fmt.Sprintf("%s/api/kics-results/?scan-id=%s&limit=%d&offset=%d&apply-predicates=true",
			checkmarxBaseURL, scanID, limit, offset)

		if debug {
			fmt.Printf("[DEBUG] FetchIaCFindings: Calling KICS API: %s\n", url)
		}

		// Create HTTP request
		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create request: %v", err)
		}

		// Set headers
		req.Header.Set("Accept", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("CorrelationId", scanID) // Use scanID as correlation ID

		// Make request using the centralized HTTP client
		resp, err := httpclient.Client.Do(req)
		if err != nil {
			return nil, fmt.Errorf("failed to execute request: %v", err)
		}
		defer resp.Body.Close()

		// Check response status
		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			return nil, fmt.Errorf("KICS API returned status %d: %s", resp.StatusCode, string(body))
		}

		// Parse response
		var response IaCResultsResponse
		if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
			return nil, fmt.Errorf("failed to decode response: %v", err)
		}

		if debug {
			fmt.Printf("[DEBUG] FetchIaCFindings: KICS API returned %d results (offset=%d, totalCount=%d)\n",
				len(response.Results), offset, response.TotalCount)
		}

		// Process and normalize severity for each result
		for i, result := range response.Results {
			// Normalize severity using our function
			result.Severity = normalizeSeverity(result.Severity, debug)

			if debug && i < 5 && offset == 0 { // Debug first 5 results from first page
				fmt.Printf("[DEBUG] KICS Result %d: ID='%s', Severity='%s', Status='%s', QueryName='%s', FileName='%s', Line=%d\n",
					i+1, result.ID, result.Severity, result.Status, result.QueryName, result.FileName, result.Line)
			}

			allResults = append(allResults, result)
		}

		// Check if we got all results
		if len(response.Results) < limit || len(allResults) >= response.TotalCount {
			if debug {
				fmt.Printf("[DEBUG] FetchIaCFindings: Completed pagination. Retrieved %d total results\n", len(allResults))
			}
			break
		}

		// Move to next page
		offset += limit
	}

	// Debug logging before sorting
	if debug {
		fmt.Printf("[DEBUG] FetchIaCFindings: Before sorting - Retrieved %d total IAC findings\n", len(allResults))

		// Count normalized severities before sorting
		preSortSeverityCounts := make(map[string]int)
		for _, finding := range allResults {
			preSortSeverityCounts[finding.Severity]++
		}
		fmt.Printf("[DEBUG] FetchIaCFindings: Pre-sort severity breakdown: %+v\n", preSortSeverityCounts)

		// Show any CRITICAL findings specifically
		criticalCount := 0
		for _, finding := range allResults {
			if strings.ToUpper(finding.Severity) == "CRITICAL" {
				criticalCount++
				if criticalCount <= 3 { // Show first 3 critical findings
					fmt.Printf("[DEBUG] CRITICAL Finding %d: ID='%s', QueryName='%s', FileName='%s', Status='%s'\n",
						criticalCount, finding.ID, finding.QueryName, finding.FileName, finding.Status)
				}
			}
		}
		if criticalCount > 0 {
			fmt.Printf("[DEBUG] FetchIaCFindings: Found %d CRITICAL findings before sorting\n", criticalCount)
		} else {
			fmt.Printf("[DEBUG] FetchIaCFindings: NO CRITICAL findings found before sorting\n")
		}
	}

	// Sort results by severity (CRITICAL, HIGH, MEDIUM, LOW, INFO)
	sort.Slice(allResults, func(i, j int) bool {
		severityOrder := map[string]int{
			"CRITICAL": 1,
			"HIGH":     2,
			"MEDIUM":   3,
			"LOW":      4,
			"INFO":     5,
		}
		return severityOrder[strings.ToUpper(allResults[i].Severity)] < severityOrder[strings.ToUpper(allResults[j].Severity)]
	})

	// Debug logging after sorting
	if debug {
		fmt.Printf("[DEBUG] FetchIaCFindings: After sorting - %d total IAC findings\n", len(allResults))

		// Count normalized severities after sorting
		postSortSeverityCounts := make(map[string]int)
		for _, finding := range allResults {
			postSortSeverityCounts[finding.Severity]++
		}
		fmt.Printf("[DEBUG] FetchIaCFindings: Post-sort severity breakdown: %+v\n", postSortSeverityCounts)

		// Show final sorted results for debugging
		if len(allResults) > 0 {
			fmt.Printf("[DEBUG] FetchIaCFindings: Final sorted results (first 10):\n")
			for i, finding := range allResults {
				if i >= 10 { // Only show first 10
					break
				}
				fmt.Printf("[DEBUG]   Result %d: Severity='%s', QueryName='%s', ID='%s', Status='%s'\n",
					i+1, finding.Severity, finding.QueryName, finding.ID, finding.Status)
			}
		}

		fmt.Printf("[DEBUG] FetchIaCFindings: Returning %d IAC findings (all severities)\n", len(allResults))
	}

	return allResults, nil
}

// normalizeSeverity ensures consistent severity naming
func normalizeSeverity(severity string, debug bool) string {
	severityUpper := strings.ToUpper(strings.TrimSpace(severity))

	// Handle common variations
	switch severityUpper {
	case "CRITICAL", "CRIT":
		return "CRITICAL"
	case "HIGH", "H":
		return "HIGH"
	case "MEDIUM", "MED", "M":
		return "MEDIUM"
	case "LOW", "L":
		return "LOW"
	case "INFO", "INFORMATION", "INFORMATIONAL", "I":
		return "INFO"
	default:
		// For unknown severities, default to INFO but log it
		if debug {
			fmt.Printf("[DEBUG] Unknown severity '%s', defaulting to INFO\n", severity)
		}
		return "INFO"
	}
}
