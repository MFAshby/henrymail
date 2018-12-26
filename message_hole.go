package main

/**
 * A black hole to throw unwanted messages into
 */
type hole struct{}

func (hole) Process(w *Wrap) error {
	return nil
}

func NewHole() MsgProcessor {
	return &hole{}
}
