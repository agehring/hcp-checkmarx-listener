Project Structure & Go Module Init

Create a clean directory structure (e.g., cmd/, internal/, pkg/, api/).
Initialize a Go module with go mod init.
Define the Payload Struct

Create a Go struct that maps to the HCP Terraform Run Task JSON payload.
Basic Web Service

Implement a simple HTTP server listening on port 80.
Add a POST endpoint to receive the JSON payload, respond with 200 OK, and enqueue the job.
Job Queue & Worker

Implement a job queue and a worker goroutine to process jobs asynchronously.
Download Function

Write a function to download a file from a URL with an Authorization header.
Checkmarx API Integration

Stub out functions for submitting a scan and checking policy violations.
Config & Proxy Support

Add config file and command-line flag parsing for proxy and “break deployment” options.
Dockerfile

Create a secure, multi-stage Dockerfile for the service.
