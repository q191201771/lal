package rtmp

type PullSession struct {
	*ClientSession
}

func NewPullSession(obs AVMessageObserver, connectTimeout int64) *PullSession {
	return &PullSession{
		ClientSession: NewClientSession(CSTPullSession, obs, connectTimeout),
	}
}

func (s *PullSession) Pull(rawURL string) error {
	return s.Do(rawURL)
}
