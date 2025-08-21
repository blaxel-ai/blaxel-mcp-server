import { existsSync, readFileSync } from "fs";
import { homedir } from "os";
import { join } from "path";

export function ensureBlaxelWorkspace(): string {
  const workspace = process.env.BL_WORKSPACE;
  const apiKey = process.env.BL_API_KEY;

  if (!workspace || !apiKey) {
    throw new Error(
      "BL_WORKSPACE and BL_API_KEY environment variables must be set"
    );
  }

  const configPath = join(homedir(), ".blaxel", "config.yaml");

  // Check if config file exists
  if (!existsSync(configPath)) {
    throw new Error(
      "Blaxel config not found. Please run 'bl login' to authenticate."
    );
  }

  try {
    // Read the config file
    const configContent = readFileSync(configPath, "utf-8");

    // Check if the workspace exists in the workspaces list
    const workspaceRegex = new RegExp(`^\\s*-\\s+name:\\s+${workspace}\\s*$`, 'm');
    const hasWorkspace = workspaceRegex.test(configContent);

    // Check if API key exists in credentials
    const apiKeyRegex = /credentials:\s*\n\s*apiKey:\s*\S+/;
    const hasApiKey = apiKeyRegex.test(configContent);

    if (!hasWorkspace) {
      throw new Error(
        `Workspace '${workspace}' not found in Blaxel config. Please run 'bl login ${workspace}' to authenticate.`
      );
    }

    if (!hasApiKey) {
      throw new Error(
        "API key not found in Blaxel config. Please run 'bl login' to authenticate."
      );
    }

    return workspace;
  } catch (error: any) {
    if (error.message.includes("Workspace") || error.message.includes("API key")) {
      throw error;
    }
    throw new Error(`Failed to read Blaxel config: ${error.message}`);
  }
}