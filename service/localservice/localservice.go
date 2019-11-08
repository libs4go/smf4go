package localservice

import (
	"sync"

	"github.com/dynamicgo/xerrors"
	"github.com/libs4go/scf4go"
	"github.com/libs4go/smf4go"
)

// F .
type F func(config scf4go.Config) (smf4go.Service, error)

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

func (extension *localServiceExtension) register(name string, f F) {
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
		return nil, xerrors.Wrapf(smf4go.ErrNotFound, "service %s not found", serviceName)
	}

	return f(config)
}

func (extension *localServiceExtension) End() error {
	return nil
}

var extension *localServiceExtension
var once sync.Once

func get() *localServiceExtension {
	once.Do(func() {
		extension = newExtension()
		smf4go.Builder().RegisterExtension(extension)
	})

	return extension
}

// Register .
func Register(name string, f F) {
	get().register(name, f)
}
