import { getAgent, listAgents } from "@blaxel/core";
import { McpServer, ResourceTemplate } from "@modelcontextprotocol/sdk/server/mcp.js";

export function registerAgentsResources(server: McpServer) {
  const template = new ResourceTemplate("blaxel://agents/{name}", {
    list: async () => {
      const res = await listAgents();
      const items = res.data ?? [];
      return {
        resources: items.map((a) => ({
          uri: `blaxel://agents/${a.metadata?.name ?? ""}`,
          name: a.metadata?.name ?? "",
          mimeType: "application/json",
          description: a.spec?.description ?? "Agent",
        })),
      };
    },
  });

  server.resource(
    "agents",
    template,
    { mimeType: "application/json" },
    async (_uri, variables) => {
      const name = String(variables.name ?? "");
      const res = await getAgent({ path: { agentName: name } } as any);
      return {
        contents: [
          {
            uri: `blaxel://agents/${name}`,
            mimeType: "application/json",
            text: JSON.stringify(res.data ?? {}, null, 2),
          },
        ],
      };
    }
  );
}


