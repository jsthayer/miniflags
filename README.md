# miniflags
Simple, flexible command line parser for golang (alternative to the Go flag package).

## Description
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

## Example usage
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
