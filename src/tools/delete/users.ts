import { removeWorkspaceUser } from "@blaxel/core";
import { McpServer } from "@modelcontextprotocol/sdk/server/mcp.js";
import { z } from "zod";
import { toolError } from "../../utils/error.js";

export function registerDeleteUserTool(server: McpServer) {
  server.tool(
    "delete_user",
    "Remove a user from the workspace (or revoke invitation)",
    {
      subOrEmail: z
        .string()
        .min(1, "User sub or email is required")
        .describe("The user's subject (sub) or email to remove"),
    },
    async ({ subOrEmail }) => {
      const res = await removeWorkspaceUser({ path: { subOrEmail } } as any);
      if ((res as any)?.error) return toolError("Error removing user", (res as any).error);
      server.sendResourceListChanged();
      return {
        content: [
          { type: "text", text: `Removed user: ${subOrEmail}` },
        ],
      };
    }
  );
}


