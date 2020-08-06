package tester

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/libs4go/scf4go"
	_ "github.com/libs4go/scf4go/codec" //
	"github.com/libs4go/scf4go/reader/file"
	"github.com/libs4go/scf4go/reader/memory"
	"github.com/libs4go/slf4go"
	_ "github.com/libs4go/slf4go/backend/console" //
	"github.com/libs4go/smf4go"
	"github.com/libs4go/smf4go/service/grpcservice"
	"github.com/libs4go/smf4go/service/localservice"
)

func isDir(path string) bool {
	fi, err := os.Stat(path)

	if err != nil {
		return false
	}

	return fi.IsDir()
}

// F .
type F func(Tester)

// Tester .
type Tester interface {
	ConfigPath(name string) Tester
	Config(data string, codec string) Tester
	Run(f F)
	Stop()
	T() *testing.T
	B() *testing.B
	LocalService() localservice.LocalService
	GRPCService() grpcservice.Register
	Context() context.Context
}

type testerImpl struct {
	t           *testing.T
	b           *testing.B
	ctx         context.Context
	cancel      context.CancelFunc
	config      scf4go.Config
	ls          localservice.LocalService
	gs          grpcservice.Register
	meshBuilder smf4go.MeshBuilder
}

func newTester(t *testing.T, b *testing.B) Tester {

	ctx, cancel := context.WithCancel(context.Background())

	meshBuilder := smf4go.NewMeshBuilder()

	localservice := localservice.New(meshBuilder)

	grpcservice := grpcservice.New(
		"tester.grpc",
		grpcservice.WithMeshBuilder(meshBuilder),
		grpcservice.WithLocalService(localservice),
	)

	tester := &testerImpl{
		t:           t,
		b:           b,
		config:      scf4go.New(),
		ctx:         ctx,
		cancel:      cancel,
		meshBuilder: meshBuilder,
		ls:          localservice,
		gs:          grpcservice,
	}

	tester.Config(`{ "slf4go": {
        "default": {
            "backend": "console"
		},
		"backend": {
			"console": {
				"formatter": {
					"timestamp": "02 Jan 2006 15:04:05",
					"output": "[@t] (@l) <@s> @m"
				}
			}
		}
    }}`, "json")

	return tester
}

func (tester *testerImpl) ConfigPath(name string) Tester {

	if isDir(name) {
		if err := tester.config.Load(file.New(file.Dir(name))); err != nil {
			println(fmt.Sprintf("load config from directory %s %s", name, err))
		}
	} else {
		if err := tester.config.Load(file.New(file.File(name))); err != nil {
			println(fmt.Sprintf("load config file %s %s", name, err))
		}
	}

	return tester
}
func (tester *testerImpl) Config(data string, codec string) Tester {

	tester.config.Load(memory.New(memory.Data(data, codec)))

	return tester
}

func (tester *testerImpl) Run(f F) {
	f(tester)

	tester.LocalService().Register("smf4go.tester", func(config scf4go.Config) (smf4go.Service, error) {
		return tester, nil
	})

	if err := slf4go.Config(tester.config.SubConfig("slf4go")); err != nil {
		println(fmt.Sprintf("set slf4go config error: %s", err))
		return
	}

	if err := tester.meshBuilder.Start(tester.config); err != nil {
		println(fmt.Sprintf("run tester error: %s", err))
		return
	}

	defer slf4go.Sync()

	<-tester.ctx.Done()
}

func (tester *testerImpl) Stop() {
	tester.cancel()
}

func (tester *testerImpl) T() *testing.T {
	return tester.t
}

func (tester *testerImpl) B() *testing.B {
	return tester.b
}

func (tester *testerImpl) LocalService() localservice.LocalService {
	return tester.ls
}

func (tester *testerImpl) GRPCService() grpcservice.Register {
	return tester.gs
}

func (tester *testerImpl) Context() context.Context {
	return tester.ctx
}

// T create function test
func T(t *testing.T) Tester {
	return newTester(t, nil)
}

// B create Benchmark test
func B(b *testing.B) Tester {
	return newTester(nil, b)
}
