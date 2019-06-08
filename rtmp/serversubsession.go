package rtmp

type SubSession struct {
	*ServerSession
}

func NewSubSession(ss *ServerSession) *SubSession {
	return &SubSession{
		ss,
	}
}
