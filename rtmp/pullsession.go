package rtmp

var chunkSize = 4096

type PullSessionObserver interface {
	// @param t: 8 audio, 9 video, 18 meta
	// after cb, PullSession will use <message>
	ReadAvMessageCB(t int, timestampAbs int, message []byte)
}

type PullSession struct {
	*ClientSession
}

func NewPullSession(obs PullSessionObserver) *PullSession {
	return &PullSession{
		ClientSession: NewClientSession(CSTPullSession, obs),
	}
}

func (s *PullSession) Pull(rawURL string) error {
	return s.Do(rawURL)
}
