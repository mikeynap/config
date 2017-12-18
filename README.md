# Config
Create command line/environment arguments from a go struct (with usage/help). Those arguments then get stored in said struct.

# Docs:
https://godoc.org/github.com/mikeynap/config

```
// desc tag is used for --help screen.
// desc/default/required are optional tags.

type Conf struct {
    Addr     string   `required:"true"`
    Ports    []string `desc:"some ports"`
    Log      LogConf
}
type LogConf struct {
	Level string `default:"info"`
}
func main() {
	cfg := Conf{}
	c, _ := config.New("conf", "A Thingy To Get Configs", &cfg)
	_, _ := c.Execute()
	fmt.Printf("%+v\n", conf)
}
```

```
export CONF_ADDR=127.0.0.1:1234
go run main.go --log-level error --config example/conf.yaml
```
#### Result:  
```
  Conf{
    Addr: 127.0.0.1:1234,
    Ports: [21,8080,9999]
    Log: LogConf{
      Level: error
    }
  }
```
