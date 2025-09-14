package checkmarx

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"

	"github.com/Checkmarx-PS/hcp-checkmarx-listener/internal/httpclient"
)

// ScanSubmissionResult holds the result of a scan submission
// including the HCP queue/run id and the Checkmarx scan id.
type ScanSubmissionResult struct {
	HCPTaskResultID string
	CheckmarxScanID string
}

// SubmitIaCScan submits a file to the Checkmarx One API for an IaC (Kics) scan.
// Returns the Checkmarx scan ID or an error.
func SubmitIaCScan(filePath, hcpTaskResultID, checkmarxToken, checkmarxProjectID, checkmarxBaseURL string) (*ScanSubmissionResult, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	part, err := writer.CreateFormFile("file", filePath)
	if err != nil {
		return nil, err
	}
	if _, err := io.Copy(part, file); err != nil {
		return nil, err
	}

	// Add additional form fields as required by Checkmarx API
	_ = writer.WriteField("projectId", checkmarxProjectID)
	_ = writer.WriteField("scanType", "iac")
	_ = writer.WriteField("engine", "kics")

	if err := writer.Close(); err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", checkmarxBaseURL+"/api/scans", &body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+checkmarxToken)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := httpclient.Client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Checkmarx scan submission failed: %s", resp.Status)
	}

	// Parse response to extract scan ID (simplified, adjust as needed)
	var result struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return &ScanSubmissionResult{
		HCPTaskResultID: hcpTaskResultID,
		CheckmarxScanID: result.ID,
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

// FetchIaCFindings fetches findings for a given scan ID from Checkmarx One.
func FetchIaCFindings(scanID, checkmarxToken, checkmarxBaseURL string) ([]IaCFinding, error) {
	url := fmt.Sprintf("%s/api/kics-results/", checkmarxBaseURL)
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
		return nil, fmt.Errorf("failed to fetch findings: %s", resp.Status)
	}
	var results IaCResultsResponse
	if err := json.NewDecoder(resp.Body).Decode(&results); err != nil {
		return nil, err
	}
	return results.Results, nil
}
