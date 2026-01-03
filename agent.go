// Package goba OneBot 11 协议 QQ 聊天 Agent
package goba

import (
	"bytes"
	"encoding/json"
	"fmt"
	"image"
	"io"
	"math"
	"net/http"
	"strings"
	"time"

	"github.com/FloatTech/imgfactory"
	"github.com/FloatTech/ttl"
	"github.com/corona10/goimagehash"
	"github.com/fumiama/deepinfra"
	"github.com/fumiama/deepinfra/chat"
	"github.com/fumiama/deepinfra/model"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	zero "github.com/wdvxdr1123/ZeroBot"
	"github.com/wdvxdr1123/ZeroBot/message"
)

const (
	// EOA is a dummy action that is used to terminate request
	EOA = "end_action"
	// SVM is a dummy action that is used to indicate that a memory has been saved
	SVM = "save_memory"
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
	imgpcache     *ttl.Cache[uint64, string]
	mem           MemoryStorage
	manualaddreq  bool
	manualaddmem  bool
	hasimageapi   bool
}

// NewAgent 创建一个 Agent 实例。
//
//   - characteristics 推荐使用 Markdown 格式，描述 Agent 个性。
//   - defaultprompt 为上下文为空时的默认提示，建议为事件的 JSON，一般不会用到，因此也可留空。
//   - manualaddreq 表示是否由用户手动添加请求。
//   - manualaddmem 表示是否由用户手动添加记忆。
func NewAgent(
	id int64, batchcap, itemscap int, imgpcachettl time.Duration,
	nickname, sex, characteristics, defaultprompt string, mem MemoryStorage,
	manualaddreq, manualaddmem bool,
) (ag Agent) {
	ag = Agent{
		id: id, nickname: nickname, sex: sex, chars: characteristics,
		imgpcache: ttl.NewCache[uint64, string](imgpcachettl),
		log:       chat.NewLog[fmt.Stringer](batchcap, itemscap, "\n", defaultprompt),
		mem:       mem, manualaddreq: manualaddreq, manualaddmem: manualaddmem,
	}
	_ = ag.LoadPermTable()
	return
}

// AddEvent 添加接收到的事件
func (ag *Agent) AddEvent(grp int64, ev *Event) {
	ag.log.Add(grp, ev, false)
}

// AddRequest 添加 API 请求, 一般无需主动调用, 由 GetAction 自动添加
func (ag *Agent) AddRequest(grp int64, req *zero.APIRequest) {
	ag.log.Add(grp, req, true)
}

// AddResponse 添加在执行完 zero.APIRequest 之后得到的响应
func (ag *Agent) AddResponse(grp int64, resp *APIResponse) {
	ag.log.Add(grp, resp, false)
}

// AddTerminus 添加会话终止符, 一般无需主动调用, 由 GetAction 自动添加
func (ag *Agent) AddTerminus(grp int64) {
	ag.log.Add(grp, Terminus{}, true)
}

// AddMemory 添加记忆, 一般无需主动调用, 由 GetAction 自动添加
func (ag *Agent) AddMemory(grp int64, text string) error {
	return ag.mem.Save(grp, strings.TrimSpace(text))
}

// CanViewImage will be true if SetViewImageAPI is called
func (ag *Agent) CanViewImage() bool {
	return ag.hasimageapi
}

// SetViewImageAPI 为 agent 增加识图功能, 需要模型支持视觉
func (ag *Agent) SetViewImageAPI(api deepinfra.API, p model.Protocol) {
	ag.log.SetPreModelize(func(s *fmt.Stringer) {
		o := *s
		if ev, ok := o.(*Event); ok {
			hasset := false
			msgs := message.ParseMessage(ev.Message)
			for i, msg := range msgs {
				if msg.Type != "image" {
					continue
				}
				if _, ok := msg.Data["__agent_desc__"]; ok {
					continue
				}
				u := msg.Data["url"]
				if !strings.HasPrefix(u, "http") {
					continue
				}
				resp, err := http.Get(u)
				if err != nil {
					logrus.Debugln("[goba] SetViewImageAPI get http err:", err)
					continue
				}
				data, err := io.ReadAll(resp.Body)
				_ = resp.Body.Close()
				if err != nil {
					logrus.Debugln("[goba] SetViewImageAPI read body err:", err)
					continue
				}
				img, _, err := image.Decode(bytes.NewReader(data))
				if err != nil {
					logrus.Debugln("[goba] SetViewImageAPI decode img err:", err)
					continue
				}
				hsh, err := goimagehash.PerceptionHash(img)
				if err != nil {
					logrus.Debugln("[goba] SetViewImageAPI calc phash err:", err)
					continue
				}
				d := math.MaxInt
				desc := ""
				_ = ag.imgpcache.Range(func(u uint64, s string) error {
					ds, err := goimagehash.NewImageHash(u, goimagehash.PHash).Distance(hsh)
					if err != nil {
						logrus.Debugln("[goba] SetViewImageAPI calc phash distance err:", err)
						return nil
					}
					if d > ds {
						d = ds
						desc = s
					}
					return nil
				})
				logrus.Debugln("[goba] SetViewImageAPI calculated min d:", d)
				if d < 8 { // very similar (>87.5%)
					msgs[i].Data["__agent_desc__"] = desc
					hasset = true
					logrus.Debugln("[goba] SetViewImageAPI hit cache.")
					continue
				}

				img = imgfactory.Limit(img, 1024, 1024)
				data, err = imgfactory.ToBytes(img)
				if err != nil {
					logrus.Debugln("[goba] SetViewImageAPI dump img err:", err)
					continue
				}
				imgs, err := model.NewContentImageDataBase64URL(data)
				if err != nil {
					logrus.Debugln("[goba] SetViewImageAPI conv b64 err:", err)
					continue
				}
				p = p.Clone() // clear protocol content
				m := p.User(
					model.NewContentImageURL(imgs),
					model.NewContentText("使用简洁清晰明确的一段中文纯文本描述图片"),
				)
				desc, err = api.Request(m)
				if err != nil {
					logrus.Debugln("[goba] SetViewImageAPI request API err:", err)
					continue
				}
				msgs[i].Data["__agent_desc__"] = desc
				ag.imgpcache.Set(hsh.GetHash(), desc)
				hasset = true
			}
			if hasset {
				raw, err := json.Marshal(&msgs)
				if err == nil {
					ev.Message = raw
				}
			}
		}
	})
	ag.hasimageapi = true
}

// ClearViewImageAPI ...
func (ag *Agent) ClearViewImageAPI() {
	ag.log.SetPreModelize(nil)
	ag.hasimageapi = false
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
	sysp, err := ag.system(role, grp)
	if err != nil {
		logrus.Debugln("[goba] GetAction get sysp err:", err)
		return
	}

	m := ag.log.Modelize(p, grp, sysp, isusersystem)

	resp, err := api.Request(m)
	if err != nil {
		logrus.Debugln("[goba] GetAction request api err:", err)
		return
	}
	if strings.HasPrefix(resp, "```") { // AI returns codeblock
		_, resp, _ = strings.Cut(resp, "\n")
		resp = strings.Trim(resp, "`")
		resp = strings.TrimSpace(resp)
	}
	reqs = make([]zero.APIRequest, 0, 2)
	dec := json.NewDecoder(strings.NewReader(resp))
	dec.UseNumber()
	for dec.More() {
		r := zero.APIRequest{}
		err = dec.Decode(&r)
		if err != nil {
			logrus.Debugln("[goba] GetAction decode api request err:", err)
			break
		}
		if r.Action == "" {
			continue
		}
		switch {
		case r.Action == EOA:
			ag.AddTerminus(grp)
			return
		case !ag.perm.allow(role, r.Action):
			err = errors.Wrap(ErrPermissionDenied, r.Action)
			return
		case !ag.manualaddreq:
			ag.AddRequest(grp, &r)
			if !ag.manualaddmem && r.Action == SVM {
				txt, err := extractMemory(&r)
				if err != nil {
					logrus.Debugln("[goba] GetAction extract memory err:", err)
					ag.AddResponse(grp, &APIResponse{
						Status:  "error",
						Message: err.Error(),
					})
					continue
				}
				logrus.Debugln("[goba] GetAction add memory:", txt, "to grp:", grp)
				err = ag.AddMemory(grp, txt)
				s := "ok"
				msg := ""
				if err != nil {
					logrus.Debugln("[goba] GetAction add memory err:", err)
					s = "error"
					msg = err.Error()
				}
				logrus.Debugln("[goba] GetAction add memory response:", s, "msg:", msg)
				ag.AddResponse(grp, &APIResponse{
					Status:  s,
					Message: msg,
				})
			}
		}
		reqs = append(reqs, r)
	}

	return
}
