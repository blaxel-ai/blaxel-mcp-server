import { listIntegrationConnectionModels } from "@blaxel/core";
import { McpServer } from "@modelcontextprotocol/sdk/server/mcp.js";
import { z } from "zod";
import { toolError } from "../../utils/error.js";

function formatModelsSummary(models: unknown[]): string {
  if (!Array.isArray(models)) return "<unexpected response>";
  if (models.length === 0) return "<no models>";
  const lines: string[] = [];
  for (let i = 0; i < models.length; i += 1) {
    const m = models[i] as Record<string, unknown>;
    const id = (m?.id as string) ?? (m?.modelId as string) ?? (m?.name as string) ?? `#${i + 1}`;
    const label = (m?.displayName as string) ?? (m?.name as string) ?? "";
    const provider = (m?.provider as string) ?? (m?.integration as string) ?? "";
    const caps = (m?.capabilities as string[]) ?? [];
    const capsStr = Array.isArray(caps) && caps.length ? ` caps=[${caps.join(", ")}]` : "";
    lines.push(`- ${id}${label && label !== id ? ` (${label})` : ""}${provider ? ` [${provider}]` : ""}${capsStr}`);
  }
  return lines.join("\n");
}

export function registerListIntegrationModelsTool(server: McpServer) {
  server.tool(
    "list_integration_models",
    "List available models for a specific integration connection",
    {
      connectionName: z
        .string()
        .min(1, "Integration connection name is required")
        .describe("Name of the integration connection to query"),
      filter: z
        .string()
        .optional()
        .describe("Optional case-insensitive substring to filter model id/name"),
    },
    async ({ connectionName, filter }) => {
      const res = await listIntegrationConnectionModels({
        // SDK uses path params for {connectionName}
        path: { connectionName },
      } as any);
      if (res.error) return toolError("Error listing integration models", res.error);

      const allModels = (res?.data as unknown[]) ?? [];
      let models = allModels;
      if (filter) {
        const q = filter.toLowerCase();
        models = allModels.filter((m: any) => {
          const fields = [m?.id, m?.modelId, m?.name, m?.displayName].filter(Boolean) as string[];
          return fields.some((f) => f.toLowerCase().includes(q));
        });
      }

      if (!Array.isArray(models) || models.length === 0) {
        return {
          content: [
            {
              type: "text",
              text: filter
                ? `No models found for connection \"${connectionName}\" matching: ${filter}`
                : `No models found for connection \"${connectionName}\"`,
            },
          ],
        };
      }

      const summary = formatModelsSummary(models);
      return {
        content: [
          { type: "text", text: `Models for connection \"${connectionName}\":\n${summary}` },
        ],
      };
    }
  );
}


