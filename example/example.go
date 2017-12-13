package main

import (
	"fmt"

	"github.com/mikeynap/config"
)

type Conf struct {
	Addr     string   `desc:"address to listen on" def:"http://0.0.0.0:9999"`
	Ports    []string `desc:"Comma Seperated list of ... ports" def:"21,23,999"`
	Required string   `required:"true"`
	Log      LogConf
	PathMap  string `mapstructure:"path_map"`
}
type LogConf struct {
	Level string `desc:"error,info,debug" def:"info" required:"true"`
	FD    int    `desc:"unix File Descriptor number"`
}

func main() {
	cfg := Conf{}
	c := config.New("configTest", "A Thingy To Run Commands", &cfg)

	conf, err := c.Execute()
	if err != nil {
		fmt.Println(err)
		return
	}
	c.SetArgs([]string{"--required", "true", "--log-level", "error"})
	conf, err = c.Execute()
	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Printf("%+v\n", conf)

}
