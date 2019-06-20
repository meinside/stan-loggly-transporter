package main

// stan-loggly log transporter
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

	"github.com/meinside/loggly-go"
	stanclient "github.com/meinside/stan-client-go"
	stan "github.com/nats-io/stan.go"
)

// Constants
const (
	configFilename = "config.json"

	queueGroup  = "unique"
	durableName = "durable"

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

	// loggly api
	LogglyToken string `json:"logglyToken"`
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
var lggly *loggly.Loggly
var whenDisconnected chan struct{}
var whenConnectError chan struct{}

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

// setup things
func init() {
	// read configs,
	conf = readConfig()

	// loggly logger
	lggly = loggly.New(conf.LogglyToken)

	// processors for incoming websocket data
	whenDisconnected = make(chan struct{}, 1)
	whenConnectError = make(chan struct{}, 1)

	// initialize connection to NATS/STAN servers,
	stanClient = stanclient.Connect(
		conf.NatsServers,
		stanclient.AuthOptionWithUsernameAndPassword(conf.NatsClientUsername, conf.NatsClientPasswd),
		stanclient.SecOptionWithCerts(conf.NatsClientCertPath, conf.NatsClientKeyPath, conf.NatsRootCaPath),
		conf.StanClusterID,
		conf.StanClientID,
		[]stanclient.ToSubscribe{
			stanclient.ToSubscribe{
				Subject:        conf.LogSubject,
				QueueGroupName: queueGroup,
				DurableName:    durableName,
				DeliverAll:     true,
			},
		},
		messageHandler,
		publishFailureHandler,
		logger,
	)
}

func main() {
	logger.Log("starting transporter...")

	// for catching signals
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	// process incoming messages from stan subscriptions
	go stanClient.Poll()

	// wait...
wait:
	for {
		select {
		case <-interrupt: // when interrupted,
			logger.Error("interrupted: signal received")

			break wait
		}
	}

	stanClient.Close()

	logger.Log("terminating transporter...")

	time.Sleep(terminationWaitSeconds * time.Second)

	logger.Log("transporter terminated")
}

// handle messages from subscriptions (time-consuming jobs)
func messageHandler(message *stan.Msg) {
	if message.Subject == conf.LogSubject {
		go func(m *stan.Msg) {
			var msg interface{}
			if err := json.Unmarshal(m.Data, &msg); err == nil {
				sendLogToLoggly(m.Subject, msg)
			} else {
				logger.Error("failed to unmarshal from subject %s: %s", m.Subject, err)
			}
		}(message)
	} else {
		logger.Error("unprocessable message with subject: %s", message.Subject)
	}
}

// called when publish fails
func publishFailureHandler(subject, nuid string, obj interface{}) {
	logger.Error("failed to publish data (nuid: %s) to %s: %+v", nuid, subject, obj)

	// resend it
	//stanClient.PublishAsync(subject, obj)
}

// send a log to Loggly API
func sendLogToLoggly(subject string, data interface{}) {
	if err := lggly.LogSync(data); err != nil {
		logger.Error("failed to send to loggly: %s (%s)", subject, err)

		// if it fails, queue it back
		stanClient.PublishAsync(subject, data)
	}
}
