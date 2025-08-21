import path from "node:path";
import { fileURLToPath } from "node:url";
import { afterAll, assert, beforeAll, describe, expect, it } from "vitest";

// MCP Client imports
import { Client } from "@modelcontextprotocol/sdk/client/index.js";
import { StdioClientTransport } from "@modelcontextprotocol/sdk/client/stdio.js";

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);

describe("Blaxel MCP e2e with MCP Client (Agent Simulation)", () => {
  const projectRoot = path.resolve(__dirname, "..");
  const tsxBin = path.resolve(projectRoot, "node_modules", ".bin", process.platform === "win32" ? "tsx.cmd" : "tsx");
  const serverEntry = path.resolve(projectRoot, "src", "index.ts");

  let transport: StdioClientTransport | null = null;
  let mcpClient: Client | null = null;

  const requires = [
    "BL_WORKSPACE",
    "BL_API_KEY",
    "BL_TEST_INTEGRATION",
    "BL_TEST_INTEGRATION_SECRET_JSON",
    "OPENAI_API_KEY"
  ] as const;
  const missingEnv = requires.filter((k) => !process.env[k] || String(process.env[k]).length === 0);

  beforeAll(async () => {
    if (missingEnv.length) return; // Skip setup; test will be skipped

    // Create transport that will spawn the server
    transport = new StdioClientTransport({
      command: tsxBin,
      args: [serverEntry],
      env: {
        ...process.env,
        BL_SERVER_PORT: "", // Force stdio transport
      },
    });

    // Create MCP client
    mcpClient = new Client({
      name: "test-agent-client",
      version: "1.0.0",
    }, {
      capabilities: {}
    });

    // Connect to the server
    await mcpClient.connect(transport);
  }, 30_000);

  afterAll(async () => {
    // Clean up
    if (mcpClient) {
      try {
        await mcpClient.close();
      } catch {}
      mcpClient = null;
    }
    if (transport) {
      try {
        await transport.close();
      } catch {}
      transport = null;
    }
  });

  it(
    missingEnv.length
      ? `skipped: missing env ${missingEnv.join(", ")}`
      : "simulates agent workflow: list integrations, create integration, then create linked MCP",
    async () => {
      assert(mcpClient !== null, "MCP client not initialized");

      const integrationId = String(process.env.BL_TEST_INTEGRATION);
      const integrationSecret = JSON.parse(String(process.env.BL_TEST_INTEGRATION_SECRET_JSON));
      const integrationConfig = process.env.BL_TEST_INTEGRATION_CONFIG_JSON
        ? JSON.parse(String(process.env.BL_TEST_INTEGRATION_CONFIG_JSON))
        : undefined;

      // Generate unique names for this test run
      const connectionName = `e2e-int-${Date.now()}`;
      const functionName = `e2e-mcp-${Date.now()}`;

      // Verify we have the required tools
      const toolsList = await mcpClient!.listTools();
      const tools = toolsList.tools || [];

      const hasListMcpIntegrations = tools.some(t => t.name === "list_mcp_integrations");
      const hasCreateMcpIntegration = tools.some(t => t.name === "create_mcp_integration");
      const hasCreateMcpServer = tools.some(t => t.name === "create_mcp_server");

      expect(hasListMcpIntegrations).toBe(true);
      expect(hasCreateMcpIntegration).toBe(true);
      expect(hasCreateMcpServer).toBe(true);

      console.log("\n=== Simulating Agent Workflow ===\n");

      // Step 1: List MCP integrations
      console.log("Agent: Listing MCP integrations...");
      const listIntegrationsResult = await mcpClient!.callTool({
        name: "list_mcp_integrations",
        arguments: {
          filter: integrationId
        }
      });

      const listContent = listIntegrationsResult.content?.[0];
      expect(listContent).toBeDefined();
      console.log(`Agent: Found integration type "${integrationId}"`);

      // Step 2: Create integration connection
      console.log("\nAgent: Creating integration connection...");
      const createIntegrationResult = await mcpClient!.callTool({
        name: "create_mcp_integration",
        arguments: {
          name: connectionName,
          integration: integrationId,
          secret: integrationSecret,
          config: integrationConfig
        }
      });

      const createIntContent = createIntegrationResult.content?.[0];
      expect(createIntContent).toBeDefined();
      expect(createIntContent?.text).toContain(connectionName);
      console.log(`Agent: Created integration connection "${connectionName}"`);

      // Step 3: Create MCP server
      console.log("\nAgent: Creating MCP server...");
      const createMcpResult = await mcpClient!.callTool({
        name: "create_mcp_server",
        arguments: {
          name: functionName,
          integration: connectionName
        }
      });

      const createMcpContent = createMcpResult.content?.[0];
      expect(createMcpContent).toBeDefined();
      expect(createMcpContent?.text).toContain(functionName);
      console.log(`Agent: Created MCP server "${functionName}" linked to integration "${connectionName}"`);

      // Verify via resources that the MCP server was created
      console.log("\nAgent: Verifying resources...");
      const resources = await mcpClient!.listResources();
      const hasMcpServer = resources.resources?.some((r: any) =>
        r.uri?.includes(`/mcp-servers/${functionName}`)
      );
      expect(hasMcpServer).toBe(true);

      console.log("\n=== Agent Workflow Completed Successfully ===");
      console.log(`✓ Listed integrations matching "${integrationId}"`);
      console.log(`✓ Created integration connection "${connectionName}"`);
      console.log(`✓ Created MCP server "${functionName}" linked to the integration`);
    },
    120_000
  );
});