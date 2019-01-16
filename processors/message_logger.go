package processors

import (
	"henrymail/model"
	"log"
)

/**
 * Logs out the message
 * Useful during development for debugging
 */
type logger struct {
	next MsgProcessor
}

func (l logger) Process(w *model.ReceivedMsg) error {
	log.Print("From: ", w.From)
	log.Print("To: ", w.To)
	log.Println("Content:", string(w.Content))
	return l.next.Process(w)
}

func NewLogger(next MsgProcessor) MsgProcessor {
	return &logger{next}
}