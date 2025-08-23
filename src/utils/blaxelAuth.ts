import { execSync } from "child_process";
import { fileLogger } from "./log";

// Track if we've already performed login in this session
let hasPerformedLogin = false;

/**
 * Gets the current workspace from `bl workspace` command output
 */
function getCurrentWorkspace(): string | undefined {
  try {
    const output = execSync("bl workspace", {
      encoding: "utf-8",
      stdio: "pipe"
    });

    // Parse the workspace list to find the current one (marked with *)
    // Format is:
    // NAME                           CURRENT
    // workspace1
    // workspace2                     *
    const lines = output.split('\n');

    for (let i = 1; i < lines.length; i++) {
      const line = lines[i];
      if (line.includes('*')) {
        // Extract workspace name (first column)
        const parts = line.trim().split(/\s+/);
        if (parts[0] && parts[0] !== 'NAME') {
          return parts[0];
        }
      }
    }

    return undefined;
  } catch (error: any) {
    return undefined;
  }
}

/**
 * Gets list of available workspaces from `bl workspace` command
 */
function getWorkspaceList(): string[] {
  try {
    const output = execSync("bl workspace", {
      encoding: "utf-8",
      stdio: "pipe"
    });

    // Parse the workspace list
    // Format is typically:
    // NAME                           CURRENT
    // workspace1
    // workspace2                     *
    const lines = output.split('\n');
    const workspaces: string[] = [];

    // Skip header line
    for (let i = 1; i < lines.length; i++) {
      const line = lines[i].trim();
      if (line) {
        // Extract workspace name (first column)
        const parts = line.split(/\s+/);
        if (parts[0] && parts[0] !== 'NAME') {
          workspaces.push(parts[0]);
        }
      }
    }

    return workspaces;
  } catch (error: any) {
    return [];
  }
}

export function ensureBlaxelWorkspace(): string | undefined {
  const workspace = process.env.BL_WORKSPACE;
  const apiKey = process.env.BL_API_KEY;

  // Case 1: Local user without env vars - use current workspace
  if (!workspace) {
    const currentWorkspace = getCurrentWorkspace();
    if (!currentWorkspace) {
      fileLogger.error("No current workspace found. Please run 'bl login' to set up a workspace.");
      return undefined;
    }
    return currentWorkspace;
  }

  // Case 2: Both env vars are set - validate and perform login
  if (workspace && apiKey) {
    // Get available workspaces
    const availableWorkspaces = getWorkspaceList();

    // If we can get the workspace list, validate the workspace exists
    if (availableWorkspaces.length > 0 && !availableWorkspaces.includes(workspace)) {
      throw new Error(
        `Workspace '${workspace}' not found. Available workspaces: ${availableWorkspaces.join(', ')}`
      );
    }

    // Only perform login once per session
    if (!hasPerformedLogin) {
      try {
        // Set the API key in environment for the bl login command
        execSync(`bl login ${workspace}`, {
          encoding: "utf-8",
          stdio: "pipe",
          env: { ...process.env, BL_API_KEY: apiKey }
        });
        hasPerformedLogin = true;
      } catch (error: any) {
        throw new Error(
          `Failed to login to workspace '${workspace}': ${error.message}`
        );
      }
    }

    return workspace;
  }

  // Case 3: Only one env var is set (invalid configuration)
  fileLogger.error("Both BL_WORKSPACE and BL_API_KEY must be set together, or neither should be set for local usage.");
  throw new Error(
    "Both BL_WORKSPACE and BL_API_KEY must be set together, or neither should be set for local usage."
  );
}