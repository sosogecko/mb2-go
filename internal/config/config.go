package config

import (
	"fmt"
	"os"
	"time"

	mysqldriver "github.com/go-sql-driver/mysql"
	"gopkg.in/yaml.v3"
)

type Config struct {
	Server      ServerConfig      `yaml:"server"`
	MySQL       MySQLConfig       `yaml:"mysql"`
	Cache       CacheConfig       `yaml:"cache"`
	Log         LogConfig         `yaml:"log"`
	Paymentwall PaymentwallConfig `yaml:"paymentwall"`
	Timezone    string            `yaml:"timezone"`
}

type ServerConfig struct {
	Addr string `yaml:"addr"`
	Mode string `yaml:"mode"`
}

type MySQLConfig struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
	Database string `yaml:"database"`
	Charset  string `yaml:"charset"`
}

type CacheConfig struct {
	Dir string `yaml:"dir"`
}

type LogConfig struct {
	Dir        string `yaml:"dir"`
	MaxSizeMB  int    `yaml:"max_size_mb"`
	MaxAgeDays int    `yaml:"max_age_days"`
	Compress   bool   `yaml:"compress"`
}

type PaymentwallConfig struct {
	PublicKey  string `yaml:"public_key"`
	PrivateKey string `yaml:"private_key"`
	Widget     string `yaml:"widget"`
}

func Load(path string) (*Config, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}
	cfg := &Config{}
	if err := yaml.Unmarshal(b, cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}
	cfg.applyDefaults()
	return cfg, nil
}

func (c *Config) applyDefaults() {
	if c.Server.Addr == "" {
		c.Server.Addr = "0.0.0.0:8080"
	}
	if c.Server.Mode == "" {
		c.Server.Mode = "release"
	}
	if c.MySQL.Port == 0 {
		c.MySQL.Port = 3306
	}
	if c.MySQL.Charset == "" {
		c.MySQL.Charset = "utf8mb4"
	}
	if c.Cache.Dir == "" {
		c.Cache.Dir = "data/cache"
	}
	if c.Log.Dir == "" {
		c.Log.Dir = "logs"
	}
	if c.Log.MaxSizeMB <= 0 {
		c.Log.MaxSizeMB = 100
	}
	if c.Log.MaxAgeDays <= 0 {
		c.Log.MaxAgeDays = 30
	}
	if c.Timezone == "" {
		c.Timezone = "Asia/Shanghai"
	}
	if c.Paymentwall.Widget == "" {
		c.Paymentwall.Widget = "p1_1"
	}
}

func (m MySQLConfig) DSN() string {
	cfg := mysqldriver.NewConfig()
	cfg.User = m.User
	cfg.Passwd = m.Password
	cfg.Net = "tcp"
	cfg.Addr = fmt.Sprintf("%s:%d", m.Host, m.Port)
	cfg.DBName = m.Database
	cfg.ParseTime = true
	cfg.Loc, _ = time.LoadLocation("Asia/Shanghai")
	cfg.Timeout = 10 * time.Second
	cfg.ReadTimeout = 30 * time.Second
	cfg.WriteTimeout = 30 * time.Second
	cfg.Params = map[string]string{
		"charset": m.Charset,
	}
	return cfg.FormatDSN()
}
