package commands

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/doncicuto/openuem-worker/internal/models"
	"github.com/doncicuto/openuem_utils"
	"github.com/go-playground/validator"
	"github.com/urfave/cli/v2"
)

func (command *WorkerCommand) checkCommonRequisites(cCtx *cli.Context) error {
	var err error

	cwd, err := os.Getwd()
	if err != nil {
		return err
	}

	log.Printf("🗃️   connecting to database")
	command.Model, err = models.New(cCtx.String("dburl"))
	if err != nil {
		return fmt.Errorf("could not connect to database, reason: %s", err.Error())
	}

	log.Println("📜  reading CA certificate")
	caCertPath := filepath.Join(cwd, cCtx.String("cacert"))
	command.CACert, err = openuem_utils.ReadPEMCertificate(caCertPath)
	if err != nil {
		return err
	}

	log.Println("📜  reading worker's client certificate")
	certPath := filepath.Join(cwd, cCtx.String("cert"))
	_, err = openuem_utils.ReadPEMCertificate(certPath)
	if err != nil {
		return err
	}

	log.Println("🔑  reading worker's private key")
	keyPath := filepath.Join(cwd, cCtx.String("key"))
	_, err = openuem_utils.ReadPEMPrivateKey(keyPath)
	if err != nil {
		return err
	}

	validate := validator.New()
	err = validate.Var(cCtx.String("nats-host"), "hostname")
	if err != nil {
		return err
	}

	err = validate.Var(cCtx.String("nats-port"), "numeric")
	if err != nil {
		return err
	}
	return nil
}
