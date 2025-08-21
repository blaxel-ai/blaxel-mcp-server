import { getJob, listJobs } from "@blaxel/core";
import { McpServer, ResourceTemplate } from "@modelcontextprotocol/sdk/server/mcp.js";

export function registerJobsResources(server: McpServer) {
  const template = new ResourceTemplate("blaxel://jobs/{id}", {
    list: async () => {
      const res = await listJobs();
      const items = res.data ?? [];
      return {
        resources: items.map((j) => ({
          uri: `blaxel://jobs/${j.metadata?.name ?? ""}`,
          name: j.metadata?.name ?? "",
          mimeType: "application/json",
          description: "Job",
        })),
      };
    },
  });

  server.resource(
    "jobs",
    template,
    { mimeType: "application/json" },
    async (_uri, variables) => {
      const id = String(variables.id ?? "");
      const res = await getJob({ path: { jobId: id } } as any);
      return {
        contents: [
          {
            uri: `blaxel://jobs/${id}`,
            mimeType: "application/json",
            text: JSON.stringify(res.data ?? {}, null, 2),
          },
        ],
      };
    }
  );
}


