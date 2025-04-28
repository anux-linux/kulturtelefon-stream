package main

import (
	"flag"
	"log"
	"os"

	"gopkg.in/yaml.v3"
)

func ReadConfigFile(filePath string) (*Config, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		log.Printf("Error reading config file: %v", err)
		return nil, err
	}
	var config Config
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		return nil, err
	}

	return &config, nil
}

func main() {
	configFileLocation := flag.String("config", "stream.config", "Location of config (e.g., stream.config)")
	logLevel := *flag.String("loglevel", "debug", "Log level (debug, info, warn, fatal)")
	flag.Parse()

	setLogLevel(LogType(logLevel))
	logWithCaller("Set Loglevel to %s", LogType(logLevel))

	config, err := ReadConfigFile(*configFileLocation)
	if err != nil {
		log.Fatal(err)
	}

	err = setSecretKey(config.SecretKey)
	if err != nil {
		logWithCaller("Failed to decode key: "+err.Error(), FatalLog)
		os.Exit(1)
	}

	log.Print("Starting StreamAPI server")
	streamApiServer, err := StreamAPI(":8888", *config)
	if err != nil {
		log.Fatal("Failed to create StreamAPI server")
		return
	}
	streamApiServer.Run()
}
