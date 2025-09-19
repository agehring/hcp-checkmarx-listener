#!/bin/bash
# Test script for verifying both pre_apply and post_plan stage support

echo "Testing HCP Terraform Run Task Stage Support"
echo "============================================"

# Test 1: pre_apply stage (should be processed)
echo -e "\n1. Testing pre_apply stage..."
curl -s -X POST http://localhost:8080/api/run-task \
  -H "Content-Type: application/json" \
  -d '{
    "payload_version": 1,
    "stage": "pre_apply",
    "access_token": "test-token",
    "run_id": "run-test-1",
    "workspace_name": "test-workspace",
    "organization_name": "test-org",
    "configuration_version_download_url": "https://example.com/config.tar.gz",
    "task_result_callback_url": "https://example.com/callback"
  }' | head -1

# Test 2: post_plan stage (should be processed)  
echo -e "\n2. Testing post_plan stage..."
curl -s -X POST http://localhost:8080/api/run-task \
  -H "Content-Type: application/json" \
  -d '{
    "payload_version": 1,
    "stage": "post_plan", 
    "access_token": "test-token",
    "run_id": "run-test-2",
    "workspace_name": "test-workspace",
    "organization_name": "test-org",
    "configuration_version_download_url": "https://example.com/config.tar.gz",
    "task_result_callback_url": "https://example.com/callback"
  }' | head -1

# Test 3: unsupported stage (should return 200 OK but not process)
echo -e "\n3. Testing unsupported stage (plan)..."
curl -s -X POST http://localhost:8080/api/run-task \
  -H "Content-Type: application/json" \
  -d '{
    "payload_version": 1,
    "stage": "plan",
    "access_token": "test-token", 
    "run_id": "run-test-3",
    "workspace_name": "test-workspace",
    "organization_name": "test-org"
  }'

echo -e "\n\nTest completed. Check server logs for detailed processing information."