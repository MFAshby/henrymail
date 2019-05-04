package process

/**
 * A black hole to throw unwanted messages into
 */
type hole struct{}

func (hole) Process(w *ReceivedMsg) error {
	return nil
}

func NewHole() MsgProcessor {
	return &hole{}
}
