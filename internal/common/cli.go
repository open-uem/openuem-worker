package common

import (
	"os"
	"path/filepath"

	"github.com/open-uem/openuem_utils"
	"github.com/urfave/cli/v2"
)

func (w *Worker) CheckCLICommonRequisites(cCtx *cli.Context) error {
	var err error

	cwd, err := os.Getwd()
	if err != nil {
		return err
	}

	w.DBUrl = cCtx.String("dburl")
	w.CACertPath = filepath.Join(cwd, cCtx.String("cacert"))
	w.CACert, err = openuem_utils.ReadPEMCertificate(w.CACertPath)
	if err != nil {
		return err
	}

	w.ClientCertPath = filepath.Join(cwd, cCtx.String("cert"))
	_, err = openuem_utils.ReadPEMCertificate(w.ClientCertPath)
	if err != nil {
		return err
	}

	w.ClientKeyPath = filepath.Join(cwd, cCtx.String("key"))
	_, err = openuem_utils.ReadPEMPrivateKey(w.ClientKeyPath)
	if err != nil {
		return err
	}

	w.NATSServers = cCtx.String("nats-servers")
	return nil
}
