package checkmarx

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/Checkmarx-PS/hcp-checkmarx-listener/internal/httpclient"
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

// WaitForScanCompletion polls the Checkmarx One API until the scan is complete or times out.
func WaitForScanCompletion(scanID, checkmarxToken, checkmarxBaseURL string, timeout time.Duration) error {
	end := time.Now().Add(timeout)
	for time.Now().Before(end) {
		status, err := getScanStatus(scanID, checkmarxToken, checkmarxBaseURL)
		if err != nil {
			return err
		}
		if status == "Completed" {
			return nil
		}
		if status == "Failed" {
			return fmt.Errorf("scan failed")
		}
		time.Sleep(5 * time.Second)
	}
	return fmt.Errorf("timeout waiting for scan completion")
}

func getScanStatus(scanID, checkmarxToken, checkmarxBaseURL string) (string, error) {
	url := fmt.Sprintf("%s/api/scans/%s", checkmarxBaseURL, scanID)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+checkmarxToken)
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
func CheckPolicyViolation(scanID, checkmarxToken, checkmarxBaseURL string) (*PolicyCheckResult, error) {
	url := fmt.Sprintf("%s/api/policy_management_service_uri/evaluation", checkmarxBaseURL)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+checkmarxToken)
	resp, err := httpclient.Client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}
	var result PolicyCheckResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	return &result, nil
}
