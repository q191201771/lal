package rtmp

type PushSession struct {
	*ClientSession
}

func NewPushSession(connectTimeout int64) *PushSession {
	return &PushSession{
		ClientSession: NewClientSession(CSTPushSession, nil, connectTimeout),
	}
}

func (s *PushSession) Push(rawURL string) error {
	return s.Do(rawURL)
}

// TODO chef: add function to write av data
