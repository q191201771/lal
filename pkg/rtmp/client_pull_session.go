package rtmp

type PullSessionObserver interface {
	AVMsgObserver
}

type PullSession struct {
	*ClientSession
}

type PullSessionTimeout struct {
	ConnectTimeoutMS int
	PullTimeoutMS    int
	ReadAVTimeoutMS  int
}

func NewPullSession(obs PullSessionObserver, timeout PullSessionTimeout) *PullSession {
	return &PullSession{
		ClientSession: NewClientSession(CSTPullSession, obs, ClientSessionTimeout{
			ConnectTimeoutMS: timeout.ConnectTimeoutMS,
			DoTimeoutMS:      timeout.PullTimeoutMS,
			ReadAVTimeoutMS:  timeout.ReadAVTimeoutMS,
		}),
	}
}

func (s *PullSession) Pull(rawURL string) error {
	return s.Do(rawURL)
}
