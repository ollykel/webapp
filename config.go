package webapp

import (
	"os"
	"fmt"
	"strings"
	"time"
	"encoding/json"
	"encoding/xml"
	// imported packages
	yaml "gopkg.in/yaml.v2"
)

type Config struct {
	Index string
	StaticDir string
	WaitSecs time.Duration
	Server ServerConfig//-- see server.go
	Database DatabaseConfig//-- see database.go
}

func (cfg *Config) String () string {
	output, _ := json.Marshal(cfg)
	return string(output)
}//-- end Config.String

type decoder interface {
	Decode (interface{}) error
}//-- end Decoder interface

func getDecoder (file *os.File, fileExt string) (decoder, error) {
	switch (fileExt) {
		case "json":
			return json.NewDecoder(file), nil
		case "xml":
			return xml.NewDecoder(file), nil
		case "yaml", "yml":
			return yaml.NewDecoder(file), nil
		default:
			return nil, fmt.Errorf(`Invalid file type "%s"`, fileExt)
	}//-- end switch	
}//-- end func getDecoder

func LoadConfig (filename string) (*Config, error) {
	file, err := os.Open(filename)
	if err != nil { return nil, err }
	defer file.Close()
	path := strings.Split(filename, ".")
	ext := path[len(path) - 1]
	dec, err := getDecoder(file, ext)
	if err != nil { return nil, err }
	config := &Config{}
	err = dec.Decode(config)
	if err != nil { return nil, err }
	return config, nil
}//-- end func LoadConfig

