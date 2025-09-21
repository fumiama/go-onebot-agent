package goba

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/fumiama/deepinfra"
	"github.com/fumiama/deepinfra/chat"
	"github.com/fumiama/deepinfra/model"
	"github.com/pkg/errors"
	zero "github.com/wdvxdr1123/ZeroBot"
)

var (
	ErrPermissionDenied = errors.New("permission denied")
)

// Agent is a OneBot event context, it is recommended to create one agent
// per group or per user.
type Agent struct {
	log           chat.Log[fmt.Stringer]
	id            int64
	nickname, sex string
	chars         string
	perm          *Perm
}

// NewAgent characteristics 最好是 MD 格式, defaultprompt 是上下文为空时的默认项, 建议为 Event JSON
func NewAgent(
	id int64, batchcap, itemscap int,
	nickname, sex, characteristics, defaultprompt string,
) Agent {
	return Agent{
		id: id, nickname: nickname, sex: sex, chars: characteristics,
		log: chat.NewLog[fmt.Stringer](batchcap, itemscap, "\n", defaultprompt),
	}
}

func (ag *Agent) AddEvent(grp int64, ev *Event) {
	ag.log.Add(grp, ev, false)
}

func (ag *Agent) AddRequest(grp int64, req *zero.APIRequest) {
	ag.log.Add(grp, req, true)
}

// GetAction get OneBot CallAction from LLM and add it to context.
//
// Note:
//
//  1. The response may be empty, meaning that LLM do not want
//     to react with these events. In this case, this function will
//     return io.EOF and the context will be left no change.
//
//  2. If LLM returns an invalid action, ErrPermissionDenied will be returned
//     with complete req, caller may decide whether to use this req by themselves.
//     Whatever, this req will not be added into the context. You may call
//     AddRequest to add it but it is not recommended.
func (ag *Agent) GetAction(api deepinfra.API, p model.Protocol, grp int64, role PermRole, isusersystem bool) (
	req zero.APIRequest, err error,
) {
	sysp, err := ag.system(role)
	if err != nil {
		return
	}

	m := ag.log.Modelize(p, grp, sysp, isusersystem)

	resp, err := api.Request(m)
	if err != nil {
		return
	}
	err = json.Unmarshal([]byte(resp), &req)
	if err == nil && req.Action == "" {
		err = io.EOF
	}

	if !ag.perm.allow(role, req.Action) {
		err = errors.Wrap(ErrPermissionDenied, req.Action)
	} else {
		ag.AddRequest(grp, &req)
	}

	return
}
