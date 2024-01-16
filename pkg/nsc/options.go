package nsc

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"

	"github.com/nats-io/nats.go"
)

// CABundle is similar to nats.RootCAs but accepts a byte slice instead of a file path.
func CABundle(cert []byte) nats.Option {
	return func(options *nats.Options) error {
		rootCAsCB := func() (*x509.CertPool, error) {
			pool := x509.NewCertPool()
			if ok := pool.AppendCertsFromPEM(cert); !ok {
				return nil, fmt.Errorf("failed to parse root certificate from bundle")
			}

			return pool, nil
		}

		if options.TLSConfig == nil {
			options.TLSConfig = &tls.Config{MinVersion: tls.VersionTLS12}
		}

		options.RootCAsCB = rootCAsCB
		options.Secure = true

		return nil
	}
}
