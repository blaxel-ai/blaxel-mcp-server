import { deleteWorkspaceServiceAccount } from "@blaxel/core";
import { McpServer } from "@modelcontextprotocol/sdk/server/mcp.js";
import { z } from "zod";
import { toolError } from "../../utils/error.js";

export function registerDeleteServiceAccountTool(server: McpServer) {
  server.tool(
    "delete_service_account",
    "Delete a service account by client_id",
    {
      clientId: z
        .string()
        .min(1, "Service account client_id is required")
        .describe("Client ID of the service account to delete"),
    },
    async ({ clientId }) => {
      const res = await deleteWorkspaceServiceAccount({ path: { clientId } } as any);
      if (res.error) return toolError("Error deleting service account", res.error);
      server.sendResourceListChanged();
      return {
        content: [
          { type: "text", text: `Deleted service account: ${res.data?.name ?? clientId}` },
        ],
      };
    }
  );
}


