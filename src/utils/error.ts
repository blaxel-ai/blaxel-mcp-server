export type ToolErrorResponse = {
  content: { type: "text"; text: string }[];
  isError: true;
};

function stringifyErrorDetails(error: unknown): string {
  if (error == null) return "<no details>";
  // Prefer common shapes first
  if (typeof error === "string") return error;
  if (error instanceof Error) return error.stack || error.message || String(error);
  try {
    return JSON.stringify(error, null, 2);
  } catch {
    return String(error);
  }
}

/**
 * Create a standardized MCP tool error response from a Blaxel SDK error.
 * Usage: if (res.error) return toolError("Custom message", res.error)
 */
export function toolError(message: string, error: unknown): ToolErrorResponse {
  const details = stringifyErrorDetails(error);
  const text = `${message}\n${details}`;
  return {
    content: [{ type: "text", text }],
    isError: true,
  };
}


