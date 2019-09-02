package rtmp

type PullSessionObserver interface {
	AVMsgObserver
}

type PullSession struct {
	*ClientSession
}

func NewPullSession(obs PullSessionObserver, connectTimeout int) *PullSession {
	return &PullSession{
		ClientSession: NewClientSession(CSTPullSession, obs, connectTimeout),
	}
}

func (s *PullSession) Pull(rawURL string) error {
	return s.Do(rawURL)
}
