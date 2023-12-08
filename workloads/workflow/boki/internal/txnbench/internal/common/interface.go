package common

type ReadOnlyData struct {
	ByteStream string `mapstructure:"ByteStream" json:"ByteStream"`
}

type WriteOnlyData struct {
	ByteStream string `mapstructure:"ByteStream" json:"ByteStream"`
}

type RPCInput struct {
	Function string
	Input    interface{}
}
