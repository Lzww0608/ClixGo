// Package commands provides modern command parsing and execution framework
// inspired by tmux architecture but optimized for performance
package commands

import (
	"fmt"
	"strings"
	"sync"
)

// ArgumentType represents the type of command argument
type ArgumentType int

const (
	ArgString ArgumentType = iota
	ArgInt
	ArgBool
	ArgFloat
	ArgStringSlice
)

// Context provides execution context for commands
type Context struct {
	Session   *Session
	Window    *Window
	Pane      *Pane
	Client    *Client
	Variables map[string]interface{}
	Logger    Logger
}

// Session represents a terminal session
type Session struct {
	ID      string
	Name    string
	Windows []*Window
	Active  int
	Created int64
}

// Window represents a terminal window
type Window struct {
	ID     string
	Name   string
	Panes  []*Pane
	Active int
	Index  int
}

// Pane represents a terminal pane
type Pane struct {
	ID     string
	Width  int
	Height int
	X      int
	Y      int
	Active bool
}

// Client represents a connected client
type Client struct {
	ID      string
	Address string
	TTY     string
	Width   int
	Height  int
}

// Logger interface for command logging
type Logger interface {
	Debug(msg string, args ...interface{})
	Info(msg string, args ...interface{})
	Error(msg string, args ...interface{})
}

// ArgumentSpec defines command argument specification
type ArgumentSpec struct {
	Name        string
	ShortFlag   string
	LongFlag    string
	Type        ArgumentType
	Required    bool
	Default     interface{}
	Validator   func(interface{}) error
	Description string
}

// Arguments holds parsed command arguments
type Arguments struct {
	Command    string
	Flags      map[string]interface{}
	Positional []string
	Raw        string
}

// Command interface that all commands must implement
type Command interface {
	Execute(ctx *Context, args *Arguments) error
	Validate(args *Arguments) error
	Usage() string
	Name() string
	Description() string
	ArgumentSpecs() []ArgumentSpec
}

// CommandList represents a list of commands to execute
type CommandList struct {
	Commands []Command
	Args     []*Arguments
}

// Parser interface for command parsing
type Parser interface {
	Parse(input string) (*CommandList, error)
	RegisterCommand(cmd Command) error
	GetCommand(name string) (Command, bool)
	ListCommands() []string
}

// ModernParser implements a high-performance command parser
type ModernParser struct {
	commands map[string]Command
	aliases  map[string]string
	groups   map[string][]string
	mutex    sync.RWMutex
	logger   Logger
}

// NewModernParser creates a new modern command parser
func NewModernParser(logger Logger) *ModernParser {
	return &ModernParser{
		commands: make(map[string]Command),
		aliases:  make(map[string]string),
		groups:   make(map[string][]string),
		logger:   logger,
	}
}

// RegisterCommand registers a new command
func (p *ModernParser) RegisterCommand(cmd Command) error {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	name := cmd.Name()
	if name == "" {
		return fmt.Errorf("command name cannot be empty")
	}

	if _, exists := p.commands[name]; exists {
		return fmt.Errorf("command %s already registered", name)
	}

	p.commands[name] = cmd
	p.logger.Debug("Registered command: %s", name)
	return nil
}

// GetCommand retrieves a command by name
func (p *ModernParser) GetCommand(name string) (Command, bool) {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	// Check direct command
	if cmd, exists := p.commands[name]; exists {
		return cmd, true
	}

	// Check aliases
	if alias, exists := p.aliases[name]; exists {
		if cmd, exists := p.commands[alias]; exists {
			return cmd, true
		}
	}

	return nil, false
}

// ListCommands returns list of all registered commands
func (p *ModernParser) ListCommands() []string {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	var commands []string
	for name := range p.commands {
		commands = append(commands, name)
	}
	return commands
}

// Parse parses command input and returns command list
func (p *ModernParser) Parse(input string) (*CommandList, error) {
	if input == "" {
		return nil, fmt.Errorf("empty command input")
	}

	// Tokenize input
	tokens := p.tokenize(input)
	if len(tokens) == 0 {
		return nil, fmt.Errorf("no tokens found")
	}

	// Parse first command
	cmdName := tokens[0]
	cmd, exists := p.GetCommand(cmdName)
	if !exists {
		return nil, fmt.Errorf("command not found: %s", cmdName)
	}

	// Parse arguments
	args, err := p.parseArguments(cmd, tokens[1:])
	if err != nil {
		return nil, fmt.Errorf("argument parsing failed: %v", err)
	}

	// Validate arguments
	if err := cmd.Validate(args); err != nil {
		return nil, fmt.Errorf("argument validation failed: %v", err)
	}

	return &CommandList{
		Commands: []Command{cmd},
		Args:     []*Arguments{args},
	}, nil
}

// tokenize splits input into tokens, respecting quotes
func (p *ModernParser) tokenize(input string) []string {
	var tokens []string
	var current strings.Builder
	var inQuotes bool
	var quoteChar rune

	for i, r := range input {
		switch {
		case r == '"' || r == '\'':
			if !inQuotes {
				inQuotes = true
				quoteChar = r
			} else if r == quoteChar {
				inQuotes = false
				quoteChar = 0
			} else {
				current.WriteRune(r)
			}
		case r == ' ' || r == '\t':
			if inQuotes {
				current.WriteRune(r)
			} else if current.Len() > 0 {
				tokens = append(tokens, current.String())
				current.Reset()
			}
		case r == '\\' && i+1 < len(input):
			// Handle escape sequences
			next := rune(input[i+1])
			if next == 'n' {
				current.WriteRune('\n')
			} else if next == 't' {
				current.WriteRune('\t')
			} else {
				current.WriteRune(next)
			}
			// Skip next character
			continue
		default:
			current.WriteRune(r)
		}
	}

	if current.Len() > 0 {
		tokens = append(tokens, current.String())
	}

	return tokens
}

// parseArguments parses command arguments according to command specification
func (p *ModernParser) parseArguments(cmd Command, tokens []string) (*Arguments, error) {
	args := &Arguments{
		Command:    cmd.Name(),
		Flags:      make(map[string]interface{}),
		Positional: []string{},
		Raw:        strings.Join(tokens, " "),
	}

	specs := cmd.ArgumentSpecs()
	specMap := make(map[string]*ArgumentSpec)

	// Build spec lookup maps
	for i := range specs {
		spec := &specs[i]
		if spec.ShortFlag != "" {
			specMap["-"+spec.ShortFlag] = spec
		}
		if spec.LongFlag != "" {
			specMap["--"+spec.LongFlag] = spec
		}
	}

	// Parse tokens
	for i := 0; i < len(tokens); i++ {
		token := tokens[i]

		if strings.HasPrefix(token, "-") {
			// Parse flag
			spec, exists := specMap[token]
			if !exists {
				return nil, fmt.Errorf("unknown flag: %s", token)
			}

			if spec.Type == ArgBool {
				args.Flags[spec.Name] = true
			} else {
				// Need value
				if i+1 >= len(tokens) {
					return nil, fmt.Errorf("flag %s requires value", token)
				}
				i++
				value, err := p.convertValue(tokens[i], spec.Type)
				if err != nil {
					return nil, fmt.Errorf("invalid value for %s: %v", token, err)
				}
				args.Flags[spec.Name] = value
			}
		} else {
			// Positional argument
			args.Positional = append(args.Positional, token)
		}
	}

	// Set defaults for missing flags
	for _, spec := range specs {
		if _, exists := args.Flags[spec.Name]; !exists && spec.Default != nil {
			args.Flags[spec.Name] = spec.Default
		}
	}

	return args, nil
}

// convertValue converts string value to specified type
func (p *ModernParser) convertValue(value string, argType ArgumentType) (interface{}, error) {
	switch argType {
	case ArgString:
		return value, nil
	case ArgInt:
		var i int
		if _, err := fmt.Sscanf(value, "%d", &i); err != nil {
			return nil, err
		}
		return i, nil
	case ArgFloat:
		var f float64
		if _, err := fmt.Sscanf(value, "%f", &f); err != nil {
			return nil, err
		}
		return f, nil
	case ArgBool:
		switch strings.ToLower(value) {
		case "true", "1", "yes", "on":
			return true, nil
		case "false", "0", "no", "off":
			return false, nil
		default:
			return nil, fmt.Errorf("invalid boolean value: %s", value)
		}
	case ArgStringSlice:
		return strings.Split(value, ","), nil
	default:
		return nil, fmt.Errorf("unsupported argument type: %v", argType)
	}
}

// RegisterAlias registers a command alias
func (p *ModernParser) RegisterAlias(alias, command string) error {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	if _, exists := p.commands[command]; !exists {
		return fmt.Errorf("command %s does not exist", command)
	}

	p.aliases[alias] = command
	return nil
}

// Execute executes a command list in the given context
func (cl *CommandList) Execute(ctx *Context) error {
	for i, cmd := range cl.Commands {
		if err := cmd.Execute(ctx, cl.Args[i]); err != nil {
			return fmt.Errorf("command %s failed: %v", cmd.Name(), err)
		}
	}
	return nil
}
