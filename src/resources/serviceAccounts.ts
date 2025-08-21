import { getWorkspaceServiceAccounts } from "@blaxel/core";
import { McpServer, ResourceTemplate } from "@modelcontextprotocol/sdk/server/mcp.js";

export function registerServiceAccountsResources(server: McpServer) {
  const template = new ResourceTemplate("blaxel://service-accounts/{clientId}", {
    list: async () => {
      const res = await getWorkspaceServiceAccounts();
      const items = res.data ?? [];
      return {
        resources: items.map((sa) => ({
          uri: `blaxel://service-accounts/${sa.client_id ?? ""}`,
          name: sa.name ?? sa.client_id ?? "",
          mimeType: "application/json",
          description: "Service Account",
        })),
      };
    },
  });

  server.resource(
    "service-accounts",
    template,
    { mimeType: "application/json" },
    async (_uri, variables) => {
      const clientId = String(variables.clientId ?? "");
      const items = (await getWorkspaceServiceAccounts()).data ?? [];
      const sa = items.find((x) => x.client_id === clientId || x.name === clientId);
      return {
        contents: [
          {
            uri: `blaxel://service-accounts/${clientId}`,
            mimeType: "application/json",
            text: JSON.stringify(sa ?? {}, null, 2),
          },
        ],
      };
    }
  );
}


