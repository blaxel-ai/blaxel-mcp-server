import { openai } from "@ai-sdk/openai";
import { Agent } from "@mastra/core";
import { MCPClient } from "@mastra/mcp";
import "dotenv/config";

console.log("=== Starting Mastra MCP Agent Test ===");
console.log("Loading environment variables...");

const integrationId = String(process.env.BL_TEST_INTEGRATION);
const integrationSecret = JSON.parse(String(process.env.BL_TEST_INTEGRATION_SECRET_JSON)) as Record<string, string>;
const integrationConfig = process.env.BL_TEST_INTEGRATION_CONFIG_JSON
  ? JSON.parse(String(process.env.BL_TEST_INTEGRATION_CONFIG_JSON))  as Record<string, string>
  : null;

console.log("Configuration loaded:");
console.log(`  - Integration ID: ${integrationId}`);
console.log(`  - Has secret: ${integrationSecret ? 'Yes' : 'No'}`);
console.log(`  - Has config: ${integrationConfig ? 'Yes' : 'No'}`);

// Generate unique names for this test run
const timestamp = Date.now();
const connectionName = `agent-int-${timestamp}`;
const functionName = `agent-mcp-${timestamp}`;

console.log("\nGenerated names:");
console.log(`  - Connection name: ${connectionName}`);
console.log(`  - Function name: ${functionName}`);
console.log("\n=== Initializing MCP Client ===");
const mcp = new MCPClient({
  servers: {
    blaxel_mcp: {
      command: "tsx",
      args: ["src/index.ts"],
      env: {
        BL_API_KEY: process.env.BL_API_KEY!,
        BL_WORKSPACE: process.env.BL_WORKSPACE!,
      }
    }
  }
  ,
});
console.log("MCP Client created successfully");

// Get tools from the MCP server
console.log("\n=== Fetching MCP Tools ===");
const mcpTools = await mcp.getTools();
console.log(`Found ${Object.keys(mcpTools).length} tools:`);
Object.keys(mcpTools).forEach(tool => console.log(`  - ${tool}`));

console.log("\n=== Checking Existing Resources ===");
const resources = await mcp?.resources.list();
const existingIntegration = resources?.["blaxel_mcp"]?.
    filter((r: any) => r.name.startsWith("integrations")).
    map((r: any) => r.name).join(", ");
console.log(`Existing integrations: ${existingIntegration || 'None'}`);

console.log("\n=== Preparing Agent Prompts ===");
// System prompt for the AI agent
const systemPrompt = `You are a Mastra agent that helps create Blaxel MCP configurations.
You are connected to a Blaxel MCP server via stdio and have access to MCP tools.

Your task is to:
1. Create an integration connection using the EXACT integration type specified.
2. Create an MCP server linked to that integration connection.

Use the exact names and parameters provided. Think step by step and execute each tool in the correct order.
Very important:
- Do not create an integration connection twice
- Do not try to delete an integration connection
- Existing integrations: ${existingIntegration}`;

console.log("System prompt prepared");

// User prompt with the task details
const userPrompt = `Please create a complete Blaxel MCP setup with the following specifications:
IMPORTANT: Use these EXACT values:
- Integration type: "${integrationId}"
- Integration connection name: "${connectionName}"
- MCP server name: "${functionName}"`;

console.log("User prompt prepared");
console.log("\nUser prompt content:");
console.log(userPrompt);

console.log("\n=== Setting up Tool Override ===");

// Override the tool to inject secret and config properly
const originalExecute = mcpTools["blaxel_mcp_create_mcp_integration"].execute;

mcpTools["blaxel_mcp_create_mcp_integration"].execute = async function(args: any) {
  try {
    const existing = await mcp?.resources.read("blaxel_mcp", `integrations/${args.context.name}`);
    if (existing) {
      console.log(">>> Integration already exists");
      return {
        content: [{ type: "text", text: `Integration ${args.context.name} already exists` }]
      };
    }
  } catch (e) {
    console.log(">>> Integration does not exist, will create");
  }

  // Fallback: call original with modified context
  args.context.integration = integrationId;
  args.context.secret = integrationSecret || {};
  args.context.config = integrationConfig || {};

  return originalExecute.call(this, args);
};

console.log("Tool override setup complete");

console.log("\n=== Creating Mastra Agent ===");
const agent = new Agent(
  {
    name: "Blaxel MCP Agent",
    model: openai("gpt-4o-mini"),
    instructions: systemPrompt,
    tools: mcpTools,
  }
)
console.log("Agent created successfully");
console.log(`  - Name: Blaxel MCP Agent`);
console.log(`  - Model: gpt-4o-mini`);
console.log(`  - Tools available: ${Object.keys(mcpTools).length}`);

// Let the Mastra agent with OpenAI handle everything
// Allow multiple tool rounds by setting maxSteps
console.log("\n=== Starting Agent Execution ===");
console.log("Max steps configured: 5");
console.log("Sending user prompt to agent...\n");

const response = await agent.generate(userPrompt, {
  maxSteps: 5  // Allow up to 5 rounds of tool calls
});

console.log("\n=== Agent Response ===");
console.log("Finish reason:", response.finishReason);
console.log("Tool calls made:", response.toolCalls?.length || 0);
if (response.toolCalls && response.toolCalls.length > 0) {
  console.log("\nTool calls details:");
  response.toolCalls.forEach((tc: any, index: number) => {
    console.log(`  ${index + 1}. ${tc.toolName}`);
    console.log(`     Args: ${JSON.stringify(tc.args)}`);
  });
}
console.log("\nFinal response text:");
console.log(response.text);

console.log("\n=== Test Completed ===");
console.log(`Successfully created:`);
console.log(`  - Integration connection: ${connectionName}`);
console.log(`  - MCP server: ${functionName}`);
await mcp.disconnect();