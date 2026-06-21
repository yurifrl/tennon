FROM node:22-alpine
RUN npm install -g supergateway @modelcontextprotocol/server-everything
EXPOSE 8000
# stdio MCP server (server-everything) bridged to streamable HTTP at /mcp
CMD ["supergateway", \
     "--stdio", "mcp-server-everything stdio", \
     "--outputTransport", "streamableHttp", \
     "--port", "8000", \
     "--streamableHttpPath", "/mcp"]
