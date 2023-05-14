package config

import (
	"fmt"
	"os"
)

type AppEnv struct {
	LogLvl string

	PgHost     string
	PgPort     string
	PgUser     string
	PgPassword string
	PgDbName   string
	SSLMode    string
	TimeZone   string

	AuthHost string
	AuthPort string

	SupervisorEmail        string
	SupervisorHashPassword string
}

func GetEnvironment() (env AppEnv, err error) {
	env = AppEnv{
		LogLvl:                 getEnv("LOG_LEVEL", "debug"),
		PgHost:                 getEnv("POSTGRES_HOST", ""),
		PgPort:                 getEnv("POSTGRES_PORT", ""),
		PgUser:                 getEnv("POSTGRES_USER", ""),
		PgPassword:             getEnv("POSTGRES_PASSWORD", ""),
		PgDbName:               getEnv("POSTGRES_DB", ""),
		SSLMode:                getEnv("POSTGRES_SLL_MODE", "disable"),
		TimeZone:               getEnv("POSTGRES_TIMEZONE", "Europe/Moscow"),
		AuthHost:               getEnv("AUTH_HOST", "localhost"),
		AuthPort:               getEnv("AUTH_PORT", ":8080"),
		SupervisorEmail:        getEnv("SUPERVISOR_EMAIL", ""),
		SupervisorHashPassword: getEnv("SUPERVISOR_HASHPASSWORD", ""),
	}

	if env.PgHost == "" || env.PgPort == "" || env.PgUser == "" ||
		env.PgPassword == "" || env.PgDbName == "" {
		return env, fmt.Errorf("incorrect environment params")
	}

	if env.SupervisorEmail == "" || env.SupervisorHashPassword == "" {
		return env, fmt.Errorf("incorrect environment params")
	}

	return env, nil
}

func getEnv(key string, defaultVal string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}

	return defaultVal
}
