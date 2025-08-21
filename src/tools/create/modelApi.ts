import { createModel, Model } from "@blaxel/core";
import { McpServer } from "@modelcontextprotocol/sdk/server/mcp.js";
import { z } from "zod";
import { toolError } from "../../utils/error.js";

export function registerCreateModelApiTool(server: McpServer) {
  server.tool(
    "create_model_api",
    "Create a model API, optionally bound to an integration connection",
    {
      name: z
        .string()
        .min(1, "Model name is required")
        .describe("Human-readable name for the model API"),
      integrationConnectionName: z
        .string()
        .min(1, "Integration connection name is required")
        .describe(
          "Optional integration connection to back this model (leave empty to create without integration)"
        ),
    },
    async ({ name, integrationConnectionName }) => {
      const body: Model = {
        metadata: { name },
        spec: {
          integrationConnections: integrationConnectionName
            ? [integrationConnectionName]
            : undefined,
        },
      };
      const res = await createModel({ body });
      if (res.error) return toolError("Error creating model API", res.error);
      server.sendResourceListChanged();
      return {
        content: [
          { type: "text", text: `Created model API: ${res.data?.metadata?.name ?? name}` },
        ],
      };
    }
  );
}


