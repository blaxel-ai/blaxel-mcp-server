import { deleteAgent } from "@blaxel/core";
import { McpServer } from "@modelcontextprotocol/sdk/server/mcp.js";
import { z } from "zod";
import { toolError } from "../../utils/error.js";

export function registerDeleteAgentTool(server: McpServer) {
  server.tool(
    "delete_agent",
    "Delete an agent by name",
    {
      name: z
        .string()
        .min(1, "Agent name is required")
        .describe("Name of the agent to delete"),
    },
    async ({ name }) => {
      const res = await deleteAgent({ path: { agentName: name } } as any);
      if (res.error) return toolError("Error deleting agent", res.error);
      server.sendResourceListChanged();
      return {
        content: [
          { type: "text", text: `Deleted agent: ${res.data?.metadata?.name ?? name}` },
        ],
      };
    }
  );
}


