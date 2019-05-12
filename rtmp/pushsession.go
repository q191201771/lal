package rtmp

type PushSession struct {
	*ClientSession
}

func NewPushSession() *PushSession {
	return &PushSession{
		ClientSession: NewClientSession(CSTPushSession, nil),
	}
}

func (s *PushSession) Push(rawURL string) error {
	return s.Do(rawURL)
}

// TODO chef: add function to write av data
