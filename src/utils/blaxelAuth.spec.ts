import { execSync } from "child_process";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

// Mock child_process module
vi.mock("child_process", () => ({
  execSync: vi.fn(),
}));

// Mock the logger
vi.mock("./log", () => ({
  fileLogger: {
    error: vi.fn(),
    info: vi.fn(),
    debug: vi.fn(),
  },
}));

describe("blaxelAuth", () => {
  const originalEnv = process.env;
  const mockExecSync = execSync as unknown as ReturnType<typeof vi.fn>;
  let ensureBlaxelWorkspace: () => string | undefined;

  beforeEach(async () => {
    // Reset modules to clear module-level state (hasPerformedLogin)
    vi.resetModules();

    // Re-import the function after module reset
    const module = await import("./blaxelAuth.js");
    ensureBlaxelWorkspace = module.ensureBlaxelWorkspace;

    // Reset environment variables
    process.env = { ...originalEnv };
    delete process.env.BL_WORKSPACE;
    delete process.env.BL_API_KEY;

    // Clear all mocks
    vi.clearAllMocks();
  });

  afterEach(() => {
    // Restore original environment
    process.env = originalEnv;
  });

  describe("ensureBlaxelWorkspace", () => {
    describe("without environment variables", () => {
      it("should return current workspace when available", () => {
        // Mock workspace list with current workspace marked
        mockExecSync.mockReturnValue(`NAME                           CURRENT
main
random-name                  *
`);

        const result = ensureBlaxelWorkspace();

        expect(result).toBe("random-name");
        expect(mockExecSync).toHaveBeenCalledWith("bl workspace", {
          encoding: "utf-8",
          stdio: "pipe",
        });
      });

      it("should return undefined when no current workspace is set", () => {
        // Mock workspace list without any current workspace
        mockExecSync.mockReturnValue(`NAME                           CURRENT
main
random-name
`);

        const result = ensureBlaxelWorkspace();

        expect(result).toBeUndefined();
      });

      it("should return undefined when bl command fails", () => {
        // Mock command failure
        mockExecSync.mockImplementation(() => {
          throw new Error("Command not found");
        });

        const result = ensureBlaxelWorkspace();

        expect(result).toBeUndefined();
      });

      it("should handle empty workspace list", () => {
        // Mock empty workspace list
        mockExecSync.mockReturnValue(`NAME                           CURRENT
`);

        const result = ensureBlaxelWorkspace();

        expect(result).toBeUndefined();
      });
    });

    describe("with BL_WORKSPACE environment variable", () => {
      beforeEach(() => {
        // Mock workspace list for validation
        mockExecSync.mockImplementation((command: string) => {
          if (command === "bl workspace") {
            return `NAME                           CURRENT
main
random-name                  *
production
`;
          }
          // Return empty string for login command
          return "";
        });
      });

      it("should validate and return workspace when both env vars are set", () => {
        process.env.BL_WORKSPACE = "random-name";
        process.env.BL_API_KEY = "test-api-key";

        const result = ensureBlaxelWorkspace();

        expect(result).toBe("random-name");
        // Should call workspace list for validation
        expect(mockExecSync).toHaveBeenCalledWith("bl workspace", {
          encoding: "utf-8",
          stdio: "pipe",
        });
        // Should call login
        expect(mockExecSync).toHaveBeenCalledWith("bl login random-name", {
          encoding: "utf-8",
          stdio: "pipe",
          env: expect.objectContaining({
            BL_API_KEY: "test-api-key",
          }),
        });
      });

      it("should throw error for invalid workspace", () => {
        process.env.BL_WORKSPACE = "invalid-workspace";
        process.env.BL_API_KEY = "test-api-key";

        expect(() => ensureBlaxelWorkspace()).toThrow(
          "Workspace 'invalid-workspace' not found. Available workspaces: main, random-name, production"
        );
      });

      it("should skip validation if workspace list cannot be retrieved", () => {
        process.env.BL_WORKSPACE = "some-workspace";
        process.env.BL_API_KEY = "test-api-key";

        let loginCalled = false;
        // Mock failure to get workspace list but success on login
        mockExecSync.mockImplementation((command: string) => {
          if (command === "bl workspace") {
            throw new Error("Failed to get workspaces");
          }
          if (command.startsWith("bl login")) {
            loginCalled = true;
            return "";
          }
          return "";
        });

        const result = ensureBlaxelWorkspace();

        expect(result).toBe("some-workspace");
        // Should still attempt login even when workspace list fails
        expect(loginCalled).toBe(true);
      });

      it("should throw error when login fails", () => {
        process.env.BL_WORKSPACE = "random-name";
        process.env.BL_API_KEY = "invalid-key";

        mockExecSync.mockImplementation((command: string) => {
          if (command === "bl workspace") {
            return `NAME                           CURRENT
main
random-name                  *
`;
          }
          if (command.startsWith("bl login")) {
            throw new Error("Authentication failed");
          }
          return "";
        });

        expect(() => ensureBlaxelWorkspace()).toThrow(
          "Failed to login to workspace 'random-name': Authentication failed"
        );
      });

      it("should only perform login once per session", () => {
        process.env.BL_WORKSPACE = "random-name";
        process.env.BL_API_KEY = "test-api-key";

        // First call
        const result1 = ensureBlaxelWorkspace();
        expect(result1).toBe("random-name");

        // Clear mock calls
        mockExecSync.mockClear();

        // Second call
        const result2 = ensureBlaxelWorkspace();
        expect(result2).toBe("random-name");

        // Should not call login again
        expect(mockExecSync).not.toHaveBeenCalledWith(
          expect.stringContaining("bl login"),
          expect.any(Object)
        );
      });
    });

    describe("with incomplete environment variables", () => {
      it("should throw error when only BL_WORKSPACE is set", () => {
        process.env.BL_WORKSPACE = "random-name";
        delete process.env.BL_API_KEY;

        expect(() => ensureBlaxelWorkspace()).toThrow(
          "Both BL_WORKSPACE and BL_API_KEY must be set together, or neither should be set for local usage."
        );
      });

      it("should use current workspace when only BL_API_KEY is set", () => {
        delete process.env.BL_WORKSPACE;
        process.env.BL_API_KEY = "test-api-key";

        // Mock workspace list with current workspace
        mockExecSync.mockReturnValue(`NAME                           CURRENT
main
random-name                  *
`);

        const result = ensureBlaxelWorkspace();

        expect(result).toBe("random-name");
      });
    });

    describe("workspace parsing", () => {
      it("should correctly parse workspace with spaces in output", () => {
        mockExecSync.mockReturnValue(`NAME                           CURRENT
workspace-one
workspace-two                  *
workspace-three
`);

        const result = ensureBlaxelWorkspace();

        expect(result).toBe("workspace-two");
      });

      it("should handle workspace names with hyphens and numbers", () => {
        mockExecSync.mockReturnValue(`NAME                           CURRENT
test-workspace-123
prod-env-2024                  *
dev-branch-feature
`);

        const result = ensureBlaxelWorkspace();

        expect(result).toBe("prod-env-2024");
      });

      it("should return first workspace when multiple have asterisk (edge case)", () => {
        mockExecSync.mockReturnValue(`NAME                           CURRENT
workspace-1                    *
workspace-2                    *
`);

        const result = ensureBlaxelWorkspace();

        expect(result).toBe("workspace-1");
      });
    });
  });
});
