// The subcommander package implements CLI subcommands and manages
// their flags and arguments.
package subcommander

import (
	"flag"
	"fmt"
	"os"
)

type Config interface {
	// Given a command name, add that command's flag declarations to the given FlagSet.
	DeclareFlags(string, *flag.FlagSet)
}

// A Command defines a CLI subcommand and its handler.
//
// When the subcommand of the given Name is requested, the Run
// function will be called with a Config given by the other CLI
// options, and a slice of strings containing the non-flag CLI
// arguments.
type Command struct {
	Name            string
	Description     string
	Run             func(Config, []string) error
	NumArgsRequired int
}

// Match returns true if the given CLI arguments match this command.
func (c *Command) Match(args []string) bool {
	if len(args) < 2 || args[1] != c.Name {
		return false
	}
	return true
}

// Execute parses the arguments, then runs the command handler.
func (c *Command) Execute(conf Config, args []string) error {
	flagSet := flag.NewFlagSet(c.Name, flag.ExitOnError)
	conf.DeclareFlags(c.Name, flagSet)
	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "Usage:\n\t %s %s [arguments]\n", args[0], c.Name)
		flagSet.PrintDefaults()
	}
	if !c.Match(args) {
		return fmt.Errorf("Attempted to execute the %s command with the wrong command name", c.Name)
	}
	if err := flagSet.Parse(args[2:]); err != nil {
		return err
	}
	if !flagSet.Parsed() {
		return fmt.Errorf("Could not parse arguments for the %q command.", c.Name)
	}
	if flagSet.NArg() < c.NumArgsRequired {
		return fmt.Errorf("The '%s' command should have %d or more arguments\n", c.Name, c.NumArgsRequired)
	}
	return c.Run(conf, flagSet.Args())
}

type CommandSet struct {
	Name               string
	DefaultCommandName string
	Commands           []Command
}

func (cs *CommandSet) printTopLevelUsage() {
	fmt.Fprintf(flag.CommandLine.Output(), "Usage:\n\t%s <command> [arguments]\n\n", cs.Name)
	fmt.Fprintf(flag.CommandLine.Output(), "Commands:\n\n")
	for _, command := range cs.Commands {
		fmt.Fprintf(flag.CommandLine.Output(), "%12s    %s\n", command.Name, command.Description)
	}
}

func (cs *CommandSet) runDefaultCommand(conf Config) error {
	for _, command := range cs.Commands {
		args := []string{cs.Name, cs.DefaultCommandName}
		if command.Match(args) {
			return command.Execute(conf, args)
		}
	}
	return fmt.Errorf("This command set does not define its own default command, %s", cs.DefaultCommandName)
}

type InvalidCommandError struct {
	CommandName string
}

func (e *InvalidCommandError) Error() string {
	return fmt.Sprintf("%q is not a valid command.", e.CommandName)
}

type NeededHelpError struct{}

func (e *NeededHelpError) Error() string { return "" }

// Execute matches the CLI arguments to a command, then runs that command.
func (cs *CommandSet) Execute(conf Config) error {
	if len(os.Args) < 2 {
		if cs.DefaultCommandName != "" {
			return cs.runDefaultCommand(conf)
		}
		cs.printTopLevelUsage()
		return &NeededHelpError{}
	}
	for _, command := range cs.Commands {
		if command.Match(os.Args) {
			return command.Execute(conf, os.Args)
		}
	}
	if os.Args[1] != "-h" && os.Args[1] != "--help" {
		return &InvalidCommandError{CommandName: os.Args[1]}
	}
	cs.printTopLevelUsage()
	return &NeededHelpError{}
}
