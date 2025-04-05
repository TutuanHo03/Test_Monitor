package models

// CommandInfo - Define the structure command same as server
type CommandInfo struct {
	Name        string        `json:"name"`
	Usage       string        `json:"usage"`
	Description string        `json:"description"`
	ArgsUsage   string        `json:"argsUsage"`
	Flags       []FlagInfo    `json:"flags"`
	Subcommands []CommandInfo `json:"subcommands,omitempty"`
}

// FlagInfo - Define the structure of flag same as server
type FlagInfo struct {
	Name        string `json:"name"`
	Usage       string `json:"usage"`
	DefaultText string `json:"defaultText,omitempty"`
	Required    bool   `json:"required"`
}

// CommandRequest - Structure demand same as server
type CommandRequest struct {
	NodeType    string            `json:"nodeType"`
	NodeName    string            `json:"nodeName"`
	CommandPath string            `json:"commandPath"`
	RawCommand  string            `json:"rawCommand"`
	Args        []string          `json:"args,omitempty"`
	Flags       map[string]string `json:"flags,omitempty"`
}

// CommandResponse - Structure response same as server
type CommandResponse struct {
	Response string `json:"response"`
	Error    string `json:"error"`
}

// NavigationRequest - Structure of request navigation
type NavigationRequest struct {
	CurrentContext string   `json:"currentContext"`
	Command        string   `json:"command"`
	Args           []string `json:"args"`
	ServerURL      string   `json:"serverURL"`
	NodeType       string   `json:"nodeType"`
}

// NavigationResponse - Structure of response navigation
type NavigationResponse struct {
	Context  ClientContext `json:"context"`
	Prompt   string        `json:"prompt"`
	Message  string        `json:"message"`
	Commands []CommandInfo `json:"commands"`
	Error    string        `json:"error"`
}

// ClientContext - Structure of client context
type ClientContext struct {
	Type          string   `json:"type"`
	Name          string   `json:"name"`
	ServerURL     string   `json:"serverURL"`
	Description   string   `json:"description"`
	ParentPath    string   `json:"parentPath"`
	ChildrenPaths []string `json:"childrenPaths"`
	NodeType      string   `json:"nodeType"`
	Commands      []string `json:"commands"`
}
