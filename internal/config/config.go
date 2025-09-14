package config

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
)

type Config struct {
	CheckmarxToken     string `json:"checkmarx_token"`
	CheckmarxProjectID string `json:"checkmarx_project_id"`
	ListenPort         int    `json:"listen_port"`
	ProxyURL           string `json:"proxy_url"`
	BreakDeployment    bool   `json:"break_deployment"`
	ConfigFile         string `json:"-"`
	HMACKey            string `json:"hmac_key"`

	// Region selection (e.g. US, US2, EU, EUS, DEU, ANZ, IND, SNG, MEA, IL)
	CheckmarxRegion string `json:"checkmarx_region"`

	// Derived fields (populated from region)
	CheckmarxBaseURL      string `json:"checkmarx_base_url"`
	CheckmarxBaseAuthURI  string `json:"checkmarx_base_auth_uri"`
	CheckmarxTenant       string `json:"checkmarx_tenant"`
	CheckmarxClientID     string `json:"checkmarx_client_id"`
	CheckmarxClientSecret string `json:"checkmarx_client_secret"`
}

// LoadConfig loads configuration from flags, environment, and optionally a config file.
func LoadConfig() (*Config, error) {
	cfg := &Config{}

	// Command-line flags
	flag.StringVar(&cfg.CheckmarxToken, "checkmarx-token", os.Getenv("CHECKMARX_TOKEN"), "Checkmarx API token")
	flag.StringVar(&cfg.CheckmarxProjectID, "checkmarx-project-id", os.Getenv("CHECKMARX_PROJECT_ID"), "Checkmarx Project ID")
	flag.IntVar(&cfg.ListenPort, "listen-port", 80, "Port to listen on")
	flag.StringVar(&cfg.ProxyURL, "proxy-url", os.Getenv("PROXY_URL"), "Proxy URL for outbound requests")
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
	regionMap := map[string]struct{ base, auth string }{
		"US":  {"https://ast.checkmarx.net", "https://iam.checkmarx.net"},
		"US2": {"https://us.ast.checkmarx.net", "https://us.iam.checkmarx.net"},
		"EU":  {"https://eu.ast.checkmarx.net", "https://eu.iam.checkmarx.net"},
		"EUS": {"https://eu-2.ast.checkmarx.net", "https://eu-2.iam.checkmarx.net"},
		"DEU": {"https://deu.ast.checkmarx.net", "https://deu.iam.checkmarx.net"},
		"ANZ": {"https://anz.ast.checkmarx.net", "https://anz.iam.checkmarx.net"},
		"IND": {"https://ind.ast.checkmarx.net", "https://ind.iam.checkmarx.net"},
		"SNG": {"https://sng.ast.checkmarx.net", "https://sng.iam.checkmarx.net"},
		"MEA": {"https://mea.ast.checkmarx.net", "https://mea.iam.checkmarx.net"},
		"IL":  {"https://gov-il.ast.checkmarx.net", "https://gov-il.iam.checkmarx.net"},
	}
	if v, ok := regionMap[cfg.CheckmarxRegion]; ok {
		cfg.CheckmarxBaseURL = v.base
		cfg.CheckmarxBaseAuthURI = v.auth
	}

	return cfg, nil
}
