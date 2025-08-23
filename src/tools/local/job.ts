import { McpServer } from "@modelcontextprotocol/sdk/server/mcp.js";
import { execSync } from "child_process";
import { existsSync } from "fs";
import { join } from "path";
import { z } from "zod";
import { ensureBlaxelWorkspace } from "../../utils/blaxelAuth.js";
import { toolError } from "../../utils/error.js";
import { buildWorkspaceFlag } from "../../utils/workspaceFlag.js";

export function registerLocalJobTool(server: McpServer) {
  server.tool(
    "local_create_job",
    "Create a new Blaxel job project locally using CLI",
    {
      directory: z
        .string()
        .min(1, "Directory path is required")
        .describe("Complete path to the directory to create for the job (e.g., use `pwd`/my-job for current directory)"),
      template: z
        .string()
        .optional()
        .describe(
          "Template to use for the job (e.g., 'template-jobs-ts', 'template-jobs-py')"
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

        // Build the CLI command with -y flag to skip prompts and workspace if available
        let command = `bl create-job ${directory} -y${buildWorkspaceFlag(workspace)}`;
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
              text: `Successfully created job in directory: ${directory}\n\n${output}`,
            },
          ],
        };
      } catch (error: any) {
        return toolError(
          "Error creating job",
          error.message || error.toString()
        );
      }
    }
  );
}
