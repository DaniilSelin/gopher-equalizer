package config

import (
	"fmt"
	"os"
	"time"

	"go.uber.org/zap"
	"gopkg.in/yaml.v2"
)

// Так удобнее и настройка более гибка
type LoggerConfig struct {
	zap.Config `yaml:",inline"`
}

func (lc *LoggerConfig) Build() (*zap.Logger, error) {
	return lc.Config.Build()
}

type RefillConfig struct {
	Interval string `yaml:"interval"`
	Amount   int    `yaml:"amount"`
}

type BucketConfig struct {
	Capacity int          `yaml:"capacity"`
	Refill   RefillConfig `yaml:"refill"`
}

type ServerConfig struct {
	Host string `yaml:"host"`
	Port int    `yaml:"port"`
}

type PoolConfig struct {
	MaxConns          int32    `yaml:"maxConns"`
	MinConns          int32    `yaml:"minConns"`
	MaxConnLifetime   Duration `yaml:"maxConnLifetime"`
	MaxConnIdleTime   Duration `yaml:"maxConnIdleTime"`
	HealthCheckPeriod Duration `yaml:"healthCheckPeriod"`
}

type DBConfig struct {
	Host              string     `yaml:"host"`
	Port              int        `yaml:"port"`
	User              string     `yaml:"user"`
	Password          string     `yaml:"password"`
	Dbname            string     `yaml:"dbname"`
	Sslmode           string     `yaml:"sslmode"`
	Schema            string     `yaml:"schema"`
	ConnectRetries    int        `yaml:"connectRetries"`
	ConnectRetryDelay Duration   `yaml:"connectRetryDelay"`
	Pool              PoolConfig `yaml:"pool"`
}

func (db DBConfig) ConnString() string {
	return fmt.Sprintf(
		"postgresql://%s:%s@%s:%d/%s?sslmode=%s&search_path=%s",
		db.User, db.Password, db.Host, db.Port, db.Dbname, db.Sslmode, db.Schema,
	)
}

type APIConfig struct {
	DefaultLimit int `yaml:"defaultLimit"`
}

type Config struct {
	Server ServerConfig `yaml:"server"`
	Bucket BucketConfig `yaml:"bucket"`
	DB     DBConfig     `yaml:"db"`
	Logger LoggerConfig `yaml:"logger"`
	API	   APIConfig	`yaml:"api"`
}

func LoadConfig(filename string) (*Config, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("could not open config file: %v", err)
	}
	defer file.Close()

	var config Config
	decoder := yaml.NewDecoder(file)
	err = decoder.Decode(&config)
	if err != nil {
		return nil, fmt.Errorf("could not decode config file: %v", err)
	}
	return &config, nil
}

// для bdConfig
type Duration time.Duration

func (d *Duration) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var str string
	if err := unmarshal(&str); err != nil {
		return err
	}
	parsed, err := time.ParseDuration(str)
	if err != nil {
		return err
	}
	*d = Duration(parsed)
	return nil
}
