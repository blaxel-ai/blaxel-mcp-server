import { getFunction, listFunctions } from "@blaxel/core";
import { McpServer, ResourceTemplate } from "@modelcontextprotocol/sdk/server/mcp.js";

export function registerMcpServersResources(server: McpServer) {
  const template = new ResourceTemplate("blaxel://mcp-servers/{name}", {
    list: async () => {
      const res = await listFunctions();
      const items = res.data ?? [];
      return {
        resources: items.map((f) => ({
          uri: `blaxel://mcp-servers/${f.metadata?.name ?? ""}`,
          name: f.metadata?.name ?? "",
          mimeType: "application/json",
          description: "MCP Server",
        })),
      };
    },
  });

  server.resource(
    "mcp-servers",
    template,
    { mimeType: "application/json" },
    async (_uri, variables) => {
      const name = String(variables.name ?? "");
      const res = await getFunction({ path: { functionName: name } } as any);
      return {
        contents: [
          {
            uri: `blaxel://mcp-servers/${name}`,
            mimeType: "application/json",
            text: JSON.stringify(res.data ?? {}, null, 2),
          },
        ],
      };
    }
  );
}


