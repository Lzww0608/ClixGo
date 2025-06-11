// Package commands provides tmux compatibility layer
package commands

import (
	"fmt"
	"strconv"
	"strings"
)

// TmuxCompatLayer provides tmux command compatibility
type TmuxCompatLayer struct {
	parser      *ModernParser
	mappings    map[string]string
	keyBindings map[string]*KeyBinding
	prefixKey   string
	sessions    map[string]*Session
	logger      Logger
}

// KeyBinding represents a tmux-style key binding
type KeyBinding struct {
	Key        string
	Command    string
	Args       []string
	Repeatable bool
	Context    string
}

// NewTmuxCompatLayer creates a new tmux compatibility layer
func NewTmuxCompatLayer(parser *ModernParser, logger Logger) *TmuxCompatLayer {
	compat := &TmuxCompatLayer{
		parser:      parser,
		mappings:    make(map[string]string),
		keyBindings: make(map[string]*KeyBinding),
		prefixKey:   "C-b",
		sessions:    make(map[string]*Session),
		logger:      logger,
	}

	// Initialize tmux command mappings
	compat.initCommandMappings()

	// Initialize default key bindings
	compat.initDefaultKeyBindings()

	return compat
}

// initCommandMappings initializes tmux to ClixGo command mappings
func (t *TmuxCompatLayer) initCommandMappings() {
	// Session management commands
	t.mappings["new-session"] = "session new"
	t.mappings["new"] = "session new"
	t.mappings["attach-session"] = "session attach"
	t.mappings["attach"] = "session attach"
	t.mappings["detach-client"] = "session detach"
	t.mappings["detach"] = "session detach"
	t.mappings["kill-session"] = "session kill"
	t.mappings["list-sessions"] = "session list"
	t.mappings["ls"] = "session list"
	t.mappings["rename-session"] = "session rename"

	// Window management commands
	t.mappings["new-window"] = "window new"
	t.mappings["neww"] = "window new"
	t.mappings["kill-window"] = "window kill"
	t.mappings["killw"] = "window kill"
	t.mappings["select-window"] = "window select"
	t.mappings["selectw"] = "window select"
	t.mappings["next-window"] = "window next"
	t.mappings["next"] = "window next"
	t.mappings["previous-window"] = "window previous"
	t.mappings["prev"] = "window previous"
	t.mappings["last-window"] = "window last"
	t.mappings["last"] = "window last"
	t.mappings["rename-window"] = "window rename"
	t.mappings["renamew"] = "window rename"
	t.mappings["list-windows"] = "window list"
	t.mappings["lsw"] = "window list"

	// Pane management commands
	t.mappings["split-window"] = "pane split"
	t.mappings["splitw"] = "pane split"
	t.mappings["kill-pane"] = "pane kill"
	t.mappings["killp"] = "pane kill"
	t.mappings["select-pane"] = "pane select"
	t.mappings["selectp"] = "pane select"
	t.mappings["swap-pane"] = "pane swap"
	t.mappings["swapp"] = "pane swap"
	t.mappings["move-pane"] = "pane move"
	t.mappings["movep"] = "pane move"
	t.mappings["resize-pane"] = "pane resize"
	t.mappings["resizep"] = "pane resize"
	t.mappings["break-pane"] = "pane break"
	t.mappings["breakp"] = "pane break"
	t.mappings["join-pane"] = "pane join"
	t.mappings["joinp"] = "pane join"

	// Buffer/clipboard commands
	t.mappings["copy-mode"] = "buffer copy-mode"
	t.mappings["paste-buffer"] = "buffer paste"
	t.mappings["pasteb"] = "buffer paste"
	t.mappings["list-buffers"] = "buffer list"
	t.mappings["lsb"] = "buffer list"
	t.mappings["delete-buffer"] = "buffer delete"
	t.mappings["deleteb"] = "buffer delete"
	t.mappings["save-buffer"] = "buffer save"
	t.mappings["saveb"] = "buffer save"
	t.mappings["load-buffer"] = "buffer load"
	t.mappings["loadb"] = "buffer load"

	// Configuration commands
	t.mappings["bind-key"] = "config bind"
	t.mappings["bind"] = "config bind"
	t.mappings["unbind-key"] = "config unbind"
	t.mappings["unbind"] = "config unbind"
	t.mappings["set-option"] = "config set"
	t.mappings["set"] = "config set"
	t.mappings["show-options"] = "config show"
	t.mappings["show"] = "config show"
	t.mappings["source-file"] = "config source"
	t.mappings["source"] = "config source"

	// Display/info commands
	t.mappings["display-message"] = "display message"
	t.mappings["display"] = "display message"
	t.mappings["display-panes"] = "display panes"
	t.mappings["displayp"] = "display panes"
	t.mappings["list-keys"] = "display keys"
	t.mappings["lsk"] = "display keys"
	t.mappings["list-commands"] = "display commands"
	t.mappings["lscm"] = "display commands"
	t.mappings["info"] = "display info"
	t.mappings["show-messages"] = "display messages"
	t.mappings["showmsgs"] = "display messages"

	// Other commands
	t.mappings["refresh-client"] = "client refresh"
	t.mappings["refresh"] = "client refresh"
	t.mappings["suspend-client"] = "client suspend"
	t.mappings["suspendc"] = "client suspend"
	t.mappings["lock-session"] = "session lock"
	t.mappings["locks"] = "session lock"
	t.mappings["send-keys"] = "keys send"
	t.mappings["send"] = "keys send"
	t.mappings["send-prefix"] = "keys send-prefix"
	t.mappings["run-shell"] = "shell run"
	t.mappings["run"] = "shell run"
	t.mappings["if-shell"] = "shell if"
	t.mappings["if"] = "shell if"
}

// initDefaultKeyBindings initializes tmux default key bindings
func (t *TmuxCompatLayer) initDefaultKeyBindings() {
	// Control bindings
	t.addKeyBinding("C-b", "keys send-prefix", []string{}, false)
	t.addKeyBinding("C-o", "window rotate", []string{}, false)
	t.addKeyBinding("C-z", "client suspend", []string{}, false)

	// Special keys
	t.addKeyBinding("Space", "window next-layout", []string{}, false)
	t.addKeyBinding("!", "pane break", []string{}, false)
	t.addKeyBinding("\"", "pane split", []string{}, false)
	t.addKeyBinding("#", "buffer list", []string{}, false)
	t.addKeyBinding("$", "session rename", []string{}, false)
	t.addKeyBinding("%", "pane split", []string{"-h"}, false)
	t.addKeyBinding("&", "window kill", []string{}, false)
	t.addKeyBinding("'", "window select", []string{}, false)
	t.addKeyBinding("(", "session previous", []string{}, false)
	t.addKeyBinding(")", "session next", []string{}, false)
	t.addKeyBinding(",", "window rename", []string{}, false)
	t.addKeyBinding("-", "buffer delete", []string{}, false)
	t.addKeyBinding(".", "window move", []string{}, false)

	// Number keys
	for i := 0; i <= 9; i++ {
		t.addKeyBinding(strconv.Itoa(i), "window select", []string{"-t", strconv.Itoa(i)}, false)
	}

	// Letter keys
	t.addKeyBinding(":", "command-prompt", []string{}, false)
	t.addKeyBinding(";", "pane last", []string{}, false)
	t.addKeyBinding("=", "buffer choose", []string{}, false)
	t.addKeyBinding("?", "display keys", []string{}, false)
	t.addKeyBinding("D", "client choose", []string{}, false)
	t.addKeyBinding("L", "session last", []string{}, false)
	t.addKeyBinding("[", "buffer copy-mode", []string{}, false)
	t.addKeyBinding("]", "buffer paste", []string{}, false)
	t.addKeyBinding("c", "window new", []string{}, false)
	t.addKeyBinding("d", "session detach", []string{}, false)
	t.addKeyBinding("f", "window find", []string{}, false)
	t.addKeyBinding("i", "display message", []string{}, false)
	t.addKeyBinding("l", "window last", []string{}, false)
	t.addKeyBinding("n", "window next", []string{}, false)
	t.addKeyBinding("o", "pane next", []string{}, false)
	t.addKeyBinding("p", "window previous", []string{}, false)
	t.addKeyBinding("q", "display panes", []string{}, false)
	t.addKeyBinding("r", "client refresh", []string{}, false)
	t.addKeyBinding("s", "session choose", []string{}, false)
	t.addKeyBinding("t", "display clock", []string{}, false)
	t.addKeyBinding("w", "window choose", []string{}, false)
	t.addKeyBinding("x", "pane kill", []string{}, false)
	t.addKeyBinding("z", "pane zoom", []string{}, false)
	t.addKeyBinding("{", "pane swap", []string{"-U"}, false)
	t.addKeyBinding("}", "pane swap", []string{"-D"}, false)
	t.addKeyBinding("~", "display messages", []string{}, false)

	// Arrow keys (repeatable)
	t.addKeyBinding("Up", "pane select", []string{"-U"}, true)
	t.addKeyBinding("Down", "pane select", []string{"-D"}, true)
	t.addKeyBinding("Left", "pane select", []string{"-L"}, true)
	t.addKeyBinding("Right", "pane select", []string{"-R"}, true)

	// Meta combinations
	t.addKeyBinding("M-1", "window layout", []string{"even-horizontal"}, false)
	t.addKeyBinding("M-2", "window layout", []string{"even-vertical"}, false)
	t.addKeyBinding("M-3", "window layout", []string{"main-horizontal"}, false)
	t.addKeyBinding("M-4", "window layout", []string{"main-vertical"}, false)
	t.addKeyBinding("M-5", "window layout", []string{"tiled"}, false)
	t.addKeyBinding("M-n", "window next", []string{"-a"}, false)
	t.addKeyBinding("M-o", "window rotate", []string{"-D"}, false)
	t.addKeyBinding("M-p", "window previous", []string{"-a"}, false)

	// Meta arrow keys (repeatable)
	t.addKeyBinding("M-Up", "pane resize", []string{"-U", "5"}, true)
	t.addKeyBinding("M-Down", "pane resize", []string{"-D", "5"}, true)
	t.addKeyBinding("M-Left", "pane resize", []string{"-L", "5"}, true)
	t.addKeyBinding("M-Right", "pane resize", []string{"-R", "5"}, true)

	// Control arrow keys (repeatable)
	t.addKeyBinding("C-Up", "pane resize", []string{"-U"}, true)
	t.addKeyBinding("C-Down", "pane resize", []string{"-D"}, true)
	t.addKeyBinding("C-Left", "pane resize", []string{"-L"}, true)
	t.addKeyBinding("C-Right", "pane resize", []string{"-R"}, true)

	// Page up
	t.addKeyBinding("PPage", "buffer copy-mode", []string{"-u"}, false)
}

// addKeyBinding adds a key binding to the compatibility layer
func (t *TmuxCompatLayer) addKeyBinding(key, command string, args []string, repeatable bool) {
	t.keyBindings[key] = &KeyBinding{
		Key:        key,
		Command:    command,
		Args:       args,
		Repeatable: repeatable,
		Context:    "default",
	}
}

// ParseTmuxCommand parses a tmux-style command and converts it to ClixGo format
func (t *TmuxCompatLayer) ParseTmuxCommand(input string) (*CommandList, error) {
	if input == "" {
		return nil, fmt.Errorf("empty command input")
	}

	// Handle command chaining with semicolons
	commands := t.splitCommands(input)
	var cmdList []Command
	var argsList []*Arguments

	for _, cmdStr := range commands {
		cmdStr = strings.TrimSpace(cmdStr)
		if cmdStr == "" {
			continue
		}

		// Parse individual command
		cmd, args, err := t.parseIndividualCommand(cmdStr)
		if err != nil {
			return nil, fmt.Errorf("failed to parse command '%s': %v", cmdStr, err)
		}

		cmdList = append(cmdList, cmd)
		argsList = append(argsList, args)
	}

	if len(cmdList) == 0 {
		return nil, fmt.Errorf("no valid commands found")
	}

	return &CommandList{
		Commands: cmdList,
		Args:     argsList,
	}, nil
}

// splitCommands splits input on semicolons, respecting quotes
func (t *TmuxCompatLayer) splitCommands(input string) []string {
	var commands []string
	var current strings.Builder
	var inQuotes bool
	var quoteChar rune

	for _, r := range input {
		switch {
		case r == '"' || r == '\'':
			if !inQuotes {
				inQuotes = true
				quoteChar = r
			} else if r == quoteChar {
				inQuotes = false
				quoteChar = 0
			}
			current.WriteRune(r)
		case r == ';' && !inQuotes:
			if current.Len() > 0 {
				commands = append(commands, current.String())
				current.Reset()
			}
		default:
			current.WriteRune(r)
		}
	}

	if current.Len() > 0 {
		commands = append(commands, current.String())
	}

	return commands
}

// parseIndividualCommand parses a single tmux command
func (t *TmuxCompatLayer) parseIndividualCommand(cmdStr string) (Command, *Arguments, error) {
	// Tokenize the command
	tokens := t.tokenizeTmuxCommand(cmdStr)
	if len(tokens) == 0 {
		return nil, nil, fmt.Errorf("no tokens found")
	}

	// Get tmux command name
	tmuxCmd := tokens[0]

	// Map to ClixGo command
	clixGoCmd, exists := t.mappings[tmuxCmd]
	if !exists {
		// Check if it's already a ClixGo command
		if cmd, exists := t.parser.GetCommand(tmuxCmd); exists {
			args, err := t.parseClixGoArguments(cmd, tokens[1:])
			return cmd, args, err
		}
		return nil, nil, fmt.Errorf("unknown command: %s", tmuxCmd)
	}

	// Parse the mapped ClixGo command
	clixGoTokens := strings.Fields(clixGoCmd)
	clixGoCmdName := clixGoTokens[0]
	if len(clixGoTokens) > 1 {
		clixGoCmdName = strings.Join(clixGoTokens, " ")
	}

	cmd, exists := t.parser.GetCommand(clixGoCmdName)
	if !exists {
		return nil, nil, fmt.Errorf("mapped command not found: %s", clixGoCmdName)
	}

	// Convert tmux arguments to ClixGo arguments
	args, err := t.convertTmuxArguments(tmuxCmd, cmd, tokens[1:])
	if err != nil {
		return nil, nil, fmt.Errorf("argument conversion failed: %v", err)
	}

	return cmd, args, nil
}

// tokenizeTmuxCommand tokenizes a tmux command respecting quotes and escapes
func (t *TmuxCompatLayer) tokenizeTmuxCommand(input string) []string {
	var tokens []string
	var current strings.Builder
	var inQuotes bool
	var quoteChar rune
	var escaped bool

	for _, r := range input {
		if escaped {
			current.WriteRune(r)
			escaped = false
			continue
		}

		switch {
		case r == '\\':
			escaped = true
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
		default:
			current.WriteRune(r)
		}
	}

	if current.Len() > 0 {
		tokens = append(tokens, current.String())
	}

	return tokens
}

// convertTmuxArguments converts tmux-style arguments to ClixGo arguments
func (t *TmuxCompatLayer) convertTmuxArguments(tmuxCmd string, cmd Command, tokens []string) (*Arguments, error) {

	// Handle specific tmux command argument conversions
	switch tmuxCmd {
	case "new-session", "new":
		return t.convertNewSessionArgs(cmd, tokens)
	case "split-window", "splitw":
		return t.convertSplitWindowArgs(cmd, tokens)
	case "select-window", "selectw":
		return t.convertSelectWindowArgs(cmd, tokens)
	case "bind-key", "bind":
		return t.convertBindKeyArgs(cmd, tokens)
	default:
		// Generic argument conversion
		return t.parseClixGoArguments(cmd, tokens)
	}
}

// convertNewSessionArgs converts new-session arguments
func (t *TmuxCompatLayer) convertNewSessionArgs(cmd Command, tokens []string) (*Arguments, error) {
	args := &Arguments{
		Command:    cmd.Name(),
		Flags:      make(map[string]interface{}),
		Positional: []string{},
		Raw:        strings.Join(tokens, " "),
	}

	for i := 0; i < len(tokens); i++ {
		token := tokens[i]
		switch token {
		case "-d":
			args.Flags["detached"] = true
		case "-s":
			if i+1 < len(tokens) {
				i++
				args.Flags["session-name"] = tokens[i]
			}
		case "-n":
			if i+1 < len(tokens) {
				i++
				args.Flags["window-name"] = tokens[i]
			}
		case "-c":
			if i+1 < len(tokens) {
				i++
				args.Flags["start-directory"] = tokens[i]
			}
		default:
			if !strings.HasPrefix(token, "-") {
				args.Positional = append(args.Positional, token)
			}
		}
	}

	return args, nil
}

// convertSplitWindowArgs converts split-window arguments
func (t *TmuxCompatLayer) convertSplitWindowArgs(cmd Command, tokens []string) (*Arguments, error) {
	args := &Arguments{
		Command:    cmd.Name(),
		Flags:      make(map[string]interface{}),
		Positional: []string{},
		Raw:        strings.Join(tokens, " "),
	}

	for i := 0; i < len(tokens); i++ {
		token := tokens[i]
		switch token {
		case "-h":
			args.Flags["horizontal"] = true
		case "-v":
			args.Flags["vertical"] = true
		case "-c":
			if i+1 < len(tokens) {
				i++
				args.Flags["start-directory"] = tokens[i]
			}
		case "-t":
			if i+1 < len(tokens) {
				i++
				args.Flags["target"] = tokens[i]
			}
		default:
			if !strings.HasPrefix(token, "-") {
				args.Positional = append(args.Positional, token)
			}
		}
	}

	return args, nil
}

// convertSelectWindowArgs converts select-window arguments
func (t *TmuxCompatLayer) convertSelectWindowArgs(cmd Command, tokens []string) (*Arguments, error) {
	args := &Arguments{
		Command:    cmd.Name(),
		Flags:      make(map[string]interface{}),
		Positional: []string{},
		Raw:        strings.Join(tokens, " "),
	}

	for i := 0; i < len(tokens); i++ {
		token := tokens[i]
		switch token {
		case "-t":
			if i+1 < len(tokens) {
				i++
				target := tokens[i]
				// Handle :index format
				if strings.HasPrefix(target, ":") {
					if index, err := strconv.Atoi(target[1:]); err == nil {
						args.Flags["index"] = index
					}
				} else {
					args.Flags["target"] = target
				}
			}
		case "-n":
			args.Flags["next"] = true
		case "-p":
			args.Flags["previous"] = true
		case "-l":
			args.Flags["last"] = true
		default:
			if !strings.HasPrefix(token, "-") {
				args.Positional = append(args.Positional, token)
			}
		}
	}

	return args, nil
}

// convertBindKeyArgs converts bind-key arguments
func (t *TmuxCompatLayer) convertBindKeyArgs(cmd Command, tokens []string) (*Arguments, error) {
	args := &Arguments{
		Command:    cmd.Name(),
		Flags:      make(map[string]interface{}),
		Positional: []string{},
		Raw:        strings.Join(tokens, " "),
	}

	for i := 0; i < len(tokens); i++ {
		token := tokens[i]
		switch token {
		case "-r":
			args.Flags["repeatable"] = true
		case "-n":
			args.Flags["no-prefix"] = true
		case "-T":
			if i+1 < len(tokens) {
				i++
				args.Flags["table"] = tokens[i]
			}
		default:
			if !strings.HasPrefix(token, "-") {
				args.Positional = append(args.Positional, token)
			}
		}
	}

	return args, nil
}

// parseClixGoArguments parses arguments using the modern ClixGo parser
func (t *TmuxCompatLayer) parseClixGoArguments(cmd Command, tokens []string) (*Arguments, error) {
	return t.parser.parseArguments(cmd, tokens)
}

// HandleKeyBinding handles a tmux-style key binding
func (t *TmuxCompatLayer) HandleKeyBinding(key string) (*CommandList, error) {
	binding, exists := t.keyBindings[key]
	if !exists {
		return nil, fmt.Errorf("key binding not found: %s", key)
	}

	// Build command string
	cmdStr := binding.Command
	if len(binding.Args) > 0 {
		cmdStr += " " + strings.Join(binding.Args, " ")
	}

	// Parse and return command
	return t.ParseTmuxCommand(cmdStr)
}

// SetPrefixKey sets the prefix key for tmux compatibility
func (t *TmuxCompatLayer) SetPrefixKey(prefix string) {
	t.prefixKey = prefix
}

// GetPrefixKey returns the current prefix key
func (t *TmuxCompatLayer) GetPrefixKey() string {
	return t.prefixKey
}

// IsValidTmuxCommand checks if a command is a valid tmux command
func (t *TmuxCompatLayer) IsValidTmuxCommand(command string) bool {
	_, exists := t.mappings[command]
	return exists
}

// GetTmuxCommands returns all supported tmux commands
func (t *TmuxCompatLayer) GetTmuxCommands() []string {
	var commands []string
	for cmd := range t.mappings {
		commands = append(commands, cmd)
	}
	return commands
}
