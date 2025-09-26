package lynksearch

import "github.com/lynkdb/lynkapi/go/lynkapi"

type Config struct {
	HttpPort int `toml:"http_port" json:"http_port"`
	GrpcPort int `toml:"grpc_port" json:"grpc_port"`

	Indexes []*IndexConfig `toml:"indexes" json:"indexes"`
}

type IndexConfig struct {
	Name string `toml:"name" json:"name"`

	Spec *lynkapi.TableSpec `toml:"spec" json:"spec"`
}
