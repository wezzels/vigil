// Package opir provides configuration loading for OPIR feeds
package opir

import (
	"encoding/json"
	"os"
	"time"
)

// LoadConfig loads configuration from a JSON file
func LoadConfig(path string) (*OPIRConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	
	config := DefaultConfig()
	if err := json.Unmarshal(data, config); err != nil {
		return nil, err
	}
	
	return config, nil
}

// LoadConfigFromEnv loads configuration from environment variables
func LoadConfigFromEnv() *OPIRConfig {
	config := DefaultConfig()
	
	// Endpoints
	if endpoints := os.Getenv("OPIR_ENDPOINTS"); endpoints != "" {
		var list []string
		if err := json.Unmarshal([]byte(endpoints), &list); err == nil {
			config.Endpoints = list
		}
	}
	
	// Port
	if port := os.Getenv("OPIR_PORT"); port != "" {
		if p, err := parseInt(port); err == nil {
			config.Port = p
		}
	}
	
	// Protocol
	if protocol := os.Getenv("OPIR_PROTOCOL"); protocol != "" {
		config.Protocol = protocol
	}
	
	// Authentication
	if username := os.Getenv("OPIR_USERNAME"); username != "" {
		config.Username = username
	}
	if password := os.Getenv("OPIR_PASSWORD"); password != "" {
		config.Password = password
	}
	if certFile := os.Getenv("OPIR_CERT_FILE"); certFile != "" {
		config.CertFile = certFile
	}
	if keyFile := os.Getenv("OPIR_KEY_FILE"); keyFile != "" {
		config.KeyFile = keyFile
	}
	if caFile := os.Getenv("OPIR_CA_FILE"); caFile != "" {
		config.CAFile = caFile
	}
	
	// Timeouts
	if timeout := os.Getenv("OPIR_CONNECT_TIMEOUT"); timeout != "" {
		if d, err := time.ParseDuration(timeout); err == nil {
			config.ConnectTimeout = d
		}
	}
	if timeout := os.Getenv("OPIR_READ_TIMEOUT"); timeout != "" {
		if d, err := time.ParseDuration(timeout); err == nil {
			config.ReadTimeout = d
		}
	}
	if timeout := os.Getenv("OPIR_WRITE_TIMEOUT"); timeout != "" {
		if d, err := time.ParseDuration(timeout); err == nil {
			config.WriteTimeout = d
		}
	}
	
	// Retry
	if retries := os.Getenv("OPIR_MAX_RETRIES"); retries != "" {
		if r, err := parseInt(retries); err == nil {
			config.MaxRetries = r
		}
	}
	if delay := os.Getenv("OPIR_RETRY_DELAY"); delay != "" {
		if d, err := time.ParseDuration(delay); err == nil {
			config.RetryDelay = d
		}
	}
	
	// Buffering
	if size := os.Getenv("OPIR_BUFFER_SIZE"); size != "" {
		if s, err := parseInt(size); err == nil {
			config.BufferSize = s
		}
	}
	if size := os.Getenv("OPIR_BATCH_SIZE"); size != "" {
		if s, err := parseInt(size); err == nil {
			config.BatchSize = s
		}
	}
	
	// Validation
	if conf := os.Getenv("OPIR_MIN_CONFIDENCE"); conf != "" {
		if c, err := parseFloat(conf); err == nil {
			config.MinConfidence = c
		}
	}
	if snr := os.Getenv("OPIR_MIN_SNR"); snr != "" {
		if s, err := parseFloat(snr); err == nil {
			config.MinSNR = s
		}
	}
	if alt := os.Getenv("OPIR_MAX_ALTITUDE"); alt != "" {
		if a, err := parseFloat(alt); err == nil {
			config.MaxAltitude = a
		}
	}
	
	// Processing
	if enable := os.Getenv("OPIR_ENABLE_FILTERING"); enable != "" {
		config.EnableFiltering = enable == "true" || enable == "1"
	}
	if window := os.Getenv("OPIR_DEDUPE_WINDOW"); window != "" {
		if d, err := time.ParseDuration(window); err == nil {
			config.DedupeWindow = d
		}
	}
	
	return config
}

// SaveConfig saves configuration to a JSON file
func SaveConfig(config *OPIRConfig, path string) error {
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}
	
	return os.WriteFile(path, data, 0644)
}

// MergeConfigs merges multiple configurations (later configs override earlier)
func MergeConfigs(configs ...*OPIRConfig) *OPIRConfig {
	result := DefaultConfig()
	
	for _, config := range configs {
		if len(config.Endpoints) > 0 {
			result.Endpoints = config.Endpoints
		}
		if config.Port > 0 {
			result.Port = config.Port
		}
		if config.Protocol != "" {
			result.Protocol = config.Protocol
		}
		if config.Username != "" {
			result.Username = config.Username
		}
		if config.Password != "" {
			result.Password = config.Password
		}
		if config.CertFile != "" {
			result.CertFile = config.CertFile
		}
		if config.KeyFile != "" {
			result.KeyFile = config.KeyFile
		}
		if config.CAFile != "" {
			result.CAFile = config.CAFile
		}
		if config.ConnectTimeout > 0 {
			result.ConnectTimeout = config.ConnectTimeout
		}
		if config.ReadTimeout > 0 {
			result.ReadTimeout = config.ReadTimeout
		}
		if config.WriteTimeout > 0 {
			result.WriteTimeout = config.WriteTimeout
		}
		if config.KeepAlive > 0 {
			result.KeepAlive = config.KeepAlive
		}
		if config.MaxRetries > 0 {
			result.MaxRetries = config.MaxRetries
		}
		if config.RetryDelay > 0 {
			result.RetryDelay = config.RetryDelay
		}
		if config.MaxRetryDelay > 0 {
			result.MaxRetryDelay = config.MaxRetryDelay
		}
		if config.BufferSize > 0 {
			result.BufferSize = config.BufferSize
		}
		if config.BatchSize > 0 {
			result.BatchSize = config.BatchSize
		}
		if config.BatchTimeout > 0 {
			result.BatchTimeout = config.BatchTimeout
		}
		if config.MinConfidence > 0 {
			result.MinConfidence = config.MinConfidence
		}
		if config.MinSNR > 0 {
			result.MinSNR = config.MinSNR
		}
		if config.MaxAltitude > 0 {
			result.MaxAltitude = config.MaxAltitude
		}
	}
	
	return result
}

// Helper functions

func parseInt(s string) (int, error) {
	var result int
	err := json.Unmarshal([]byte(s), &result)
	return result, err
}

func parseFloat(s string) (float64, error) {
	var result float64
	err := json.Unmarshal([]byte(s), &result)
	return result, err
}