package gb28181

const (
	psPackStartCodePackHeader       = 0x01ba
	psPackStartCodeSystemHeader     = 0x01bb
	psPackStartCodeProgramStreamMap = 0x01bc
	psPackStartCodeAudioStream      = 0x01c0
	psPackStartCodeVideoStream      = 0x01e0
	psPackStartCodeHikStream        = 0x01bd

	psPackStartCodePesPsd      = 0x01ff // program_stream_directory
	psPackStartCodePesPadding  = 0x01be // padding_stream
	psPackStartCodePesPrivate2 = 0x01bf // padding_stream_2
	psPackStartCodePesEcm      = 0x01f0 // ECM_stream
	psPackStartCodePesEmm      = 0x01f1 // EMM_stream

	psPackStartCodePackEnd = 0x01b9
)

type PsStreamType int

const (
	StreamTypeH264    PsStreamType = 0x1b
	StreamTypeH265                 = 0x24
	StreamTypeAAC                  = 0x0f
	StreamTypeG711A                = 0x90 //PCMA
	StreamTypeG7221                = 0x92
	StreamTypeG7231                = 0x93
	StreamTypeG729                 = 0x99
	StreamTypeUnknown              = -1
)

const psBufInitSize = 4096
const (
	PsHeaderlen     int = 14
	SysHeaderlen    int = 18
	SysMapHeaderLen int = 24
	PesHeaderLen    int = 19
)
const (
	MaxPesLen = 0xFFFF
)
const (
	StreamIdVideo = 0xe0
	StreamIdAudio = 0xc0
)
const adtsMinLen = 7
