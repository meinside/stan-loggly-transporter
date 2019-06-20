# stan-loggly-transporter

A log transporter which polls messages from [STAN](https://github.com/nats-io/nats-streaming-server) and sends them to [loggly](https://www.loggly.com/).

## build

```bash
$ git clone github.com/meinside/stan-loggly-transporter
$ cd stan-loggly-transporter
$ go build
```

## configure

Duplicate the sample config file and edit it:

```bash
$ cp config.json.sample config.json
$ vi config.json
```

```json
{
	"stanClusterId": "my-stan-cluster-id",
	"stanClientId": "this-client-id",
	"natsServers": [
		"tls://localhost:4242",
		"tls://localhost:4243"
	],
	"natsClientUsername": "USER",
	"natsClientPasswd": "PASSWORD",
	"natsClientCertPath": "/path/to/certs/cert.pem",
	"natsClientKeyPath": "/path/to/certs/key.pem",
	"natsRootCaPath": "/path/to/certs/ca.pem",

	"logSubject": "logmsgs.loggly",

	"logglyToken": "0123-4567-abcd"
}
```

Edit values to yours.

## how to run

### run it directly

```bash
$ ./stan-loggly-transporter
```

or,

### run it with systemd

Copy systemd service file and edit it:

```bash
$ sudo cp systemd/stan-loggly-transporter.service /lib/systemd/stan-loggly-transporter.service
$ sudo vi /lib/systemd/stan-loggly-transporter.service
```

```
[Unit]
Description=STAN-to-Loggly Log Transporter
Wants=stan.service
After=network.target syslog.target stan.service

[Service]
Type=simple
User=some_user
Group=some_user
WorkingDirectory=/path/to/stan-loggly-transporter
ExecStart=/path/to/stan-loggly-transporter/stan-loggly-transporter
Restart=always
RestartSec=5
Environment=

[Install]
WantedBy=multi-user.target
```

then enable/start it:

```bash
$ sudo systemctl enable stan-loggly-transporter.service
$ sudo systemctl start stan-loggly-transporter.service
```

## how to log messages

Send messages with the same subject name in the `config.json` file.
(eg. **"logmsgs.loggly"**)

You can see a sample logger application [here](https://github.com/meinside/stan-loggly-transporter/blob/master/sample/logger).

## license

MIT

