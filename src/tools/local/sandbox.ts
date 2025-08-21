import { McpServer } from "@modelcontextprotocol/sdk/server/mcp.js";
import { execSync } from "child_process";
import { existsSync } from "fs";
import { join } from "path";
import { z } from "zod";
import { ensureBlaxelWorkspace } from "../../utils/blaxelAuth.js";
import { toolError } from "../../utils/error.js";

export function registerLocalSandboxTool(server: McpServer) {
  server.tool(
    "local_create_sandbox",
    "Create a new Blaxel sandbox project locally using CLI",
    {
      directory: z
        .string()
        .min(1, "Directory name is required")
        .describe("Name of the directory to create for the sandbox"),
      template: z
        .string()
        .optional()
        .describe(
          "Template to use for the sandbox (e.g., 'template-sandbox-ts', 'template-sandbox-py')"
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
        let command = `bl create-sandbox ${directory} -y -w ${workspace}`;
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
              text: `Successfully created sandbox in directory: ${directory}\n\n${output}`,
            },
          ],
        };
      } catch (error: any) {
        return toolError(
          "Error creating sandbox",
          error.message || error.toString()
        );
      }
    }
  );
}
