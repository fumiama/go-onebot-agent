package goba

import (
	"encoding/json"
	"strings"

	"github.com/pkg/errors"
	zero "github.com/wdvxdr1123/ZeroBot"
)

// Event is the simplified OneBot event that dumped to the agent in JSON format
type Event struct {
	Time        int64           `json:"time"`         // 事件发生的时间戳
	PostType    string          `json:"post_type"`    // 上报类型: message / notice / request
	MessageType string          `json:"message_type"` // message 类型: group / private
	SubType     string          `json:"sub_type"`     // message 子类型: normal (一般消息) / notice (灰色小字通知)
	MessageID   int64           `json:"message_id"`   // 消息 ID, 唯一标识该事件
	GroupID     int64           `json:"group_id"`     // QQ群号
	UserID      int64           `json:"user_id"`      // 事件发送者QQ号
	TargetID    int64           `json:"target_id"`
	SelfID      int64           `json:"self_id"` // 收到事件的QQ号 (你的ID)
	NoticeType  string          `json:"notice_type,omitempty"`
	OperatorID  int64           `json:"operator_id"` // This field is used for Notice Event
	File        *zero.File      `json:"file,omitempty"`
	RequestType string          `json:"request_type,omitempty"`
	Flag        string          `json:"flag,omitempty"`
	Comment     string          `json:"comment,omitempty"` // This field is used for Request Event
	Sender      *zero.User      `json:"sender,omitempty"`  // 事件发送者个人信息
	Message     json.RawMessage `json:"message,omitempty"` // JSON 格式的消息内容
}

func (ev *Event) String() string {
	sb := strings.Builder{}
	err := json.NewEncoder(&sb).Encode(ev)
	if err != nil {
		panic(errors.Wrap(err, "unexpected"))
	}
	return strings.TrimSpace(sb.String())
}
