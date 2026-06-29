package config

import "os"

const (
	defaultServerPort = "8081"

	defaultDBHost     = "notification-db"
	defaultDBPort     = "5432"
	defaultDBUser     = "postgres"
	defaultDBPassword = "password"
	defaultDBName     = "notifications"

	// Переменные окружения используют NOTIFICATION_ префикс,
	// чтобы не конфликтовать с DATABASE_* booking-service в одном .env.


	defaultJWTSecret = "supersecretkey"

	defaultKafkaBrokers            = "kafka:9092"
	defaultKafkaTopicBookingEvents = "booking.events"
	defaultKafkaGroupID            = "notification-service"

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

	KafkaBrokers            string
	KafkaTopicBookingEvents string
	KafkaGroupID            string
}

func Load() Config {
	return Config{
		ServerPort: getEnv("SERVER_PORT", defaultServerPort),

		DBHost:     getEnv("NOTIFICATION_DATABASE_HOST", defaultDBHost),
		DBPort:     getEnv("NOTIFICATION_DATABASE_PORT", defaultDBPort),
		DBUser:     getEnv("NOTIFICATION_DATABASE_USER", defaultDBUser),
		DBPassword: getEnv("NOTIFICATION_DATABASE_PASSWORD", defaultDBPassword),
		DBName:     getEnv("NOTIFICATION_DATABASE_NAME", defaultDBName),

		JWTSecret: getEnv("JWT_SECRET", defaultJWTSecret),

		KafkaBrokers:            getEnv("KAFKA_BROKERS", defaultKafkaBrokers),
		KafkaTopicBookingEvents: getEnv("KAFKA_TOPIC_BOOKING_EVENTS", defaultKafkaTopicBookingEvents),
		KafkaGroupID:            getEnv("KAFKA_GROUP_ID", defaultKafkaGroupID),
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
