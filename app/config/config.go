package config

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"sync"

	"github.com/fsnotify/fsnotify"
	"gopkg.in/yaml.v3"
)

// C 为 yaml 原始结构，供外部需要完整配置时使用
type yamlConfig struct {
	Server   serverCfg   `yaml:"server"`
	Database databaseCfg `yaml:"database"`
	Redis    redisCfg    `yaml:"redis"`
	Log      logCfg      `yaml:"log"`
	JWT      jwtCfg      `yaml:"jwt"`
}

type serverCfg struct {
	Host string `yaml:"host"`
	Port int    `yaml:"port"`
	Mode string `yaml:"mode"`
}

type databaseCfg struct {
	Host         string `yaml:"host"`
	Port         int    `yaml:"port"`
	User         string `yaml:"user"`
	Password     string `yaml:"password"`
	Name         string `yaml:"name"`
	SSLMode      string `yaml:"sslmode"`
	Timezone     string `yaml:"timezone"`
	MaxIdleConns int    `yaml:"max_idle_conns"`
	MaxOpenConns int    `yaml:"max_open_conns"`
}

type redisCfg struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	Password string `yaml:"password"`
	DB       int    `yaml:"db"`
	PoolSize int    `yaml:"pool_size"`
}

type logCfg struct {
	Level    string `yaml:"level"`
	Encoding string `yaml:"encoding"`
}

type jwtCfg struct {
	Secret     string `yaml:"secret"`
	ExpireHour int    `yaml:"expire_hour"`
}

var (
	cfg     yamlConfig
	cfgPath string
	mu      sync.RWMutex
)

func applyDefaults(c *yamlConfig) {
	if c.Server.Host == "" {
		c.Server.Host = "0.0.0.0"
	}
	if c.Server.Port == 0 {
		c.Server.Port = 8080
	}
	if c.Server.Mode == "" {
		c.Server.Mode = "debug"
	}
	if c.Database.Host == "" {
		c.Database.Host = "localhost"
	}
	if c.Database.Port == 0 {
		c.Database.Port = 5432
	}
	if c.Database.User == "" {
		c.Database.User = "postgres"
	}
	if c.Database.Name == "" {
		c.Database.Name = "graduation_project"
	}
	if c.Database.SSLMode == "" {
		c.Database.SSLMode = "disable"
	}
	if c.Database.Timezone == "" {
		c.Database.Timezone = "PRC"
	}
	if c.Database.MaxIdleConns == 0 {
		c.Database.MaxIdleConns = 10
	}
	if c.Database.MaxOpenConns == 0 {
		c.Database.MaxOpenConns = 100
	}
	if c.Redis.Host == "" {
		c.Redis.Host = "localhost"
	}
	if c.Redis.Port == 0 {
		c.Redis.Port = 6379
	}
	if c.Log.Level == "" {
		c.Log.Level = "info"
	}
	if c.Log.Encoding == "" {
		c.Log.Encoding = "console"
	}
	if c.JWT.Secret == "" {
		c.JWT.Secret = "your-secret-key-change-in-production"
	}
	if c.JWT.ExpireHour == 0 {
		c.JWT.ExpireHour = 24
	}
}

func resolveConfigPath() (string, error) {
	for _, p := range []string{"/app/config.yaml", "config.yaml", "./config.yaml"} {
		if _, err := os.Stat(p); err == nil {
			return p, nil
		}
	}
	return "", fmt.Errorf("未找到 config.yaml，请复制 config.example.yaml 为 config.yaml 并修改")
}

func loadFromFile(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read config: %w", err)
	}
	var c yamlConfig
	if err := yaml.Unmarshal(data, &c); err != nil {
		return fmt.Errorf("parse config: %w", err)
	}
	applyDefaults(&c)
	mu.Lock()
	cfg = c
	mu.Unlock()
	return nil
}

// LoadConfig 从 config.yaml 加载配置，并启动监听热更新
func LoadConfig() error {
	path, err := resolveConfigPath()
	if err != nil {
		return err
	}
	cfgPath = path
	if err := loadFromFile(cfgPath); err != nil {
		return err
	}
	go watchConfig()
	return nil
}

func watchConfig() {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Printf("config watcher create failed: %v", err)
		return
	}
	defer watcher.Close()

	dir := filepath.Dir(cfgPath)
	if err := watcher.Add(dir); err != nil {
		log.Printf("config watcher add dir %s: %v", dir, err)
		return
	}

	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				return
			}
			if event.Has(fsnotify.Write) && filepath.Clean(event.Name) == filepath.Clean(cfgPath) {
				if err := loadFromFile(cfgPath); err != nil {
					log.Printf("config reload failed: %v", err)
				} else {
					log.Printf("config reloaded from %s", cfgPath)
				}
			}
		case err, ok := <-watcher.Errors:
			if !ok {
				return
			}
			log.Printf("config watcher error: %v", err)
		}
	}
}

// Server
func ServerHost() string { mu.RLock(); defer mu.RUnlock(); return cfg.Server.Host }
func ServerPort() string { mu.RLock(); defer mu.RUnlock(); return strconv.Itoa(cfg.Server.Port) }
func ServerMode() string { mu.RLock(); defer mu.RUnlock(); return cfg.Server.Mode }

// Database
func DBHost() string     { mu.RLock(); defer mu.RUnlock(); return cfg.Database.Host }
func DBPort() string     { mu.RLock(); defer mu.RUnlock(); return strconv.Itoa(cfg.Database.Port) }
func DBUser() string     { mu.RLock(); defer mu.RUnlock(); return cfg.Database.User }
func DBPassword() string { mu.RLock(); defer mu.RUnlock(); return cfg.Database.Password }
func DBName() string     { mu.RLock(); defer mu.RUnlock(); return cfg.Database.Name }
func DBSSLMode() string  { mu.RLock(); defer mu.RUnlock(); return cfg.Database.SSLMode }
func DBTimezone() string { mu.RLock(); defer mu.RUnlock(); return cfg.Database.Timezone }

// Redis
func RedisHost() string     { mu.RLock(); defer mu.RUnlock(); return cfg.Redis.Host }
func RedisPort() string     { mu.RLock(); defer mu.RUnlock(); return strconv.Itoa(cfg.Redis.Port) }
func RedisPassword() string { mu.RLock(); defer mu.RUnlock(); return cfg.Redis.Password }
func RedisDB() string       { mu.RLock(); defer mu.RUnlock(); return strconv.Itoa(cfg.Redis.DB) }

// Log
func LogLevel() string    { mu.RLock(); defer mu.RUnlock(); return cfg.Log.Level }
func LogEncoding() string { mu.RLock(); defer mu.RUnlock(); return cfg.Log.Encoding }

// JWT
func JWTSecret() string  { mu.RLock(); defer mu.RUnlock(); return cfg.JWT.Secret }
func JWTExpireHour() int { mu.RLock(); defer mu.RUnlock(); return cfg.JWT.ExpireHour }
