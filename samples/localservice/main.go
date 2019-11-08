package main

import (
	"github.com/libs4go/scf4go"
	_ "github.com/libs4go/scf4go/codec"
	"github.com/libs4go/slf4go"
	_ "github.com/libs4go/slf4go/backend/console"
	"github.com/libs4go/smf4go"
	"github.com/libs4go/smf4go/app"
	"github.com/libs4go/smf4go/service/localservice"
)

var logger = slf4go.Get("test")

type serviceA struct {
	B    *serviceB `inject:"B"`
	Name string
}

func (a *serviceA) Start() error {
	logger.I("A:{@a}, B:{@b}", a.Name, a.B.Name)
	return nil
}

type serviceB struct {
	A    *serviceA `inject:"A"`
	Name string
}

func (b *serviceB) Start() error {
	logger.I("B:{@b}, A:{@a}", b.Name, b.A.Name)
	return nil
}

func main() {

	localservice.Register("A", func(config scf4go.Config) (smf4go.Service, error) {
		return &serviceA{
			Name: config.Get("Name").String(""),
		}, nil
	})

	localservice.Register("B", func(config scf4go.Config) (smf4go.Service, error) {
		return &serviceB{
			Name: config.Get("Name").String(""),
		}, nil
	})

	app.Run("test")
}
