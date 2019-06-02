package rtmp

var chunkSize = 4096

type PullSessionObserver interface {
	// @param t: 8 audio, 9 video, 18 meta
	// after cb, PullSession will use <message>
	ReadAVMessageCB(t int, timestampAbs int, message []byte)
}

type PullSession struct {
	*ClientSession
}

func NewPullSession(obs PullSessionObserver, connectTimeout int64) *PullSession {
	return &PullSession{
		ClientSession: NewClientSession(CSTPullSession, obs, connectTimeout),
	}
}

func (s *PullSession) Pull(rawURL string) error {
	return s.Do(rawURL)
}
