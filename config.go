package config

import (
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
}

// New creates a config parser using a provided cfg struct.
// name, desc are for the help window.
func New(name string, desc string, cfg interface{}) (*Config, error) {
	return NewWithCommand(
		&cobra.Command{
			Use:  name,
			Long: desc,
			Run:  func(cmd *cobra.Command, args []string) {},
		}, cfg)
}

// NewWithCommand creates the config parser using an existing cobra command.
func NewWithCommand(cmd *cobra.Command, cfg interface{}) (*Config, error) {
	c := &Config{
		Viper: viper.New(),
		Cmd:   cmd,
		cfg:   cfg,
	}
	cmd.PreRunE = func(cmd *cobra.Command, args []string) error {
		return checkRequiredFlags(cmd.Flags())
	}

	c.Cmd.PersistentFlags().String("config", "", "The configuration file")
	c.Viper.BindPFlag("config", c.Cmd.PersistentFlags().Lookup("config"))

	return c, c.setupEnvAndFlags(c.cfg)

}

// Execute runs the command (if provided) and populates the config struct.
func (c *Config) Execute() (interface{}, error) {
	c.Cmd.Flags().Visit(func(arg0 *pflag.Flag) {
		if arg0.Name == "help" {
			os.Exit(0)
		}
	})
	_, flags, _ := c.Cmd.Find(os.Args[1:])
	c.Cmd.ParseFlags(flags)

	configFile := c.Viper.GetString("config")
	if configFile != "" {
		if _, err := os.Stat(configFile); err == nil {
			c.Viper.SetConfigFile(configFile) // name of config file
			err := c.Viper.ReadInConfig()     // Find and read the config file
			if err != nil {                   // Handle errors reading the config file
				return nil, err
			}
			err = c.Viper.Unmarshal(c.cfg)
			if err != nil { // Handle errors reading the config file
				return nil, err
			}
		}
	}
	var err error
	if err = c.getCfg(c.cfg); err != nil {
		return c.cfg, err
	}

	err = c.Cmd.Execute()

	return c.cfg, err
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
				v := c.Viper.GetString(str)
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
			case reflect.Slice:
				v := c.Viper.GetStringSlice(str)
				if len(v) == 0 || len(v[0]) == 0 {
					return nil
				}
				subField.Set(reflect.ValueOf(v))
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
		_def := subField.Tag.Get("def")
		_, req := subField.Tag.Lookup("required")
		switch subField.Type.Kind() {
		case reflect.Bool:
			def := false
			if b, err := strconv.ParseBool(_def); err != nil {
				def = b
			}
			c.Cmd.PersistentFlags().Bool(flagStr, def, desc)
		case reflect.Int:
			var def int
			if b, err := strconv.ParseInt(_def, 10, 32); err != nil {
				def = int(b)
			}
			c.Cmd.PersistentFlags().Int(flagStr, def, desc)
		case reflect.Int64:
			var def int64
			if b, err := strconv.ParseInt(_def, 10, 64); err != nil {
				def = b
			}
			c.Cmd.PersistentFlags().Int64(flagStr, def, desc)
		case reflect.String:
			c.Cmd.PersistentFlags().String(flagStr, _def, desc)
		case reflect.Float64:
			var def float64
			if b, err := strconv.ParseFloat(_def, 64); err != nil {
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
		panic("eachSubField can only be called on a pointer-to-struct")
	}
	// Sanity check. Should be true if it is a pointer-to-struct
	if !t.Elem().CanSet() {
		panic("eachSubField can only be called on a settable struct of structs")
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

func checkRequiredFlags(flags *pflag.FlagSet) error {
	requiredError := false
	flagName := ""

	flags.VisitAll(func(flag *pflag.Flag) {
		requiredAnnotation := flag.Annotations[cobra.BashCompOneRequiredFlag]
		if len(requiredAnnotation) == 0 {
			return
		}

		flagRequired := requiredAnnotation[0] == "true"

		if flagRequired && !flag.Changed {
			requiredError = true
			flagName = flag.Name
		}
	})

	if requiredError {
		fmt.Println("Required flag `" + flagName + "` has not been set")
		os.Exit(0)
	}

	return nil
}
