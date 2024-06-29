/*
Real-time Online/Offline Charging System (OCS) for Telecom & ISP environments
Copyright (C) ITsysCOM GmbH

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU General Public License for more details.

You should have received a copy of the GNU General Public License
along with this program.  If not, see <http://www.gnu.org/licenses/>
*/
package ees

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"os"

	"github.com/cgrates/cgrates/config"
	"github.com/cgrates/cgrates/utils"
	kafka "github.com/segmentio/kafka-go"
)

// NewKafkaEE creates a kafka poster
func NewKafkaEE(cfg *config.EventExporterCfg, dc *utils.SafeMapStorage) (*KafkaEE, error) {
	pstr := &KafkaEE{
		cfg:   cfg,
		dc:    dc,
		topic: utils.DefaultQueueID,
		reqs:  newConcReq(cfg.ConcurrentRequests),
	}
	opts := cfg.Opts.Kafka
	if opts.Topic != nil {
		pstr.topic = *opts.Topic
	}
	var tlsCfg *tls.Config
	if opts.TLS != nil && *opts.TLS {
		rootCAs, err := x509.SystemCertPool()
		if err != nil {
			return nil, err
		}
		if rootCAs == nil {
			rootCAs = x509.NewCertPool()
		}

		// Load additional CA certificates if a path is provided.
		if opts.CAPath != nil && *opts.CAPath != "" {
			ca, err := os.ReadFile(*opts.CAPath)
			if err != nil {
				return nil, err
			}
			if !rootCAs.AppendCertsFromPEM(ca) {
				return nil, errors.New("failed to append certificates from PEM file")
			}
		}
		tlsCfg = &tls.Config{
			RootCAs:            rootCAs,
			InsecureSkipVerify: opts.SkipTLSVerify != nil && *opts.SkipTLSVerify,
		}
	}
	pstr.writer = &kafka.Writer{
		Addr:  kafka.TCP(pstr.Cfg().ExportPath), // Kafka broker address
		Topic: pstr.topic,                       // Kafka topic to write to

		// Leave it to the ExportWithAttempts function
		// to handle the connect attempts.
		MaxAttempts: 1,

		// Set up the Kafka transport (DefaultTransport + optional TLS configuration).
		Transport: &kafka.Transport{
			// Dial: (&net.Dialer{
			// 	Timeout: 3 * time.Second,
			// }).DialContext,
			TLS: tlsCfg,
		},
	}
	return pstr, nil
}

// KafkaEE is a kafka poster
type KafkaEE struct {
	topic  string // identifier of the CDR queue where we publish
	writer *kafka.Writer

	cfg  *config.EventExporterCfg
	dc   *utils.SafeMapStorage
	reqs *concReq
	bytePreparing
}

func (pstr *KafkaEE) Cfg() *config.EventExporterCfg { return pstr.cfg }

func (pstr *KafkaEE) Connect() error { return nil }

func (pstr *KafkaEE) ExportEvent(content any, key string) error {
	pstr.reqs.get()
	defer pstr.reqs.done()
	return pstr.writer.WriteMessages(context.Background(), kafka.Message{
		Key:   []byte(key),
		Value: content.([]byte),
	})
}

func (pstr *KafkaEE) Close() error {
	utils.Logger.Debug("closing kafka...")
	pstr.writer.Transport.(*kafka.Transport).CloseIdleConnections()
	return pstr.writer.Close()
}

func (pstr *KafkaEE) GetMetrics() *utils.SafeMapStorage { return pstr.dc }
