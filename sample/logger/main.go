package main

// (sample) stan-loggly log pusher
//
// created on 2019.06.20.

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"time"

	stanclient "github.com/meinside/stan-client-go"
	"github.com/meinside/stan-loggly-transporter/types"
	stan "github.com/nats-io/stan.go"
)

// Constants
const (
	applicationName = "sample-logger"
	configFilename  = "config.json"

	messageIntervalSeconds = 5
	terminationWaitSeconds = 5
)

// struct for config file
type config struct {
	// values for NATS/STAN connection
	StanClusterID      string   `json:"stanClusterId"`
	StanClientID       string   `json:"stanClientId"`
	NatsServers        []string `json:"natsServers"`
	NatsClientUsername string   `json:"natsClientUsername"`
	NatsClientPasswd   string   `json:"natsClientPasswd"`
	NatsClientCertPath string   `json:"natsClientCertPath"`
	NatsClientKeyPath  string   `json:"natsClientKeyPath"`
	NatsRootCaPath     string   `json:"natsRootCaPath"`

	// log subject
	LogSubject string `json:"logSubject"`
}

// read config from file
func readConfig() config {
	var err error
	var execPath string
	if execPath, err = os.Executable(); err == nil {
		var bytes []byte
		if bytes, err = ioutil.ReadFile(filepath.Join(filepath.Dir(execPath), configFilename)); err == nil {
			var conf config
			if err = json.Unmarshal(bytes, &conf); err == nil {
				return conf
			}
		}
	}

	panic(err)
}

var conf config
var stanClient *stanclient.Client

type stdlogger struct {
	_stdout *log.Logger
	_stderr *log.Logger
}

func (l stdlogger) Log(format string, args ...interface{}) {
	l._stdout.Printf(format, args...)
}

func (l stdlogger) Error(format string, args ...interface{}) {
	l._stderr.Printf(format, args...)
}

var logger = stdlogger{
	_stdout: log.New(os.Stdout, "", log.LstdFlags),
	_stderr: log.New(os.Stderr, "", log.LstdFlags),
}

type logMessage struct {
	types.LogMessage

	ApplicationName string `json:"app"`
	Severity        string `json:"severity,omitempty"`
	Message         string `json:"message"`
}

func main() {
	// read configs,
	conf = readConfig()

	// initialize connection to NATS/STAN servers,
	stanClient = stanclient.Connect(
		conf.NatsServers,
		stanclient.AuthOptionWithUsernameAndPassword(conf.NatsClientUsername, conf.NatsClientPasswd),
		stanclient.SecOptionWithCerts(conf.NatsClientCertPath, conf.NatsClientKeyPath, conf.NatsRootCaPath),
		conf.StanClusterID,
		conf.StanClientID,
		[]stanclient.ToSubscribe{},
		messageHandler,
		publishFailureHandler,
		logger,
	)

	// for catching signals
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	// process incoming messages from stan subscriptions
	go stanClient.Poll()

	ticker := time.NewTicker(messageIntervalSeconds * time.Second)

	// wait...
wait:
	for {
		select {
		case <-interrupt: // when interrupted,
			logger.Error("interrupted: signal received")

			break wait
		case <-ticker.C:
			msg := logMessage{
				ApplicationName: applicationName,
				Severity:        "Log",
				Message:         "log message for test",
			}
			msg.MarkTimestamp() // mark timestamp as now

			if err := stanClient.Publish(conf.LogSubject, msg); err == nil {
				logger.Log("sent a log message")
			} else {
				logger.Error("failed to publish message: %s", err)
			}
		}
	}

	time.Sleep(terminationWaitSeconds * time.Second)

	stanClient.Close()
}

// handle messages from subscriptions (time-consuming jobs)
func messageHandler(message *stan.Msg) {
	// do nothing
}

// called when publish fails
func publishFailureHandler(subject, nuid string, obj interface{}) {
	// do nothing
}
