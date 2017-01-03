package miniflags

import (
	"flag"
	"fmt"
	"os"
	"reflect"
	"strings"
	"testing"
)

func TestMain(m *testing.M) {
	flag.Parse()
	// Turn off default error handling
	OnError = testOnError
	os.Exit(m.Run())
}

// no-op error handler prevents exiting process
func testOnError(*OptionSet, ...interface{}) {
}

// checkValErr function checks that both value and error return from a function
// are as expected.  If errPrefix is not empty, check that the error returned
// from the function is not nil and has a message starting with the prefix.
// Otherwise, make sure the error is nil and do a deep comparison of the
// expected and actual returned values. Returns the test error message to log
// on failure, empty string on success.
func checkValErr(t *testing.T, want, got interface{}, errPrefix string, gotErr error) string {
	switch {
	case errPrefix == "" && gotErr != nil:
		return fmt.Sprintf("Got unexpected error '%v'", gotErr)

	case errPrefix != "" && gotErr == nil:
		return fmt.Sprintf("Expected error starting with '%s', got no error", errPrefix)

	case errPrefix != "" && !strings.HasPrefix(gotErr.Error(), errPrefix):
		return fmt.Sprintf("Expected error starting with '%s', got error '%v'", errPrefix, gotErr)

	case !reflect.DeepEqual(want, got):
		return fmt.Sprintf("Got '%v', expected '%v'", got, want)

	default:
		return ""
	}
}

func Test_IncDecOption(t *testing.T) {
	n := 0
	tests := []struct {
		input []string
		want  int
	}{
		{[]string{"-i", "-x"}, 1},
		{[]string{"-x", "-d"}, -1},
		{[]string{"-ii", "-x", "-i"}, 3},
		{[]string{"-iidid"}, 1},
	}
	oSet := NewOptionSet().
		Option("i", IncOption(&n), "").
		Option("d", DecOption(&n), "").
		Option("x", func() {}, "")

	for _, test := range tests {
		n = 0
		_, err := oSet.ParseArgs(test.input)
		if m := checkValErr(t, test.want, n, "", err); m != "" {
			t.Error(m)
		}
	}
}

func Test_FlagResetOption(t *testing.T) {
	b := true
	_, err := NewOptionSet().
		Option("r", FlagResetOption(&b), "").
		ParseArgs([]string{"-r"})

	if m := checkValErr(t, false, b, "", err); m != "" {
		t.Error(m)
	}
}

func Test_AlternativesOption(t *testing.T) {
	s := ""
	tests := []struct {
		input     []string
		want      string
		errPrefix string
	}{
		{[]string{"-s", "foo", "-x"}, "foo", ""},
		{[]string{"-s", "bar", "-x"}, "bar", ""},
		{[]string{"-s", "baz", "-x"}, "", "Error with command line option '-s': Invalid parameter value 'baz'"},
	}
	oSet := NewOptionSet().
		Option("s", AlternativesOption(&s, []string{"foo", "bar"}), "").
		Option("x", func() {}, "")

	for _, test := range tests {
		s = ""
		_, err := oSet.ParseArgs(test.input)
		if m := checkValErr(t, test.want, s, test.errPrefix, err); m != "" {
			t.Error(m)
		}
	}
}

func Test_OptionDef_takesParameter(t *testing.T) {
	var b bool
	var i int
	var tests = []struct {
		input *OptionDef
		want  bool
	}{
		{Option("x", &b, ""), false},
		{Option("x", &i, ""), true},
		{Option("x", func() {}, ""), false},
		{Option("x", func(s string) {}, ""), true},
	}
	for _, test := range tests {
		got := test.input.takesParameter()
		if m := checkValErr(t, test.want, got, "", nil); m != "" {
			t.Error(m)
		}
	}
}

func Test_isTargetOk(t *testing.T) {
	var i int
	var s string
	var a []string
	var tests = []struct {
		input *OptionDef
		want  bool
	}{
		{&OptionDef{"", nil, ""}, true},
		{&OptionDef{"a", nil, ""}, false},
		{&OptionDef{"a", 3, ""}, false},
		{&OptionDef{"a", &i, ""}, true},
		{&OptionDef{"a", &s, ""}, true},
		{&OptionDef{"a", &a, ""}, true},
		{&OptionDef{"a", func() {}, ""}, true},
		{&OptionDef{"a", func(int) {}, ""}, false},
	}
	for _, test := range tests {
		got := test.input.isTargetOk()
		if m := checkValErr(t, test.want, got, "", nil); m != "" {
			t.Error(m)
		}
	}
}

func Test_OptionDef_formatOptionNames(t *testing.T) {
	var tests = []struct {
		input *OptionDef
		want  string
	}{
		{Option("", nil, ""), ""},
		{Option("o", nil, ""), "-o"},
		{Option("opt", nil, ""), "--opt"},
		{Option("o opt", nil, ""), "-o, --opt"},
		{Option("  opt o p  ", nil, ""), "--opt, -o, -p"},
		{Section("foo"), ""},
	}
	for _, test := range tests {
		got := test.input.formatOptionNames()
		if m := checkValErr(t, test.want, got, "", nil); m != "" {
			t.Error(m)
		}
	}
}

func Test_OptionSet_ArgAction(t *testing.T) {
	var val string
	var tests = []struct {
		input     interface{}
		want      string
		errPrefix string
	}{
		{func(arg string) { val += arg }, "foobar", ""},
		{"BAD TARGET", "", "Unsupported target type for argument action"},
	}
	for _, test := range tests {
		val = ""
		args, err := NewOptionSet().
			ArgAction(test.input).
			Option("x", func() {}, "").
			ParseArgs([]string{"-x", "foo", "-x", "bar"})
		if m := checkValErr(t, test.want, val, test.errPrefix, err); m != "" {
			t.Error(m)
		}
		if m := checkValErr(t, 0, len(args), "", nil); m != "" {
			t.Error(m)
		}
	}
}

func Test_OptionSet_Add(t *testing.T) {
	e1 := Option("x", func() {}, "")
	e2 := Option(" y yyy ", func() {}, "")
	e3 := Section("foo")
	e4 := Option("z", "BAD TARGET", "")

	var tests = []struct {
		input     []*OptionDef // list of options to add
		names     []string     // names to check against index
		lookups   []*OptionDef // entries that should be found in index for each name
		errPrefix string       // expected text of setupError, if any
	}{
		{
			// one single name option, plus section entry
			[]*OptionDef{e1, e3},
			[]string{"x", "NA", ""},
			[]*OptionDef{e1, nil, nil},
			"",
		},
		{
			// single and dual-name options
			[]*OptionDef{e1, e2},
			[]string{"x", "y", "yyy", "NA"},
			[]*OptionDef{e1, e2, e2, nil},
			""},
		// redundant name error
		{[]*OptionDef{e1, e1}, nil, nil, "Option name 'x' defined more than once"},
		// bad target type in entry
		{[]*OptionDef{e1, e4}, nil, nil, "Unsupported target type"},
	}
	for _, test := range tests {
		opSet := NewOptionSet()
		opSet.Add(test.input...)
		// check for any expected setupError
		if m := checkValErr(t, nil, nil, test.errPrefix, opSet.setupError); m != "" {
			t.Error(m)
		}
		if test.errPrefix == "" {
			// list in OptionSet should match the input entries
			if m := checkValErr(t, test.input, opSet.list, "", nil); m != "" {
				t.Error(m)
			}
			// make sure that index contains expected entries
			for i, name := range test.names {
				got := opSet.lookupDef(name)
				if m := checkValErr(t, test.lookups[i], got, "", nil); m != "" {
					t.Error(m)
				}
			}
		}
	}
}

func Test_formatOptionsHelp(t *testing.T) {
	want := `
  -x                help for x
  -y, --yyy=YYY     help for y
Section:
  -z, --zzzzz1, --zzzzz2=ZZZ
                    help for z`

	lines := NewOptionSet().
		Option("x", func() {}, "help for x").
		Option(" y yyy ", func() {}, "=YYY; help for y").
		Section("Section:").
		Option(" z zzzzz1 zzzzz2 ", func() {}, "=ZZZ; help for z").
		ArgAction(func() {}).
		FormatOptionsHelp()

	if m := checkValErr(t, strings.Trim(want, "\n"), strings.Join(lines, "\n"), "", nil); m != "" {
		t.Error(m)
	}
}

func Test_OptionDef_set(t *testing.T) {
	var (
		s   string
		u   uint
		i   int
		i64 int64
		u64 uint64
		f   float64
		b   bool
		a   []string
	)
	var tests = []struct {
		def       *OptionDef
		arg       string
		checker   func() bool
		errPrefix string
	}{
		{Option("x", &s, ""), "foo", func() bool { return s == "foo" }, ""},
		{Option("x", &u, ""), "3", func() bool { return u == 3 }, ""},
		{Option("x", &u, ""), "bad", func() bool { return true }, "strconv.ParseUint"},
		{Option("x", &i, ""), "3", func() bool { return i == 3 }, ""},
		{Option("x", &i, ""), "bad", func() bool { return true }, "strconv.ParseInt"},
		{Option("x", &u64, ""), "3", func() bool { return u64 == 3 }, ""},
		{Option("x", &u64, ""), "bad", func() bool { return true }, "strconv.ParseUint"},
		{Option("x", &i64, ""), "3", func() bool { return i64 == 3 }, ""},
		{Option("x", &i64, ""), "bad", func() bool { return true }, "strconv.ParseInt"},
		{Option("x", &f, ""), "3", func() bool { return f == 3 }, ""},
		{Option("x", &f, ""), "bad", func() bool { return true }, "strconv.ParseFloat"},
		{Option("x", &b, ""), "", func() bool { return b }, ""},
		{Option("x", &a, ""), "foo", func() bool { return len(a) == 1 && a[0] == "foo" }, ""},
		{Option("x", 3, ""), "foo", func() bool { return true }, "Unsupported type"},
	}
	for _, test := range tests {
		s = ""
		u = 0
		i = 0
		i64 = 0
		u64 = 0
		f = 0
		b = false
		a = nil
		err := test.def.set(test.arg)
		if m := checkValErr(t, true, test.checker(), test.errPrefix, err); m != "" {
			t.Error(m)
		}
	}
}

func Test_OptionSet_ParseArgs(t *testing.T) {
	var i int
	var s string
	var b bool
	var tests = []struct {
		input     []string
		wantI     int
		wantS     string
		wantB     bool
		wantArgs  []string
		errPrefix string
	}{
		{[]string{}, 0, "", false, []string{}, ""},
		{[]string{"a", "-i", "3"}, 3, "", false, []string{"a"}, ""},
		{[]string{"a", "-sfoo", "-bi3", "--double"}, 6, "foo", true, []string{"a"}, ""},
		{[]string{"a", "-s", "foo", "-bi", "3"}, 3, "foo", true, []string{"a"}, ""},
		{[]string{"a", "-s=foo", "-bi=3"}, 3, "foo", true, []string{"a"}, ""},
		{[]string{"--str=foo", "a", "--double", "-bi=3"}, 3, "foo", true, []string{"a"}, ""},
		{[]string{"--str=foo", "--double", "-bi=3"}, 3, "foo", true, []string{}, ""},
		{[]string{"--str=foo", "a", "--integer=3", "--double", "--double", "b"}, 12, "foo", false, []string{"a", "b"}, ""},
		{[]string{"--str=foo", "--integer=bad"}, 0, "foo", false, []string{}, "Error with command line option '--integer=bad': strconv.ParseInt"},
		{[]string{"--str=foo", "--integer"}, 0, "foo", false, []string{}, "Expected a parameter after option '--integer"},
		{[]string{"--str=foo", "-sbar"}, 0, "bar", false, []string{}, ""},
	}

	for _, test := range tests {
		i = 0
		s = ""
		b = false
		gotArgs, err := NewOptionSet().
			Option("i integer", &i, "").
			Option("  s str", &s, "").
			Option("b    ", &b, "").
			Option("double", func() { i *= 2 }, "").
			ParseArgs(test.input)
		if m := checkValErr(t, test.wantArgs, gotArgs, test.errPrefix, err); m != "" {
			t.Error(m)
		}
		if m := checkValErr(t, test.wantI, i, "", nil); m != "" {
			t.Error(m)
		}
		if m := checkValErr(t, test.wantS, s, "", nil); m != "" {
			t.Error(m)
		}
		if m := checkValErr(t, test.wantB, b, "", nil); m != "" {
			t.Error(m)
		}
	}
}
