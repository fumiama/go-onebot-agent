// Package goba OneBot 11 协议 QQ 聊天 Agent
package goba

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/fumiama/deepinfra"
	"github.com/fumiama/deepinfra/chat"
	"github.com/fumiama/deepinfra/model"
	"github.com/pkg/errors"
	zero "github.com/wdvxdr1123/ZeroBot"
)

var (
	// ErrPermissionDenied LLM 调用了不该调用的 action
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
	manualaddreq  bool
}

// NewAgent 创建一个 Agent 实例。
//
//   - characteristics 推荐使用 Markdown 格式，描述 Agent 个性。
//   - defaultprompt 为上下文为空时的默认提示，建议为事件的 JSON，一般不会用到，因此也可留空。
//   - manualaddreq 表示是否由用户手动添加请求。
func NewAgent(
	id int64, batchcap, itemscap int,
	nickname, sex, characteristics, defaultprompt string,
	manualaddreq bool,
) (ag Agent) {
	ag = Agent{
		id: id, nickname: nickname, sex: sex, chars: characteristics,
		log:          chat.NewLog[fmt.Stringer](batchcap, itemscap, "\n", defaultprompt),
		manualaddreq: manualaddreq,
	}
	_ = ag.LoadPermTable()
	return
}

// AddEvent 添加接收到的事件
func (ag *Agent) AddEvent(grp int64, ev *Event) {
	ag.log.Add(grp, ev, false)
}

// AddRequest 一般无需主动调用, 由 GetAction 自动添加
func (ag *Agent) AddRequest(grp int64, req *zero.APIRequest) {
	ag.log.Add(grp, req, true)
}

// GetAction get OneBot CallAction from LLM and add it to context.
//
// Note:
//
//   - If LLM returns an invalid action, ErrPermissionDenied will be returned
//     with complete reqs before invalid call, caller may decide whether to use
//     these reqs by themselves. Whatever, invalid req will not be added into
//     the context. You may call AddRequest to add it but it is not recommended.
func (ag *Agent) GetAction(api deepinfra.API, p model.Protocol, grp int64, role PermRole, isusersystem bool) (
	reqs []zero.APIRequest, err error,
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
	reqs = make([]zero.APIRequest, 0, 2)
	dec := json.NewDecoder(strings.NewReader(resp))
	for dec.More() {
		r := zero.APIRequest{}
		err = dec.Decode(&r)
		if err != nil {
			break
		}
		if r.Action == "" {
			continue
		}
		switch {
		case !ag.perm.allow(role, r.Action):
			err = errors.Wrap(ErrPermissionDenied, r.Action)
			return
		case !ag.manualaddreq:
			ag.AddRequest(grp, &r)
		}
		reqs = append(reqs, r)
	}

	return
}
