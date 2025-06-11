// Package commands provides session management commands
package commands

import (
	"fmt"
	"time"
)

// NewSessionCommand implements the session new command
type NewSessionCommand struct {
	logger Logger
}

// NewNewSessionCommand creates a new session command
func NewNewSessionCommand(logger Logger) *NewSessionCommand {
	return &NewSessionCommand{logger: logger}
}

// Name returns the command name
func (c *NewSessionCommand) Name() string {
	return "session new"
}

// Description returns command description
func (c *NewSessionCommand) Description() string {
	return "Create a new terminal session"
}

// Usage returns usage string
func (c *NewSessionCommand) Usage() string {
	return "session new [-d] [-s session-name] [-n window-name] [-c start-directory] [command]"
}

// ArgumentSpecs returns argument specifications
func (c *NewSessionCommand) ArgumentSpecs() []ArgumentSpec {
	return []ArgumentSpec{
		{
			Name:        "detached",
			ShortFlag:   "d",
			Type:        ArgBool,
			Default:     false,
			Description: "Create session in detached mode",
		},
		{
			Name:        "session-name",
			ShortFlag:   "s",
			Type:        ArgString,
			Description: "Name of the new session",
		},
		{
			Name:        "window-name",
			ShortFlag:   "n",
			Type:        ArgString,
			Description: "Name of the initial window",
		},
		{
			Name:        "start-directory",
			ShortFlag:   "c",
			Type:        ArgString,
			Description: "Working directory for the new session",
		},
	}
}

// Validate validates command arguments
func (c *NewSessionCommand) Validate(args *Arguments) error {
	if sessionName, exists := args.Flags["session-name"]; exists {
		if name, ok := sessionName.(string); ok && name == "" {
			return fmt.Errorf("session name cannot be empty")
		}
	}
	return nil
}

// Execute executes the command
func (c *NewSessionCommand) Execute(ctx *Context, args *Arguments) error {
	// Generate session ID
	sessionID := fmt.Sprintf("session_%d", time.Now().UnixNano())

	// Get session name
	sessionName := sessionID
	if name, exists := args.Flags["session-name"]; exists {
		if nameStr, ok := name.(string); ok {
			sessionName = nameStr
		}
	}

	// Create new session
	session := &Session{
		ID:      sessionID,
		Name:    sessionName,
		Windows: []*Window{},
		Active:  0,
		Created: time.Now().Unix(),
	}

	// Create initial window
	windowName := "0"
	if name, exists := args.Flags["window-name"]; exists {
		if nameStr, ok := name.(string); ok {
			windowName = nameStr
		}
	}

	window := &Window{
		ID:     fmt.Sprintf("window_%d", time.Now().UnixNano()),
		Name:   windowName,
		Panes:  []*Pane{},
		Active: 0,
		Index:  0,
	}

	// Create initial pane
	pane := &Pane{
		ID:     fmt.Sprintf("pane_%d", time.Now().UnixNano()),
		Width:  80,
		Height: 24,
		X:      0,
		Y:      0,
		Active: true,
	}

	window.Panes = append(window.Panes, pane)
	session.Windows = append(session.Windows, window)

	// Update context
	ctx.Session = session
	ctx.Window = window
	ctx.Pane = pane

	c.logger.Info("Created new session: %s", sessionName)

	// Handle detached mode
	if detached, exists := args.Flags["detached"]; exists {
		if detachedBool, ok := detached.(bool); ok && detachedBool {
			c.logger.Info("Session created in detached mode")
			return nil
		}
	}

	// Attach to session (in real implementation, this would start the terminal)
	c.logger.Info("Attaching to session: %s", sessionName)

	return nil
}

// AttachSessionCommand implements the session attach command
type AttachSessionCommand struct {
	logger Logger
}

// NewAttachSessionCommand creates a new attach session command
func NewAttachSessionCommand(logger Logger) *AttachSessionCommand {
	return &AttachSessionCommand{logger: logger}
}

// Name returns the command name
func (c *AttachSessionCommand) Name() string {
	return "session attach"
}

// Description returns command description
func (c *AttachSessionCommand) Description() string {
	return "Attach to an existing session"
}

// Usage returns usage string
func (c *AttachSessionCommand) Usage() string {
	return "session attach [-d] [-r] [-t target-session]"
}

// ArgumentSpecs returns argument specifications
func (c *AttachSessionCommand) ArgumentSpecs() []ArgumentSpec {
	return []ArgumentSpec{
		{
			Name:        "detach-others",
			ShortFlag:   "d",
			Type:        ArgBool,
			Default:     false,
			Description: "Detach other clients attached to the session",
		},
		{
			Name:        "read-only",
			ShortFlag:   "r",
			Type:        ArgBool,
			Default:     false,
			Description: "Attach in read-only mode",
		},
		{
			Name:        "target-session",
			ShortFlag:   "t",
			Type:        ArgString,
			Description: "Target session name or ID",
		},
	}
}

// Validate validates command arguments
func (c *AttachSessionCommand) Validate(args *Arguments) error {
	return nil
}

// Execute executes the command
func (c *AttachSessionCommand) Execute(ctx *Context, args *Arguments) error {
	targetSession := ""
	if target, exists := args.Flags["target-session"]; exists {
		if targetStr, ok := target.(string); ok {
			targetSession = targetStr
		}
	}

	c.logger.Info("Attaching to session: %s", targetSession)

	// In real implementation, this would:
	// 1. Find the target session
	// 2. Attach the current client to it
	// 3. Handle detach-others and read-only flags

	return nil
}

// ListSessionsCommand implements the session list command
type ListSessionsCommand struct {
	logger Logger
}

// NewListSessionsCommand creates a new list sessions command
func NewListSessionsCommand(logger Logger) *ListSessionsCommand {
	return &ListSessionsCommand{logger: logger}
}

// Name returns the command name
func (c *ListSessionsCommand) Name() string {
	return "session list"
}

// Description returns command description
func (c *ListSessionsCommand) Description() string {
	return "List all sessions"
}

// Usage returns usage string
func (c *ListSessionsCommand) Usage() string {
	return "session list [-F format]"
}

// ArgumentSpecs returns argument specifications
func (c *ListSessionsCommand) ArgumentSpecs() []ArgumentSpec {
	return []ArgumentSpec{
		{
			Name:        "format",
			ShortFlag:   "F",
			Type:        ArgString,
			Default:     "#{session_name}: #{session_windows} windows",
			Description: "Output format",
		},
	}
}

// Validate validates command arguments
func (c *ListSessionsCommand) Validate(args *Arguments) error {
	return nil
}

// Execute executes the command
func (c *ListSessionsCommand) Execute(ctx *Context, args *Arguments) error {
	format := "#{session_name}: #{session_windows} windows"
	if f, exists := args.Flags["format"]; exists {
		if formatStr, ok := f.(string); ok {
			format = formatStr
		}
	}

	c.logger.Info("Listing sessions with format: %s", format)

	// In real implementation, this would:
	// 1. Get all active sessions
	// 2. Format each session according to the format string
	// 3. Output the formatted list

	return nil
}

// KillSessionCommand implements the session kill command
type KillSessionCommand struct {
	logger Logger
}

// NewKillSessionCommand creates a new kill session command
func NewKillSessionCommand(logger Logger) *KillSessionCommand {
	return &KillSessionCommand{logger: logger}
}

// Name returns the command name
func (c *KillSessionCommand) Name() string {
	return "session kill"
}

// Description returns command description
func (c *KillSessionCommand) Description() string {
	return "Kill a session"
}

// Usage returns usage string
func (c *KillSessionCommand) Usage() string {
	return "session kill [-t target-session] [-a]"
}

// ArgumentSpecs returns argument specifications
func (c *KillSessionCommand) ArgumentSpecs() []ArgumentSpec {
	return []ArgumentSpec{
		{
			Name:        "target-session",
			ShortFlag:   "t",
			Type:        ArgString,
			Description: "Target session name or ID",
		},
		{
			Name:        "all",
			ShortFlag:   "a",
			Type:        ArgBool,
			Default:     false,
			Description: "Kill all sessions",
		},
	}
}

// Validate validates command arguments
func (c *KillSessionCommand) Validate(args *Arguments) error {
	if allFlag, exists := args.Flags["all"]; exists {
		if all, ok := allFlag.(bool); ok && all {
			// If killing all sessions, target-session should not be specified
			if _, exists := args.Flags["target-session"]; exists {
				return fmt.Errorf("cannot specify target session when using -a flag")
			}
		}
	}
	return nil
}

// Execute executes the command
func (c *KillSessionCommand) Execute(ctx *Context, args *Arguments) error {
	if allFlag, exists := args.Flags["all"]; exists {
		if all, ok := allFlag.(bool); ok && all {
			c.logger.Info("Killing all sessions")
			// In real implementation, this would kill all sessions
			return nil
		}
	}

	targetSession := ""
	if target, exists := args.Flags["target-session"]; exists {
		if targetStr, ok := target.(string); ok {
			targetSession = targetStr
		}
	}

	if targetSession == "" {
		// Use current session if no target specified
		if ctx.Session != nil {
			targetSession = ctx.Session.Name
		} else {
			return fmt.Errorf("no target session specified and no current session")
		}
	}

	c.logger.Info("Killing session: %s", targetSession)

	// In real implementation, this would:
	// 1. Find the target session
	// 2. Close all windows and panes
	// 3. Clean up resources
	// 4. Remove from session list

	return nil
}

// RenameSessionCommand implements the session rename command
type RenameSessionCommand struct {
	logger Logger
}

// NewRenameSessionCommand creates a new rename session command
func NewRenameSessionCommand(logger Logger) *RenameSessionCommand {
	return &RenameSessionCommand{logger: logger}
}

// Name returns the command name
func (c *RenameSessionCommand) Name() string {
	return "session rename"
}

// Description returns command description
func (c *RenameSessionCommand) Description() string {
	return "Rename a session"
}

// Usage returns usage string
func (c *RenameSessionCommand) Usage() string {
	return "session rename [-t target-session] new-name"
}

// ArgumentSpecs returns argument specifications
func (c *RenameSessionCommand) ArgumentSpecs() []ArgumentSpec {
	return []ArgumentSpec{
		{
			Name:        "target-session",
			ShortFlag:   "t",
			Type:        ArgString,
			Description: "Target session name or ID",
		},
	}
}

// Validate validates command arguments
func (c *RenameSessionCommand) Validate(args *Arguments) error {
	if len(args.Positional) == 0 {
		return fmt.Errorf("new session name is required")
	}

	newName := args.Positional[0]
	if newName == "" {
		return fmt.Errorf("new session name cannot be empty")
	}

	return nil
}

// Execute executes the command
func (c *RenameSessionCommand) Execute(ctx *Context, args *Arguments) error {
	if len(args.Positional) == 0 {
		return fmt.Errorf("new session name is required")
	}

	newName := args.Positional[0]

	targetSession := ""
	if target, exists := args.Flags["target-session"]; exists {
		if targetStr, ok := target.(string); ok {
			targetSession = targetStr
		}
	}

	if targetSession == "" {
		// Use current session if no target specified
		if ctx.Session != nil {
			targetSession = ctx.Session.Name
		} else {
			return fmt.Errorf("no target session specified and no current session")
		}
	}

	c.logger.Info("Renaming session '%s' to '%s'", targetSession, newName)

	// Update current session name if it's the target
	if ctx.Session != nil && ctx.Session.Name == targetSession {
		ctx.Session.Name = newName
	}

	// In real implementation, this would:
	// 1. Find the target session
	// 2. Update its name
	// 3. Update any references to the old name

	return nil
}
