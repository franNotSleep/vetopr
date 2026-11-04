package vetopr

import (
	"io"
	"os"

	"github.com/moby/moby/client"
)

type ServiceType int

const (
	DatabaseType ServiceType = iota
	UserType
	CacheType
)

var serviceTypeName = map[ServiceType]string{
	DatabaseType: "database",
	UserType:     "user",
	CacheType:    "cache",
}

func (st ServiceType) String() string {
	return serviceTypeName[st]
}

type ServiceConfig struct {
	Base     string `yaml:"base"`
	Build    string `yaml:"build"`
	Start    string `yaml:"start"`
	Workpath string
	Name     string
}

// type VetoprConfig map[any]any
type VetoprConfig struct {
	Services  map[any]ServiceConfig `yaml:"services"`
	Databases []ServiceConfig       `yaml:"databases"`
}

type VetoprD struct {
	Services     []*VetoprService
	Workpath     string
	Config       VetoprConfig
	dockerClient *client.Client
	boxID        string
}

type VetoprService struct {
	Process    *os.Process
	Config     ServiceConfig
	StdoutRead *io.ReadCloser
	StderrRead *io.ReadCloser
	Type       ServiceType
}
