package config

import (
	"io"
	"os"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"

	_ "embed"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
)

type PortForwardConfiguration struct {
	Name      string   `json:"name"`
	Ports     []string `json:"ports"`

	Namespace string   `json:"namespace"`
	Resource  string   `json:"resource"`
}

type LogsConfiguration struct {
	Level string `json:"level"`
	Pretty bool   `json:"pretty"`
}

type Configuration struct {
	Forwards []PortForwardConfiguration `json:"forwards"`

	Logs    LogsConfiguration `json:"logs"`
}

//go:embed schema.cue
var schemaContent string

func ReadConfiguration(filename string) (Configuration, error) {
	var configuration Configuration

	ctx := cuecontext.New()

	schemalVal := ctx.CompileString(schemaContent, cue.Filename("schema.cue"))
	if schemalVal.Err() != nil {
		return configuration, schemalVal.Err()
	}

	buf, err := loadConfiguration(filename)
	if err != nil {
		return configuration, err
	}

	configVal := ctx.CompileBytes(buf, cue.Filename(filename))
	if configVal.Err() != nil {
		return configuration, configVal.Err()
	}
		
	unified := schemalVal.Unify(configVal)
	if unified.Err() != nil {
		return configuration, unified.Err()
	}

	err = unified.Decode(&configuration)
	if err != nil {
		return configuration, err
	}

	return validateConfiguration(configuration)
}

func loadConfiguration(finename string) ([]byte, error) {
	f, err := os.Open(filepath.Clean(finename))
	if err != nil {
		return nil, err
	}
	defer f.Close()

	return io.ReadAll(f)
}

func validateConfiguration(cfg Configuration) (Configuration, error) {
	for _, pf := range cfg.Forwards {
		if pf.Name == "" {
			return cfg, fmt.Errorf("each port forward must have a name")
		}

		for _, port := range pf.Ports {
			parts := strings.SplitN(port, ":", 2) 
			
			if len(parts) >= 1 {
				p1, err := strconv.Atoi(parts[0])
				if err != nil || p1 < 1 || p1 > 65535 {
					return cfg, fmt.Errorf("invalid port specification in port forward %s : %s", pf.Name, port)
				}					
			}
			if len(parts) == 2 {
				p2, err := strconv.Atoi(parts[1])
				if err != nil || p2 < 1 || p2 > 65535 {
					return cfg, fmt.Errorf("invalid port specification in port forward %s : %s", pf.Name, port)
				}
			}
		}
	}
	return cfg, nil
}
