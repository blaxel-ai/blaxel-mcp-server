import { deleteFunction } from "@blaxel/core";
import { McpServer } from "@modelcontextprotocol/sdk/server/mcp.js";
import { z } from "zod";
import { toolError } from "../../utils/error.js";

export function registerDeleteMcpServerTool(server: McpServer) {
  server.tool(
    "delete_mcp_server",
    "Delete an MCP server (function) by name",
    {
      name: z
        .string()
        .min(1, "Function name is required")
        .describe("Name of the MCP server (function) to delete"),
    },
    async ({ name }) => {
      const res = await deleteFunction({ path: { functionName: name } } as any);
      if (res.error) return toolError("Error deleting MCP server", res.error);
      server.sendResourceListChanged();
      return {
        content: [
          { type: "text", text: `Deleted MCP server: ${res.data?.metadata?.name ?? name}` },
        ],
      };
    }
  );
}


