package config

import (
	"log"
	"os"
)

const sslModeDisable = "sslmode=disable"

type Config struct {
	ServerPort string

	DBHost     string
	DBPort     string
	DBUser     string
	DBPassword string
	DBName     string

	JWTSecret string

	KafkaBrokers            string
	KafkaTopicBookingEvents string
}

func Load() Config {
	return Config{
		ServerPort: optEnv("BOOKING_SERVER_PORT", "8080"),

		DBHost:     mustEnv("BOOKING_DATABASE_HOST"),
		DBPort:     optEnv("BOOKING_DATABASE_PORT", "5432"),
		DBUser:     mustEnv("BOOKING_DATABASE_USER"),
		DBPassword: mustEnv("BOOKING_DATABASE_PASSWORD"),
		DBName:     mustEnv("BOOKING_DATABASE_NAME"),

		JWTSecret: mustEnv("JWT_SECRET"),

		KafkaBrokers:            mustEnv("KAFKA_BROKERS"),
		KafkaTopicBookingEvents: optEnv("KAFKA_TOPIC_BOOKING_EVENTS", "booking.events"),
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

func mustEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		log.Fatalf("required environment variable %q is not set", key)
	}
	return v
}

func optEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
