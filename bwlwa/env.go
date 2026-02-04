package bwlwa

import (
	"fmt"

	"github.com/caarlos0/env/v11"
	"go.uber.org/zap/zapcore"
)

// Environment defines the interface that all environment configurations must implement.
// Embed BaseEnvironment in your struct to satisfy this interface.
type Environment interface {
	port() int
	serviceName() string
	readinessCheckPath() string
	logLevel() zapcore.Level
	otelExporter() string
}

// BaseEnvironment contains the required LWA environment variables.
// Embed this in your custom environment struct.
type BaseEnvironment struct {
	Port               int           `env:"AWS_LWA_PORT,required"`
	ServiceName        string        `env:"SERVICE_NAME,required"`
	ReadinessCheckPath string        `env:"AWS_LWA_READINESS_CHECK_PATH,required"`
	LogLevel           zapcore.Level `env:"LOG_LEVEL" envDefault:"info"`
	OtelExporter       string        `env:"OTEL_EXPORTER" envDefault:"stdout"`
}

func (e BaseEnvironment) port() int {
	return e.Port
}
func (e BaseEnvironment) serviceName() string {
	return e.ServiceName
}
func (e BaseEnvironment) readinessCheckPath() string {
	return e.ReadinessCheckPath
}
func (e BaseEnvironment) logLevel() zapcore.Level {
	return e.LogLevel
}
func (e BaseEnvironment) otelExporter() string {
	return e.OtelExporter
}

var _ Environment = BaseEnvironment{}

// ParseEnv parses environment variables into the given Environment type.
func ParseEnv[E Environment]() func() (E, error) {
	return func() (e E, err error) {
		if err := env.Parse(&e); err != nil {
			return e, fmt.Errorf("failed to parse environment: %w", err)
		}
		return e, nil
	}
}
