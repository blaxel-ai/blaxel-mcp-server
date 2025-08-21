import { listMcpHubDefinitions, McpDefinition } from "@blaxel/core";
import { McpServer } from "@modelcontextprotocol/sdk/server/mcp.js";
import { z } from "zod";
import { toolError } from "../../utils/error.js";

function formatDefinitionSummary(def: McpDefinition): string {
  const name = def.displayName ?? def.name ?? def.integration ?? "unknown";
  const id = def.integration ?? def.name ?? "";
  const description = def.description ?? def.longDescription ?? "";
  const configs = def.form?.config as Record<string, { description: string, required: boolean }> ?? {};
  const secrets = def.form?.secrets as Record<string, { description: string, required: boolean }> ?? {};
  const secretsStr = Object.entries(secrets).map(([key, value]) => `\n\t- ${key}: description=${value.description}, required=${value.required}`).join("").replace("\n", "");
  const configsStr = Object.entries(configs).map(([key, value]) => `\n\t- ${key}: description=${value.description}, required=${value.required}`).join("").replace("\n", "");
  return `- ${name}${id ? ` (${id})` : ""}\n  Description: ${description}${secretsStr.length ? `\n  Secrets:\n${secretsStr}` : ""}${configsStr.length ? `\n  Config keys:\n${configsStr}` : ""}`;
}

export function registerListMcpHubDefinitionsTool(server: McpServer) {
  server.tool(
    "list_mcp_integrations",
    "List available MCP Hub integrations and show required secrets and config keys",
    {
      filter: z
        .string()
        .optional()
        .describe(
          "Optional filter: substring to match against integration identifier, name, or displayName"
        ),
    },
    async ({ filter }) => {
      const res = await listMcpHubDefinitions();
      if (res.error) return toolError("Error listing MCP Hub definitions", res.error);
      let defs = res.data.filter((d: McpDefinition) => d.coming_soon !== true) ?? [];
      if (filter) {
        const q = filter.toLowerCase();
        defs = defs.filter((d: McpDefinition) => {
          const fields = [d.integration, d.name, d.displayName].filter(Boolean) as string[];
          return fields.some((f) => f.toLowerCase().includes(q));
        });
      }
      if (!defs.length) {
        return {
          content: [
            {
              type: "text",
              text: filter
                ? `No MCP definition matched filter: ${filter}`
                : "No MCP definition found",
            },
          ],
        };
      }
      const lines = defs.map((d: McpDefinition) => formatDefinitionSummary(d));
      return {
        content: [
          {
            type: "text",
            text: lines.join("\n\n"),
          },
        ],
      };
    }
  );
}

export function registerGetMcpHubDefinitionTool(server: McpServer) {
  server.tool(
    "get_mcp_integration",
    "Get details for a specific MCP Hub integration, including required secrets and config schema",
    {
      name: z
        .string()
        .min(1, "MCP Hub name is required")
        .describe("MCP Hub name, or display name to lookup"),
    },
    async ({ name }) => {
      const res = await listMcpHubDefinitions();
      if (res.error) return toolError("Error retrieving MCP Hub definition", res.error);
      const defs = res.data ?? [];
      const q = name.toLowerCase();
      const match = defs.find((d: McpDefinition) => {
        const fields = [d.integration, d.name, d.displayName].filter(Boolean) as string[];
        return fields.some((f) => f.toLowerCase() === q || f.toLowerCase().includes(q));
      });
      if (!match) {
        return {
          content: [
            { type: "text", text: `No MCP Hub integration found for: ${name}` },
          ],
        };
      }
      const summary = formatDefinitionSummary(match);
      return {
        content: [
          { type: "text", text: summary }
        ],
      };
    }
  );
}


