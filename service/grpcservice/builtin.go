package grpcservice

import (
	"context"
	"net"

	"github.com/libs4go/scf4go"
	"github.com/libs4go/slf4go"
)

type builtinProvider struct {
	slf4go.Logger
	config   scf4go.Config
	listener net.Listener
}

func newBuiltinProvider(config scf4go.Config) (Provider, error) {

	listener, err := net.Listen("tcp", config.Get("laddr").String(":8080"))

	if err != nil {
		return nil, err
	}

	return &builtinProvider{
		Logger:   slf4go.Get("grpservice.default"),
		listener: listener,
	}, nil
}

func (provider *builtinProvider) Listener() net.Listener {
	return provider.listener
}

func (provider *builtinProvider) Connect(ctx context.Context, remote string) (net.Conn, error) {
	provider.D("try connect to {@remote}", remote)
	return net.Dial("tcp", remote)
}
