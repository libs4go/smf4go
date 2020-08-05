package main

import (
	"fmt"

	"github.com/libs4go/scf4go"
	_ "github.com/libs4go/scf4go/codec"
	"github.com/libs4go/slf4go"
	_ "github.com/libs4go/slf4go/backend/console"
	_ "github.com/libs4go/slf4go/filter/cached"
	"github.com/libs4go/smf4go"
	"github.com/libs4go/smf4go/app"
	"github.com/libs4go/smf4go/service/grpcservice"
	"google.golang.org/grpc"
)

var logger = slf4go.Get("test")

type grpcClient struct {
}

type grpcServer struct {
	Client grpcservice.Client `inject:"test.grpc"`
}

func (s *grpcServer) GrpcHandler(server *grpc.Server) error {
	logger.D("bind grpc server {@client}", fmt.Sprintf("%p", s.Client))

	return nil
}

func main() {
	grpcService := grpcservice.New("test.grpc")

	grpcService.Local("test.grpc.server", func(config scf4go.Config) (grpcservice.Service, error) {
		return &grpcServer{}, nil
	})

	grpcService.Remote("test.grpc.client", func(conn *grpc.ClientConn) (smf4go.Service, error) {
		return &grpcClient{}, nil
	})

	app.Run("test")
}
