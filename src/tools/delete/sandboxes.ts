import { deleteSandbox } from "@blaxel/core";
import { McpServer } from "@modelcontextprotocol/sdk/server/mcp.js";
import { z } from "zod";
import { toolError } from "../../utils/error.js";

export function registerDeleteSandboxTool(server: McpServer) {
  server.tool(
    "delete_sandbox",
    "Delete a sandbox by name",
    {
      name: z
        .string()
        .min(1, "Sandbox name is required")
        .describe("Name of the sandbox to delete"),
    },
    async ({ name }) => {
      const res = await deleteSandbox({ path: { sandboxName: name } } as any);
      if (res.error) return toolError("Error deleting sandbox", res.error);
      server.sendResourceListChanged();
      return {
        content: [
          { type: "text", text: `Deleted sandbox: ${res.data?.metadata?.name ?? name}` },
        ],
      };
    }
  );
}


