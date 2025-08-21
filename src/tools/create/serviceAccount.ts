import { createWorkspaceServiceAccount } from "@blaxel/core";
import { McpServer } from "@modelcontextprotocol/sdk/server/mcp.js";
import { z } from "zod";
import { toolError } from "../../utils/error.js";

export function registerCreateServiceAccountTool(server: McpServer) {
  server.tool(
    "create_service_account",
    "Create a new service account",
    {
      name: z
        .string()
        .min(1, "Name is required")
        .describe("Display name for the service account"),
    },
    async ({ name }) => {
      const res = await createWorkspaceServiceAccount({ body: { name } });
      if (res.error) return toolError("Error creating service account", res.error);
      server.sendResourceListChanged();
      return {
        content: [
          {
            type: "text",
            text: `Created service account: ${res.data?.name ?? name}\nclient_id=${res.data?.client_id ?? ""}\nclient_secret=${res.data?.client_secret ?? ""}`,
          },
        ],
      };
    }
  );
}


