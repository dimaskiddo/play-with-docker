package config

import (
	"errors"
	"os"
	"strconv"
	"strings"

	_ "github.com/joho/godotenv/autoload"
)

func SanitizeEnv(envName string) (string, error) {
	if len(envName) == 0 {
		return "", errors.New("Environment Variable Name Should Not Empty")
	}

	retValue := strings.TrimSpace(os.Getenv(envName))
	if len(retValue) == 0 {
		return "", errors.New("Environment Variable '" + envName + "' Has an Empty Value")
	}

	return retValue, nil
}

func GetEnvString(envName string, envDefault string) string {
	envValue, err := SanitizeEnv(envName)
	if err != nil {
		return envDefault
	}

	return envValue
}

func GetEnvBool(envName string, envDefault bool) bool {
	envValue, err := SanitizeEnv(envName)
	if err != nil {
		return envDefault
	}

	retValue, err := strconv.ParseBool(envValue)
	if err != nil {
		return envDefault
	}

	return retValue
}

func GetEnvInt(envName string, envDefault int) int {
	envValue, err := SanitizeEnv(envName)
	if err != nil {
		return envDefault
	}

	retValue, err := strconv.ParseInt(envValue, 0, 0)
	if err != nil {
		return envDefault
	}

	return int(retValue)
}

func GetEnvFloat32(envName string, envDefault float32) float32 {
	envValue, err := SanitizeEnv(envName)
	if err != nil {
		return envDefault
	}

	retValue, err := strconv.ParseFloat(envValue, 32)
	if err != nil {
		return envDefault
	}

	return float32(retValue)
}

func GetEnvFloat64(envName string, envDefault float64) float64 {
	envValue, err := SanitizeEnv(envName)
	if err != nil {
		return envDefault
	}

	retValue, err := strconv.ParseFloat(envValue, 64)
	if err != nil {
		return envDefault
	}

	return float64(retValue)
}
