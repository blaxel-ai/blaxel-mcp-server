import { deleteModel } from "@blaxel/core";
import { McpServer } from "@modelcontextprotocol/sdk/server/mcp.js";
import { z } from "zod";
import { toolError } from "../../utils/error.js";

export function registerDeleteModelApiTool(server: McpServer) {
  server.tool(
    "delete_model_api",
    "Delete a model API by name",
    {
      name: z
        .string()
        .min(1, "Model name is required")
        .describe("Name of the model API to delete"),
    },
    async ({ name }) => {
      const res = await deleteModel({ path: { modelName: name } } as any);
      if (res.error) return toolError("Error deleting model API", res.error);
      server.sendResourceListChanged();
      return {
        content: [
          { type: "text", text: `Deleted model API: ${res.data?.metadata?.name ?? name}` },
        ],
      };
    }
  );
}


