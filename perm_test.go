package goba

import "testing"

const fulltab = `|功能|action|params|
|---|---|---|
|发送私聊消息|send_private_msg|user_id 对方QQ号；message 要发送的内容 (json.RawMessage)|
|发送群消息|send_group_msg|group_id 群号；message 要发送的内容 (json.RawMessage)|
|撤回消息|delete_msg|message_id 消息ID|
|发送好友赞|send_like|user_id 对方QQ号；times 赞的次数，每个好友每天最多10次 (number)|
|群组踢人|set_group_kick|group_id 群号；user_id 要踢的QQ号；reject_add_request 拒绝此人的加群请求 (boolean)|
|群组单人禁言|set_group_ban|group_id 群号；user_id 要禁言的QQ号；duration 禁言时长（秒），0表示取消禁言|
|群组全员禁言|set_group_whole_ban|group_id 群号；enable 是否禁言 (boolean)|
|群组设置管理员|set_group_admin|group_id 群号；user_id 要设置管理员的QQ号；enable true为设置，false为取消|
|设置群名片|set_group_card|group_id 群号；user_id 要设置的QQ号；card 群名片内容，不填或空字符串表示删除群名片|
|设置群名|set_group_name|group_id 群号；group_name 新群名|
|退出群组|set_group_leave|group_id 群号；is_dismiss 是否解散 (boolean)|
|设置群组专属头衔|set_group_special_title|group_id 群号；user_id 要设置的QQ号；special_title 专属头衔，不填或空字符串表示删除；duration 专属头衔有效期（秒），-1表示永久|
|处理加好友请求|set_friend_add_request|flag 加好友请求的flag (string)；approve 是否同意请求 (boolean)；remark 添加后的好友备注（仅同意时有效）|
|处理加群请求/邀请|set_group_add_request|flag 加群请求的flag (string)；sub_type/type add或invite 请求类型（需与上报一致）；approve 是否同意请求/邀请 (boolean)；reason 拒绝理由（仅拒绝时有效）|`

func TestMDTable(t *testing.T) {
	ag := new(Agent)
	err := ag.LoadPermTable()
	if err != nil {
		t.Fatal(err)
	}
	tab, err := ag.perm.mdtable(PermRoleOwner)
	if err != nil {
		t.Fatal(err)
	}
	t.Log(tab)
	if tab != fulltab {
		t.Fail()
	}
}
