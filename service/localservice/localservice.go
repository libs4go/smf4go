package localservice

import (
	"sync"

	"github.com/libs4go/errors"
	"github.com/libs4go/scf4go"
	"github.com/libs4go/smf4go"
)

// F .
type F func(config scf4go.Config) (smf4go.Service, error)

// LocalService .
type LocalService interface {
	Register(name string, f F)
}

type registerEntry struct {
	Name string
	F    F
}

type localServiceExtension struct {
	creators map[string]F
	orders   []registerEntry
}

func newExtension() *localServiceExtension {
	return &localServiceExtension{
		creators: make(map[string]F),
	}
}

func (extension *localServiceExtension) Register(name string, f F) {
	extension.creators[name] = f
	extension.orders = append(extension.orders, registerEntry{
		Name: name,
		F:    f,
	})
}

func (extension *localServiceExtension) Name() string {
	return "smf4go.extension.local"
}

func (extension *localServiceExtension) Begin(config scf4go.Config, builder smf4go.MeshBuilder) error {

	for _, entry := range extension.orders {
		builder.RegisterService(extension.Name(), entry.Name)
	}

	return nil
}

func (extension *localServiceExtension) CreateSerivce(serviceName string, config scf4go.Config) (smf4go.Service, error) {
	f, ok := extension.creators[serviceName]

	if !ok {
		return nil, errors.Wrap(smf4go.ErrNotFound, "service %s not found", serviceName)
	}

	return f(config)
}

func (extension *localServiceExtension) End() error {
	return nil
}

var extension *localServiceExtension
var once sync.Once

// Get get sigleton LocalService
func Get() LocalService {
	once.Do(func() {
		extension = newExtension()
		smf4go.Builder().RegisterExtension(extension)
	})

	return extension
}

// Register .
func Register(name string, f F) {
	Get().Register(name, f)
}

// New create LocalService with provider smf4go.MeshBuilder
func New(builder smf4go.MeshBuilder) LocalService {
	extension := newExtension()
	builder.RegisterExtension(extension)

	return extension
}
