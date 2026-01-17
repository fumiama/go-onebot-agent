package goba

import (
	_ "embed"
	"fmt"
	"time"
)

//go:embed README.md
var sysp string

// Config stores mutable characteristics of the agent.
type Config struct {
	Nickname string // Nickname 昵称
	Sex      string // Sex 性别
	Chars    string // Chars 个性
}

func (ag *Agent) system(role PermRole, iter int, grp int64) (string, error) {
	tab, err := ag.perm.mdtable(role)
	if err != nil {
		return "", err
	}
	t := time.Now()
	typ := "群聊"
	if grp < 0 {
		typ = "私聊"
	}
	return fmt.Sprintf(
		sysp, ag.cfg.Nickname, ag.cfg.Sex,
		ag.cfg.Chars, tab, ag.memoryof(grp),
		t.Format(time.RFC3339), t.Unix(), typ, iter,
	), nil
}
