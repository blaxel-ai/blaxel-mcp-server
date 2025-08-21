import { createIntegrationConnection, IntegrationConnection } from "@blaxel/core";
import { McpServer } from "@modelcontextprotocol/sdk/server/mcp.js";
import { z } from "zod";
import { toolError } from "../../utils/error.js";
import { fileLogger } from "../../utils/log.js";

export function registerCreateIntegrationTool(server: McpServer) {
  server.tool(
    "create_mcp_integration",
    "Create an integration connection for an MCP server from credentials/config",
    {
      name: z
        .string()
        .min(1, "Connection name is required")
        .describe("Human-readable name for this integration connection"),
      integration: z
        .string()
        .min(1, "Integration type is required")
        .describe("Integration type identifier (e.g., 'openai', 'slack')"),
      // Note: Using z.object({}).catchall(z.string()) instead of z.record(z.string())
      // to ensure proper serialization through the Mastra MCP framework.
      // This explicitly defines a string-to-string map that preserves all key-value pairs.
      secret: z
        .object({})
        .catchall(z.string())
        .optional()
        .describe("Credentials map; keys depend on integration (e.g., apiKey)"),
      config: z
        .object({})
        .catchall(z.string())
        .optional()
        .describe(
          "Non-secret configuration map; keys depend on integration (e.g., endpoint)"
        ),
    },
    async ({ name, integration, secret, config }) => {
      fileLogger.info("Creating integration connection", { name, integration, secret, config });
      const body: IntegrationConnection = {
        metadata: { name },
        spec: { integration, secret, config },
      };
      const res = await createIntegrationConnection({ body });
      if (res.error) return toolError("Error creating integration connection", res.error);
      server.sendResourceListChanged();
      return {
        content: [
          {
            type: "text",
            text: `Created integration connection: ${res.data?.metadata?.name ?? name}`,
          },
        ],
      };
    }
  );
  server.tool(
    "create_model_api_integration",
    "Create an integration connection for a Model API from credentials/config",
    {
      name: z
        .string()
        .min(1, "Connection name is required")
        .describe("Name for the model API integration connection"),
      integration: z
        .string()
        .min(1, "Model API type is required")
        .describe("Model API provider identifier (e.g., 'openai')"),
      apiKey: z
        .string()
        .min(1, "API key is required")
        .describe("API key/token used to authenticate with the provider"),
      endpoint: z.string().optional().describe("Base URL for the provider API"),
    },
    async ({ name, integration, apiKey, endpoint }) => {
      const body: IntegrationConnection = {
        metadata: { name },
        spec: { integration, secret: { apiKey } },
      };
      if (endpoint) {
        body.spec.config = { endpoint };
      }
      const res = await createIntegrationConnection({ body });
      if (res.error) return toolError("Error creating integration connection", res.error);
      server.sendResourceListChanged();
      return {
        content: [
          {
            type: "text",
            text: `Created integration connection: ${res.data?.metadata?.name ?? name}`,
          },
        ],
      };
    }
  );
}


