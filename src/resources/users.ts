import { listWorkspaceUsers } from "@blaxel/core";
import { McpServer, ResourceTemplate } from "@modelcontextprotocol/sdk/server/mcp.js";

export function registerUsersResources(server: McpServer) {
  const template = new ResourceTemplate("blaxel://users/{sub}", {
    list: async () => {
      const res = await listWorkspaceUsers();
      const items = res.data ?? [];
      return {
        resources: items.map((u) => ({
          uri: `blaxel://users/${u.sub ?? ""}`,
          name: u.email ?? u.sub ?? "",
          mimeType: "application/json",
          description: `role=${u.role ?? ""}`,
        })),
      };
    },
  });

  server.resource(
    "users",
    template,
    { mimeType: "application/json" },
    async (_uri, variables) => {
      const subOrEmail = String(variables.sub ?? "");
      // We only expose the list via resources; single read returns a pointer for now.
      const user = (await listWorkspaceUsers()).data?.find(
        (u) => u.sub === subOrEmail || u.email === subOrEmail
      );
      return {
        contents: [
          {
            uri: `blaxel://users/${subOrEmail}`,
            mimeType: "application/json",
            text: JSON.stringify(user ?? {}, null, 2),
          },
        ],
      };
    }
  );
}


