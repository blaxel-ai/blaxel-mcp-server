import { McpServer } from "@modelcontextprotocol/sdk/server/mcp.js";
import { execSync } from "child_process";
import { existsSync } from "fs";
import { z } from "zod";
import { ensureBlaxelWorkspace } from "../../utils/blaxelAuth.js";
import { toolError } from "../../utils/error.js";
import { buildWorkspaceFlag } from "../../utils/workspaceFlag.js";

export function registerLocalRunTool(server: McpServer) {
  server.tool(
    "local_run_deployed_resource",
    "Run a deployed resource (agent, model, function/mcp or job) on Blaxel",
    {
      resourceType: z
        .enum(["agent", "model", "job", "function"])
        .describe("Type of resource to run (function is for MCP servers)"),
      resourceName: z
        .string()
        .min(1, "Resource name is required")
        .describe("Name of the deployed resource to run"),
      data: z
        .string()
        .optional()
        .describe("JSON body data for the inference request"),
      file: z
        .string()
        .optional()
        .describe("Input from a file (alternative to data)"),
      local: z
        .boolean()
        .optional()
        .default(false)
        .describe("Run locally instead of on Blaxel cloud"),
      headers: z
        .array(z.string())
        .optional()
        .describe("Request headers in 'Key: Value' format"),
      params: z
        .array(z.string())
        .optional()
        .describe("Query params sent to the inference request"),
      path: z
        .string()
        .optional()
        .describe("Path for the inference request"),
      method: z
        .string()
        .optional()
        .default("POST")
        .describe("HTTP method for the inference request"),
      debug: z
        .boolean()
        .optional()
        .default(false)
        .describe("Enable debug mode"),
    },
    async ({
      resourceType,
      resourceName,
      data,
      file,
      local,
      headers,
      params,
      path,
      method,
      debug,
    }) => {
      try {
        // Ensure Blaxel workspace is configured
        const workspace = ensureBlaxelWorkspace();

        // Validate that either data or file is provided, but not both
        if (data && file) {
          return toolError(
            "Cannot specify both 'data' and 'file' parameters",
            null
          );
        }

        // Check if file exists if specified
        if (file && !existsSync(file)) {
          return toolError(`File '${file}' does not exist`, null);
        }


        // Build the CLI command with workspace if available
        let command = `bl run ${resourceType} ${resourceName}${buildWorkspaceFlag(workspace)}`;

        if (data) {
          // Escape single quotes in JSON data
          const escapedData = data.replace(/'/g, "'\\''");
          command += ` --data '${escapedData}'`;
        }

        if (file) {
          command += ` --file ${file}`;
        }

        if (local) {
          command += " --local";
        }


        if (headers && headers.length > 0) {
          headers.forEach((header) => {
            command += ` --header "${header}"`;
          });
        }

        if (params && params.length > 0) {
          params.forEach((param) => {
            command += ` --params ${param}`;
          });
        }

        if (path) {
          command += ` --path ${path}`;
        }

        if (method && method !== "POST") {
          command += ` --method ${method}`;
        }

        if (debug) {
          command += " --debug";
        }

        // Execute the CLI command
        const output = execSync(command, {
          encoding: "utf-8",
          cwd: process.cwd(),
          maxBuffer: 10 * 1024 * 1024, // 10MB buffer for larger outputs
        });

        const locationText = local ? "locally" : "on Blaxel";

        return {
          content: [
            {
              type: "text",
              text: `Successfully ran ${resourceType} '${resourceName}' ${locationText}\n\n${output}`,
            },
          ],
        };
      } catch (error: any) {
        // Extract more meaningful error messages from run failures
        const errorMessage = error.stderr || error.message || error.toString();
        return toolError("Error running resource", errorMessage);
      }
    }
  );
}
