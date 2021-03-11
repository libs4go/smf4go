package grpcservice

import (
	"context"
	"net"
	"time"

	"github.com/libs4go/errors"
	"github.com/libs4go/scf4go"
	"github.com/libs4go/slf4go"
	"github.com/libs4go/smf4go"
	"github.com/libs4go/smf4go/service/localservice"
	"google.golang.org/grpc"
)

// Provider .
type Provider interface {
	Listener() net.Listener
	Connect(ctx context.Context, remote string) (net.Conn, error)
}

// Service .
type Service interface {
	smf4go.Service
	GrpcHandler(server *grpc.Server) error
}

// CreatorF .
type CreatorF func(config scf4go.Config) (Service, error)

// ConnectorF .
type ConnectorF func(conn *grpc.ClientConn) (smf4go.Service, error)

// Register .
type Register interface {
	Client
	Local(name string, creator CreatorF)
	Remote(name string, connector ConnectorF)
}

// Client .
type Client interface {
	Dial(ctx context.Context, url string, dialOpts ...grpc.DialOption) (*grpc.ClientConn, error)
}

type registerImpl struct {
	slf4go.Logger
	provider     string // provider serivce name
	local        map[string]CreatorF
	remote       map[string]ConnectorF
	server       *grpc.Server
	config       scf4go.Config
	servces      []Service
	meshBulder   smf4go.MeshBuilder
	localservice localservice.LocalService
}

// Option .
type Option func(*registerImpl)

// WithProvider .
func WithProvider(name string) Option {
	return func(register *registerImpl) {
		register.provider = name
	}
}

// WithMeshBuilder .
func WithMeshBuilder(builder smf4go.MeshBuilder) Option {
	return func(register *registerImpl) {
		register.meshBulder = builder
	}
}

// WithLocalService .
func WithLocalService(localservice localservice.LocalService) Option {
	return func(register *registerImpl) {
		register.localservice = localservice
	}
}

// New .
func New(name string, options ...Option) Register {

	providerName := "grpcservice.default"

	impl := &registerImpl{
		Logger:     slf4go.Get("mxwservice"),
		local:      make(map[string]CreatorF),
		remote:     make(map[string]ConnectorF),
		provider:   providerName,
		meshBulder: smf4go.Builder(),
	}

	for _, option := range options {
		option(impl)
	}

	if impl.meshBulder == smf4go.Builder() {
		impl.localservice = localservice.Get()
	} else if impl.localservice == nil {
		impl.localservice = localservice.New(impl.meshBulder)
	}

	if impl.provider == providerName {
		impl.localservice.Register(providerName, func(config scf4go.Config) (smf4go.Service, error) {
			return newBuiltinProvider(config)
		})
	}

	impl.meshBulder.RegisterExtension(impl)

	localservice.Register(name, func(config scf4go.Config) (smf4go.Service, error) {
		impl.config = config
		return impl, nil
	})

	return impl
}

func (extension *registerImpl) Start() error {

	go func() {
		for {
			if err := extension.server.Serve(extension); err != nil {
				extension.E("grpc serve err {@err}", err)
			}

			time.Sleep(extension.config.Get("backoff").Duration(time.Second * 5))
		}
	}()

	return nil
}

func (extension *registerImpl) Name() string {
	return "smf4go.extension.mxwservice"
}

// Accept waits for and returns the next connection to the listener.
func (extension *registerImpl) Accept() (net.Conn, error) {
	for {
		provider := extension.getProvider()

		if provider != nil {
			return provider.Listener().Accept()
		}

		time.Sleep(time.Second)
	}
}

// Close closes the listener.
// Any blocked Accept operations will be unblocked and return errors.
func (extension *registerImpl) Close() error {
	provider := extension.getProvider()

	if provider != nil {
		return provider.Listener().Close()
	}

	return nil
}

func fakeLocalAddr() net.Addr {
	localIP := net.ParseIP("127.0.0.1")
	return &net.TCPAddr{IP: localIP, Port: 0}
}

// Addr returns the listener's network address.
func (extension *registerImpl) Addr() net.Addr {
	return fakeLocalAddr()
}

func (extension *registerImpl) Begin(config scf4go.Config, builder smf4go.MeshBuilder) error {

	for name := range extension.local {
		builder.RegisterService(extension.Name(), name)
	}

	for name := range extension.remote {
		builder.RegisterService(extension.Name(), name)
	}

	extension.server = grpc.NewServer()

	return nil
}

func (extension *registerImpl) CreateSerivce(serviceName string, config scf4go.Config) (smf4go.Service, error) {
	f, ok := extension.local[serviceName]

	if ok {
		service, err := f(config)

		if err != nil {
			return nil, err
		}

		grpcService, ok := service.(Service)

		if ok {
			extension.servces = append(extension.servces, grpcService)
		}

		return service, nil

	}

	f2, ok := extension.remote[serviceName]

	if ok {

		remote := config.Get("remote").String("")

		extension.D("[{@serviceName}] grpc dial to {@remote}", serviceName, remote)

		conn, err := extension.Dial(context.Background(), remote, grpc.WithInsecure())

		if err != nil {
			return nil, err
		}

		return f2(conn)
	}

	return nil, errors.Wrap(smf4go.ErrNotFound, "service %s not found", serviceName)
}

func (extension *registerImpl) getProvider() Provider {
	var provider Provider
	smf4go.Builder().FindService(extension.provider, &provider)

	return provider
}

func (extension *registerImpl) dialOption(ctx context.Context) grpc.DialOption {
	return grpc.WithDialer(func(remote string, timeout time.Duration) (net.Conn, error) {

		provider := extension.getProvider()

		if provider == nil {
			extension.D("grpc provider not exists ...")
			return nil, errors.New("grpc provider not valid")
		}

		subCtx, subCtxCancel := context.WithTimeout(ctx, timeout)
		defer subCtxCancel()

		conn, err := provider.Connect(subCtx, remote)

		if err != nil {
			extension.E("grpc dial to {@remote} error: {@err}", remote, err)
			return nil, err
		}

		return conn, nil
	})
}

func (extension *registerImpl) Dial(ctx context.Context, url string, dialOpts ...grpc.DialOption) (*grpc.ClientConn, error) {

	dialOpsPrepended := append([]grpc.DialOption{extension.dialOption(ctx)}, dialOpts...)

	return grpc.DialContext(ctx, url, dialOpsPrepended...)
}

func (extension *registerImpl) End() error {

	for _, service := range extension.servces {
		if err := service.GrpcHandler(extension.server); err != nil {
			return err
		}
	}

	return nil
}

func (extension *registerImpl) Local(name string, creator CreatorF) {
	extension.local[name] = creator
}

func (extension *registerImpl) Remote(name string, connector ConnectorF) {
	extension.remote[name] = connector
}
