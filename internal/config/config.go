package config

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"
)

// Config holds all configuration values for the application
type Config struct {
	CheckmarxToken        string `json:"checkmarx_token"`
	CheckmarxRegion       string `json:"checkmarx_region"`
	CheckmarxProjectID    string `json:"checkmarx_project_id"`
	CheckmarxTenant       string `json:"checkmarx_tenant"`
	CheckmarxClientID     string `json:"checkmarx_client_id"`
	CheckmarxClientSecret string `json:"checkmarx_client_secret"`
	CheckmarxBaseURL      string `json:"checkmarx_base_url"`
	CheckmarxBaseAuthURI  string `json:"checkmarx_base_auth_uri"`
	ListenPort            int    `json:"listen_port"`
	ProxyURL              string `json:"proxy_url"`
	BreakDeployment       bool   `json:"break_deployment"`
	HMACKey               string `json:"hmac_key"`
	DisableHMAC           bool   `json:"disable_hmac"`
	ConfigFile            string `json:"-"`
	Debug                 bool   `json:"debug"`
	// TLS Configuration
	TLSEnabled    bool   `json:"tls_enabled"`
	TLSCertFile   string `json:"tls_cert_file"`
	TLSKeyFile    string `json:"tls_key_file"`
	TLSSelfSigned bool   `json:"tls_self_signed"`
}

// Supported Checkmarx One Regions
var RegionBaseURLs = map[string]string{
	"US":  "https://ast.checkmarx.net",
	"US2": "https://us.ast.checkmarx.net",
	"EU":  "https://eu.ast.checkmarx.net",
	"EUS": "https://eu-2.ast.checkmarx.net",
	"DEU": "https://deu.ast.checkmarx.net",
	"ANZ": "https://anz.ast.checkmarx.net",
	"IND": "https://ind.ast.checkmarx.net",
	"SNG": "https://sng.ast.checkmarx.net",
	"MEA": "https://mea.ast.checkmarx.net",
	"IL":  "https://gov-il.ast.checkmarx.net",
}

var RegionAuthURLs = map[string]string{
	"US":  "https://iam.checkmarx.net",
	"US2": "https://us.iam.checkmarx.net",
	"EU":  "https://eu.iam.checkmarx.net",
	"EUS": "https://eu-2.iam.checkmarx.net",
	"DEU": "https://deu.iam.checkmarx.net",
	"ANZ": "https://anz.iam.checkmarx.net",
	"IND": "https://ind.iam.checkmarx.net",
	"SNG": "https://sng.iam.checkmarx.net",
	"MEA": "https://mea.iam.checkmarx.net",
	"IL":  "https://gov-il.iam.checkmarx.net",
}

// LoadConfig loads configuration from flags, environment variables, and config file
func LoadConfig() (*Config, error) {
	cfg := &Config{
		ListenPort: 80,
	}

	// Command-line flags
	flag.StringVar(&cfg.CheckmarxToken, "checkmarx-token", os.Getenv("CHECKMARX_TOKEN"), "Checkmarx API token")
	flag.StringVar(&cfg.CheckmarxProjectID, "checkmarx-project-id", os.Getenv("CHECKMARX_PROJECT_ID"), "Checkmarx project ID")
	flag.IntVar(&cfg.ListenPort, "listen-port", 80, "Port to listen on")
	flag.StringVar(&cfg.ProxyURL, "proxy-url", os.Getenv("PROXY_URL"), "HTTPS proxy URL")
	flag.BoolVar(&cfg.BreakDeployment, "break-deployment", false, "Break deployment on policy violation")
	flag.StringVar(&cfg.ConfigFile, "config", "", "Path to config file (optional)")

	// Region flag
	flag.StringVar(&cfg.CheckmarxRegion, "checkmarx-region", os.Getenv("CHECKMARX_REGION"), "Checkmarx region (e.g. US, US2, EU, EUS, DEU, ANZ, IND, SNG, MEA, IL)")

	// OAuth2 flags (tenant, client id/secret)
	flag.StringVar(&cfg.CheckmarxTenant, "checkmarx-tenant", os.Getenv("CHECKMARX_TENANT"), "Checkmarx tenant name")
	flag.StringVar(&cfg.CheckmarxClientID, "checkmarx-client-id", os.Getenv("CHECKMARX_CLIENT_ID"), "Checkmarx OAuth2 client ID")
	flag.StringVar(&cfg.CheckmarxClientSecret, "checkmarx-client-secret", os.Getenv("CHECKMARX_CLIENT_SECRET"), "Checkmarx OAuth2 client secret")

	// HMAC key flag
	flag.StringVar(&cfg.HMACKey, "hmac-key", os.Getenv("HMAC_KEY"), "HMAC key for validating requests from HCP Terraform")
	flag.BoolVar(&cfg.DisableHMAC, "disable-hmac", false, "Disable HMAC validation entirely (for testing only)")

	// TLS Configuration flags
	defaultTLSEnabled := false
	if v := os.Getenv("TLS_ENABLED"); v == "1" || strings.ToLower(v) == "true" || strings.ToLower(v) == "yes" {
		defaultTLSEnabled = true
	}
	flag.BoolVar(&cfg.TLSEnabled, "tls", defaultTLSEnabled, "Enable TLS/HTTPS server")
	flag.StringVar(&cfg.TLSCertFile, "tls-cert", os.Getenv("TLS_CERT_FILE"), "Path to TLS certificate file")
	flag.StringVar(&cfg.TLSKeyFile, "tls-key", os.Getenv("TLS_KEY_FILE"), "Path to TLS private key file")

	defaultTLSSelfSigned := false
	if v := os.Getenv("TLS_SELF_SIGNED"); v == "1" || strings.ToLower(v) == "true" || strings.ToLower(v) == "yes" {
		defaultTLSSelfSigned = true
	}
	flag.BoolVar(&cfg.TLSSelfSigned, "tls-self-signed", defaultTLSSelfSigned, "Generate self-signed certificate if cert files not provided")

	// Debug flag (also supports DEBUG env var)
	defaultDebug := false
	if v := os.Getenv("DEBUG"); v == "1" || strings.ToLower(v) == "true" || strings.ToLower(v) == "yes" {
		defaultDebug = true
	}
	flag.BoolVar(&cfg.Debug, "debug", defaultDebug, "Enable verbose request/traffic logging (use for troubleshooting only)")

	flag.Parse()

	// If config file is specified, load it and override values
	if cfg.ConfigFile != "" {
		file, err := os.Open(cfg.ConfigFile)
		if err != nil {
			return nil, fmt.Errorf("failed to open config file: %w", err)
		}
		defer file.Close()
		dec := json.NewDecoder(file)
		if err := dec.Decode(cfg); err != nil {
			return nil, fmt.Errorf("failed to decode config file: %w", err)
		}
	}

	// Set HTTPS_PROXY env var if ProxyURL is set
	if cfg.ProxyURL != "" {
		os.Setenv("HTTPS_PROXY", cfg.ProxyURL)
	}

	// Map region to URLs
	if cfg.CheckmarxRegion != "" {
		if baseURL, ok := RegionBaseURLs[cfg.CheckmarxRegion]; ok {
			cfg.CheckmarxBaseURL = baseURL
		}
		if authURL, ok := RegionAuthURLs[cfg.CheckmarxRegion]; ok {
			cfg.CheckmarxBaseAuthURI = authURL
		}
	}

	return cfg, nil
}
