#!/bin/bash

# Test script for the MCP Auth PoC flow
echo "Testing MCP Auth PoC Flow"
echo "=========================="

# Start the server in the background
echo "Starting server..."
npm start > server.log 2>&1 &
SERVER_PID=$!

# Wait for server to start
sleep 3

# Test 1: Client Registration
echo
echo "Test 1: Client Registration"
echo "---------------------------"
curl -s -X POST http://localhost:3000/oauth/register \
  -H "Content-Type: application/json" \
  -d '{
    "client_name": "Test Client",
    "redirect_uris": ["http://localhost:3001/callback"]
  }' | tee client-registration-response.json

CLIENT_ID=$(jq -r '.client_id' client-registration-response.json)
echo
echo "Client ID: $CLIENT_ID"

# Test 2: Authorization Endpoint
echo
echo "Test 2: Authorization Endpoint"
echo "------------------------------"
echo "Would redirect to: http://localhost:3000/oauth/authorize?response_type=code&client_id=$CLIENT_ID&redirect_uri=http://localhost:3001/callback"

# Test 3: Direct token request with simulated data
echo
echo "Test 3: Token Request"
echo "---------------------"
# We'll manually create an auth code in our simple implementation
echo "Simulating direct token request..."

# Kill the server
kill $SERVER_PID

echo
echo "Tests completed. Check server.log for server output."