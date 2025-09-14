package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/Checkmarx-PS/hcp-checkmarx-listener/api"
	"github.com/Checkmarx-PS/hcp-checkmarx-listener/internal/checkmarx"
	"github.com/Checkmarx-PS/hcp-checkmarx-listener/internal/config"
	"github.com/Checkmarx-PS/hcp-checkmarx-listener/internal/hmac"
	"github.com/Checkmarx-PS/hcp-checkmarx-listener/internal/httpclient"
	"github.com/Checkmarx-PS/hcp-checkmarx-listener/internal/secrets"
	"github.com/Checkmarx-PS/hcp-checkmarx-listener/internal/terraform"
)

type DownloadHeaders struct {
	ForwardedFor   string
	ForwardedProto string
	AgentAuth      string
	AccessToken    string
}

type JobWithHeaders struct {
	Payload api.RunTaskPayload
	Headers DownloadHeaders
}

var JobQueue = make(chan JobWithHeaders, 100)

var appConfig *config.Config

var validRunTaskStatuses = map[string]struct{}{
	"running": {},
	"passed":  {},
	"failed":  {},
}

func main() {
	var err error
	appConfig, err = config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Start worker goroutine
	go jobWorker()

	http.HandleFunc("/run-task", runTaskHandler)
	addr := fmt.Sprintf(":%d", appConfig.ListenPort)
	log.Printf("Listening on %s...", addr)
	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

// sendRunTaskStatus posts a status update to the HCP Terraform run task callback URL.
func sendRunTaskStatus(callbackURL, accessToken, status, message string, details map[string]interface{}) error {
	if _, ok := validRunTaskStatuses[status]; !ok {
		return fmt.Errorf("invalid run task status: %s", status)
	}

	attributes := map[string]interface{}{
		"status": status,
	}
	if message != "" {
		attributes["message"] = message
	}
	if url, ok := details["url"]; ok {
		attributes["url"] = url
	}

	payload := map[string]interface{}{
		"data": map[string]interface{}{
			"type":       "task-results",
			"attributes": attributes,
		},
	}
	if outcomes, ok := details["outcomes"]; ok {
		payload["data"].(map[string]interface{})["relationships"] = map[string]interface{}{
			"outcomes": map[string]interface{}{
				"data": outcomes,
			},
		}
	}

	b, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("PATCH", callbackURL, strings.NewReader(string(b)))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/vnd.api+json")
	if accessToken != "" {
		req.Header.Set("Authorization", "Bearer "+accessToken)
	}
	resp, err := httpclient.Client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case 200, 204:
		// No Content or OK, success
		return nil
	case 401, 422:
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("callback returned %d: %s", resp.StatusCode, string(body))
	default:
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("callback returned unexpected status %d: %s", resp.StatusCode, string(body))
	}
}

func jobWorker() {
	for job := range JobQueue {
		var checkmarxSecrets *secrets.CheckmarxSecrets
		log.Printf("Processing job: %+v", job.Payload)

		// Send initial status
		err := sendRunTaskStatus(job.Payload.TaskResultCallbackURL, job.Payload.AccessToken, "running", "Job started", nil)
		if err != nil {
			log.Printf("Failed to send status update: %v", err)
		}

		// Download the file from Terraform HCP with request forwarding headers
		filePath, err := downloadFileWithHeaders(job.Payload.ConfigurationVersionDownloadURL, job.Headers)
		if err != nil {
			log.Printf("Failed to download file: %v", err)
			sendRunTaskStatus(job.Payload.TaskResultCallbackURL, job.Payload.AccessToken, "failed", "Failed to download file", map[string]interface{}{"error": err.Error()})
			continue
		}
		sendRunTaskStatus(job.Payload.TaskResultCallbackURL, job.Payload.AccessToken, "running", "Downloaded configuration archive", nil)
		// Load secrets (path can be from config or env)
		secretsPath := os.Getenv("CHECKMARX_SECRETS_FILE")
		if secretsPath == "" {
			secretsPath = "checkmarx.secrets.json"
		}
		checkmarxSecrets, err = secrets.LoadSecrets(secretsPath)
		if err != nil {
			log.Fatalf("Failed to load Checkmarx secrets: %v", err)
		}
		log.Printf("Downloaded file to: %s", filePath)

		// Extract Checkmarx project ID from Terraform configuration
		projectID := appConfig.CheckmarxProjectID // Default from config
		if dynamicProjectID, err := terraform.ExtractCheckmarxProjectID(filePath); err == nil && dynamicProjectID != "" {
			projectID = dynamicProjectID
			log.Printf("Using dynamic project ID from Terraform config: %s", projectID)
		} else {
			log.Printf("Using default project ID from config: %s", projectID)
		}

		// Submit file to Checkmarx One for IaC (Kics) scan
		scanResult, err := checkmarx.SubmitIaCScan(
			filePath,
			job.Payload.TaskResultID,
			appConfig.CheckmarxToken,
			projectID,
			appConfig.CheckmarxBaseURL,
		)
		if err != nil {
			log.Printf("Checkmarx scan submission failed: %v", err)
			sendRunTaskStatus(job.Payload.TaskResultCallbackURL, job.Payload.AccessToken, "failed", "Checkmarx scan submission failed", map[string]interface{}{"error": err.Error()})
			continue
		}
		sendRunTaskStatus(job.Payload.TaskResultCallbackURL, job.Payload.AccessToken, "running", "Submitted to Checkmarx for scan", map[string]interface{}{"scan_id": scanResult.CheckmarxScanID})

		// Wait for scan completion
		log.Printf("Waiting for scan %s to complete...", scanResult.CheckmarxScanID)
		err = checkmarx.WaitForScanCompletion(scanResult.CheckmarxScanID, appConfig.CheckmarxToken, appConfig.CheckmarxBaseURL, 10*60*time.Second)
		if err != nil {
			log.Printf("Scan did not complete successfully: %v", err)
			sendRunTaskStatus(job.Payload.TaskResultCallbackURL, job.Payload.AccessToken, "failed", "Scan did not complete successfully", map[string]interface{}{"error": err.Error()})
			continue
		}
		sendRunTaskStatus(job.Payload.TaskResultCallbackURL, job.Payload.AccessToken, "running", "Scan completed", map[string]interface{}{"scan_id": scanResult.CheckmarxScanID})

		// Determine auth method for Checkmarx
		var token string
		if checkmarxSecrets.AuthType == secrets.AuthTypeAPIKey {
			token = checkmarxSecrets.APIKey
		} else if checkmarxSecrets.AuthType == secrets.AuthTypeOAuth2 {
			// Get OAuth2 token (fetch if needed)
			t, err := getOAuth2Token(checkmarxSecrets)
			if err != nil {
				log.Printf("Failed to get OAuth2 token: %v", err)
				sendRunTaskStatus(job.Payload.TaskResultCallbackURL, job.Payload.AccessToken, "failed", "Failed to get OAuth2 token", map[string]interface{}{"error": err.Error()})
				continue
			}
			token = t
		} else {
			log.Printf("Unknown Checkmarx auth type: %s", checkmarxSecrets.AuthType)
			sendRunTaskStatus(job.Payload.TaskResultCallbackURL, job.Payload.AccessToken, "failed", "Unknown Checkmarx auth type", map[string]interface{}{"error": checkmarxSecrets.AuthType})
			continue
		}
		err = checkmarx.WaitForScanCompletion(scanResult.CheckmarxScanID, token, appConfig.CheckmarxBaseURL, 10*60*time.Second)
		if err != nil {
			log.Printf("Scan did not complete successfully: %v", err)
			sendRunTaskStatus(job.Payload.TaskResultCallbackURL, job.Payload.AccessToken, "failed", "Scan did not complete successfully (auth)", map[string]interface{}{"error": err.Error()})
			continue
		}
		// Check for policy violation
		policyResult, err := checkmarx.CheckPolicyViolation(scanResult.CheckmarxScanID, token, appConfig.CheckmarxBaseURL)
		if err != nil {
			log.Printf("Policy check failed: %v", err)
			sendRunTaskStatus(job.Payload.TaskResultCallbackURL, job.Payload.AccessToken, "failed", "Policy check failed", map[string]interface{}{"error": err.Error()})
			continue
		}
		var policyOutcomes []interface{}
		for _, policy := range policyResult.Policies {
			level := "info"
			if strings.ToUpper(policy.Status) == "FAILED" || policy.BreakBuild {
				level = "error"
			}
			outcome := map[string]interface{}{
				"type": "task-result-outcomes",
				"attributes": map[string]interface{}{
					"outcome-id":  policy.PolicyName,
					"description": policy.PolicyName + ": " + policy.Description,
					"body":        fmt.Sprintf("**Rules Violated:** %v\n**Break Build:** %v\n**Status:** %s", policy.RulesViolated, policy.BreakBuild, policy.Status),
					"tags": map[string]interface{}{
						"Status": []map[string]interface{}{{"label": policy.Status, "level": level}},
					},
				},
			}
			policyOutcomes = append(policyOutcomes, outcome)
		}
		status := "passed"
		msg := "No policy violations."
		if policyResult.BreakBuild {
			status = "failed"
			msg = "Policy violation(s) detected."
		}
		sendRunTaskStatus(job.Payload.TaskResultCallbackURL, job.Payload.AccessToken, status, msg, map[string]interface{}{"outcomes": policyOutcomes})

		// TODO: Handle findings retrieval and tagging
		/*
			findings, err := checkmarx.GetIaCFindings(scanResult.CheckmarxScanID, token, appConfig.CheckmarxBaseURL)
			if err != nil {
				log.Printf("Failed to fetch IaC findings: %v", err)
				sendRunTaskStatus(job.Payload.TaskResultCallbackURL, job.Payload.AccessToken, "failed", "Failed to fetch IaC findings: "+err.Error(), nil)
				continue
			}
			var outcomes []interface{}
			for _, finding := range findings {
				outcome := map[string]interface{}{
					"type": "task-result-outcomes",
					"attributes": map[string]interface{}{
						"outcome-id":  finding.ID,
						"description":  finding.Description,
						"body":        fmt.Sprintf("**Severity:** %s\n**Status:** %s", finding.Severity, finding.Status),
						"tags": map[string]interface{}{
							"Status":   []map[string]interface{}{{"label": finding.Status, "level": mapSeverityLevel(finding.Severity)}},
						},
					},
				}
				outcomes = append(outcomes, outcome)
			}
			if len(outcomes) == 0 {
				outcomes = append(outcomes, map[string]interface{}{
					"type": "task-result-outcomes",
					"attributes": map[string]interface{}{
						"outcome-id":  "no-findings",
						"description":  "No findings",
						"body":        "No issues were found during the scan.",
						"tags": map[string]interface{}{
							"Status":   []map[string]interface{}{{"label": "Passed", "level": "info"}},
						},
					},
				})
			}
			if policyResult.BreakBuild {
				sendRunTaskStatus(job.Payload.TaskResultCallbackURL, job.Payload.AccessToken, "failed", "Policy violated", map[string]interface{}{"outcomes": outcomes})
			} else {
				sendRunTaskStatus(job.Payload.TaskResultCallbackURL, job.Payload.AccessToken, "passed", "No policy violation", map[string]interface{}{"outcomes": outcomes})
			}
		*/
	}
}

func getOAuth2Token(checkmarxSecrets *secrets.CheckmarxSecrets) (string, error) {
	// TODO: Implement token caching/refresh as needed
	if checkmarxSecrets == nil || checkmarxSecrets.AuthType != secrets.AuthTypeOAuth2 {
		return "", fmt.Errorf("OAuth2 secrets not loaded")
	}
	data := fmt.Sprintf("grant_type=client_credentials&client_id=%s&client_secret=%s",
		checkmarxSecrets.OAuthClientID, checkmarxSecrets.OAuthSecret)
	req, err := http.NewRequest("POST", checkmarxSecrets.OAuthTokenURL, strings.NewReader(data))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := httpclient.Client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}
	var result struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}
	if result.AccessToken == "" {
		return "", fmt.Errorf("no access_token in response")
	}
	return result.AccessToken, nil
}

func runTaskHandler(w http.ResponseWriter, r *http.Request) {
	log.Printf("Received /run-task request from %s", r.RemoteAddr)
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	// HMAC validation if configured
	if appConfig.HMACKey != "" {
		signature := r.Header.Get("X-TFC-Task-Signature")
		if signature == "" {
			log.Printf("Missing HMAC signature in request from %s", r.RemoteAddr)
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte("Missing signature"))
			return
		}

		bodyBytes, err := io.ReadAll(r.Body)
		if err != nil {
			log.Printf("Error reading request body for HMAC validation: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Unable to read request"))
			return
		}

		if !hmac.ValidateSignature(bodyBytes, signature, appConfig.HMACKey) {
			log.Printf("Invalid HMAC signature from %s", r.RemoteAddr)
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte("Invalid signature"))
			return
		}

		// Reset body for further processing
		r.Body = io.NopCloser(bytes.NewReader(bodyBytes))
		log.Printf("HMAC signature validation successful for %s", r.RemoteAddr)
	}

	var payload api.RunTaskPayload
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&payload); err != nil {
		log.Printf("Invalid payload: %v", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// Validate access token presence
	if payload.AccessToken == "" {
		log.Printf("Missing access token in payload from %s", r.RemoteAddr)
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte("Missing access token"))
		return
	}

	// Validate stage - only accept pre-apply tasks
	if payload.Stage != "pre_apply" {
		log.Printf("Rejecting run task with unsupported stage '%s' from %s (only pre_apply supported)", payload.Stage, r.RemoteAddr)
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Only pre_apply stage is supported"))
		return
	}

	// Log only non-sensitive fields
	log.Printf("Run task for org: %s, workspace: %s, run_id: %s, stage: %s", payload.OrganizationName, payload.WorkspaceName, payload.RunID, payload.Stage)

	headers := DownloadHeaders{
		ForwardedFor:   r.Header.Get("X-TFC-Forwarded-For"),
		ForwardedProto: r.Header.Get("X-TFC-Forwarded-Proto"),
		AgentAuth:      r.Header.Get("X-TFC-Agent-Auth"),
		AccessToken:    payload.AccessToken,
	}

	job := JobWithHeaders{
		Payload: payload,
		Headers: headers,
	}

	// Enqueue the job
	select {
	case JobQueue <- job:
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	default:
		w.WriteHeader(http.StatusServiceUnavailable)
		w.Write([]byte("Queue full"))
	}
}

// downloadFileWithHeaders downloads a file from the given URL using the provided headers for request forwarding and authentication.
// It returns the path to the downloaded file or an error.
func downloadFileWithHeaders(url string, headers DownloadHeaders) (string, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", err
	}
	// Prefer AgentAuth if present, otherwise use AccessToken
	if headers.AgentAuth != "" {
		req.Header.Set("X-TFC-Agent-Auth", headers.AgentAuth)
	} else if headers.AccessToken != "" {
		req.Header.Set("Authorization", "Bearer "+headers.AccessToken)
	}
	if headers.ForwardedFor != "" {
		req.Header.Set("X-TFC-Forwarded-For", headers.ForwardedFor)
	}
	if headers.ForwardedProto != "" {
		req.Header.Set("X-TFC-Forwarded-Proto", headers.ForwardedProto)
	}

	resp, err := httpclient.Client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	// Create a temp file
	tmpFile, err := os.CreateTemp("", "hcp-config-*.tar.gz")
	if err != nil {
		return "", err
	}
	defer tmpFile.Close()

	// Write response body to file
	_, err = io.Copy(tmpFile, resp.Body)
	if err != nil {
		return "", err
	}

	return tmpFile.Name(), nil
}
