package goba

import (
	_ "embed"
	"os"
	"strings"

	"github.com/RomiChan/syncx"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"
)

var (
	ErrNoSuchPermRole   = errors.New("no such perm role")
	ErrUnexpectedAction = errors.New("unexpected action")
)

type PermRole string

const (
	PermRoleOwner PermRole = "owner"
	PermRoleAdmin PermRole = "admin"
	PermRoleUser  PermRole = "user"
)

//go:embed actions.yaml
var innerpermtable []byte

type PermAction struct {
	Desc   string `yaml:"desc"`
	Params string `yaml:"params"`
}

type Perm struct {
	Actions map[string]PermAction        `yaml:"actions"`
	Config  map[PermRole][]string        `yaml:"config"`
	cache   syncx.Map[PermRole, actions] `yaml:"-"`
}

func (p *Perm) mdtable(role PermRole) (string, error) {
	acs, ok := p.Config[role]
	if !ok {
		return "", errors.Wrap(ErrNoSuchPermRole, string(role))
	}
	table := strings.Builder{}
	table.WriteString("|功能|action|params|\n|---|---|---|")
	var ac actions
	if _, ok := p.cache.Load(role); ok {
		ac = actions{}
	}
	for _, act := range acs {
		a, ok := p.Actions[act]
		if !ok {
			panic(errors.Wrap(ErrUnexpectedAction, act))
		}
		if ac != nil {
			ac[act] = struct{}{}
		}
		table.WriteString("\n|")
		table.WriteString(a.Desc)
		table.WriteByte('|')
		table.WriteString(act)
		table.WriteByte('|')
		table.WriteString(a.Params)
		table.WriteByte('|')
	}
	if ac != nil {
		p.cache.Store(role, ac)
	}
	return table.String(), nil
}

func (p *Perm) allow(role PermRole, action string) bool {
	ac, ok := p.cache.Load(role)
	if ok {
		return ac.allow(action)
	}
	acs, ok := p.Config[role]
	if !ok {
		return false
	}
	ac = actions{}
	for _, act := range acs {
		ac[act] = struct{}{}
	}
	p.cache.Store(role, ac)
	return ac.allow(action)
}

type actions map[string]struct{}

func (ac actions) allow(action string) bool {
	_, ok := ac[action]
	return ok
}

// LoadPermTable 读取 yaml file 并根据权限生成 MD 表格, 参数为空则加载内嵌配置
func (ag *Agent) LoadPermTable(file ...string) error {
	var data []byte
	if len(file) == 0 {
		data = innerpermtable
	} else {
		var err error
		data, err = os.ReadFile(file[0])
		if err != nil {
			return err
		}
	}
	var cfg Perm
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return err
	}
	ag.perm = &cfg
	return nil
}
