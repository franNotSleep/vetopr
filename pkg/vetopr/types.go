package vetopr

import (
	"io"
	"os"
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

type DatabaseConfig struct {
	Name string `yaml:"name"`
}

// type VetoprConfig map[any]any
type VetoprConfig struct {
	Services  map[any]ServiceConfig `yaml:"services"`
	Databases []DatabaseConfig      `yaml:"databases"`
}

type VetoprD struct {
	Services []*VetoprService
	Workpath string
	Config   VetoprConfig
}

type VetoprService struct {
	Process    *os.Process
	Config     ServiceConfig
	StdoutRead *io.ReadCloser
	StderrRead *io.ReadCloser
}
