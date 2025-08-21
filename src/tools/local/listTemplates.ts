import { listTemplates } from "@blaxel/core";
import { McpServer } from "@modelcontextprotocol/sdk/server/mcp.js";
import { z } from "zod";
import { toolError } from "../../utils/error.js";

export function registerLocalListTemplatesTool(server: McpServer) {
  server.tool(
    "local_list_templates",
    "List available templates for a specific resource type (agent, job, sandbox, or mcp-server)",
    {
      resourceType: z
        .enum(["agent", "job", "sandbox", "mcp-server", "all"])
        .describe("Type of resource to list templates for (or 'all' for all templates)"),
    },
    async ({ resourceType }) => {
      try {
        // Fetch all templates from the API
        const response = await listTemplates();

        if (response.error) {
          return toolError("Error fetching templates", response.error);
        }

        const templates = response.data || [];

        // Filter templates based on resource type
        // Templates typically have topics that indicate their type
        let filteredTemplates = templates;

        if (resourceType !== "all") {
          // Map resource types to topic keywords
          const topicKeywords: Record<string, string[]> = {
            "agent": ["agent", "agents", "adk", "langgraph", "pydantic", "crewai", "mastra", "controlflow"],
            "job": ["job", "jobs", "batch"],
            "sandbox": ["sandbox", "sandboxes", "vm"],
            "mcp-server": ["mcp", "mcp-server", "function", "functions", "tool", "tools"]
          };

          const keywords = topicKeywords[resourceType] || [];

          filteredTemplates = templates.filter(template => {
            const topicMatch = template.topics?.some(topic =>
              keywords.some(keyword => topic.toLowerCase().includes(keyword))
            );

            return topicMatch;
          });
        }

        // Sort templates by star count and download count for better relevance
        filteredTemplates.sort((a, b) => {
          const aScore = (a.starCount || 0) + (a.downloadCount || 0);
          const bScore = (b.starCount || 0) + (b.downloadCount || 0);
          return bScore - aScore;
        });

        // Format the response
        let responseText = resourceType === "all"
          ? "All available templates:\n\n"
          : `Available templates for ${resourceType}:\n\n`;

        if (filteredTemplates.length > 0) {
          filteredTemplates.forEach(template => {
            responseText += `• ${template.name}`;
            if (template.description) {
              responseText += ` - ${template.description}`;
            }
            if (template.starCount || template.downloadCount) {
              responseText += ` (⭐ ${template.starCount || 0}, 📥 ${template.downloadCount || 0})`;
            }
            if (template.topics && template.topics.length > 0) {
              responseText += `\n  Topics: ${template.topics.join(", ")}`;
            }
            responseText += "\n\n";
          });

          if (resourceType !== "all") {
            responseText += `Use any of these templates when creating a new ${resourceType} by specifying the template name.`;
          }
        } else {
          responseText += `No templates found for ${resourceType}. The API may not have templates available or they may be categorized differently.`;
        }

        return {
          content: [
            {
              type: "text",
              text: responseText,
            },
          ],
        };
      } catch (error: any) {
        return toolError(
          `Error listing templates for ${resourceType}`,
          error.message || error.toString()
        );
      }
    }
  );
}