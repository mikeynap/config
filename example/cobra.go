package main

import (
	"fmt"
	"os"

	"github.com/mikeynap/config"
	"github.com/spf13/cobra"
)

type LogConfig struct {
	Level string `desc:"error,info,debug" def:"info"`
	FD    int    `desc:"unix File Descriptor number"`
}

type Config struct {
	Addr     string `desc:"address to listen on" def:"http://0.0.0.0:9999"`
	Log      LogConfig
	Ports    []string `desc:"Comma Seperated list of ... ports" def:"21,23,999"`
	Required string   `required:"true"`
}

var Cmd = &cobra.Command{
	Use:   "configTest",
	Short: "Run configTest",
	Long:  "Config Test!",
	Run:   configRun,
}

var VersionCmd = &cobra.Command{
	Use:   "version",
	Short: "get version",
	Long:  "Gets the stupid version",
	Run:   versionRun,
}

func ConfigExit(err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "--- Config encountered an Error: ---\n")
		fmt.Fprintf(os.Stderr, "\t%v\n", err)
		os.Exit(1)
	}
}

func versionRun(cmd *cobra.Command, args []string) {
	fmt.Printf("v1.2.0")
}

func configRun(cmd *cobra.Command, args []string) {
	var globalErr error
	defer func() {
		ConfigExit(globalErr)
	}()
	// DO STUFF
	fmt.Printf("%+v\n", cfg)
}

var cfg Config

func main() {
	cfg = Config{}
	Cmd.AddCommand(VersionCmd)
	c := config.NewWithCommand(Cmd, &cfg)
	_, err := c.Execute() // cfgInterface === cfg
	if err != nil {
		fmt.Println(err)
		return
	}
}
