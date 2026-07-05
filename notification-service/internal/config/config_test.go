package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestKafkaBrokersList(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []string
	}{
		{
			name:  "single broker",
			input: "kafka:9092",
			want:  []string{"kafka:9092"},
		},
		{
			name:  "multiple brokers no spaces",
			input: "kafka1:9092,kafka2:9092,kafka3:9092",
			want:  []string{"kafka1:9092", "kafka2:9092", "kafka3:9092"},
		},
		{
			name:  "spaces around commas trimmed",
			input: "kafka1:9092, kafka2:9092 , kafka3:9092",
			want:  []string{"kafka1:9092", "kafka2:9092", "kafka3:9092"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := Config{KafkaBrokers: tt.input}
			assert.Equal(t, tt.want, cfg.KafkaBrokersList())
		})
	}
}
