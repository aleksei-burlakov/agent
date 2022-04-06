package cmd

import (
	"fmt"
	"os"
	"time"

	"github.com/pkg/errors"
	"github.com/spf13/viper"
	"github.com/trento-project/agent/internal"
	"github.com/trento-project/agent/internal/discovery/collector"
)

func LoadConfig() (*internal.Config, error) {
	enablemTLS := viper.GetBool("enable-mtls")
	cert := viper.GetString("cert")
	key := viper.GetString("key")
	ca := viper.GetString("ca")

	if enablemTLS {
		var err error

		if cert == "" {
			err = fmt.Errorf("you must provide a server ssl certificate")
		}
		if key == "" {
			err = errors.Wrap(err, "you must provide a key to enable mTLS")
		}
		if ca == "" {
			err = errors.Wrap(err, "you must provide a CA ssl certificate")
		}
		if err != nil {
			return nil, err
		}
	}

	hostname, err := os.Hostname()
	if err != nil {
		return nil, errors.Wrap(err, "could not read the hostname")
	}

	sshAddress := viper.GetString("ssh-address")
	if sshAddress == "" {
		return nil, errors.New("ssh-address is required, cannot start agent")
	}

	return &internal.Config{
		CollectorConfig: &collector.Config{
			CollectorHost: viper.GetString("collector-host"),
			CollectorPort: viper.GetInt("collector-port"),
			EnablemTLS:    enablemTLS,
			Cert:          cert,
			Key:           key,
			CA:            ca,
		},
		InstanceName:    hostname,
		SSHAddress:      sshAddress,
		DiscoveryPeriod: time.Duration(viper.GetInt("discovery-period")) * time.Second,
	}, nil
}
