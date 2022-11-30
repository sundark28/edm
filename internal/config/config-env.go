package config

// Code generated by go generate; DO NOT EDIT.
// This file was generated with Python robot.

import "os"

func (cfg *Config) readOSEnv() {
	if v, ok := os.LookupEnv("EDM_DB_TYPE"); ok {
		cfg.DBType = v
	}
	if v, ok := os.LookupEnv("EDM_DB_NAME"); ok {
		cfg.DBName = v
	}
	if v, ok := os.LookupEnv("EDM_DB_HOST"); ok {
		cfg.DBHost = v
	}
	if v, ok := os.LookupEnv("EDM_DB_PORT"); ok {
		cfg.DBPort = v
	}
	if v, ok := os.LookupEnv("EDM_DB_USER"); ok {
		cfg.DBUser = v
	}
	if v, ok := os.LookupEnv("EDM_DB_PASSWORD"); ok {
		cfg.DBPassword = v
	}
	if v, ok := os.LookupEnv("EDM_REDIS_CONNECT"); ok {
		cfg.REDISConnect = v
	}
	if v, ok := os.LookupEnv("EDM_REDIS_PASSWORD"); ok {
		cfg.REDISPassword = v
	}
	if v, ok := os.LookupEnv("EDM_REDIS_FLUSH"); ok {
		cfg.REDISFlush = v
	}
	if v, ok := os.LookupEnv("EDM_SMTP_EMAIL"); ok {
		cfg.SMTPEmail = v
	}
	if v, ok := os.LookupEnv("EDM_SMTP_HOST"); ok {
		cfg.SMTPHost = v
	}
	if v, ok := os.LookupEnv("EDM_SMTP_PORT"); ok {
		cfg.SMTPPort = v
	}
	if v, ok := os.LookupEnv("EDM_SMTP_USER"); ok {
		cfg.SMTPUser = v
	}
	if v, ok := os.LookupEnv("EDM_SMTP_PASSWORD"); ok {
		cfg.SMTPPassword = v
	}

}
