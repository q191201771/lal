package main

import (
	"bufio"
	"context"
	ts "github.com/asticode/go-astits"
	"github.com/haivision/srtgo"
)

type Subscriber struct {
	ctx    context.Context
	socket *srtgo.SrtSocket
	muxer  *ts.Muxer
}

func NewSubscriber(ctx context.Context, socket *srtgo.SrtSocket) *Subscriber {
	return &Subscriber{
		ctx:    ctx,
		socket: socket,
		muxer:  ts.NewMuxer(ctx, bufio.NewWriter(socket)),
	}
}
