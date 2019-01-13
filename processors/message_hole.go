package processors

import "henrymail/model"

/**
 * A black hole to throw unwanted messages into
 */
type hole struct{}

func (hole) Process(w *model.ReceivedMsg) error {
	return nil
}

func NewHole() MsgProcessor {
	return &hole{}
}
