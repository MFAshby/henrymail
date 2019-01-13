package processors

import "henrymail/model"

type MsgProcessor interface {
	Process(*model.ReceivedMsg) error
}
