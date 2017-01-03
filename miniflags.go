/*
Package miniflags implements a command line option parser. It is intended
to be small and simple to use for developers, and to offer ease of use for
for end users.

This package has the following features:

- Arguments and options may be freely interleaved. This saves keystrokes for
end users who are making small changes to a command on each run.

- There is support for both short and long option styles. Short option strings
may be a concatenation of zero-or more flag options, joined by up to one
option that takes a parameter, optionally joined by the parameter's value.
(Alternatively, the parameter value may be provided in the next argument.)

- Options may be defined by stringing together Option method calls on an
OptionSet object. Each call specifies one or more names (long and/or short),
any help text, and the "target" of the option.

- Option parameters may be joined to the option name with a '=' character,
or may be placed separately in the next argument.

- A target may be a pointer to one of several supported primitive types.  The
option then attempts to convert any parameter to that type and update the
referenced variable. The target variables can be normal values within the
client program, and do not themselves need to be pointers or of special types.
Their initial values are the defaults if the corresponding option(s) are not
specified on the command line.

- A target may instead be a "setter" function that can perform an arbitrary
action.  Some factory functions are included in this package to make setters
that do common custom operations such as incrementing a variable each time an
option is seen, or setting a string using a finite set of choices. Custom
actions may be implemented with a symple anonymous function.

- If no help options are defined, this package has an optional default
implementation that prints the usage text and exits the program.

- A custom action may be defined to handle non-option arguments instead
of the default of appending them to an arguments list.

- Section headers may be defined to separate groups of options in the help
output.

Example usage:
	import "github.com/jsthayer/miniflags"

	var (
		num = 3
		color string
		str []string
		flag bool
	)

	args, err := miniflags.NewOptionSet().
		Option("n number", &num, "=NUM; Number value (default=3)").
		Option("c color ", miniflags.AlternativesOption(&color, []string{"red", "green", "blue"}),
			"=COLOR; Color (red, green or blue)").
		Option("  list  ", &str, "=ITEM; String list value").
		Option("f flag  ", &flag, "Boolean flag").
		Option("E eight ", func() { num = 8 }, "Set number value to 8").
		ParseArgs(nil)

Using the above definitions in a test such as "test_prog -h" produces the following output:

	Usage: test_prog [ options and/or arguments ]
	Options:
	  -n, --number=NUM  Number value (default=3)
	  -c, --color=COLOR Color (red, green or blue)
	  --list=ITEM       String list value
	  -f, --flag        Boolean flag
	  -E, --eight       Set number value to 8

In the above example, the following command line arguments all set num to 8 and flag to true:

	--number 8 --flag
	--number=8 -f
	-n 8 -f
	-n=8 --flag
	-n8 -f
	-fn8
	-fn=8
	--eight --flag
	-Ef
*/
package miniflags

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// OptionDef structs are used to specify options.
type OptionDef struct {
	names  string      // Space-separated long and/or short option names
	target interface{} // The variable receiving the option or a setter function
	help   string      // Description of this option in the usage help text
}

// OptionSet holds a set of OptionDef structures that defines the valid options
// for a parsing operation.
type OptionSet struct {
	list       []*OptionDef          // The options in this set in original order
	index      map[string]*OptionDef // Options indexed by names
	argAction  *OptionDef            // Optional action for non-option arguments
	setupError error                 // Any error detected in the definition phase
}

// Emit is called when the option parser needs to write a user-visible message
// line. The default action is to write the line to stderr. This function can
// be replaced by the client to substitute different behavior.
var Emit = func(a ...interface{}) {
	fmt.Fprintln(os.Stderr, a...)
}

// OnError is called when the option parser encounters an error. Defs is the
// current list of OptionDef structures. The default action is to print the
// usage message, followed by any values provided in "a", then calling
// os.Exit(1).  This function can be replaced by the client to substitute
// different behavior.
var OnError func(defs *OptionSet, a ...interface{})

// Args contains the non-option arguments found by the most recent call to
// ParseArgs.  A copy of this list is also returned by the ParseArgs function.
var Args []string

// UsageHeader is the first part of the message displayed by the Usage
// function.  The default shows "Usage:", followed by the program name,
// followed by generic options choices. This string can be replaced by the
// client to get a different header.
var UsageHeader = fmt.Sprintf("Usage: %s [ options and/or arguments ]", filepath.Base(os.Args[0]))

// Usage displays the command line usage help for the program, using the given
// list of OptionDef structures. The default is to print the usage header,
// followed by any help for non-option arguments, followed by help text for
// each option. This function can be replaced by the client to substitute
// different behavior.
var Usage func(defs *OptionSet)

// AutoHelp enables the automatic generation of help options. If neither "-h"
// or "--help" options are defined and AutoHelp is true, then the above named
// options will automatically be added to the option definition list. The
// action for these options will be to print the usage message and exit the
// program with a zero status. If AutoHelp is set to false, then the automatic
// help options will not be added.
var AutoHelp = true

// Set the implementations for OnError and Usage here so they don't clutter the
// documentation.
func init() {
	OnError = func(defs *OptionSet, a ...interface{}) {
		Usage(defs)
		Emit()
		Emit(a...)
		os.Exit(1)
	}

	Usage = func(defs *OptionSet) {
		Emit(UsageHeader)
		Emit("Options:")
		for _, line := range defs.FormatOptionsHelp() {
			Emit(line)
		}
	}
}

// IncOption is a factory function that can be called to create an
// OptionDef.target value that will increment the referenced integer variable
// each time the option flag is encountered. The option will not take a
// parameter.
func IncOption(target *int) func() {
	return func() {
		*target++
	}
}

// DecOption is a factory function that can be called to create an
// OptionDef.target value that will decrement the referenced integer variable
// each time the option flag is encountered. The option will not take a
// parameter.
func DecOption(target *int) func() {
	return func() {
		*target--
	}
}

// FlagResetOption is a factory function that can be called to create a
// OptionDef.target value that will reset the referenced boolean value to false
// when the option flag is encountered. The option will not take a parameter.
func FlagResetOption(target *bool) func() {
	return func() {
		*target = false
	}
}

// AlternativesOption is a factory function that can be called to create an
// Option target value that will only accept one of the set of
// alternative values specified in choices.
func AlternativesOption(target *string, choices []string) func(val string) error {
	return func(val string) error {
		for _, choice := range choices {
			if choice == val {
				*target = val
				return nil
			}
		}
		return fmt.Errorf("Invalid parameter value '%s'", val)
	}
}

// Test whether the option defined by def consumes a parameter. Returns true
// unless the target is either a bool variable or is a setter function that
// takes no parameters.
func (def *OptionDef) takesParameter() bool {
	switch def.target.(type) {
	case *bool, func() error, func(), func() (*OptionSet, error):
		return false
	default:
		return true
	}
}

// Check if this entry is just a section header for help outptut
func (self *OptionDef) isSectionHeader() bool {
	return self.target == nil && self.names == ""
}

// Check if the target in this entry is of a supported type
func (self *OptionDef) isTargetOk() bool {
	if self.isSectionHeader() {
		return true
	}
	switch self.target.(type) {
	case func(string) error, func() error, func(string), func(),
		*string, *uint, *uint64, *int, *int64, *float64, *bool, *[]string:
		return true
	default:
		return false
	}
}

// Given the space-separated names field from an OptionDef, return a string
// with those names formatted for output in a usage message. Single-character
// names get prefixed with "-", longer ones with "--". The names are then
// separated by commas. For the special case of the non-option arguments
// def, the returned value is "<Arguments>".
func (self *OptionDef) formatOptionNames() string {
	names := []string{}
	for _, name := range strings.Split(self.names, " ") {
		switch {
		case len(name) == 1:
			names = append(names, "-"+name)
		case len(name) > 1:
			names = append(names, "--"+name)
		}
	}
	return strings.Join(names, ", ")
}

// Option returns a new OptionDef structure given one or more space-separated
// names (long and/or short), the target action, and a help string. If
// the help string starts with a prefix of the form "=NAME; ", then that
// argument name is used for the parameter name in the help output after
// the option names are listed.
//
// The supported types of target are any of the following:
//   *string, *uint, *uint64, *int, *int64, *float64, *bool, *[]string
//   func(), func() error, func(string), func(string) error
// For most pointers, an attempt is made to convert the string parameter to the
// target type. If successful, the new value is stored in the target.  For the
// bool pointer, there is no parameter and the value is set to true.  For the
// []string pointer, the parameter is appended to the slice each time the
// option is parsed.  The function types specify custom actions with and
// without parameters, which may or may not return errors.
func Option(names string, target interface{}, help string) *OptionDef {
	return &OptionDef{names, target, help}
}

// Section returns a new OptionDef that is only used as a section header
// in the help output.
func Section(header string) *OptionDef {
	return &OptionDef{help: header}
}

// NewOptionSet returns a new option set, optionally containing all of the
// OptionDef structures in entires.
func NewOptionSet(entries ...*OptionDef) *OptionSet {
	defs := &OptionSet{index: map[string]*OptionDef{}}
	return defs.Add(entries...)
}

// Option is equivalent to calling Add(Option(names, target, help)) on this OptionSet.
// Returns self so that calls can be chained.
func (self *OptionSet) Option(names string, target interface{}, help string) *OptionSet {
	return self.Add(Option(names, target, help))
}

// ArgAction sets a custom target action for non-option arguments in this OptionSet.
// The requirements for target are the same as those for the Option call. Returns
// self so that calls can be chained.
func (self *OptionSet) ArgAction(target interface{}) *OptionSet {
	self.argAction = &OptionDef{target: target}
	if self.setupError == nil && !self.argAction.isTargetOk() {
		self.setupError = fmt.Errorf("Unsupported target type for argument action")
	}
	return self
}

// Section is equivalent to calling Add(Section(header)) on this OptionSet.
// Returns self so that calls can be chained.
func (self *OptionSet) Section(header string) *OptionSet {
	return self.Add(Section(header))
}

// Add a number of OptionDef entries to this option set. The target of each
// entry is checked for validity, and the names are checked for redundant
// definitions within this option set. If an error is detected, it is saved for
// reporting later when ParseArgs is called.  The return value is self so that
// calls to this method can be chained together.
func (self *OptionSet) Add(entries ...*OptionDef) *OptionSet {
	// process each entry
	for _, entry := range entries {
		// add to in-order list
		self.list = append(self.list, entry)

		// check that target has a supported type
		if !entry.isTargetOk() {
			if self.setupError == nil {
				self.setupError = fmt.Errorf("Unsupported target type for option '%s'", entry.formatOptionNames())
			}
			return self
		}

		// process each name
		for _, name := range strings.Split(entry.names, " ") {
			if name != "" {
				// check for redundant name definition
				if self.index[name] != nil {
					if self.setupError == nil {
						self.setupError = fmt.Errorf("Option name '%s' defined more than once", name)
					}
					return self
				}
				self.index[name] = entry
			}
		}
	}
	return self
}

// FormatOptionsHelp creates a list of lines of help output from the list of
// OptionDef structures.  Generally, each line consists of the option
// names followed by the help text, with the help text aligned in its own
// column. If the help text starts with "=ARGNAME; ...", then the ARGNAME is
// removed from the help text and appended to the options name list. If the
// length of the option names and any ARGNAME exceeds the column width, then
// the help text is output on the following line. The help text for any section
// header entries are output as-is left justified.
func (self *OptionSet) FormatOptionsHelp() []string {
	out := []string{}
	const padding = 20 // width of left column
	for _, def := range self.list {
		if def.isSectionHeader() {
			// Section separator comment
			out = append(out, def.help)
		} else {
			// Look for "=ARGNAME; help text", pull out ARGNAME if found
			help := def.help
			semi := strings.IndexByte(help, ';')
			valName := ""
			if semi > 0 && strings.HasPrefix(help, "=") {
				valName = help[:semi]
				help = strings.TrimLeft(help[semi+1:], " ")
			}

			// Format the option names followed by any ARGNAME
			leftText := fmt.Sprintf("%-*s", padding, "  "+def.formatOptionNames()+valName)
			if strings.HasSuffix(leftText, " ") {
				// Fits within the left column, add the help text
				out = append(out, leftText+help)
			} else {
				// Doesn't fit, output on separate lines
				out = append(out, leftText)
				out = append(out, strings.Repeat(" ", padding)+help)
			}
		}
	}
	return out
}

// Set the target in the OptionDef with the given value. If the target is a
// setter function, call it. Otherwise, in most cases convert the string to the
// type of the target and set it. For the case of bool, the value is ignored
// and the target is set to true. In the case of a string list, append the
// value to the list. Returns an error if a conversion fails or the
// setter function returns an error.
func (self *OptionDef) set(value string) error {
	var err error
	var i int64
	var u uint64
	var f float64

	switch target := self.target.(type) {
	// setter that takes a parameter and never has errors
	case func(string):
		target(value)
	// setter that takes no parameter and never has errors
	case func():
		target()
	// setter that takes a parameter and may have errors
	case func(string) error:
		err = target(value)
	// setter that takes no parameter and may have errors
	case func() error:
		err = target()
	// string target: no conversion
	case *string:
		*target = value
	// numeric targets: attempt to convert to number
	case *uint:
		u, err = strconv.ParseUint(value, 0, 0)
		if err == nil {
			*target = uint(u)
		}
	case *uint64:
		u, err = strconv.ParseUint(value, 0, 64)
		if err == nil {
			*target = u
		}
	case *int:
		i, err = strconv.ParseInt(value, 0, 0)
		if err == nil {
			*target = int(i)
		}
	case *int64:
		i, err = strconv.ParseInt(value, 0, 64)
		if err == nil {
			*target = i
		}
	case *float64:
		f, err = strconv.ParseFloat(value, 64)
		if err == nil {
			*target = f
		}
	// bool target: set it to true
	case *bool:
		*target = true
	// string slice target: append to slice
	case *[]string:
		*target = append(*target, value)
	default:
		err = fmt.Errorf("Unsupported type given as target to to ParseArgs for option '%s'", self.formatOptionNames())
	}
	return err
}

// Search the given list of OptionDef structures to find one with a name
// matching the given name.  If any of an option's names matches name, then
// return a pointer to that option. If the entry kind is for the non-option
// argument handler or for the undefined option handler, then name is ignored.
// If no matching OptionDef is found, return nil.
func (self *OptionSet) lookupDef(name string) *OptionDef {
	return self.index[name]
}

// ParseArgs parses the given command line arguments in args according to these
// option definitions. If args is nil, the values in os.Args[1:] are parsed
// instead. For each option found in the arguments, the specified action is
// taken (such as setting a variable). Any remaining non-option arguments are
// returned (unless a custom argument handler was set with ArgAction). If an
// error is encountered, parsing stops and a non-nil error is also returned. If
// an error is returned, only some of the side effects may have been performed,
// and the returned argument list may be incomplete.  A copy of the returned
// non-option argument list is also stored in the global variable Args.
func (self *OptionSet) ParseArgs(args []string) ([]string, error) {
	// default to args from os if nil
	if args == nil {
		args = os.Args[1:]
	}

	// If there was an error detected during setup, report it now and quit
	if self.setupError != nil {
		OnError(self, self.setupError)
		return nil, self.setupError
	}

	var err error
	argsOut := []string{}
	moreShorts := ""    // for a short option, any chars found after the first
	terminated := false // the "--" terminator has been encountered
	i := 0
argLoop:
	// parse each argument
	for i < len(args) {
		var parameter string // if an option has a parameter, its value
		var name string      // the name of this option
		var arg string       // the current argument
		var def *OptionDef   // the relevant option definition for this arg, if any

		if moreShorts != "" {
			// we have more short options that were concatenated with previous short option; use them
			arg = moreShorts
			moreShorts = ""
		} else {
			// normal case: use next command line arg
			arg = args[i]
		}

		// take action based on dashes
		switch {
		case !terminated && arg == "--":
			// end of options marker
			terminated = true
			continue argLoop
		case !terminated && strings.HasPrefix(arg, "--"):
			// long option name; look for a '=' delimiter
			split := strings.Index(arg, "=")
			if split >= 0 {
				// has an '=' and parameter, split name from param
				name = arg[2:split]
				parameter = arg[split:]
			} else {
				// no '=', just get name
				name = arg[2:]
			}
			def = self.lookupDef(name)
		case !terminated && len(arg) > 1 && strings.HasPrefix(arg, "-"):
			// short option, any parameter or more shorts are after 1-character name
			parameter = arg[2:]
			name = arg[1:2]
			def = self.lookupDef(name)
		default:
			// non-option argument (includes "-")
			if self.argAction == nil {
				// Normal case; add arg to arguments list and go on
				i++
				argsOut = append(argsOut, arg)
				continue argLoop
			} else {
				// Custom non-option action found, handle below
				def = self.argAction
				parameter = arg
				arg = "<Argument>"
			}
		}

		if def == nil {
			// no definition found, check if automatic help should be shown
			if AutoHelp && (name == "h" || name == "help") &&
				self.lookupDef("h") == nil && self.lookupDef("help") == nil {
				Usage(self)
				os.Exit(0)
			}
			// report not found error
			err = fmt.Errorf("Unknown option '%s'", arg)
			OnError(self, err)
			break argLoop
		}

		// option definition was found; process it
		if def.takesParameter() {
			// option has a parameter
			if parameter == "" {
				// parameter was not concatenated with option, get the next command line arg as parameter
				if i >= len(args)-1 {
					err = fmt.Errorf("Expected a parameter after option '%s'", arg)
					OnError(self, err)
					break argLoop
				}
				i++
				parameter = args[i]
			} else if strings.HasPrefix(parameter, "=") {
				// parameter was concatenated with option; strip any '=' delimiter
				parameter = parameter[1:]
			}
			// use the parameter to perform the specified action
			err = def.set(parameter)
		} else {
			// option has no parameter
			if parameter != "" {
				// any extra chars must be more short options; save them for the next iteration
				moreShorts = "-" + parameter
			}
			// perform the specified action
			err = def.set("")
		}
		// check for an error with the action
		if err != nil {
			err = fmt.Errorf("Error with command line option '%s': %v", arg, err)
			OnError(self, err)
			break argLoop
		}
		if moreShorts == "" {
			// go on to next argument unless we had extra shorts concatenated with this option
			i++
		}
	}
	// copy output list to Args
	Args = append([]string{}, argsOut...)
	return argsOut, err
}
