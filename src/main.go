package main

import (
	"flag"
	"fmt"
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
	logWithCaller(fmt.Sprintf("Set Loglevel to %s", logLevel), InfoLog)

	config, err := ReadConfigFile(*configFileLocation)
	if err != nil {
		log.Fatal(err)
	}

	err = setSecretKey(config.SecretKey)
	if err != nil {
		logWithCaller("Failed to decode key: "+err.Error(), FatalLog)
		os.Exit(1)
	}

	port := os.Getenv("STREAM_API_PORT")
	if port == "" {
		port = "8080" // Default port if not set
	}

	logWithCaller(fmt.Sprintf("Using port: %s", port), InfoLog)

	logWithCaller("Starting StreamAPI server", InfoLog)
	streamApiServer, err := StreamAPI(":"+port, *config)
	if err != nil {
		logWithCaller("Failed to create StreamAPI server", FatalLog)
		logWithCaller(err.Error(), FatalLog)
		os.Exit(1)
	}
	// Handle the error from Run instead of letting it return
	err = streamApiServer.Run()
	if err != nil {
		logWithCaller(fmt.Sprintf("StreamAPI server error: %s", err), FatalLog)
		os.Exit(1)
	}

	logWithCaller("StreamAPI server stopped", InfoLog)
}
