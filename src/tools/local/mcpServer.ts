import { McpServer } from "@modelcontextprotocol/sdk/server/mcp.js";
import { execSync } from "child_process";
import { existsSync } from "fs";
import { join } from "path";
import { z } from "zod";
import { ensureBlaxelWorkspace } from "../../utils/blaxelAuth.js";
import { toolError } from "../../utils/error.js";

export function registerLocalMcpServerTool(server: McpServer) {
  server.tool(
    "local_create_mcp_server",
    "Create a new Blaxel MCP server project locally using CLI",
    {
      directory: z
        .string()
        .min(1, "Directory name is required")
        .describe("Name of the directory to create for the MCP server"),
      template: z
        .string()
        .optional()
        .describe(
          "Template to use for the MCP server (e.g., 'template-mcp-hello-world-py', 'template-mcp-hello-world-ts')"
        ),
    },
    async ({ directory, template }) => {
      try {
        // Ensure Blaxel workspace is configured
        const workspace = ensureBlaxelWorkspace();

        // Check if directory already exists
        const targetPath = join(process.cwd(), directory);
        if (existsSync(targetPath)) {
          return toolError(`Directory '${directory}' already exists`, null);
        }

        // Build the CLI command with -y flag to skip prompts and workspace
        let command = `bl create-mcp-server ${directory} -y -w ${workspace}`;
        if (template) {
          command += ` --template ${template}`;
        }

        // Execute the CLI command
        const output = execSync(command, {
          encoding: "utf-8",
          cwd: process.cwd(),
        });

        return {
          content: [
            {
              type: "text",
              text: `Successfully created MCP server in directory: ${directory}\n\n${output}`,
            },
          ],
        };
      } catch (error: any) {
        return toolError(
          "Error creating MCP server",
          error.message || error.toString()
        );
      }
    }
  );
}
