/**
 * Builds a workspace flag for CLI commands
 * @param workspace - The workspace name (can be undefined)
 * @returns The workspace flag string if workspace is defined, empty string otherwise
 */
export function buildWorkspaceFlag(workspace: string | undefined): string {
  return workspace ? ` -w ${workspace}` : "";
}
