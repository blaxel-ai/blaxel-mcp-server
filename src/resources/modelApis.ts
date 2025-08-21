import { getModel, listModels } from "@blaxel/core";
import { McpServer, ResourceTemplate } from "@modelcontextprotocol/sdk/server/mcp.js";

export function registerModelApisResources(server: McpServer) {
  const template = new ResourceTemplate("blaxel://model-apis/{name}", {
    list: async () => {
      const res = await listModels();
      const items = res.data ?? [];
      return {
        resources: items.map((m) => ({
          uri: `blaxel://model-apis/${m.metadata?.name ?? ""}`,
          name: m.metadata?.name ?? "",
          mimeType: "application/json",
          description: "Model API",
        })),
      };
    },
  });

  server.resource(
    "model-apis",
    template,
    { mimeType: "application/json" },
    async (_uri, variables) => {
      const name = String(variables.name ?? "");
      const res = await getModel({ path: { modelName: name } } as any);
      return {
        contents: [
          {
            uri: `blaxel://model-apis/${name}`,
            mimeType: "application/json",
            text: JSON.stringify(res.data ?? {}, null, 2),
          },
        ],
      };
    }
  );
}


