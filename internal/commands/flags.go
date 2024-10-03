package commands

import "github.com/urfave/cli/v2"

func CommonFlags() []cli.Flag {
	return []cli.Flag{
		&cli.StringFlag{
			Name:    "cacert",
			Value:   "certificates/ca.cer",
			Usage:   "the path to your CA certificate file in PEM format",
			EnvVars: []string{"CA_CRT_FILENAME"},
		},
		&cli.StringFlag{
			Name:    "cert",
			Value:   "certificates/worker.cer",
			Usage:   "the path to your worker's certificate file in PEM format",
			EnvVars: []string{"CERT_FILENAME"},
		},
		&cli.StringFlag{
			Name:    "key",
			Value:   "certificates/worker.key",
			Usage:   "the path to your worker's private key file in PEM format",
			EnvVars: []string{"KEY_FILENAME"},
		},
		&cli.StringFlag{
			Name:     "nats-host",
			Usage:    "the NATS server hostname",
			EnvVars:  []string{"NATS_HOST"},
			Required: true,
		},
		&cli.StringFlag{
			Name:     "nats-port",
			Usage:    "the port where NATS server is listening on",
			EnvVars:  []string{"NATS_PORT"},
			Required: true,
		},
		&cli.StringFlag{
			Name:     "dburl",
			Usage:    "the Postgres database connection url e.g (postgres://user:password@host:5432/openuem)",
			EnvVars:  []string{"DATABASE_URL"},
			Required: true,
		},
	}
}
