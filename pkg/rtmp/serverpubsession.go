package rtmp

type PubSession struct {
	*ServerSession
}

func NewPubSession(ss *ServerSession) *PubSession {
	return &PubSession{
		ss,
	}
}

func (s *ServerSession) SetAVMessageObserver(obs AVMessageObserver) {
	s.avObs = obs
}
