package main

import (
	"bufio"
	"context"
	"errors"
	ts "github.com/asticode/go-astits"
	"github.com/haivision/srtgo"
	"github.com/q191201771/lal/pkg/aac"
	"github.com/q191201771/lal/pkg/base"
	"github.com/q191201771/lal/pkg/logic"
	"github.com/q191201771/naza/pkg/nazalog"
	"log"
)

type Publisher struct {
	ctx           context.Context
	srv           *Server
	ss            logic.ICustomizePubSessionContext
	streamName    string
	streamUser    string
	demuxer       *ts.Demuxer
	pkts          map[uint16]*base.AvPacket
	pmts          map[uint16]*ts.PMTData
	pat           *ts.PATData
	socket        *srtgo.SrtSocket
	gotAllPMTs    bool
	isSerVideoCfg bool
	subscribers   []*Subscriber
}

func NewPublisher(ctx context.Context, host, user string, socket *srtgo.SrtSocket, srv *Server) *Publisher {
	pub := &Publisher{
		ctx:           ctx,
		srv:           srv,
		streamName:    host,
		streamUser:    user,
		pkts:          make(map[uint16]*base.AvPacket),
		pmts:          make(map[uint16]*ts.PMTData),
		socket:        socket,
		demuxer:       ts.NewDemuxer(ctx, bufio.NewReader(socket)),
		gotAllPMTs:    false,
		isSerVideoCfg: false,
	}
	return pub
}

func (p *Publisher) SetSession(session logic.ICustomizePubSessionContext) {
	p.ss = session
}

func (p *Publisher) Run() {
	defer p.socket.Close()
	for {
		d, err := p.demuxer.NextData()
		if err != nil {
			if err == ts.ErrNoMorePackets {
				break
			}

			if errors.Is(err, srtgo.EConnLost) {
				log.Printf("stream [%s] disconnected", p.streamName)
				p.srv.Remove(p.streamName, p.ss)
				break
			}

		}

		if d.PAT != nil {
			p.pat = d.PAT
			p.gotAllPMTs = false
			continue
		}

		if d.PMT != nil {
			p.pmts[d.PMT.ProgramNumber] = d.PMT

			p.gotAllPMTs = true
			for _, pro := range p.pat.Programs {
				_, ok := p.pmts[pro.ProgramNumber]
				if !ok {
					p.gotAllPMTs = false
					break
				}
			}

			if !p.gotAllPMTs {
				continue
			}

			for _, pmt := range p.pmts {
				for _, es := range pmt.ElementaryStreams {
					_, ok := p.pkts[es.ElementaryPID]
					if ok {
						continue
					}
					var payloadType base.AvPacketPt
					switch es.StreamType {
					case ts.StreamTypeH264Video:
						payloadType = base.AvPacketPtAvc
					case ts.StreamTypeH265Video:
						payloadType = base.AvPacketPtHevc
					case ts.StreamTypeAACAudio:
						payloadType = base.AvPacketPtAac
					}

					p.pkts[es.ElementaryPID] = &base.AvPacket{
						PayloadType: payloadType,
					}
				}
			}
		}
		if !p.gotAllPMTs {
			continue
		}

		if d.PES != nil {

			pid := d.FirstPacket.Header.PID
			pkt, ok := p.pkts[pid]
			if !ok {
				log.Printf("Got payload for unknown PID %d", pid)
				continue
			}

			if d.PES.Header.IsVideoStream() {
				//pkt.Timestamp = int64(videoTimestamp)
				pkt.Payload = d.PES.Data
				pkt.Pts = d.PES.Header.OptionalHeader.PTS.Base / 90
				if d.PES.Header.OptionalHeader.DTS != nil {
					pkt.Timestamp = d.PES.Header.OptionalHeader.DTS.Base / 90
				} else {
					pkt.Timestamp = pkt.Pts
				}
				p.ss.FeedAvPacket(*pkt)
			} else {
				if d.PES.Header.PacketLength == 0 {
					continue
				}
				if pkt.PayloadType == base.AvPacketPtAac {
					data := d.PES.Data
					for remainLen := len(data); remainLen > 0; {
						// AACFrameLength(13)
						// xx xxxxxxxx xxx
						frameLen := (int(data[3]&3) << 11) | (int(data[4]) << 3) | (int(data[5]) >> 5)
						if frameLen > remainLen {
							break
						}
						payload := data[:frameLen]
						if !p.isSerVideoCfg {
							asc, err := aac.MakeAscWithAdtsHeader(payload[:aac.AdtsHeaderLength])
							nazalog.Assert(nil, err)
							p.ss.FeedAudioSpecificConfig(asc)
							p.isSerVideoCfg = true
						}
						pkt.Payload = payload[aac.AdtsHeaderLength:]
						pkt.Pts = d.PES.Header.OptionalHeader.PTS.Base / 90
						if d.PES.Header.OptionalHeader.DTS != nil {
							pkt.Timestamp = d.PES.Header.OptionalHeader.DTS.Base / 90
						} else {
							pkt.Timestamp = pkt.Pts
						}
						p.ss.FeedAvPacket(*pkt)
						data = data[frameLen:remainLen]
						remainLen -= frameLen
					}

				}
			}
		}

	}
}
