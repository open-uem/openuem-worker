package models

import (
	"context"
	"time"

	"github.com/doncicuto/openuem_ent"
	"github.com/doncicuto/openuem_ent/certificate"
	"golang.org/x/crypto/ocsp"
)

func (m *Model) SaveCertificate(serial int64, certType certificate.Type, uid, description string, expiry time.Time) error {

	if uid != "" {
		_, err := m.Client.Certificate.Create().SetID(serial).SetType(certType).SetDescription(description).SetExpiry(expiry).SetUID(uid).Save(context.Background())
		if err != nil {
			return err
		}

		if _, err := m.Client.User.UpdateOneID(uid).SetExpiry(expiry).Save(context.Background()); err != nil {
			return err
		}
	} else {
		_, err := m.Client.Certificate.Create().SetID(serial).SetType(certType).SetDescription(description).SetExpiry(expiry).Save(context.Background())
		if err != nil {
			return err
		}
	}

	return nil
}

func (m *Model) RevokePreviousCertificates(description string) error {
	cert, err := m.Client.Certificate.Query().Where(certificate.DescriptionEQ(description)).Only(context.Background())
	if err != nil {
		if openuem_ent.IsNotFound(err) {
			return nil
		}
		return err
	}

	if err := m.Client.Certificate.DeleteOneID(cert.ID).Exec(context.Background()); err != nil {
		return err
	}

	return m.AddRevocation(cert.ID, ocsp.Superseded, "new certificate requested from console", cert.Expiry)
}
