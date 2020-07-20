package smf4go

import (
	"sync"
	"sync/atomic"

	"github.com/libs4go/errors"
	"github.com/libs4go/scf4go"
	"github.com/libs4go/sdi4go"
	"github.com/libs4go/slf4go"
)

// ScopeOfAPIError .
const errVendor = "smf4go"

// errors
var (
	ErrInternal = errors.New("the internal error", errors.WithVendor(errVendor))
	ErrAgent    = errors.New("agent implement not found", errors.WithVendor(errVendor))
	ErrExists   = errors.New("target resource exists", errors.WithVendor(errVendor))
	ErrNotFound = errors.New("target resource not found", errors.WithVendor(errVendor))
)

// Service smf4go service base interface has nothing
type Service interface{}

// Runnable smf4go service base interface has nothing
type Runnable interface {
	Service
	Start() error
}

// ServiceRegisterEntry .
type ServiceRegisterEntry struct {
	Name    string  // service name
	Service Service // service impl
}

// MeshBuilder .
type MeshBuilder interface {
	RegisterService(extensionName string, serviceName string) error
	RegisterExtension(extension Extension) error
	Start(config scf4go.Config) error
	FindService(name string, service interface{})
}

// Extension smf4go service handle extension
type Extension interface {
	Name() string // extension name
	Begin(config scf4go.Config, builder MeshBuilder) error
	CreateSerivce(serviceName string, config scf4go.Config) (Service, error)
	End() error
}

type meshBuilderImpl struct {
	slf4go.Logger                        // mixin logger
	injector        sdi4go.Injector      // injector context
	registers       map[string]string    // registers services
	orderServices   []string             //order service name
	extensions      map[string]Extension // extensions
	orderExtensions []Extension          // order extension names
	started         atomic.Value         // started
}

// NewMeshBuilder create new mesh builder
func NewMeshBuilder() MeshBuilder {
	impl := &meshBuilderImpl{
		Logger:     slf4go.Get("smf4go"),
		registers:  make(map[string]string),
		extensions: make(map[string]Extension),
		injector:   sdi4go.New(),
	}

	impl.started.Store(false)

	return impl
}

func (builder *meshBuilderImpl) RegisterService(extensionName string, serviceName string) error {

	_, ok := builder.registers[serviceName]

	if ok {
		return errors.Wrap(ErrExists, "service %s exists", serviceName)
	}

	if _, ok := builder.extensions[extensionName]; !ok {
		return errors.Wrap(ErrNotFound, "extension %s not found", extensionName)
	}

	builder.registers[serviceName] = extensionName
	builder.orderServices = append(builder.orderServices, serviceName)

	return nil
}

func (builder *meshBuilderImpl) RegisterExtension(extension Extension) error {

	_, ok := builder.extensions[extension.Name()]

	if ok {
		return errors.Wrap(ErrExists, "extension %s exists", extension.Name())
	}

	builder.extensions[extension.Name()] = extension
	builder.orderExtensions = append(builder.orderExtensions, extension)

	return nil
}

func (builder *meshBuilderImpl) FindService(name string, service interface{}) {

	if !builder.started.Load().(bool) {
		builder.D("mesh builder processing, FindService not valid")
		return
	}

	builder.injector.Create(name, service)
}

func (builder *meshBuilderImpl) Start(config scf4go.Config) error {

	for _, extension := range builder.extensions {
		subconfig := config.SubConfig("smf4go", "extension", extension.Name())

		builder.D("call extension {@ext} initialize routine", extension.Name())

		if err := extension.Begin(subconfig, builder); err != nil {
			return errors.Wrap(err, "start extension %s error", extension.Name())
		}

		builder.D("call extension {@ext} initialize routine -- success", extension.Name())
	}

	var services []ServiceRegisterEntry

	for _, serviceName := range builder.orderServices {
		subconfig := config.SubConfig("smf4go", "service", serviceName)

		extension := builder.extensions[builder.registers[serviceName]]

		builder.D("create service {@service} by extension {@ext}", serviceName, extension.Name())

		service, err := extension.CreateSerivce(serviceName, subconfig)

		if err != nil {
			return errors.Wrap(err, "create service %s by extension %s error", serviceName, extension.Name())
		}

		builder.D("create service {@service} by extension {@ext} -- success", serviceName, extension.Name())

		services = append(services, ServiceRegisterEntry{Name: serviceName, Service: service})
	}

	for _, entry := range services {
		builder.injector.Bind(entry.Name, sdi4go.Singleton(entry.Service))
	}

	for _, entry := range services {

		builder.D("bind service {@service}", entry.Name)

		if err := builder.injector.Inject(entry.Service); err != nil {
			return errors.Wrap(err, "service %s bind error", entry.Name)
		}

		builder.D("bind service {@service} -- success", entry.Name)
	}

	for _, extension := range builder.extensions {

		builder.D("call extension {@ext} finally routine", extension.Name())

		if err := extension.End(); err != nil {
			return errors.Wrap(err, "extension %s finally routine error", extension.Name())
		}

		builder.D("call extension {@ext} finally routine -- success", extension.Name())
	}

	for _, entry := range services {
		if runnable, ok := entry.Service.(Runnable); ok {
			builder.D("start runnable service {@service}", entry.Name)
			if err := runnable.Start(); err != nil {
				return errors.Wrap(err, "start service %s error", entry.Name)
			}
			builder.D("start runnable service {@service} -- success", entry.Name)
		}
	}

	builder.started.Store(true)

	return nil
}

var meshBuilder MeshBuilder
var once sync.Once

// Builder get mesh builder instance
func Builder() MeshBuilder {
	once.Do(func() {
		meshBuilder = NewMeshBuilder()
	})

	return meshBuilder
}
