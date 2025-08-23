import { openai } from "@ai-sdk/openai";
import { Agent } from "@mastra/core/agent";
import { MCPClient } from "@mastra/mcp";
import path from "node:path";
import { fileURLToPath } from "node:url";
import { afterAll, beforeAll, describe, expect, it } from "vitest";

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);

describe("Blaxel MCP e2e with Mastra Agent + MCP + OpenAI", () => {
  const projectRoot = path.resolve(__dirname, "..");
  const tsxBin = path.resolve(projectRoot, "node_modules", ".bin", process.platform === "win32" ? "tsx.cmd" : "tsx");
  const serverEntry = path.resolve(projectRoot, "src", "index.ts");

  let mcp: MCPClient | null = null;
  let mcpTools: any = null;

  const requires = [
    "BL_WORKSPACE",
    "BL_API_KEY",
    "BL_TEST_INTEGRATION",
    "BL_TEST_INTEGRATION_SECRET_JSON",
    "OPENAI_API_KEY"
  ] as const;
  const missingEnv = requires.filter((k) => !process.env[k] || String(process.env[k]).length === 0);

  beforeAll(async () => {
    if (missingEnv.length) return; // Skip setup

    // Initialize Mastra MCP client as per documentation
    // https://mastra.ai/en/docs/tools-mcp/mcp-overview
    mcp = new MCPClient({
      servers: {
        "blaxel_mcp": {
          command: tsxBin,
          args: [serverEntry],
          env: {
            BL_API_KEY: process.env.BL_API_KEY!,
            BL_WORKSPACE: process.env.BL_WORKSPACE!,
          },
        }
      }
      ,
    });

    // Get tools from the MCP server
    mcpTools = await mcp.getTools();
  }, 30_000);

  afterAll(async () => {
    if (mcp) {
      try {
        await mcp.disconnect();
      } catch {}
      mcp = null;
    }
  });

  if (missingEnv.length > 0) {
    it("requires environment variables", () => {
    });
  }

  it.skipIf(missingEnv.length > 0)(
    "Mastra Agent with MCP tools and OpenAI autonomously creates Blaxel setup",
    async () => {
      const integrationId = String(process.env.BL_TEST_INTEGRATION);
      const integrationSecret = JSON.parse(String(process.env.BL_TEST_INTEGRATION_SECRET_JSON));
      const integrationConfig = process.env.BL_TEST_INTEGRATION_CONFIG_JSON
        ? JSON.parse(String(process.env.BL_TEST_INTEGRATION_CONFIG_JSON))
        : undefined;

      // Generate unique names for this test run
      const timestamp = Date.now();
      const connectionName = `agent-int-${timestamp}`;
      const functionName = `agent-mcp-${timestamp}`;

      const existingIntegration = (await mcp?.resources.list())?.
          ["blaxel_mcp"]?.
          filter((r: any) => r.name.startsWith("integrations")).
          map((r: any) => r.name).join(", ");

      // System prompt for the AI agent
      const systemPrompt = `You are a Mastra agent that helps create Blaxel MCP configurations.
You are connected to a Blaxel MCP server via stdio and have access to MCP tools.

Your task is to:
1. First check GET MCP integration to see which secret and config are required
2. Create an integration connection using the EXACT integration type specified.
3. Create an MCP server linked to that integration connection.

Use the exact names and parameters provided. Think step by step and execute each tool in the correct order.
Very important:
- Do not create an integration connection twice
- Do not try to delete an integration connection
- Existing integrations: ${existingIntegration}`;

    // User prompt with the task details
    const userPrompt = `Please create a complete Blaxel MCP setup with the following specifications:
    IMPORTANT: Use these EXACT values:
    - Integration type: "${integrationId}"
    - Integration connection name: "${connectionName}"
    - MCP server name: "${functionName}"`;

  // Store the original execute function
      const originalExecute = mcpTools["blaxel_mcp_create_mcp_integration"].execute;
      // Override to inject secret and config
      // You don't want to give your secret to an LLM, that's why we're overriding the execute function.
      mcpTools["blaxel_mcp_create_mcp_integration"].execute = async (args: any) => {
        try {
          const existingIntegration = await mcp?.resources.read("blaxel_mcp", `integrations/${connectionName}`);
          if (existingIntegration) {
            return;
          }
        } catch {
        }
        // Check if this is being called with context (from Mastra Agent)
        const context = args?.context || args;

        // Force the correct integration type and add secret/config
        const enhancedContext = {
          ...context,
          integration: integrationId, // Always use the correct integration
          secret: integrationSecret,
          config: integrationConfig || {}
        };

        // Call the original execute with enhanced arguments
        const enhancedArgs = args?.context ? { ...args, context: enhancedContext } : { context: enhancedContext };
        const response = await originalExecute(enhancedArgs);
        return response;
      }

      const agent = new Agent(
        {
          name: "Blaxel MCP Agent",
          model: openai("gpt-4o-mini"),
          instructions: systemPrompt,
          tools: mcpTools,
        }
      )

      // Let the Mastra agent with OpenAI handle everything
      // Allow multiple tool rounds by setting maxSteps
      const response = await agent.generate(userPrompt, {
        maxSteps: 5  // Allow up to 5 rounds of tool calls
      });

      // The agent should report success in its response
      expect(response.text).toBeDefined();
      expect(response.text.length).toBeGreaterThan(0);

      // Verify in the response text that everything was created
      expect(response.text).toContain(connectionName);
      expect(response.text).toContain(functionName);
      expect(response.text.toLowerCase()).toContain("successfully");

      // Verify via MCP client that resources were actually created
      const resources = await mcp?.resources.list();

      // Resources are returned under the server name key 'blaxel_mcp'
      const blaxelResources = resources?.["blaxel_mcp"] || [];
      const hasMcpServer = blaxelResources.some((r: any) =>
        r.uri?.includes(`mcp-servers/${functionName}`) || r.name === functionName
      );


      expect(hasMcpServer).toBe(true);

    },
    120_000
  );
});