package config

import (
	"github.com/kelseyhightower/envconfig"
)

var singleConfig *Config = nil

type Config struct {
	Database *dbConfig
	Service  *svcConfig
	Provider *providerConfig
}

type dbConfig struct {
	Type     string `envconfig:"DB_TYPE" default:"pgsql"`
	Hostname string `envconfig:"DB_HOST" default:"localhost"`
	Port     string `envconfig:"DB_PORT" default:"5432"`
	Name     string `envconfig:"DB_NAME" default:"placement"`
	User     string `envconfig:"DB_USER" default:"admin"`
	Password string `envconfig:"DB_PASS" default:"adminpass"`
}

type svcConfig struct {
	Address   string `envconfig:"DCM_ADDRESS" default:":8080"`
	BaseUrl   string `envconfig:"DCM_BASE_URL" default:"https://localhost:8080"`
	LogLevel  string `envconfig:"DCM_LOG_LEVEL" default:"info"`
	OpaServer string `envconfig:"DCM_OPA_SERVER" default:"http://localhost:8181"`
}

type providerConfig struct {
	Address  string `envconfig:"PROVIDER_ADDRESS" default:":8081"`
	BaseUrl  string `envconfig:"PROVIDER_BASE_URL" default:"https://localhost:8081"`
	LogLevel string `envconfig:"PROVIDER_LOG_LEVEL" default:"debug"`
}

func New() (*Config, error) {
	if singleConfig == nil {
		singleConfig = new(Config)
		if err := envconfig.Process("", singleConfig); err != nil {
			return nil, err
		}
	}
	return singleConfig, nil
}
