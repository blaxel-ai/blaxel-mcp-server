import { McpServer } from "@modelcontextprotocol/sdk/server/mcp.js";
import { z } from "zod";

export function registerLocalQuickStartTool(server: McpServer) {
  server.tool(
    "local_quick_start_guide",
    "Get a quick start guide for creating Blaxel resources without credentials",
    {
      resourceType: z
        .enum(["agent", "job", "mcp-server", "sandbox", "all"])
        .optional()
        .default("all")
        .describe("Type of resource to get quick start guide for"),
    },
    async ({ resourceType }) => {
      const guides: Record<string, string> = {
        agent: `🚀 Quick Start: Creating an Agent

1. Create an agent project:
   - Use: local_create_agent
   - Example: { directory: "my-agent", template: "template-pydantic-py" }

2. Deploy your agent:
   - Use: local_deploy_directory
   - Example: { directory: "my-agent" }

3. Run your agent:
   - Use: local_run_deployed_resource
   - Example: { resourceType: "agent", resourceName: "my-agent" }

Popular Agent Templates:
- template-pydantic-py: Python agent with Pydantic models
- template-langgraph-py: LangGraph-based agent
- template-google-adk-py: Google ADK agent
- template-mastra-ts: Mastra TypeScript agent`,

        "mcp-server": `🔧 Quick Start: Creating an MCP Server

1. Create an MCP server project:
   - Use: local_create_mcp_server
   - Example: { directory: "my-mcp", template: "template-mcp-hello-world-ts" }

2. Deploy your MCP server:
   - Use: local_deploy_directory
   - Example: { directory: "my-mcp" }

3. Run your MCP server (optional):
   - Use: local_run_deployed_resource
   - Example: { resourceType: "function", resourceName: "my-mcp" }

Note: Custom MCP servers don't require integration connections.
If you need to connect to external services, create an integration:
   - Use: create_mcp_integration
   - Then link with: create_mcp_server

Popular MCP Templates:
- template-mcp-hello-world-ts: TypeScript starter
- template-mcp-hello-world-py: Python starter
- template-mcp-tools-ts: TypeScript with tools
- template-mcp-resources-py: Python with resources`,

        job: `⚡ Quick Start: Creating a Job

1. Create a job project:
   - Use: local_create_job
   - Example: { directory: "my-job", template: "template-jobs-ts" }

2. Deploy your job:
   - Use: local_deploy_directory
   - Example: { directory: "my-job" }

3. Run your job:
   - Use: local_run_deployed_resource
   - Example: { resourceType: "job", resourceName: "my-job" }

Popular Job Templates:
- template-jobs-ts: TypeScript job template
- template-jobs-py: Python job template

Jobs are perfect for:
- Scheduled tasks
- Batch processing
- Data pipelines
- Background operations`,

        sandbox: `📦 Quick Start: Creating a Sandbox

1. Create a sandbox project:
   - Use: local_create_sandbox
   - Example: { directory: "my-sandbox", template: "template-sandbox-codegen" }

2. Deploy your sandbox:
   - Use: local_deploy_directory
   - Example: { directory: "my-sandbox" }

3. Run code in your sandbox:
   - Use: local_run_deployed_resource
   - Example: { resourceType: "sandbox", resourceName: "my-sandbox" }

Popular Sandbox Templates:
- template-sandbox-py: Python sandbox
- template-sandbox-ts: TypeScript sandbox

Sandboxes are ideal for:
- Secure code execution
- Testing environments
- Isolated computations`,

        all: `🎯 Blaxel Quick Start Guide

📌 Key Tools for Local Development:
1. local_list_templates - See all available templates
2. local_create_agent/job/mcp_server/sandbox - Create projects
3. local_deploy_directory - Deploy to Blaxel
4. local_run_deployed_resource - Run deployed resources

🔄 Typical Workflow:
1. List templates to see options
2. Create a local project with a template
3. Customize the code as needed
4. Deploy your project
5. Run and test

💡 Pro Tips:
- Custom MCP servers don't need integration connections
- For external services, use create_mcp_integration to set up credentials
- Start with template projects and customize
- Deploy frequently to test changes
- Use the --local flag to run locally first

📚 Resource Types:
- Agents: AI-powered applications
- MCP Servers: Model Context Protocol servers (custom or from MCP Hub)
- Jobs: Scheduled or triggered tasks
- Sandboxes: Secure execution environments

For specific guides, query with resourceType: "agent", "job", "mcp-server", or "sandbox"`
      };

      const guide = resourceType === "all"
        ? guides.all
        : guides[resourceType] || guides.all;

      return {
        content: [
          {
            type: "text",
            text: guide,
          },
        ],
      };
    }
  );
}
