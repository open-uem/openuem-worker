package commands

import (
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"

	"github.com/open-uem/openuem-worker/internal/common"
	"github.com/open-uem/utils"
	"github.com/urfave/cli/v2"
)

func CertManagerWorker() *cli.Command {
	return &cli.Command{
		Name:  "cert-manager",
		Usage: "Manage OpenUEM's Cert-Manager worker",
		Subcommands: []*cli.Command{
			{
				Name:   "start",
				Usage:  "Start an OpenUEM's Cert-Manager worker",
				Action: startCertManagerWorker,
				Flags:  StartCertManagerWorkerFlags(),
			},
			{
				Name:   "stop",
				Usage:  "Stop an OpenUEM's Cert-Manager worker",
				Action: stopWorker,
			},
		},
	}
}

func StartCertManagerWorkerFlags() []cli.Flag {
	flags := CommonFlags()

	flags = append(flags, &cli.StringFlag{
		Name:     "ocsp",
		Usage:    "the url of the OCSP responder, e.g https://ocsp.example.com",
		EnvVars:  []string{"OCSP"},
		Required: true,
	})

	return append(flags, &cli.StringFlag{
		Name:    "cakey",
		Value:   "certificates/ca.key",
		Usage:   "the path to your CA private key file in PEM format",
		EnvVars: []string{"CA_KEY_FILENAME"},
	})
}

func startCertManagerWorker(cCtx *cli.Context) error {
	worker := common.NewWorker("")

	// Specific requisites
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}

	caKeyPath := filepath.Join(cwd, cCtx.String("cakey"))
	worker.CAPrivateKey, err = utils.ReadPEMPrivateKey(caKeyPath)
	if err != nil {
		return err
	}

	// get ocsp servers
	ocspServers := []string{}
	for _, ocsp := range strings.Split(cCtx.String("ocsp"), ",") {
		ocspServers = append(ocspServers, strings.TrimSpace(ocsp))
	}
	worker.OCSPResponders = ocspServers

	if err := worker.CheckCLICommonRequisites(cCtx); err != nil {
		log.Printf("[ERROR]: could not generate config for Cert Manager Worker: %v", err)
	}

	if err := os.WriteFile("PIDFILE", []byte(strconv.Itoa(os.Getpid())), 0666); err != nil {
		return err
	}

	worker.StartWorker(worker.SubscribeToCertManagerWorkerQueues)

	// Keep the connection alive
	done := make(chan os.Signal, 1)
	signal.Notify(done, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	log.Println("[INFO]: cert manager worker is ready")
	<-done

	worker.StopWorker()
	log.Println("[INFO]: cert manager worker has been shutdown")
	return nil
}
