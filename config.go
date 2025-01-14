package main

import (
	"fmt"
	flag "github.com/spf13/pflag"
	"net"
	"strconv"
)

const portMinValue int = 1024
const portMaxValue int = 65535
const portDefaultValue int = 3000

type Config struct {
	Port                int
	ShouldFormatJson    bool
	IsHelpRequested     bool
	MaxFormBodySizeInMB int
}

func (c *Config) PrintUsage() {
	flag.Usage()
}

func (c *Config) Validate() error {
	if !isPortInValidRange(c.Port, portMinValue, portMaxValue) {
		return fmt.Errorf("expecting port to be in the range between %d and %d", portMinValue, portMaxValue)
	}

	if !isPortAvailable(c.Port) {
		return fmt.Errorf("can't listen on %d, port already in use", c.Port)
	}

	return nil
}

func NewConfig() (*Config, error) {
	config := new(Config)

	flag.BoolVarP(&config.IsHelpRequested, "help", "h", false, "Print usage information and exit.")
	flag.IntVarP(&config.Port, "port", "p", portDefaultValue, "Port to listen on.")
	flag.BoolVar(&config.ShouldFormatJson, "format-json", true, "Format JSON.")
	flag.IntVar(&config.MaxFormBodySizeInMB, "form-data-size", 10, "Maximum size of form-data body in MB that will be stored in memory. If body is greater, it's still should be parsed fully but stored in temp file on disk.")

	flag.Parse()

	err := config.Validate()
	if err != nil {
		return nil, err
	}

	return config, nil
}

func isPortAvailable(port int) bool {
	portStr := strconv.FormatUint(uint64(port), 10)

	// Attempt to listen on the specified port
	listener, err := net.Listen("tcp", ":"+portStr)
	if err != nil {
		return false
	}

	err = listener.Close()
	if err != nil {
		panic(fmt.Errorf("failed to close port listener: %v", err))
	}

	return true
}

func isPortInValidRange(port int, min int, max int) bool {
	return port > min && port < max
}
