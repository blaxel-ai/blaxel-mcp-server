import { deleteIntegrationConnection } from "@blaxel/core";
import { McpServer } from "@modelcontextprotocol/sdk/server/mcp.js";
import { z } from "zod";
import { toolError } from "../../utils/error.js";

export function registerDeleteIntegrationTool(server: McpServer) {
  server.tool(
    "delete_integration",
    "Delete an integration connection by name",
    {
      name: z
        .string()
        .min(1, "Integration connection name is required")
        .describe("Name of the integration connection to delete"),
    },
    async ({ name }) => {
      const res = await deleteIntegrationConnection({ path: { connectionName: name } } as any);
      if (res.error) return toolError("Error deleting integration connection", res.error);
      server.sendResourceListChanged();
      return {
        content: [
          { type: "text", text: `Deleted integration: ${res.data?.metadata?.name ?? name}` },
        ],
      };
    }
  );
}


