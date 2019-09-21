package rtmp

type PushSession struct {
	*ClientSession
}

type PushSessionTimeout struct {
	ConnectTimeoutMS int
	PushTimeoutMS    int
	WriteAVTimeoutMS int
}

func NewPushSession(timeout PushSessionTimeout) *PushSession {
	return &PushSession{
		ClientSession: NewClientSession(CSTPushSession, nil, ClientSessionTimeout{
			ConnectTimeoutMS: timeout.ConnectTimeoutMS,
			DoTimeoutMS:      timeout.PushTimeoutMS,
			WriteAVTimeoutMS: timeout.WriteAVTimeoutMS,
		}),
	}
}

func (s *PushSession) Push(rawURL string) error {
	return s.Do(rawURL)
}
