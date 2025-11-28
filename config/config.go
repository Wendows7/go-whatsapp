package config

import (
	"github.com/joho/godotenv"
	logs "log"
)

func LoadEnv() map[string]string {
	envMap, err := godotenv.Read("app.env")
	if err != nil {
		logs.Fatal("Error loading app.env file")
	}
	return envMap
}

func ProtoString(s string) *string {
	return &s
}
