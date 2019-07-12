package rtmp

type PubSessionObserver interface {
	AVMsgObserver
}

type PubSession struct {
	*ServerSession
}

func NewPubSession(ss *ServerSession) *PubSession {
	return &PubSession{
		ss,
	}
}

func (s *ServerSession) SetPubSessionObserver(obs PubSessionObserver) {
	s.avObs = obs
}
