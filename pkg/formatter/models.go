package formatter

import (
	"time"
)

// Simple model structs for formatting

// AgentModel represents a simple agent model
type AgentModel struct {
	Name       string
	Status     string
	Image      *string
	Memory     *int
	Generation *string
	MaxTasks   *int
	Labels     map[string]string
	CreatedAt  *time.Time
}

// JobModel represents a simple job model
type JobModel struct {
	Name       string
	Status     string
	Image      *string
	Memory     *int
	MaxTasks   *int
	MaxRetries *int
	Labels     map[string]string
	CreatedAt  *time.Time
}

// ModelAPI represents a simple model API model
type ModelAPI struct {
	Name      string
	Status    string
	Type      *string
	ModelName *string
	Memory    *int
	Labels    map[string]string
	CreatedAt *time.Time
}

// FunctionModel represents a simple function/MCP server model
type FunctionModel struct {
	Name                   string
	Status                 string
	Image                  *string
	Memory                 *int
	Generation             *string
	IntegrationConnections []string
	Labels                 map[string]string
	CreatedAt              *time.Time
}

// SandboxModel represents a simple sandbox model
type SandboxModel struct {
	Name       string
	Status     string
	Image      *string
	Memory     *int
	Generation *string
	TTL        *string
	Expires    *time.Time
	Ports      []int
	Labels     map[string]string
	CreatedAt  *time.Time
}

// IntegrationModel represents a simple integration model
type IntegrationModel struct {
	Name      string
	Secrets   map[string]string
	Config    map[string]string
	Labels    map[string]string
	CreatedAt *time.Time
}

// UserModel represents a simple user model
type UserModel struct {
	Email         string
	Name          string
	Role          string
	Accepted      bool
	EmailVerified bool
}

// ServiceAccountModel represents a simple service account model
type ServiceAccountModel struct {
	Name        string
	ClientID    string
	Description string
	CreatedAt   *time.Time
}

// TemplateModel represents a simple template model
type TemplateModel struct {
	Name          string
	Description   *string
	Topics        []string
	StarCount     *int
	DownloadCount *int
}
