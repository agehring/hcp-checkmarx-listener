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
	"github.com/Checkmarx-PS/hcp-checkmarx-listener/internal/terraform"
	tlsutil "github.com/Checkmarx-PS/hcp-checkmarx-listener/internal/tls"
	"github.com/cxpsemea/Cx1ClientGo"
	"github.com/sirupsen/logrus"
)

// maskToken returns a masked version of a token for debug logging
func maskToken(token string) string {
	if len(token) <= 8 {
		return "****"
	}
	return token[:4] + "..." + token[len(token)-4:]
}

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
	if appConfig.Debug {
		log.Printf("DEBUG MODE ENABLED: All requests and errors will be logged verbosely.")
		log.Printf("DEBUG: Configuration loaded - Tenant: '%s', Token: %s, ClientID: %s",
			appConfig.CheckmarxTenant,
			maskToken(appConfig.CheckmarxToken),
			maskToken(appConfig.CheckmarxClientID))
	}

	// Startup: acquire Checkmarx client or token
	var cx1client *Cx1ClientGo.Cx1Client
	if appConfig.CheckmarxToken != "" && appConfig.CheckmarxToken != "your-api-key-here" {
		// API Key mode
		if appConfig.Debug {
			log.Printf("DEBUG: Using API Key for Checkmarx One authentication.")
		}
		httpClient := httpclient.NewClient()
		// Create a logger compatible with Cx1ClientGo
		logger := logrus.New()
		logger.SetLevel(logrus.ErrorLevel) // Suppress SDK warnings
		logger.SetOutput(os.Stdout)
		logger.SetFormatter(&logrus.TextFormatter{
			FullTimestamp: true,
		})
		cx1client, err = Cx1ClientGo.NewAPIKeyClient(httpClient, appConfig.CheckmarxBaseURL, appConfig.CheckmarxBaseAuthURI, appConfig.CheckmarxTenant, appConfig.CheckmarxToken, logger)
		if err != nil {
			log.Fatalf("Startup: failed to create CheckmarxOne SDK client: %v", err)
		}
		// Set custom User-Agent for SDK requests as well
		cx1client.SetUserAgent("CxOne-HCP")
		// Verify connectivity via SDK
		_, err = cx1client.GetGroups()
		if err != nil {
			log.Fatalf("Startup: Checkmarx One API Key authentication failed: %v", err)
		}
		tenantInfo := appConfig.CheckmarxTenant
		if tenantInfo == "" {
			tenantInfo = "[not configured]"
		}
		log.Printf("Auth: Checkmarx One API Key authentication successful")
	} else if appConfig.CheckmarxClientID != "" && appConfig.CheckmarxClientSecret != "" {
		// OAuth2 mode
		if appConfig.Debug {
			log.Printf("DEBUG: Using OAuth2 for Checkmarx One authentication.")
		}
		token, err := checkmarx.GetCheckmarxToken(
			"",
			appConfig.CheckmarxTenant,
			appConfig.CheckmarxClientID,
			appConfig.CheckmarxClientSecret,
			appConfig.CheckmarxBaseAuthURI,
		)
		if err != nil {
			log.Fatalf("Startup: failed to acquire Checkmarx OAuth2 token: %v", err)
		}
		appConfig.CheckmarxToken = token
		statesURL := appConfig.CheckmarxBaseURL + "/api/lists/states"
		req, err := http.NewRequest("GET", statesURL, nil)
		if err != nil {
			log.Fatalf("Startup: failed to create states list request for Checkmarx One: %v", err)
		}
		req.Header.Set("Authorization", "Bearer "+appConfig.CheckmarxToken)
		req.Header.Set("Accept", "application/json; version=1.0")
		resp, err := httpclient.Client.Do(req)
		if err != nil {
			if appConfig.Debug {
				log.Printf("DEBUG: Checkmarx One connectivity/authentication FAILED: %v", err)
			}
			log.Fatalf("Startup: unable to reach Checkmarx One tenant at %s: %v", statesURL, err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			if appConfig.Debug {
				log.Printf("DEBUG: Checkmarx One authentication failed, status %d at %s", resp.StatusCode, statesURL)
			}
			log.Fatalf("Startup: Checkmarx One states list failed (%d) at %s", resp.StatusCode, statesURL)
		}
		var states []string
		if err := json.NewDecoder(resp.Body).Decode(&states); err != nil {
			if appConfig.Debug {
				log.Printf("DEBUG: Checkmarx One response decode failed: %v", err)
			}
			log.Fatalf("Startup: failed to decode states list from Checkmarx One: %v", err)
		}
		requiredStates := map[string]bool{
			"TO_VERIFY":                true,
			"NOT_EXPLOITABLE":          true,
			"PROPOSED_NOT_EXPLOITABLE": true,
			"CONFIRMED":                true,
			"URGENT":                   true,
		}
		missing := []string{}
		for reqState := range requiredStates {
			found := false
			for _, s := range states {
				if s == reqState {
					found = true
					break
				}
			}
			if !found {
				missing = append(missing, reqState)
			}
		}
		if len(missing) > 0 {
			if appConfig.Debug {
				log.Printf("DEBUG: Checkmarx One states list missing required states: %v", missing)
			}
			log.Fatalf("Startup: Checkmarx One states list missing required states: %v", missing)
		}
		if appConfig.Debug {
			log.Printf("DEBUG: Checkmarx One authentication/connectivity SUCCESS. Endpoint: %s, Token: %s", statesURL, maskToken(appConfig.CheckmarxToken))
		}
		tenantInfo := appConfig.CheckmarxTenant
		if tenantInfo == "" {
			tenantInfo = "[not configured]"
		}
		log.Printf("Auth: Checkmarx One OAuth2 authentication successful (tenant: %s)", tenantInfo)
	} else {
		// Neither API key nor OAuth2 credentials provided
		log.Fatalf("Startup: No valid Checkmarx One credentials provided. Please configure either:\n" +
			"  - API Key: set checkmarx_token and checkmarx_tenant in config.json\n" +
			"  - OAuth2: set checkmarx_client_id, checkmarx_client_secret, and checkmarx_tenant in config.json")
	}

	// Start worker goroutine
	go jobWorker(cx1client)

	http.HandleFunc("/api/run-task", runTaskHandler)
	http.HandleFunc("/health", healthHandler)
	// Root handler last to avoid shadowing more specific paths
	http.HandleFunc("/", rootHandler)
	addr := fmt.Sprintf(":%d", appConfig.ListenPort)

	// Setup TLS configuration
	tlsConfig, err := tlsutil.LoadTLSConfig(appConfig.TLSEnabled, appConfig.TLSCertFile, appConfig.TLSKeyFile, appConfig.TLSSelfSigned)
	if err != nil {
		log.Fatalf("Failed to setup TLS configuration: %v", err)
	}

	// Wrap default mux with security and logging middleware
	var handler http.Handler = http.DefaultServeMux
	handler = securityMiddleware(handler)
	handler = loggingMiddleware(handler)

	// Create server with optional TLS
	server := &http.Server{
		Addr:      addr,
		Handler:   handler,
		TLSConfig: tlsConfig,
	}

	// Start server with proper port logging
	if appConfig.TLSEnabled {
		log.Printf("Server: Starting HTTPS server on %s (TLS enabled)", addr)
		if err := server.ListenAndServeTLS("", ""); err != nil {
			log.Fatalf("Failed to start HTTPS server: %v", err)
		}
	} else {
		log.Printf("Server: Starting HTTP server on %s", addr)
		if err := server.ListenAndServe(); err != nil {
			log.Fatalf("Failed to start HTTP server: %v", err)
		}
	}
}

// securityMiddleware adds security headers including HSTS for production security compliance
func securityMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// HSTS header - always set for security compliance
		// For HTTPS: standard HSTS with subdomains
		// For HTTP: still set to indicate HTTPS preference (some SAST tools require this)
		if r.TLS != nil {
			w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		} else {
			// Even for HTTP, set HSTS to indicate HTTPS is required for security
			w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		}

		// Additional security headers
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("X-XSS-Protection", "1; mode=block")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")

		// Content Security Policy for API endpoints
		w.Header().Set("Content-Security-Policy", "default-src 'none'; script-src 'none'; object-src 'none'")

		next.ServeHTTP(w, r)
	})
}

// writeErrorWithHSTS ensures HSTS header is set before writing error responses (for SAST compliance)
func writeErrorWithHSTS(w http.ResponseWriter, statusCode int, message string) {
	// Explicitly set HSTS header for error responses to satisfy SAST requirements
	w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
	w.WriteHeader(statusCode)
	w.Write([]byte(message))
}

// loggingMiddleware logs incoming requests when debug is enabled, masking sensitive headers and safely logging bodies
// Also logs connection source for supported endpoints in production
func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Log connection source for supported endpoints (always)
		if r.URL.Path == "/api/run-task" || r.URL.Path == "/health" {
			log.Printf("Connection: %s %s from %s", r.Method, r.URL.Path, r.RemoteAddr)
		}

		// Debug logging (when debug mode is enabled)
		if appConfig != nil && appConfig.Debug {
			log.Printf("DEBUG incoming: %s %s from %s", r.Method, r.URL.String(), r.RemoteAddr)
			for k, vv := range r.Header {
				if len(vv) == 0 {
					continue
				}
				log.Printf("DEBUG header: %s: %s", k, maskHeader(k, vv[0]))
			}
			bodyBytes, err := io.ReadAll(r.Body)
			if err == nil {
				max := 64 * 1024
				bb := bodyBytes
				if len(bb) > max {
					bb = bb[:max]
				}
				if len(bb) > 0 {
					log.Printf("DEBUG body (truncated %d/%d): %s", len(bb), len(bodyBytes), string(bb))
				}
				r.Body = io.NopCloser(bytes.NewReader(bodyBytes))
			} else {
				log.Printf("DEBUG failed to read body for logging: %v", err)
			}
		}
		next.ServeHTTP(w, r)
	})
}

// maskHeader redacts sensitive headers in debug logs
func maskHeader(key, value string) string {
	kl := strings.ToLower(key)
	if strings.Contains(kl, "authorization") || strings.Contains(kl, "token") || strings.Contains(kl, "secret") || strings.Contains(kl, "signature") {
		return "[redacted]"
	}
	return value
}

// rootHandler returns 418 I'm a teapot for any endpoint other than /api/run-task
func rootHandler(w http.ResponseWriter, r *http.Request) {
	// Only allow requests to /api/run-task and /health, return 418 for everything else
	if r.URL.Path != "/api/run-task" && r.URL.Path != "/health" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusTeapot) // 418 I'm a teapot
		_ = json.NewEncoder(w).Encode(map[string]string{
			"error": "I'm a teapot",
		})
		return
	}

	// This should not be reached since specific handlers are registered for /api/run-task and /health
	// But keeping it as a fallback
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusTeapot) // 418 I'm a teapot
	_ = json.NewEncoder(w).Encode(map[string]string{
		"error": "I'm a teapot",
	})
}

// healthHandler returns a simple liveness response.
func healthHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
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

func jobWorker(cx1client *Cx1ClientGo.Cx1Client) {
	for job := range JobQueue {
		log.Printf("Processing: %s stage job for workspace %s (run: %s)",
			job.Payload.Stage, job.Payload.WorkspaceName, job.Payload.RunID)

		if appConfig != nil && appConfig.Debug {
			log.Printf("DEBUG: Processing job payload: %+v", job.Payload)
		}

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

		log.Printf("Downloaded file to: %s", filePath)

		// Parse CxOne configuration from Terraform variables
		cxOneConfig, err := terraform.ParseCxOneConfig(filePath)
		if err != nil {
			log.Printf("Error parsing CxOne config from Terraform, using defaults: %v", err)
			// Use default configuration from app config
			cxOneConfig = terraform.CxOneConfig{
				ProjectName: appConfig.CheckmarxProjectID, // Use existing project ID as default
				Group:       "hcp-tasks",                  // Default group name
				Branch:      "main",                       // Default branch
			}
		}

		// Ensure group is not empty (Checkmarx API requirement)
		if cxOneConfig.Group == "" {
			cxOneConfig.Group = "hcp-tasks"
			log.Printf("Empty group name detected, using default: %s", cxOneConfig.Group)
		}

		log.Printf("Using CxOne configuration - Project: %s, Group: %s, Branch: %s",
			cxOneConfig.ProjectName, cxOneConfig.Group, cxOneConfig.Branch)

		// Submit file to Checkmarx One for IaC (Kics) scan using the authenticated SDK client
		scanResult, err := checkmarx.SubmitIaCScan(
			cx1client,
			filePath,
			job.Payload.TaskResultID,
			cxOneConfig.ProjectName,
			cxOneConfig.Group,
			cxOneConfig.Branch,
			appConfig.CheckmarxBaseURL,
			appConfig.CheckmarxToken,
		)
		if err != nil {
			log.Printf("Checkmarx scan submission failed: %v", err)
			sendRunTaskStatus(job.Payload.TaskResultCallbackURL, job.Payload.AccessToken, "failed", "Checkmarx scan submission failed", map[string]interface{}{"error": err.Error()})
			continue
		}
		sendRunTaskStatus(job.Payload.TaskResultCallbackURL, job.Payload.AccessToken, "running", "Submitted to Checkmarx for scan", map[string]interface{}{"scan_id": scanResult.CheckmarxScanID})

		// Wait for scan completion
		log.Printf("Waiting for scan %s to complete...", scanResult.CheckmarxScanID)
		err = checkmarx.WaitForScanCompletion(cx1client, scanResult.CheckmarxScanID, 10*60*time.Second)
		if err != nil {
			log.Printf("Scan did not complete successfully: %v", err)
			sendRunTaskStatus(job.Payload.TaskResultCallbackURL, job.Payload.AccessToken, "failed", "Scan did not complete successfully", map[string]interface{}{"error": err.Error()})
			continue
		}
		sendRunTaskStatus(job.Payload.TaskResultCallbackURL, job.Payload.AccessToken, "running", "Scan completed", map[string]interface{}{"scan_id": scanResult.CheckmarxScanID})

		// Use authenticated SDK client from startup
		err = checkmarx.WaitForScanCompletion(cx1client, scanResult.CheckmarxScanID, 10*60*time.Second)
		if err != nil {
			log.Printf("Scan did not complete successfully: %v", err)
			sendRunTaskStatus(job.Payload.TaskResultCallbackURL, job.Payload.AccessToken, "failed", "Scan did not complete successfully (auth)", map[string]interface{}{"error": err.Error()})
			continue
		}
		// Check for policy violation
		log.Printf("Policy: Checking policy violations for scan ID: %s", scanResult.CheckmarxScanID)
		policyResult, err := checkmarx.CheckPolicyViolation(cx1client, scanResult.CheckmarxScanID, appConfig.CheckmarxBaseURL, appConfig.CheckmarxToken)
		if err != nil {
			log.Printf("Policy check failed: %v", err)
			sendRunTaskStatus(job.Payload.TaskResultCallbackURL, job.Payload.AccessToken, "failed", "Policy check failed", map[string]interface{}{"error": err.Error()})
			continue
		}

		log.Printf("Policy: Check completed - Status: %s, BreakBuild: %t, Policies evaluated: %d",
			policyResult.Status, policyResult.BreakBuild, len(policyResult.Policies))

		if appConfig.Debug {
			log.Printf("[DEBUG] Policy check completed successfully. Status: %s, BreakBuild: %t, Policies: %d",
				policyResult.Status, policyResult.BreakBuild, len(policyResult.Policies))
			log.Printf("[DEBUG] Configuration: BreakDeployment=%t", appConfig.BreakDeployment)
			log.Printf("[DEBUG] Build will break: %t (policy: %t OR config: %t)",
				policyResult.BreakBuild || appConfig.BreakDeployment, policyResult.BreakBuild, appConfig.BreakDeployment)
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
		// Break the build if EITHER the policy indicates breaking OR the config requires breaking deployments
		if policyResult.BreakBuild || appConfig.BreakDeployment {
			status = "failed"
			if policyResult.BreakBuild && appConfig.BreakDeployment {
				msg = "Policy violation(s) detected (policy and configuration both require build break)."
			} else if policyResult.BreakBuild {
				msg = "Policy violation(s) detected (policy requires build break)."
			} else {
				msg = "Build break enforced by configuration."
			}
		}
		// Fetch IaC findings from Checkmarx One
		findings, err := checkmarx.FetchIaCFindings(cx1client, scanResult.CheckmarxScanID, appConfig.CheckmarxBaseURL, appConfig.Debug)
		if err != nil {
			log.Printf("Failed to fetch IaC findings: %v", err)
			sendRunTaskStatus(job.Payload.TaskResultCallbackURL, job.Payload.AccessToken, "failed", "Failed to fetch IaC findings: "+err.Error(), nil)
			continue
		}

		log.Printf("Scan: Retrieved %d IaC findings from Checkmarx One (scan ID: %s)", len(findings), scanResult.CheckmarxScanID)

		// Calculate severity breakdown
		severityCount := make(map[string]int)
		criticalFindings := []checkmarx.IaCFinding{}
		for _, finding := range findings {
			severityCount[finding.Severity]++
			if strings.ToUpper(finding.Severity) == "CRITICAL" {
				criticalFindings = append(criticalFindings, finding)
			}
		}

		// Log severity breakdown and critical findings only in debug mode
		if appConfig != nil && appConfig.Debug {
			log.Printf("DEBUG: Severity breakdown - %+v", severityCount)
			if len(criticalFindings) > 0 {
				log.Printf("DEBUG: Found %d CRITICAL findings:", len(criticalFindings))
				for i, finding := range criticalFindings {
					if i >= 3 { // Only show first 3
						log.Printf("DEBUG: ... and %d more critical findings", len(criticalFindings)-3)
						break
					}
					log.Printf("DEBUG:   CRITICAL[%d]: %s in %s (Line %d) - Status: %s",
						i+1, finding.QueryName, finding.FileName, finding.Line, finding.Status)
				}
			} else {
				log.Printf("DEBUG: NO CRITICAL findings found in retrieved results")
			}
		} else {
			// Production mode: no detailed scan result logging
			// Critical findings will be reported through HCP Terraform outcomes
		} // Convert findings to outcomes format
		var findingOutcomes []interface{}
		for _, finding := range findings {
			severityLevel := "info"
			switch strings.ToUpper(finding.Severity) {
			case "CRITICAL":
				severityLevel = "error"
			case "HIGH":
				severityLevel = "error"
			case "MEDIUM":
				severityLevel = "warning"
			case "LOW":
				severityLevel = "info"
			case "INFO":
				severityLevel = "info"
			default:
				severityLevel = "info"
			}

			outcome := map[string]interface{}{
				"type": "task-result-outcomes",
				"attributes": map[string]interface{}{
					"outcome-id":  finding.ID,
					"description": finding.QueryName,
					"body":        fmt.Sprintf("**Severity:** %s\n**File:** %s\n**Line:** %d\n\n%s", finding.Severity, finding.FileName, finding.Line, finding.Description),
					"tags": map[string]interface{}{
						"Severity": []map[string]interface{}{{"label": finding.Severity, "level": severityLevel}},
						"Status":   []map[string]interface{}{{"label": finding.Status, "level": "info"}},
					},
				},
			}
			findingOutcomes = append(findingOutcomes, outcome)
		}

		// Combine policy outcomes and finding outcomes
		allOutcomes := policyOutcomes
		allOutcomes = append(allOutcomes, findingOutcomes...)

		// If no findings, add a success outcome
		if len(findingOutcomes) == 0 {
			successOutcome := map[string]interface{}{
				"type": "task-result-outcomes",
				"attributes": map[string]interface{}{
					"outcome-id":  "no-iac-findings",
					"description": "No IaC Security Issues Found",
					"body":        "The Checkmarx IaC scan completed successfully with no security issues found.",
					"tags": map[string]interface{}{
						"Status": []map[string]interface{}{{"label": "Passed", "level": "info"}},
					},
				},
			}
			allOutcomes = append(allOutcomes, successOutcome)
		}

		// Send final status with all outcomes
		sendRunTaskStatus(job.Payload.TaskResultCallbackURL, job.Payload.AccessToken, status, msg, map[string]interface{}{"outcomes": allOutcomes})
	}
}

func runTaskHandler(w http.ResponseWriter, r *http.Request) {
	log.Printf("Received /api/run-task request from %s", r.RemoteAddr)
	if appConfig != nil && appConfig.Debug {
		log.Printf("DEBUG: runTaskHandler invoked. Method: %s, Remote: %s", r.Method, r.RemoteAddr)
	}
	if r.Method == http.MethodGet {
		// Allow GET for HCP handshake/validation
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"endpoint": "api/run-task", "status": "ok", "note": "POST only for processing"})
		return
	}
	if r.Method != http.MethodPost {
		if appConfig != nil && appConfig.Debug {
			log.Printf("Method not allowed: %s %s", r.Method, r.URL.String())
		}
		writeErrorWithHSTS(w, http.StatusMethodNotAllowed, "Method not allowed; use POST for /api/run-task")
		return
	}

	// Read body once for subsequent validation and decoding
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("Error reading request body: %v", err)
		writeErrorWithHSTS(w, http.StatusBadRequest, "Unable to read request body")
		return
	}
	// Reset body for any downstream readers
	r.Body = io.NopCloser(bytes.NewReader(bodyBytes))

	// Decode payload first to decide handling path
	var payload api.RunTaskPayload
	if err := json.Unmarshal(bodyBytes, &payload); err != nil {
		log.Printf("Invalid payload: %v", err)
		writeErrorWithHSTS(w, http.StatusBadRequest, "Invalid payload")
		return
	}
	if appConfig != nil && appConfig.Debug {
		log.Printf("DEBUG: Decoded payload. Stage: %s, RunID: %s", payload.Stage, payload.RunID)
	}

	// Supported stages: pre_apply and post_plan
	supportedStages := map[string]bool{
		"pre_apply": true,
		"post_plan": true,
	}

	// Explicitly unsupported stages that should receive "stage not supported" response
	unsupportedStages := map[string]bool{
		"pre_plan":   true,
		"post_apply": true,
	}

	// HMAC validation (enforced for ALL stages, including non-supported ones)
	if appConfig.HMACKey != "" && !appConfig.DisableHMAC {
		signature := r.Header.Get("X-TFC-Task-Signature")
		if appConfig != nil && appConfig.Debug {
			log.Printf("DEBUG: Enforcing HMAC for %s stage. Signature header: '%s'", payload.Stage, signature)
		}
		if signature == "" {
			log.Printf("Security: HMAC validation failed - missing signature from %s (stage: %s)", r.RemoteAddr, payload.Stage)
			if appConfig != nil && appConfig.Debug {
				log.Printf("DEBUG: 401 due to missing signature header.")
			}
			writeErrorWithHSTS(w, http.StatusUnauthorized, "Missing signature")
			return
		}
		if !hmac.ValidateSignature(bodyBytes, signature, appConfig.HMACKey) {
			log.Printf("Security: HMAC validation failed - invalid signature from %s (stage: %s)", r.RemoteAddr, payload.Stage)
			if appConfig != nil && appConfig.Debug {
				log.Printf("DEBUG: 401 due to invalid HMAC signature. Payload: %s", string(bodyBytes))
			}
			writeErrorWithHSTS(w, http.StatusUnauthorized, "Invalid signature")
			return
		}
		log.Printf("Security: HMAC validation successful for %s stage from %s", payload.Stage, r.RemoteAddr)
	} else if appConfig.DisableHMAC {
		log.Printf("Security: HMAC validation disabled - allowing request from %s (stage: %s)", r.RemoteAddr, payload.Stage)
		if appConfig.Debug {
			log.Printf("DEBUG: HMAC validation disabled - skipping signature check")
		}
	}

	// Handle explicitly unsupported stages (pre_plan, post_apply) with specific response
	if unsupportedStages[payload.Stage] {
		if appConfig != nil && appConfig.Debug {
			log.Printf("DEBUG: Stage '%s' is explicitly not supported. Returning stage not supported response.", payload.Stage)
		}
		log.Printf("Stage '%s' is not supported by this service from %s", payload.Stage, r.RemoteAddr)

		// Send callback to HCP Terraform if we have the necessary information
		if payload.TaskResultCallbackURL != "" && payload.AccessToken != "" {
			go func() {
				if appConfig != nil && appConfig.Debug {
					log.Printf("DEBUG: Sending 'stage not supported' callback for stage '%s'", payload.Stage)
				}
				sendRunTaskStatus(payload.TaskResultCallbackURL, payload.AccessToken, "passed", "Stage not supported", nil)
			}()
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Stage not supported"))
		return
	}

	// Other non-supported stages receive 200 OK no-op after HMAC validation
	if !supportedStages[payload.Stage] {
		if appConfig != nil && appConfig.Debug {
			log.Printf("DEBUG: Stage '%s' passed HMAC validation but is not supported. Returning 200.", payload.Stage)
		}
		log.Printf("Ignoring run task with unsupported stage '%s' from %s (supported: pre_apply, post_plan)", payload.Stage, r.RemoteAddr)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Stage not processed; only pre_apply and post_plan are handled"))
		return
	}

	// Validate access token presence (required for callbacks)
	if payload.AccessToken == "" {
		log.Printf("Missing access token in payload from %s", r.RemoteAddr)
		writeErrorWithHSTS(w, http.StatusUnauthorized, "Missing access token")
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
