package goba

import (
	"encoding/json"
	"io"

	"github.com/fumiama/deepinfra"
	"github.com/fumiama/deepinfra/model"
	"github.com/pkg/errors"
	zero "github.com/wdvxdr1123/ZeroBot"
)

var (
	ErrZeroOrNegContextCap = errors.New("ctxcap <= 0")
	ErrZeroOrNegEventCap   = errors.New("evcap <= 0")
	ErrPermissionDenied    = errors.New("permission denied")
)

// Agent is a OneBot event context, it is recommended to create one agent
// per group or per user.
type Agent struct {
	id            int64
	nickname, sex string
	chars         string
	ctxcap, evcap int
	// 64 bits or 32 bits gap
	ctx  generalctx
	perm *Perm
}

// NewAgent characteristics 最好是 MD 格式
func NewAgent(id int64, nickname, sex string, characteristics string) Agent {
	return Agent{
		id: id, nickname: nickname, sex: sex, chars: characteristics,
		ctxcap: 16, evcap: 8,
	}
}

func (ag *Agent) SetContextCap(n int) {
	if n <= 0 {
		panic(ErrZeroOrNegContextCap)
	}
	ag.ctxcap = n
}

func (ag *Agent) SetEventCap(n int) {
	if n <= 0 {
		panic(ErrZeroOrNegEventCap)
	}
	ag.evcap = n
}

func (ag *Agent) AddEvent(ev *Event) {
	addctx(&ag.ctx, ev, ag.ctxcap, ag.evcap)
}

func (ag *Agent) AddRequest(req *zero.APIRequest) {
	addctx(&ag.ctx, req, ag.ctxcap, ag.evcap)
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
func (ag *Agent) GetAction(api deepinfra.API, m model.Protocol, role PermRole) (
	req zero.APIRequest, err error,
) {
	p, err := ag.system(role)
	if err != nil {
		return
	}
	m.System(p)

	ag.ctx.mu.Lock()
	for i, evs := range ag.ctx.ctx {
		if i%2 == 0 { // is user input
			m.User(evs.String())
		} else { // is agent callback
			m.Assistant(evs.String())
		}
	}
	ag.ctx.mu.Unlock()

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
		ag.AddRequest(&req)
	}
	return
}
