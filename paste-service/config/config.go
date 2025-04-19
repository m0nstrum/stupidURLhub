package config

import (
	"os"
	"reflect"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	Cache    CacheConfig
	Tagger   TaggerConfig
	SlugGen  SlugGenConfig
}

type ServerConfig struct {
	Port            string
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	ShutdownTimeout time.Duration
	MaxRequestSize  int64
	RateLimit       int
	TestMode        bool
}

type DatabaseConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	DBName   string
	SSLMode  string
}

type CacheConfig struct {
	Type            string
	RedisURL        string
	DefaultTTL      time.Duration
	GCInterval      time.Duration
	RefreshTTLOnGet bool
}

type TaggerConfig struct {
	BaseURL     string
	Timeout     time.Duration
	MaxTextSize int
}

type SlugGenConfig struct {
	Address     string
	Timeout     time.Duration
	MaxTextSize int
}

func DefaultConfig() *Config {
	return &Config{
		Server: ServerConfig{
			Port:            "8080",
			ReadTimeout:     10 * time.Second,
			WriteTimeout:    10 * time.Second,
			ShutdownTimeout: 5 * time.Second,
			MaxRequestSize:  1024 * 1024 * 5,
			RateLimit:       100,
			TestMode:        false,
		},
		Database: DatabaseConfig{
			Host:     "localhost",
			Port:     "5432",
			User:     "postgres",
			Password: "postgres",
			DBName:   "paste_service",
			SSLMode:  "disable",
		},
		Cache: CacheConfig{
			Type:            "inmemory",
			RedisURL:        "redis://localhost:6379/0",
			DefaultTTL:      10 * time.Minute,
			GCInterval:      1 * time.Minute,
			RefreshTTLOnGet: true,
		},
		Tagger: TaggerConfig{
			BaseURL:     "http://tagger-ml:8000",
			Timeout:     5 * time.Second,
			MaxTextSize: 10000, // 10KB
		},
		SlugGen: SlugGenConfig{
			Address:     "slug-generator:50051",
			Timeout:     5 * time.Second,
			MaxTextSize: 10000, // 10KB
		},
	}
}

func LoadFromEnv() *Config {
	cfg := DefaultConfig()
	loadEnvToStruct(cfg)
	return cfg
}

func loadEnvToStruct(cfg interface{}) {
	v := reflect.ValueOf(cfg).Elem()
	loadEnvToValue("", v)
}

func loadEnvToValue(prefix string, v reflect.Value) {
	t := v.Type()

	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		fieldType := t.Field(i)
		fieldName := fieldType.Name

		if !field.CanSet() {
			continue
		}

		var envName string
		if prefix == "" {
			envName = strings.ToUpper(fieldName)
		} else {
			envName = prefix + "_" + strings.ToUpper(fieldName)
		}

		if field.Kind() == reflect.Struct {
			loadEnvToValue(envName, field)
			continue
		}

		value := os.Getenv(envName)
		if value == "" {
			continue
		}

		setValueFromEnv(field, value)
	}
}

func setValueFromEnv(field reflect.Value, value string) {
	switch field.Kind() {
	case reflect.String:
		field.SetString(value)

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		if field.Type().String() == "time.Duration" {
			if duration, err := time.ParseDuration(value); err == nil {
				field.SetInt(int64(duration))
			}
		} else {
			if intVal, err := strconv.ParseInt(value, 10, 64); err == nil {
				field.SetInt(intVal)
			}
		}

	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		if uintVal, err := strconv.ParseUint(value, 10, 64); err == nil {
			field.SetUint(uintVal)
		}

	case reflect.Float32, reflect.Float64:
		if floatVal, err := strconv.ParseFloat(value, 64); err == nil {
			field.SetFloat(floatVal)
		}

	case reflect.Bool:
		field.SetBool(strings.ToLower(value) == "true")

	case reflect.Slice:
		if field.Type().Elem().Kind() == reflect.String {
			values := strings.Split(value, ",")
			slice := reflect.MakeSlice(field.Type(), len(values), len(values))
			for i, v := range values {
				slice.Index(i).SetString(strings.TrimSpace(v))
			}
			field.Set(slice)
		}
	}
}
