#!/bin/bash

# Demo script for Qwen MCP Auth PoC
echo "Qwen MCP Auth PoC Demo"
echo "======================"

# Start the server in the background
echo "1. Starting the MCP Auth server..."
npm start > server.log 2>&1 &
SERVER_PID=$!

# Wait for server to start
sleep 3

# Test the well-known endpoints
echo
echo "2. Testing well-known endpoints..."
echo "----------------------------------"
echo "RFC 8414 Authorization Server Metadata:"
curl -s http://localhost:3000/.well-known/oauth-authorization-server | jq 'with_entries(select([.key] | inside(["issuer", "authorization_endpoint", "token_endpoint"])) )'
echo
echo "RFC 9728 Protected Resource Metadata:"
curl -s http://localhost:3000/.well-known/oauth-protected-resource | jq .

# Test dynamic client registration
echo
echo "3. Registering a test client..."
echo "-------------------------------"
curl -s -X POST http://localhost:3000/oauth/register \
  -H "Content-Type: application/json" \
  -d '{
    "client_name": "Demo Client",
    "redirect_uris": ["http://localhost:3001/callback"]
  }' | tee client-registration.json

CLIENT_ID=$(jq -r '.client_id' client-registration.json)
echo
echo "Registered client with ID: $CLIENT_ID"

# Test unauthenticated MCP call
echo
echo "4. Testing unauthenticated MCP call..."
echo "--------------------------------------"
curl -i -X POST http://localhost:3000/mcp \
  -H "Content-Type: application/json" \
  -d '{}' | head -5

# Clean up
echo
echo "5. Cleaning up..."
echo "-----------------"
kill $SERVER_PID 2>/dev/null
rm -f client-registration.json
echo "Demo completed. Server log available in server.log"

echo
echo "To test with a real MCP client:"
echo "1. Create a .mcp.json file with the following content:"
echo '{'
echo '  "mcpServers": {'
echo '    "qwen-auth-poc": {'
echo '      "type": "http",'
echo '      "url": "http://localhost:3000/mcp"'
echo '    }'
echo '  }'
echo '}'
echo "2. Launch Claude Code in the directory with .mcp.json"
echo "3. Try to use a tool from the server to trigger the auth flow"