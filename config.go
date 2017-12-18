package config

import (
	"errors"
	"fmt"
	"os"
	"reflect"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

type Config struct {
	Viper          *viper.Viper
	configLocation string
	cfg            interface{}
	Cmd            *cobra.Command
	parsed         bool
	Args           []string
}

/* New creates a config parser using a provided cfg struct.
   name, desc are for the help window.
   cfg can be decorated with the following tags:
	 	- `required:"true"`: must be specified and non-zero.
		   Most naturally used on strings, but can be used on numbers, but
			 CANNOT be used if zero is a possible value. (Not supported for bools)
		- `default:"val"`: if a value is not specified, replace with tag value.
		  The same zero caviat as above applies here as well.
		- `description:"this is the desc"`: description to use in help menu.
*/
func New(name string, desc string, cfg interface{}) *Config {
	return NewWithCommand(
		&cobra.Command{
			Use:  name,
			Long: desc,
			Run:  func(cmd *cobra.Command, args []string) {},
		}, cfg)
}

// NewWithCommand creates the config parser using an existing cobra command.
func NewWithCommand(cmd *cobra.Command, cfg interface{}) *Config {
	// To avoid panics vvv
	cmd.ResetFlags()
	cmd.ResetCommands()

	c := &Config{
		Viper: viper.New(),
		Cmd:   cmd,
		cfg:   cfg,
	}
	cmd.PreRunE = func(cmd *cobra.Command, args []string) error {
		return c.checkRequiredFlags(cmd.Flags())
	}
	c.Cmd.PersistentFlags().String("config", "", "The configuration file")
	c.Viper.BindPFlag("config", c.Cmd.PersistentFlags().Lookup("config"))

	return c
}

// SetArgs sets args to use instead of the default os.Args (useful for testing).
// This is used instead of cobraCommand.SetArgs()
func (c *Config) SetArgs(args []string) {
	c.Args = args
}

// do pre-execute parsing of config file.
func (c *Config) parse() (interface{}, error) {
	defer func() { c.parsed = true }()
	if c.parsed {
		c.Reset()
	}
	c.setupEnvAndFlags(c.cfg)
	c.Cmd.Flags().Visit(func(arg0 *pflag.Flag) {
		if arg0.Name == "help" {
			os.Exit(0)
		}
	})

	if len(os.Args) > 1 && len(c.Args) == 0 {
		c.Args = os.Args[1:]
	}
	_, flags, _ := c.Cmd.Find(c.Args)
	c.Cmd.ParseFlags(flags)
	configFile := c.Viper.GetString("config")
	if configFile != "" {
		_, err := os.Stat(configFile)
		if err != nil {
			return nil, err
		}
		c.Viper.SetConfigFile(configFile) // name of config file
		// Find and read the config file
		if err = c.Viper.ReadInConfig(); err != nil { // Handle errors reading the config file
			return nil, err
		}
		if err = c.Viper.Unmarshal(c.cfg); err != nil { // Handle errors reading the config file
			return nil, err
		}
	}
	var err error
	if err = c.getCfg(c.cfg); err != nil {
		return c.cfg, err
	}
	return c.cfg, nil
}

// Parse is an alias for Execute().
func (c *Config) Parse() (interface{}, error) {
	return c.Execute()
}

// Execute runs the command (if provided) and populates the config struct.
func (c *Config) Execute() (interface{}, error) {
	cfg, err := c.parse()
	if err != nil {
		return cfg, err
	}
	return c.cfg, c.Cmd.Execute()
}

// SilenceUsage will not print the help screen on error.
func (c *Config) SilenceUsage() {
	c.Cmd.SilenceUsage = true
}

// SilenceUsage will not print errors found during parse/load.
func (c *Config) SilenceErrors() {
	c.Cmd.SilenceErrors = true
}

// Reset flags/command to reuse.
func (c *Config) Reset() {
	c.Cmd.ResetCommands()
	c.Cmd.ResetFlags()
	c.Viper = viper.New()
	c.Args = nil
	c.Cmd.PersistentFlags().String("config", "", "The configuration file")
	c.Viper.BindPFlag("config", c.Cmd.PersistentFlags().Lookup("config"))
}

/* NOTE: Due to a bug in Viper, all boolean flags MUST DEFAULT TO FALSE.
   That is, all boolean flags should be to ENABLE features.
	 --use-db vs --dont-use-db.
*/

func (c *Config) getCfg(gCfg interface{}) error {
	return eachSubField(gCfg, func(parent reflect.Value, subFieldName string, crumbs []string) error {
		p := strings.Join(crumbs, "")
		envStr := envString(p, subFieldName)
		flagStr := flagString(p, subFieldName)

		// eachSubField only calls this function if  subFieldName exists
		// and can be set
		subField := parent.FieldByName(subFieldName)
		str := ""
		if v := c.Viper.Get(envStr); v != nil {
			str = envStr
		} else if c.Viper.Get(flagStr) != nil {
			str = flagStr
		}
		var v, ogV string
		lup := c.Cmd.PersistentFlags().Lookup(strings.ToLower(str))
		if lup != nil {
			v = lup.Value.String()
			ogV = v
		}
		subFieldAsString := fmt.Sprintf("%v", subField)
		// If the struct has a value filled in that wasn't provided
		// as a flag, then set it as the flag value.
		// This allows the required check to pass.
		if subField.Type().Kind() != reflect.Bool && lup != nil {
			if !isZero(subField) && isZeroStr(v) {
				v = subFieldAsString
				lup.Value.Set(v)
			}
		}

		// AHHHHHHHHHHHHHH. This line next line took forever.
		// Don't "reset" the default value if it's been specified differently.
		if lup != nil && lup.DefValue == ogV && ogV != "" && !isZeroStr(subFieldAsString) {
			return nil
		}

		if len(str) != 0 && subField.CanSet() {
			switch subField.Type().Kind() {
			case reflect.Bool:
				v := c.Viper.GetBool(str)
				subField.SetBool(v || subField.Bool()) // IsSet is broken with bools, see NOTE above ^^^
			case reflect.Int:
				v := c.Viper.GetInt(str)
				if v == 0 {
					return nil
				}
				subField.SetInt(int64(v))
			case reflect.Int64:
				v := c.Viper.GetInt64(str)
				if v == 0 {
					return nil
				}
				subField.SetInt(v)
			case reflect.String:
				v = c.Viper.GetString(str)
				if len(v) == 0 {
					return nil
				}
				subField.SetString(v)
			case reflect.Float64:
				v := c.Viper.GetFloat64(str)
				if v == 0 {
					return nil
				}
				subField.SetFloat(v)
			case reflect.Float32:
				v := c.Viper.GetFloat64(str)
				if v == 0 {
					return nil
				}
				subField.SetFloat(v)
			case reflect.Slice:
				v := c.Viper.GetStringSlice(str)
				if len(v) == 0 || len(v[0]) == 0 || v[0] == "[]" {
					return nil
				}
				subField.Set(reflect.Zero(reflect.TypeOf(v)))
			default:
				return fmt.Errorf("%s is unsupported by config @ %s.%s", subField.Type().String(), p, subFieldName)
			}
		}
		return nil
	})
}

// Process env var overrides for all values
func (c *Config) setupEnvAndFlags(gCfg interface{}) error {
	// Supports fetching value from env for all config of type: int, float64, bool, and string
	c.Viper.AutomaticEnv()
	c.Viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	c.Viper.SetEnvPrefix(c.Cmd.Name())
	return eachSubField(gCfg, func(parent reflect.Value, subFieldName string, crumbs []string) error {
		p := strings.Join(crumbs, "")
		envStr := envString(p, subFieldName)
		flagStr := flagString(p, subFieldName)
		c.Viper.BindEnv(envStr)

		subField, _ := parent.Type().FieldByName(subFieldName)

		desc := subField.Tag.Get("desc")
		if desc == "" {
			desc = subField.Tag.Get("description")
		}
		_def := subField.Tag.Get("def")
		if _def == "" {
			_def = subField.Tag.Get("default")
		}
		_, req := subField.Tag.Lookup("required")
		switch subField.Type.Kind() {
		case reflect.Bool:
			c.Cmd.PersistentFlags().Bool(flagStr, false, desc)
		case reflect.Int:
			var def int
			if b, err := strconv.ParseInt(_def, 10, 32); err == nil {
				def = int(b)
			}
			c.Cmd.PersistentFlags().Int(flagStr, def, desc)
		case reflect.Int64:
			var def int64
			if b, err := strconv.ParseInt(_def, 10, 64); err == nil {
				def = b
			}
			c.Cmd.PersistentFlags().Int64(flagStr, def, desc)
		case reflect.String:
			c.Cmd.PersistentFlags().String(flagStr, _def, desc)
		case reflect.Float32:
			var def float64
			if b, err := strconv.ParseFloat(_def, 32); err == nil {
				def = b
			}
			c.Cmd.PersistentFlags().Float64(flagStr, def, desc)
		case reflect.Float64:
			var def float64
			if b, err := strconv.ParseFloat(_def, 64); err == nil {
				def = b
			}
			c.Cmd.PersistentFlags().Float64(flagStr, def, desc)
		case reflect.Slice:
			def := strings.Split(_def, ",")
			if len(def[0]) == 0 {
				def = nil
			}
			if subField.Type.Elem().Kind() != reflect.String {
				return fmt.Errorf("%s is unsupported by config @ %s.%s", subField.Type.String(), p, subFieldName)
			}
			c.Cmd.PersistentFlags().StringSlice(flagStr, def, desc)
		default:
			return fmt.Errorf("%s is unsupported by config @ %s.%s", subField.Type.String(), p, subFieldName)
		}
		if req {
			c.Cmd.MarkPersistentFlagRequired(flagStr)
		}
		c.Viper.BindPFlag(flagStr, c.Cmd.PersistentFlags().Lookup(flagStr))
		return nil
	})

}

// eachSubField is used for a struct of structs (like GlobalConfig). fn is called
// with each field from each sub-struct of the parent. Fields are skipped if they
// are not settable, or unexported OR are marked with `flag:"false"`
func eachSubField(i interface{}, fn func(reflect.Value, string, []string) error, crumbs ...string) error {
	t := reflect.ValueOf(i)
	if t.Kind() != reflect.Ptr || t.Elem().Kind() != reflect.Struct {
		return errors.New("eachSubField can only be called on a pointer-to-struct")
	}
	// Sanity check. Should be true if it is a pointer-to-struct
	if !t.Elem().CanSet() {
		return errors.New("eachSubField can only be called on a settable struct of structs")
	}

	t = t.Elem()
	nf := t.NumField()
	for i := 0; i < nf; i++ {
		field := t.Field(i)
		sf := t.Type().Field(i)
		if sf.Tag.Get("flag") == "false" {
			continue
		}

		if field.Kind() == reflect.Struct && field.CanSet() {
			eachSubField(field.Addr().Interface(), fn, append(crumbs, sf.Name)...)
		} else if field.CanSet() {
			if err := fn(t, sf.Name, crumbs); err != nil {
				return err
			}
		}
	}
	return nil
}

func (c *Config) checkRequiredFlags(flags *pflag.FlagSet) error {
	requiredError := false
	flagName := ""

	flags.VisitAll(func(flag *pflag.Flag) {
		requiredAnnotation := flag.Annotations[cobra.BashCompOneRequiredFlag]
		if len(requiredAnnotation) == 0 {
			return
		}

		flagRequired := requiredAnnotation[0] == "true"
		val := c.Viper.Get(flag.Name)
		if flagRequired && (!flag.Changed && isZero(val)) {
			requiredError = true
			flagName = flag.Name
		}
	})

	if requiredError {
		return fmt.Errorf("Required flag `%s` has not been set", flagName)
	}

	return nil
}
