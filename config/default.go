package config

import (
	"github.com/spf13/viper"
	"time"
)

type Config struct {
	DBUri                  string        `mapstructure:"MONGODB_LOCAL_URI"`
	RedisUri               string        `mapstructure:"REDIS_URI"`
	Port                   string        `mapstructure:"PORT"`
	AccessTokenPrivateKey  string        `mapstructure:"ACCESS_TOKEN_PRIVATE_KEY"`
	AccessTokenPublicKey   string        `mapstructure:"ACCESS_TOKEN_PUBLIC_KEY"`
	RefreshTokenPrivateKey string        `mapstructure:"REFRESH_TOKEN_PRIVATE_KEY"`
	RefreshTokenPublicKey  string        `mapstructure:"REFRESH_TOKEN_PUBLIC_KEY"`
	AccessTokenExpiresIn   time.Duration `mapstructure:"ACCESS_TOKEN_EXPIRED_IN"`
	RefreshTokenExpiresIn  time.Duration `mapstructure:"REFRESH_TOKEN_EXPIRED_IN"`
	AccessTokenMaxAge      int           `mapstructure:"ACCESS_TOKEN_MAXAGE"`
	RefreshTokenMaxAge     int           `mapstructure:"REFRESH_TOKEN_MAXAGE"`
}

// LoadConfig will load the .env variables
func LoadConfig(path string) (config Config, err error) {

	viper.AddConfigPath(path)  // path to look for the config file, in this case the current working dir
	viper.SetConfigType("env") // name of the config file (without extension)
	viper.SetConfigName("app") // REQUIRED if the config file does not have the extension in the name

	viper.AutomaticEnv()

	err = viper.ReadInConfig() // Find and read the config file
	if err != nil {
		return
	}

	err = viper.Unmarshal(&config) // Unmarshalling all or a specific value to a struct, map, etc.
	return
}
