package api

// RunTaskPayload represents the HCP Terraform Run Task JSON payload.
type RunTaskPayload struct {
	PayloadVersion                  int          `json:"payload_version"`
	Stage                           string       `json:"stage"`
	AccessToken                     string       `json:"access_token"`
	Capabilities                    Capabilities `json:"capabilities"`
	ConfigurationVersionDownloadURL string       `json:"configuration_version_download_url"`
	ConfigurationVersionID          string       `json:"configuration_version_id"`
	IsSpeculative                   bool         `json:"is_speculative"`
	OrganizationName                string       `json:"organization_name"`
	PlanJSONAPIURL                  string       `json:"plan_json_api_url"`
	RunAppURL                       string       `json:"run_app_url"`
	RunCreatedAt                    string       `json:"run_created_at"`
	RunCreatedBy                    string       `json:"run_created_by"`
	RunID                           string       `json:"run_id"`
	RunMessage                      string       `json:"run_message"`
	TaskResultCallbackURL           string       `json:"task_result_callback_url"`
	TaskResultEnforcementLevel      string       `json:"task_result_enforcement_level"`
	TaskResultID                    string       `json:"task_result_id"`
	VCSBranch                       string       `json:"vcs_branch"`
	VCSCommitURL                    string       `json:"vcs_commit_url"`
	VCSPullRequestURL               *string      `json:"vcs_pull_request_url"`
	VCSRepoURL                      string       `json:"vcs_repo_url"`
	WorkspaceAppURL                 string       `json:"workspace_app_url"`
	WorkspaceID                     string       `json:"workspace_id"`
	WorkspaceName                   string       `json:"workspace_name"`
	WorkspaceWorkingDirectory       string       `json:"workspace_working_directory"`
}

type Capabilities struct {
	Outcomes bool `json:"outcomes"`
}
