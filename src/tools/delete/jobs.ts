import { deleteJob } from "@blaxel/core";
import { McpServer } from "@modelcontextprotocol/sdk/server/mcp.js";
import { z } from "zod";
import { toolError } from "../../utils/error.js";

export function registerDeleteJobTool(server: McpServer) {
  server.tool(
    "delete_job",
    "Delete a job by id",
    {
      id: z
        .string()
        .min(1, "Job id is required")
        .describe("Identifier of the job to delete"),
    },
    async ({ id }) => {
      const res = await deleteJob({ path: { jobId: id } } as any);
      if (res.error) return toolError("Error deleting job", res.error);
      server.sendResourceListChanged();
      return {
        content: [
          { type: "text", text: `Deleted job: ${res.data?.metadata?.name ?? id}` },
        ],
      };
    }
  );
}


