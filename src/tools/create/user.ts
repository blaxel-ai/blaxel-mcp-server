import { inviteWorkspaceUser } from "@blaxel/core";
import { McpServer } from "@modelcontextprotocol/sdk/server/mcp.js";
import { z } from "zod";
import { toolError } from "../../utils/error.js";

export function registerCreateUserTool(server: McpServer) {
  server.tool(
    "invite_user",
    "Invite a user to the workspace",
    {
      email: z
        .string()
        .min(1, "Email is required")
        .email("Invalid email address")
        .describe("Email address of the user to invite"),
    },
    async ({ email }) => {
      const res = await inviteWorkspaceUser({ body: { email } });
      if (res.error) return toolError("Error inviting user", res.error);
      server.sendResourceListChanged();
      return {
        content: [
          { type: "text", text: `Invited user: ${res.data?.email ?? email}` },
        ],
      };
    }
  );
}


