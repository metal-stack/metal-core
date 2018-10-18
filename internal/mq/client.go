package mq

import (
	"crypto/tls"
	"fmt"
	"git.f-i-ts.de/cloud-native/maas/metal-core/internal/domain"
	"git.f-i-ts.de/cloud-native/maas/metal-core/internal/logging"
	"github.com/bitly/go-nsq"
	log "github.com/sirupsen/logrus"
)

type (
	Client interface {
		Subscribe(topic string, channel string, callback func(message string))
	}
	client struct {
		Config *domain.Config
	}
)

func NewClient(cfg *domain.Config) Client {
	return client{
		Config: cfg,
	}
}

func (c client) Subscribe(topic string, channel string, callback func(message string)) {
	q, _ := nsq.NewConsumer(topic, channel, c.createConfig())
	q.AddHandler(nsq.HandlerFunc(func(msg *nsq.Message) error {
		log.WithFields(log.Fields{
			"topic":   topic,
			"channel": channel,
			"timestamp": msg.Timestamp,
			"attempts": msg.Attempts,
			"nsqdAddress": msg.NSQDAddress,
			"message": string(msg.Body),
		}).Info("Got message")
		callback(string(msg.Body))
		return nil
	}))
	mqServer := fmt.Sprintf("%v:%d", c.Config.MQAddress, c.Config.MQPort)
	if err := q.ConnectToNSQLookupd(mqServer); err != nil {
		logging.Decorate(log.WithFields(log.Fields{
			"nsqlookupd": mqServer,
			"error":      err,
		})).Fatal("Cannot connect to MQ server")
	}
}

func (c client) createConfig() *nsq.Config {
	config := nsq.NewConfig()
	if c.Config.MQCert != "" {
		cert, _ := tls.LoadX509KeyPair(c.Config.MQCert, c.Config.MQKey)
		tlsConfig := &tls.Config{
			Certificates:       []tls.Certificate{cert},
			InsecureSkipVerify: true,
		}
		config.TlsConfig = tlsConfig
		config.TlsV1 = true
	}
	return config
}
