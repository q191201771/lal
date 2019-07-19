package rtmp

type SubSession struct {
	*ServerSession

	isFresh     bool
	waitKeyNalu bool
}

func NewSubSession(ss *ServerSession) *SubSession {
	return &SubSession{
		ServerSession: ss,
		isFresh:       true,
		waitKeyNalu:   true,
	}
}
