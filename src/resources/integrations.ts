import { getIntegrationConnection, listIntegrationConnections } from "@blaxel/core";
import { McpServer, ResourceTemplate } from "@modelcontextprotocol/sdk/server/mcp.js";

export function registerIntegrationsResources(server: McpServer) {
  const template = new ResourceTemplate("blaxel://integrations/{name}", {
    list: async () => {
      const res = await listIntegrationConnections();
      const items = res.data ?? [];
      return {
        resources: items.map((i) => ({
          uri: `blaxel://integrations/${i.metadata?.name ?? ""}`,
          name: i.metadata?.name ?? "",
          mimeType: "application/json",
          description: `Integration: ${i.spec?.integration ?? ""}`,
        })),
      };
    },
  });

  server.resource(
    "integrations",
    template,
    { mimeType: "application/json" },
    async (_uri, variables) => {
      const name = String(variables.name ?? "");
      const res = await getIntegrationConnection({ path: { connectionName: name } } as any);
      return {
        contents: [
          {
            uri: `blaxel://integrations/${name}`,
            mimeType: "application/json",
            text: JSON.stringify(res.data ?? {}, null, 2),
          },
        ],
      };
    }
  );
}


