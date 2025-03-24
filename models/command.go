package models

// Command represents a command that can be executed
type Command struct {
	Name      string     `json:"name"`
	Help      string     `json:"help"`
	Usage     string     `json:"usage"`
	Arguments []Argument `json:"arguments"`
	Func      func(map[string]string) string
}

// Argument represents an argument to a subcommand
type Argument struct {
	Tag          string `json:"Tag"`
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
	Arguments  map[string]string `json:"arguments"`
	RawCommand string            `json:"rawCommand"`
}
