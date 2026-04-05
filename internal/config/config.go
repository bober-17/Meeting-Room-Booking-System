package config

import "os"

const (
	defaultServerPort = "8080"

	defaultDBHost     = "db"
	defaultDBPort     = "5432"
	defaultDBUser     = "postgres"
	defaultDBPassword = "password"
	deaultDBName      = "booking"

	defaultJWTSecret = "supersecretkey"

	sslModeDisable = "sslmode=disable"
)

type Config struct {
	ServerPort string

	DBHost     string
	DBPort     string
	DBUser     string
	DBPassword string
	DBName     string

	JWTSecret string
}

func Load() Config {
	return Config{
		ServerPort: getEnv("SERVER_PORT", defaultServerPort),

		DBHost:     getEnv("DATABASE_HOST", defaultDBHost),
		DBPort:     getEnv("DATABASE_PORT", defaultDBPort),
		DBUser:     getEnv("DATABASE_USER", defaultDBUser),
		DBPassword: getEnv("DATABASE_PASSWORD", defaultDBPassword),
		DBName:     getEnv("DATABASE_NAME", deaultDBName),

		JWTSecret: getEnv("JWT_SECRET", defaultJWTSecret),
	}
}

func (c Config) DSN() string {
	return "host=" + c.DBHost +
		" port=" + c.DBPort +
		" user=" + c.DBUser +
		" password=" + c.DBPassword +
		" dbname=" + c.DBName +
		" " + sslModeDisable
}

func (c Config) MigrateDSN() string {
	return "pgx5://" + c.DBUser + ":" + c.DBPassword +
		"@" + c.DBHost + ":" + c.DBPort + "/" + c.DBName + "?" + sslModeDisable
}

func getEnv(key, defaultValue string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultValue
}
