package client

type Config struct {
	rpcEndpoint string
}

func LoadConfig() *Config {
	return &Config{
		rpcEndpoint: "http://localhost:26657",
	}
}
