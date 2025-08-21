import { createFunction, Function } from "@blaxel/core";
import { McpServer } from "@modelcontextprotocol/sdk/server/mcp.js";
import { z } from "zod";
import { toolError } from "../../utils/error.js";

export function registerCreateMcpServerTool(server: McpServer) {
  server.tool(
    "create_mcp_server",
    "Create an MCP server (function) from config",
    {
      name: z
        .string()
        .min(1, "Function name is required")
        .describe("Human-readable name for the MCP server (function)"),
      integration: z
        .string()
        .min(1, "Integration is required")
        .describe(
          "Integration connection name to bind to this MCP server (must already exist)"
        ),
    },
    async ({ name, integration }) => {
      const body: Function = {
        metadata: { name },
        spec: { integrationConnections: [integration], runtime: { type: "mcp", memory: 2048 } },
      };
      const res = await createFunction({ body });
      if (res.error) return toolError("Error creating MCP server", res.error);
      server.sendResourceListChanged();
      return {
        content: [
          { type: "text", text: `Created MCP server: ${res.data?.metadata?.name ?? name}` },
        ],
      };
    }
  );
}


