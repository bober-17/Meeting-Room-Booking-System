package config

import "os"

const sslModeDisable = "sslmode=disable"

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
		ServerPort: getEnv("SERVER_PORT", "8080"),

		DBHost:     getEnv("DATABASE_HOST", "db"),
		DBPort:     getEnv("DATABASE_PORT", "5432"),
		DBUser:     getEnv("DATABASE_USER", "postgres"),
		DBPassword: getEnv("DATABASE_PASSWORD", "password"),
		DBName:     getEnv("DATABASE_NAME", "booking"),

		JWTSecret: getEnv("JWT_SECRET", "supersecretkey"),
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
