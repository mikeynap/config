package main

import (
	"fmt"

	"github.com/mikeynap/config"
)

type LogConf struct {
	Level string `desc:"error,info,debug" def:"info"`
	FD    int    `desc:"unix File Descriptor number"`
}

type Conf struct {
	Addr     string `desc:"address to listen on" def:"http://0.0.0.0:9999"`
	Log      LogConf
	Ports    []string `desc:"Comma Seperated list of ... ports" def:"21,23,999"`
	Required string   `required:"true"`
}

func main() {
	cfg := Conf{}
	c, err := config.New("configTest", "A Thingy To Run Commands", &cfg)
	if err != nil {
		fmt.Println(err)
		return
	}

	conf, err := c.Execute()
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Printf("%+v\n", conf)

}
