#!/bin/bash

# Delete functions starting with "agent-mcp-"
bl get functions -o json | \
  jq -r '.[] |
    select(.metadata.name |
      type == "string" and
      startswith("agent-mcp-")
    ) |
    .metadata.name' | \
  xargs -I {} bl delete function {}

# Delete integration connections starting with "agent-int-"
bl get ic -o json | \
  jq -r '.[] |
    select(.metadata.name |
      type == "string" and
      startswith("agent-int-")
    ) |
    .metadata.name' | \
  xargs -I {} bl delete ic {}