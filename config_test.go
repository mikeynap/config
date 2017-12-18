package config

import (
	"fmt"
	"io/ioutil"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type TestStruct struct {
	Int     int
	Bool    bool
	String  string `required:"true"`
	Dstring string `default:"string"`
	Sub     TestSubStruct
}

type TestSubStruct struct {
	Int     int
	Bool    bool
	String  string `required:"true"`
	Dstring string `default:"substring"`
	SubSub  TestSubSubStruct
}

type TestSubSubStruct struct {
	Rint int `required:"true"`
	Dint int `default:"1"`
}

type TestArgs struct {
	args         []string
	env          []string
	conf         string
	shouldPass   bool
	expectedRes  *TestStruct
	expectedRes2 *TestStruct2
}

type TestStruct2 struct {
	Float  float64
	Floatr float64 `required:"true"`
	Floatd float32 `default:"2.0"`
	Slice  []string
	Slicer []string `required:"true"`
	Sliced []string `default:"7,8,9"`
}

var testArgs = []TestArgs{
	{
		[]string{},
		nil,
		"",
		false,
		nil,
		nil,
	},
	{
		[]string{"--string", "string"},
		nil,
		"",
		false,
		nil,
		nil,
	},
	{ //2
		[]string{"--sub-string", "string"},
		nil,
		"",
		false,
		nil,
		nil,
	},
	{
		[]string{"--string", "string", "--sub-int", "9"},
		nil,
		"",
		false,
		nil,
		nil,
	},
	{
		nil,
		nil,
		`{"string": "string"}`,
		false,
		nil,
		nil,
	},
	{
		nil,
		nil,
		`{"sub": {"string": "string"}}`,
		false,
		nil,
		nil,
	},
	{
		[]string{"--sub-string", "substring", "--sub-sub-sub-rint", "2"},
		[]string{"TEST_INT=9"},
		`{"string": "string"}`,
		true,
		withStrings(&TestStruct{
			Int: 9,
		}),
		nil,
	},
	{
		nil,
		nil,
		`{"string": "stringg", "sub": {"string": "string", "subsub": {"rint": 2}}}`,
		true,
		nil,
		nil,
	},
	{
		[]string{"--sub-string", "substring", "--sub-sub-sub-rint", "2"},
		[]string{"TEST_INT=9"},
		`{"string": "string"}`,
		true,
		withStrings(&TestStruct{
			Int: 9,
		}),
		nil,
	},
	{
		withStringsSlice([]string{"--int", "9", "--sub-bool", "true"}),
		nil,
		"",
		true,
		withStrings(&TestStruct{
			Int: 9,
			Sub: TestSubStruct{
				Bool: true,
			},
		}),
		nil,
	},
	{
		withStringsSlice([]string{"--dstring", "blahhh"}),
		[]string{"TEST_SUB_INT=8", "TEST_SUB_DSTRING=subblahhh"},
		"",
		true,
		withStrings(&TestStruct{
			Dstring: "blahhh",
			Sub: TestSubStruct{
				Dstring: "subblahhh",
				Int:     8,
			},
		}),
		nil,
	},
	// Test having a different default in JUST the config.
	{
		nil,
		nil,
		`
    {
        "Int": 9, "Bool": true, "String": "string", "Dstring": "blahhhh",
        "sub":{
          "Int": 8, "String": "substring", "Dstring": "subblahhhh",
					"subsub": {"rint": 5, "dint": 8}
          }
    }`,
		true,
		withStrings(&TestStruct{
			Int:     9,
			Bool:    true,
			Dstring: "blahhhh",
			Sub: TestSubStruct{
				Bool:    false,
				Int:     8,
				String:  "substring",
				Dstring: "subblahhhh",
				SubSub: TestSubSubStruct{
					Rint: 5,
					Dint: 8,
				},
			},
		}),
		nil,
	},
	{
		[]string{"--string", "poop!", "--sub-dstring", "poop2!", "--sub-sub-sub-rint", "2"},
		[]string{"TEST_SUB_STRING=asdf", "TEST_SUB_SUB_SUB_DINT=3"},
		`{"string": "string", "sub":{"subsub": {"dint": 9}}}`,
		true,
		&TestStruct{
			String:  "poop!",
			Dstring: "string",
			Sub: TestSubStruct{
				String:  "asdf",
				Dstring: "poop2!",
				SubSub: TestSubSubStruct{
					Rint: 2,
					Dint: 3,
				},
			},
		},
		nil,
	},
	{
		[]string{"--string", "poop!", "--sub-dstring", "poop2!"},
		[]string{"TEST_SUB_STRING=asdf", "TEST_SUB_SUB_SUB_RINT=2"},
		`{"string": "string", "sub": {"string": "string2", "Dstring": "string2"}}`,
		true,
		&TestStruct{
			String:  "poop!",
			Dstring: "string",
			Sub: TestSubStruct{
				String:  "asdf",
				Dstring: "poop2!",
				SubSub: TestSubSubStruct{
					Rint: 2,
					Dint: 1,
				},
			},
		},
		nil,
	},
	{
		nil,
		nil,
		`{"floatr": 1.0, "slicer": [4,5,6]}`,
		true,
		nil,
		&TestStruct2{
			0.0,
			1.0,
			2.0,
			[]string{},
			[]string{"4", "5", "6"},
			[]string{"7", "8", "9"},
		},
	},
	{
		nil,
		nil,
		`{"floatr": 1.0, "slicer": []}`,
		false,
		nil,
		&TestStruct2{
			0.0,
			1.0,
			2.0,
			[]string{},
			[]string{},
			[]string{"7", "8", "9"},
		},
	},
	{
		nil,
		nil,
		`{"floatr": 1.0, "slice":[1,2,3]`,
		false,
		nil,
		nil,
	},
	{
		nil,
		nil,
		`{"floatr": 1.0, "floatd": 2.0, "slice":[1,2,3], "slicer": [4,5,6], "sliced":[7,8,9]}`,
		true,
		nil,
		&TestStruct2{
			0.0,
			1.0,
			2.0,
			[]string{"1", "2", "3"},
			[]string{"4", "5", "6"},
			[]string{"7", "8", "9"},
		},
	},
}

func TestConfig(t *testing.T) {
	for ti, tc := range testArgs {
		cmd := &cobra.Command{
			Use:           "test",
			Long:          "test desc",
			Run:           func(cmd *cobra.Command, args []string) {},
			SilenceUsage:  true,
			SilenceErrors: true,
		}
		var cfg TestStruct
		var cfg2 TestStruct2
		os.Args = append([]string{os.Args[0]}, tc.args...)
		for _, v := range tc.env {
			vars := strings.Split(v, "=")
			os.Setenv(vars[0], vars[1])
		}
		if len(tc.conf) > 0 {
			f, _ := ioutil.TempFile("", string(ti))
			newName := f.Name() + ".json"
			defer os.Remove(newName)
			f.Write([]byte(tc.conf))
			f.Close()
			os.Rename(f.Name(), newName)
			os.Args = append(os.Args, "--config", newName)
		}
		var c *Config
		if tc.expectedRes2 != nil {
			c = NewWithCommand(cmd, &cfg2)
		} else {
			c = NewWithCommand(cmd, &cfg)
		}

		if _, err := c.Execute(); err == nil && !tc.shouldPass {
			t.Errorf("Test %d) Should have errored.", ti)
		} else if err != nil && tc.shouldPass {
			t.Errorf("Test %d) Shouldn't have errored: %v", ti, err)
		}
		os.Args = os.Args[:1]
		var res interface{}
		var dacfg interface{}
		if tc.expectedRes != nil {
			res = *tc.expectedRes
			dacfg = cfg
		} else if tc.expectedRes2 != nil {
			res = *tc.expectedRes2
			dacfg = cfg2
		}
		if res != nil {
			if !reflect.DeepEqual(dacfg, res) {
				t.Errorf("Test %d) Structs should be equal.\nGot       %+v\n expected %+v", ti, dacfg, res)
			}
		}
		// unset env.
		for _, v := range tc.env {
			vars := strings.Split(v, "=")
			os.Unsetenv(vars[0])
		}
		c.Reset()
		viper.Reset()
	}
}

type noFlags struct {
	Val  int
	Val2 int
}

func TestEachSubField(t *testing.T) {
	tests := []struct {
		testStruct  interface{}
		shouldPanic bool
		shouldError bool
		f           func(reflect.Value, string, []string) error
	}{
		// Struct must be settable
		{
			testStruct:  struct{}{},
			shouldPanic: true,
			shouldError: true,
			f:           func(reflect.Value, string, []string) error { return nil },
		},
		// Must be a struct of structs
		{
			testStruct:  1,
			shouldPanic: true,
			shouldError: true,
			f:           func(reflect.Value, string, []string) error { return nil },
		},
		{
			testStruct:  &struct{}{},
			shouldPanic: false,
			shouldError: false,
			f:           func(reflect.Value, string, []string) error { return nil },
		},
		{
			testStruct: &struct {
				Test      int
				TestSlice []string
			}{1, []string{"Hi", "Mom"}},
			shouldPanic: false,
			shouldError: false,
			f:           func(reflect.Value, string, []string) error { return nil },
		},
		{
			testStruct: &struct {
				Test struct{ Val int }
			}{
				Test: struct{ Val int }{1},
			},
			shouldPanic: false,
			shouldError: false,
			f: func(p reflect.Value, f string, bc []string) error {
				if strings.Join(bc, "") != "Test" || f != "Val" {
					t.Errorf("Expected single call for Test.Val. Got %s.%s", strings.Join(bc, ""), f)
					return fmt.Errorf("Expected single call for Test.Val. Got %s.%s", strings.Join(bc, ""), f)
				}
				return nil
			},
		},
		{
			testStruct: &struct {
				Test struct {
					Val   int `flag:"false"`
					Test2 struct {
						Val2     int
						ValSlice []string
					}
					unexported string
					SkipMe     string `flag:"false"`
				}
			}{
				Test: struct {
					Val   int `flag:"false"`
					Test2 struct {
						Val2     int
						ValSlice []string
					}
					unexported string
					SkipMe     string `flag:"false"`
				}{1, struct {
					Val2     int
					ValSlice []string
				}{2, []string{"hi", "mom"}}, "hello, you failed!", "BadFlag"},
			},
			shouldPanic: false,
			shouldError: false,
			f: func(p reflect.Value, f string, bc []string) error {
				if f != "Val2" && f != "IgnoreThis" && f != "SkipMe" && f != "ValSlice" {
					t.Errorf("Expected single call for Test.Val. Got %s.%s", strings.Join(bc, ""), f)
					return fmt.Errorf("Expected single call for Test.Val. Got %s.%s", strings.Join(bc, ""), f)
				}
				if f == "SkipMe" || f == "Val" {
					t.Error("Should not have gotten to SkipMe, since flag:false")
				}
				return nil
			},
		},

		{
			testStruct: &struct {
				NoFlags        noFlags `flag:"false"`
				DontIgnoreThis string
			}{
				NoFlags: noFlags{
					1, 2},
			},
			shouldPanic: false,
			shouldError: false,
			f: func(p reflect.Value, f string, bc []string) error {
				if f == "Val1" || f == "Val2" {
					t.Error("should not have parsed NoFlags.Val1 or NoFlags.Val2 since `flags:'false'` is set.")
				}
				return nil
			},
		},
		{
			testStruct: &struct {
				Test struct{ Val int }
			}{
				Test: struct{ Val int }{1},
			},
			shouldPanic: false,
			shouldError: false,
			f:           func(reflect.Value, string, []string) error { return fmt.Errorf("error") },
		},
	}

	//func eachSubField(i interface{}, fn func(reflect.Value, string, []string))

	for i, test := range tests {
		assertPanic(i, t, test.testStruct, test.f, test.shouldPanic, test.shouldError)
	}
}

// Wrapper function to catch panics
func assertPanic(index int, t *testing.T, i interface{}, f func(reflect.Value, string, []string) error, shouldPanic bool, shouldError bool) {
	err := eachSubField(i, f)
	if err != nil {
		if !shouldError {
			t.Errorf("Test: %d Received Err : %v, expected none.", index, err)
		}
	} else {
		if shouldError {
			t.Errorf("Test %d No Err received when one was expected", index)
		}
	}
}

func TestFlagString(t *testing.T) {
	tests := []struct {
		parent   string
		field    string
		expected string
	}{
		{"Docker", "Foo", "docker-foo"},
		{"Foo", "Bar", "foo-bar"},
		{"FOO", "FooBAR", "foo-foo-bar"},
		{"FoOo", "FooBARBaZ", "fo-oo-foo-bar-ba-z"},
		{"FOO", "FOoBARBaZa", "foo-f-oo-bar-ba-za"},
		{"", "bar", "bar"},
		{"bar", "", "bar"},
		{"", "", ""},
	}

	for _, test := range tests {
		if flagString(test.parent, test.field) != test.expected {
			t.Errorf("Str %s%s: Expected %s, got %s", test.parent, test.field, test.expected, flagString(test.parent, test.field))
		}
	}
}

func TestEnvString(t *testing.T) {
	tests := []struct {
		parent   string
		field    string
		expected string
	}{
		{"Docker", "Foo", "DOCKER_FOO"},
		{"Foo", "Bar", "FOO_BAR"},
		{"FOO", "FooBAR", "FOO_FOO_BAR"},
		{"FoOo", "FooBARBaZ", "FO_OO_FOO_BAR_BA_Z"},
		{"FOO", "FOoBARBaZa", "FOO_F_OO_BAR_BA_ZA"},
		{"", "bar", "BAR"},
		{"bar", "", "BAR"},
		{"", "", ""},
	}

	for _, test := range tests {
		if envString(test.parent, test.field) != test.expected {
			t.Errorf("Expected %s, got %s", test.expected, envString(test.parent, test.field))
		}
	}
}

func TestCmdArgs(t *testing.T) {
	cobCmd := &cobra.Command{
		Use:           "test",
		Long:          "test desc",
		Run:           func(cmd *cobra.Command, args []string) {},
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	type s struct {
		Test  bool
		Test2 bool
	}
	cfg := s{}
	c := NewWithCommand(cobCmd, &cfg)
	c.SetArgs([]string{"--test", "true"})
	_, err := c.Execute()
	if err != nil {
		t.Error("Error Executing with cmdArgs: ", err)
	}
	if !cfg.Test {
		t.Error("test should be true.")
	}
}

func TestSplitCamel(t *testing.T) {
	cases := map[string]string{
		"Test":         "test",
		"TestThing":    "test-thing",
		"TestThingT":   "test-thing-t",
		"TestURLStuff": "test-url-stuff",
		"TestURL":      "test-url",
		"TestUrls":     "test-urls",
		"TesturLs":     "testur-ls",
		"TestURLs":     "test-urls",
		"TestURLS":     "test-urls",
		"aURLTest":     "a-url-test",
		"S":            "s",
		"s":            "s",
		"aS":           "a-s",
		"Sa":           "sa",
		"as":           "as",
		"lowerUpper":   "lower-upper",
		"FOO":          "foo",
		"foo":          "foo",
		"":             "",
		"tEsTiNg":      "t-es-ti-ng",
		"TeStInG":      "te-st-in-g",
		"TeStINg":      "te-st-ing",
	}
	for test, exp := range cases {
		res := splitCamel(test, '-')
		if strings.ToLower(res) != exp {
			t.Errorf("Splitting %s should have given %s, got %s", test, exp, res)
		}
	}
}

func withStrings(s *TestStruct) *TestStruct {
	if s.Dstring == "" {
		s.Dstring = "string"
	}
	if s.String == "" {
		s.String = "string"
	}
	if s.Sub.String == "" {
		s.Sub.String = "substring"
	}
	if s.Sub.Dstring == "" {
		s.Sub.Dstring = "substring"
	}
	if s.Sub.SubSub.Rint == 0 {
		s.Sub.SubSub.Rint = 2
	}
	if s.Sub.SubSub.Dint == 0 {
		s.Sub.SubSub.Dint = 1
	}
	return s
}

func withStringsSlice(s []string) []string {
	if !contains(s, "--string") {
		s = append(s, "--string", "string")
	}
	if !contains(s, "--sub-string") {
		s = append(s, "--sub-string", "substring")
	}

	if !contains(s, "--sub-sub-sub-rint") {
		s = append(s, "--sub-sub-sub-rint", "2")
	}
	return s
}
