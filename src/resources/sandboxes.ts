import { getSandbox, listSandboxes } from "@blaxel/core";
import { McpServer, ResourceTemplate } from "@modelcontextprotocol/sdk/server/mcp.js";

export function registerSandboxesResources(server: McpServer) {
  const template = new ResourceTemplate("blaxel://sandboxes/{name}", {
    list: async () => {
      const res = await listSandboxes();
      const items = res.data ?? [];
      return {
        resources: items.map((s) => ({
          uri: `blaxel://sandboxes/${s.metadata?.name ?? ""}`,
          name: s.metadata?.name ?? "",
          mimeType: "application/json",
          description: "Sandbox",
        })),
      };
    },
  });

  server.resource(
    "sandboxes",
    template,
    { mimeType: "application/json" },
    async (_uri, variables) => {
      const name = String(variables.name ?? "");
      const res = await getSandbox({ path: { sandboxName: name } } as any);
      return {
        contents: [
          {
            uri: `blaxel://sandboxes/${name}`,
            mimeType: "application/json",
            text: JSON.stringify(res.data ?? {}, null, 2),
          },
        ],
      };
    }
  );
}


