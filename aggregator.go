/*
Copyright © 2020 Red Hat, Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Implementation of insights rules aggregator
package main

import (
	"flag"
	"fmt"
	"github.com/spf13/viper"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/RedHatInsights/insights-results-aggregator/broker"
	"github.com/RedHatInsights/insights-results-aggregator/consumer"
	"github.com/RedHatInsights/insights-results-aggregator/producer"
	"github.com/RedHatInsights/insights-results-aggregator/server"
	"github.com/RedHatInsights/insights-results-aggregator/storage"
)

const (
	// ExitStatusOK means that the tool finished with success
	ExitStatusOK = iota
	// ExitStatusNoCommand is returned when no command (-consumer, -server) is provided via CLI
	ExitStatusNoCommand
	// ExitStatusProducerError is returned in case of any producer-related error
	ExitStatusProducerError
	// ExitStatusConsumerError is returned in case of any consumer-related error
	ExitStatusConsumerError
	// ExitStatusServerError is returned in case of any REST API server-related error
	ExitStatusServerError
)

func loadConfiguration(defaultConfigFile string) {
	configFile, specified := os.LookupEnv("INSIGHTS_RESULTS_AGGREGATOR_CONFIG_FILE")
	if specified {
		// we need to separate the directory name and filename without extension
		directory, basename := filepath.Split(configFile)
		file := strings.TrimSuffix(basename, filepath.Ext(basename))
		// parse the configuration
		viper.SetConfigName(file)
		viper.AddConfigPath(directory)
	} else {
		// parse the configuration
		viper.SetConfigName(defaultConfigFile)
		viper.AddConfigPath(".")
	}
	err := viper.ReadInConfig()
	if err != nil {
		panic(fmt.Errorf("Fatal error config file: %s", err))
	}
}

func loadBrokerConfiguration() broker.Configuration {
	brokerCfg := viper.Sub("broker")
	return broker.Configuration{
		Address: brokerCfg.GetString("address"),
		Topic:   brokerCfg.GetString("topic"),
		Group:   brokerCfg.GetString("group"),
	}
}

func loadStorageConfiguration() storage.Configuration {
	storageCfg := viper.Sub("storage")
	return storage.Configuration{
		Driver:     storageCfg.GetString("driver"),
		DataSource: storageCfg.GetString("datasource"),
	}
}

func loadServerConfiguration() server.Configuration {
	serverCfg := viper.Sub("server")
	return server.Configuration{
		Address:   serverCfg.GetString("address"),
		APIPrefix: serverCfg.GetString("api_prefix"),
	}
}

func produceMessages() error {
	const testMessage = `
{"OrgID":1,
 "ClusterName":"aaaaaaaa-bbbb-cccc-dddd-000000000000",
 "Report":"{}"}
`
	brokerCfg := loadBrokerConfiguration()

	_, _, err := producer.ProduceMessage(brokerCfg, testMessage)
	if err != nil {
		log.Fatal(err)
		return err
	}
	return nil
}

func startConsumer() error {
	storageCfg := loadStorageConfiguration()
	storage, err := storage.New(storageCfg)
	if err != nil {
		log.Fatal(err)
		return err
	}
	defer storage.Close()

	brokerCfg := loadBrokerConfiguration()
	consumerInstance, err := consumer.New(brokerCfg, storage)
	if err != nil {
		log.Fatal(err)
		return err
	}
	defer consumerInstance.Close()
	err = consumerInstance.Start()
	if err != nil {
		log.Fatal(err)
		return err
	}
	return nil
}

func startServer() error {
	storageCfg := loadStorageConfiguration()
	storage, err := storage.New(storageCfg)
	if err != nil {
		log.Fatal(err)
		return err
	}
	defer storage.Close()

	serverCfg := loadServerConfiguration()
	serverInstance := server.New(serverCfg, storage)
	err = serverInstance.Start()
	if err != nil {
		log.Fatal(err)
		return err
	}
	return nil
}

func startService(produceMessagesCmd bool, startConsumerCmd bool, startServerCmd bool) int {
	switch {
	case produceMessagesCmd:
		err := produceMessages()
		if err != nil {
			return ExitStatusProducerError
		}
	case startConsumerCmd:
		err := startConsumer()
		if err != nil {
			return ExitStatusConsumerError
		}
	case startServerCmd:
		err := startServer()
		if err != nil {
			return ExitStatusServerError
		}
	default:
		fmt.Println("Setup error: use -consumer, -server, or -produce CLI flag to select startup mode")
		return ExitStatusNoCommand
	}
	return ExitStatusOK
}

func main() {
	loadConfiguration("config")

	// parse command line arguments and flags
	var produceMessagesCmd = flag.Bool("produce", false, "produce messages for testing purposes")
	var startConsumerCmd = flag.Bool("consumer", false, "start the service in consumer mode")
	var startServerCmd = flag.Bool("server", false, "start the service in HTTP server mode")
	flag.Parse()

	statusCode := startService(*produceMessagesCmd, *startConsumerCmd, *startServerCmd)
	os.Exit(statusCode)
}
