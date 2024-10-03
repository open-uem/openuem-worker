package commands

import (
	"crypto/rsa"
	"crypto/x509"

	"github.com/doncicuto/openuem-worker/internal/models"
	"github.com/doncicuto/openuem_ent"
	"github.com/doncicuto/openuem_nats"
)

type WorkerCommand struct {
	MessageServer  *openuem_nats.MessageServer
	Model          *models.Model
	CACert         *x509.Certificate
	CAPrivateKey   *rsa.PrivateKey
	PKCS12         []byte
	UserCert       *x509.Certificate
	CertRequest    *openuem_nats.CertificateRequest
	Settings       *openuem_ent.Settings
	OCSPResponders []string
}
