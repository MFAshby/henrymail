package main

type Usr struct {
	Id    int64
	Email string
}

type Mbx struct {
	Id          int64
	UserId      int64
	Name        string
	Messages    uint32
	Recent      uint32
	Unseen      uint32
	UidNext     uint32
	UidValidity uint32
}

type Msg struct {
	Id      int64
	MbxId   int64
	Content []byte
	Uid     uint32
	Flags   []string
}
