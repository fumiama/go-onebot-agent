package goba

import (
	_ "embed"
	"fmt"
)

//go:embed README.md
var sysp string

func (ag *Agent) system(role PermRole) (string, error) {
	tab, err := ag.perm.mdtable(role)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf(sysp, ag.id, ag.nickname, ag.sex, ag.chars, tab), nil
}
