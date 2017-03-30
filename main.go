package main

import (
	"encoding/json"
	"flag"
	"io/ioutil"
	"os"
	"os/signal"
	"strings"
	"syscall"

	log "github.com/Sirupsen/logrus"
)

func init() {
	log.SetLevel(log.InfoLevel)
}

func main() {
	type Configuration struct {
		Port      string
		Address   string
		PgUsers   string
		Debug     bool
		Cleartext bool
	}
	var config Configuration

	configFile := flag.String("config", "pghoney.conf", "JSON configuration file")
	flag.Parse()

	jsonConfig, err := ioutil.ReadFile(*configFile)
	if err != nil {
		panic(err)
	}

	err = json.Unmarshal(jsonConfig, &config)
	if err != nil {
		panic(err)
	}

	port := config.Port
	addr := config.Address
	pgUsers := config.PgUsers
	debug := config.Debug
	cleartext := config.Cleartext

	if debug {
		log.SetLevel(log.DebugLevel)
	}

	hpFeedsConfig := &HpFeedsConfig{
		Enabled: false,
	}
	hpfeedsChannel := make(chan []byte)
	if hpFeedsConfig.Enabled {
		go hpfeedsConnect(hpFeedsConfig, hpfeedsChannel)
	}

	postgresServer := NewPostgresServer(
		port,
		addr,
		strings.Split(pgUsers, ","),
		cleartext,
		hpfeedsChannel,
		hpFeedsConfig.Enabled,
	)

	// Capture 'shutdown' signals and shutdown gracefully.
	shutdownSignal := make(chan os.Signal)
	signal.Notify(shutdownSignal, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	go func() {
		sig := <-shutdownSignal
		log.Infof("Process got signal: %s", sig)
		log.Infof("Shutting down...")

		postgresServer.Close()

		os.Exit(0)
	}()

	postgresServer.Listen()
}
