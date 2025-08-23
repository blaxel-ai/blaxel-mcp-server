import { McpServer } from "@modelcontextprotocol/sdk/server/mcp.js";
import { execSync } from "child_process";
import { existsSync } from "fs";
import { join } from "path";
import { z } from "zod";
import { ensureBlaxelWorkspace } from "../../utils/blaxelAuth.js";
import { toolError } from "../../utils/error.js";
import { buildWorkspaceFlag } from "../../utils/workspaceFlag.js";

export function registerLocalDeployTool(server: McpServer) {
  server.tool(
    "local_deploy_directory",
    "Deploy a local directory containing agent, MCP server, or job code to Blaxel",
    {
      directory: z
        .string()
        .optional()
        .describe(
          "Path to the directory to deploy (defaults to current directory)"
        ),
      name: z
        .string()
        .optional()
        .describe("Optional name for the deployment"),
      envFile: z
        .array(z.string())
        .optional()
        .describe("Environment file(s) to load (defaults to .env)"),
      secrets: z
        .array(z.string())
        .optional()
        .describe("Secrets to deploy (format: KEY=VALUE)"),
      skipBuild: z
        .boolean()
        .optional()
        .default(false)
        .describe("Skip the build step"),
      dryRun: z
        .boolean()
        .optional()
        .default(false)
        .describe("Perform a dry run without actually deploying"),
      recursive: z
        .boolean()
        .optional()
        .default(true)
        .describe("Deploy recursively (default: true)"),
    },
    async ({
      directory,
      name,
      envFile,
      secrets,
      skipBuild,
      dryRun,
      recursive,
    }) => {
      try {
        // Ensure Blaxel workspace is configured
        const workspace = ensureBlaxelWorkspace();

        // Determine the target directory
        const targetPath = directory
          ? join(process.cwd(), directory)
          : process.cwd();

        // Check if directory exists
        if (!existsSync(targetPath)) {
          return toolError(`Directory '${targetPath}' does not exist`, null);
        }

        // Check if it's a valid Blaxel project directory
        // (should contain blaxel.yaml or similar configuration)
        const blaxelConfigPath = join(targetPath, "blaxel.yaml");
        const blaxelYmlPath = join(targetPath, "blaxel.yml");
        if (!existsSync(blaxelConfigPath) && !existsSync(blaxelYmlPath)) {
          return toolError(
            `Directory '${targetPath}' does not appear to be a valid Blaxel project (missing blaxel.yaml)`,
            null
          );
        }

        // Build the CLI command with workspace if available
        let command = `bl deploy${buildWorkspaceFlag(workspace)}`;

        if (directory) {
          command += ` --directory ${directory}`;
        }

        if (name) {
          command += ` --name ${name}`;
        }

        if (envFile && envFile.length > 0) {
          envFile.forEach((file) => {
            command += ` --env-file ${file}`;
          });
        }

        if (secrets && secrets.length > 0) {
          secrets.forEach((secret) => {
            command += ` --secrets ${secret}`;
          });
        }

        if (skipBuild) {
          command += " --skip-build";
        }

        if (dryRun) {
          command += " --dryrun";
        }

        if (!recursive) {
          command += " --recursive=false";
        }

        // Execute the CLI command
        const output = execSync(command, {
          encoding: "utf-8",
          cwd: process.cwd(),
          maxBuffer: 10 * 1024 * 1024, // 10MB buffer for larger outputs
        });

        const statusMessage = dryRun
          ? "Dry run completed successfully"
          : "Successfully deployed";

        return {
          content: [
            {
              type: "text",
              text: `${statusMessage} from directory: ${targetPath}\n\n${output}`,
            },
          ],
        };
      } catch (error: any) {
        // Extract more meaningful error messages from deploy failures
        const errorMessage = error.stderr || error.message || error.toString();
        return toolError("Error deploying directory", errorMessage);
      }
    }
  );
}
