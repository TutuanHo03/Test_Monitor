package models

// Command represents a command that can be executed
type Command struct {
	Name        string       `json:"name"`
	Help        string       `json:"help"`
	Usage       string       `json:"usage"`
	Subcommands []Subcommand `json:"subcommands"`
}

// Subcommand represents a subcommand of a command
type Subcommand struct {
	Name      string     `json:"name"`
	Help      string     `json:"help"`
	Usage     string     `json:"usage"`
	Response  string     `json:"response"`
	Arguments []Argument `json:"arguments"`
}

// Argument represents an argument to a subcommand
type Argument struct {
	Name         string `json:"name"`
	Description  string `json:"description"`
	Type         string `json:"type"`
	Required     bool   `json:"required"`
	Value        string
	AllowMutiple bool `json:"allowMutiple"`
}

// FormArgs represents the arguments passed from the client to the server
type FormArgs struct {
	NodeType   string            `json:"nodeType"`
	NodeName   string            `json:"nodeName"`
	Command    string            `json:"command"`
	Subcommand string            `json:"subcommand"`
	Arguments  map[string]string `json:"arguments"`
	RawCommand string            `json:"rawCommand"`
}
