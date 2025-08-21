import { BlaxelMcpServerTransport, env } from "@blaxel/core";
import "@blaxel/telemetry";
import { McpServer } from "@modelcontextprotocol/sdk/server/mcp.js";
import { StdioServerTransport } from "@modelcontextprotocol/sdk/server/stdio.js";
import "dotenv/config";
// Initialize file-based logging
import { registerAgentsResources } from "./resources/agents.js";
import { registerIntegrationsResources } from "./resources/integrations.js";
import { registerJobsResources } from "./resources/jobs.js";
import { registerMcpServersResources } from "./resources/mcpServers.js";
import { registerModelApisResources } from "./resources/modelApis.js";
import { registerSandboxesResources } from "./resources/sandboxes.js";
import { registerServiceAccountsResources } from "./resources/serviceAccounts.js";
import { registerUsersResources } from "./resources/users.js";
import { registerCreateIntegrationTool } from "./tools/create/integration.js";
import { registerCreateMcpServerTool } from "./tools/create/mcpServer.js";
import { registerCreateModelApiTool } from "./tools/create/modelApi.js";
import { registerCreateServiceAccountTool } from "./tools/create/serviceAccount.js";
import { registerCreateUserTool } from "./tools/create/user.js";
import { registerDeleteAgentTool } from "./tools/delete/agents.js";
import { registerDeleteIntegrationTool } from "./tools/delete/integrations.js";
import { registerDeleteJobTool } from "./tools/delete/jobs.js";
import { registerDeleteMcpServerTool } from "./tools/delete/mcpServers.js";
import { registerDeleteModelApiTool } from "./tools/delete/modelApis.js";
import { registerDeleteSandboxTool } from "./tools/delete/sandboxes.js";
import { registerDeleteServiceAccountTool } from "./tools/delete/serviceAccounts.js";
import { registerDeleteUserTool } from "./tools/delete/users.js";
import { registerListIntegrationModelsTool } from "./tools/list/integrationModels.js";
import { registerGetMcpHubDefinitionTool, registerListMcpHubDefinitionsTool } from "./tools/list/mcpHub.js";
import { registerLocalAgentTool } from "./tools/local/agent.js";
import { registerLocalDeployTool } from "./tools/local/deploy.js";
import { registerLocalJobTool } from "./tools/local/job.js";
import { registerLocalListTemplatesTool } from "./tools/local/listTemplates.js";
import { registerLocalMcpServerTool } from "./tools/local/mcpServer.js";
import { registerLocalRunTool } from "./tools/local/run.js";
import { registerLocalSandboxTool } from "./tools/local/sandbox.js";
import { fileLogger } from "./utils/log.js";

const server = new McpServer({
  name: "blaxel-server",
  version: "1.0.0",
});

function registerResources() {
  registerAgentsResources(server);
  registerModelApisResources(server);
  registerMcpServersResources(server);
  registerSandboxesResources(server);
  registerJobsResources(server);
  registerIntegrationsResources(server);
  registerUsersResources(server);
  registerServiceAccountsResources(server);
}

async function start() {
  const transport = env.BL_SERVER_PORT
    ? new BlaxelMcpServerTransport()
    : new StdioServerTransport();
  registerResources();
  // destructive operations behind explicit tools
  registerDeleteAgentTool(server);
  registerDeleteModelApiTool(server);
  registerDeleteMcpServerTool(server);
  registerDeleteSandboxTool(server);
  registerDeleteJobTool(server);
  registerDeleteIntegrationTool(server);
  registerDeleteUserTool(server);
  registerDeleteServiceAccountTool(server);
  // create operations
  registerCreateIntegrationTool(server);
  registerCreateModelApiTool(server);
  registerCreateMcpServerTool(server);
  registerCreateUserTool(server);
  registerCreateServiceAccountTool(server);
  // listing MCP Hub integrations
  registerListMcpHubDefinitionsTool(server);
  registerGetMcpHubDefinitionTool(server);
  // list models for a given integration connection
  registerListIntegrationModelsTool(server);
  // local tools for creating projects
  registerLocalAgentTool(server);
  registerLocalJobTool(server);
  registerLocalMcpServerTool(server);
  registerLocalSandboxTool(server);
  // local tools for deploying and running
  registerLocalDeployTool(server);
  registerLocalRunTool(server);
  registerLocalListTemplatesTool(server);
  await server.connect(transport);
  fileLogger.info("Server started; resources and delete tools registered");
}

start().catch((err) => {
  fileLogger.error("Failed to start MCP server", err);
  process.exit(1);
});
