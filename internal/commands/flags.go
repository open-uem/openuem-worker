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
			Name:     "nats-servers",
			Usage:    "comma-separated list of NATS servers urls e.g (tls://localhost:4433)",
			EnvVars:  []string{"NATS_SERVERS"},
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
