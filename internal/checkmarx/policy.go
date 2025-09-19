package checkmarx

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/Checkmarx-PS/hcp-checkmarx-listener/internal/httpclient"
	"github.com/cxpsemea/Cx1ClientGo"
)

// ScanStatus represents the status of a Checkmarx scan
// (simplified for this example)
type ScanStatus struct {
	Status string `json:"status"`
}

type PolicyInfo struct {
	PolicyName    string   `json:"policyName"`
	Description   string   `json:"description"`
	RulesViolated []string `json:"rulesViolated"`
	Tags          []string `json:"tags"`
	BreakBuild    bool     `json:"breakBuild"`
	Status        string   `json:"status"`
}

type PolicyCheckResult struct {
	Status     string       `json:"status"`
	BreakBuild bool         `json:"breakBuild"`
	Policies   []PolicyInfo `json:"policies"`
}

// WaitForScanCompletion polls the Checkmarx One API until the scan is complete or times out using the provided authenticated client.
func WaitForScanCompletion(cx1client *Cx1ClientGo.Cx1Client, scanID string, timeout time.Duration) error {
	end := time.Now().Add(timeout)
	for time.Now().Before(end) {
		scan, err := cx1client.GetScanByID(scanID)
		if err != nil {
			return fmt.Errorf("failed to get scan status: %w", err)
		}
		if scan.Status == "Completed" {
			return nil
		}
		if scan.Status == "Failed" {
			return fmt.Errorf("scan failed")
		}
		time.Sleep(5 * time.Second)
	}
	return fmt.Errorf("timeout waiting for scan completion")
}

func getScanStatus(cx1client *Cx1ClientGo.Cx1Client, scanID, checkmarxToken, checkmarxBaseURL string) (string, error) {
	url := fmt.Sprintf("%s/api/scans/%s", checkmarxBaseURL, scanID)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", err
	}
	// Use the SDK client's access token instead of the config token
	req.Header.Set("Authorization", "Bearer "+cx1client.GetAccessToken())
	resp, err := httpclient.Client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}
	var status ScanStatus
	if err := json.NewDecoder(resp.Body).Decode(&status); err != nil {
		return "", err
	}
	return status.Status, nil
}

// CheckPolicyViolation checks if the scan result violates any policies.
func CheckPolicyViolation(cx1client *Cx1ClientGo.Cx1Client, scanID, checkmarxBaseURL, checkmarxToken string) (*PolicyCheckResult, error) {
	debug := os.Getenv("DEBUG") == "1" || os.Getenv("DEBUG") == "true"

	if debug {
		fmt.Printf("[DEBUG] Starting policy check for scan ID: %s\n", scanID)
	}

	// The policy API requires both astProjectId and scanId parameters
	// First, get the scan details to extract the project ID
	scan, err := cx1client.GetScanByID(scanID)
	if err != nil {
		return nil, fmt.Errorf("failed to get scan details for policy check: %w", err)
	}

	// The correct policy API endpoint with both required parameters
	url := fmt.Sprintf("%s/api/policy_management_service_uri/evaluation?astProjectId=%s&scanId=%s",
		checkmarxBaseURL, scan.ProjectID, scanID)

	if debug {
		fmt.Printf("[DEBUG] Policy check URL: %s\n", url)
	}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Accept", "application/json; version=1.0")
	// Use the SDK client's access token instead of the config token
	req.Header.Set("Authorization", "Bearer "+cx1client.GetAccessToken())

	resp, err := httpclient.Client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		// Read response body for debugging
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("policy check failed with status %d: %s", resp.StatusCode, string(body))
	}

	var result PolicyCheckResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	if debug {
		fmt.Printf("[DEBUG] Policy retrieval completed successfully. Status: %s, BreakBuild: %t, Policies: %d\n",
			result.Status, result.BreakBuild, len(result.Policies))
	}

	return &result, nil
}
