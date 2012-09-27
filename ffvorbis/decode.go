// Copyright (c) 2012, Jorge Acereda Maciá. All rights reserved.  
//
// Use of this source code is governed by a BSD-style license that can
// be found in the LICENSE file.

// Package ffvorbis provides a wrapper around the vorbis codec in ffmpeg.
package ffvorbis

// #cgo LDFLAGS: -lavcodec -lavutil
/*
#include "libavcodec/avcodec.h"
#include "libavutil/samplefmt.h"
extern AVCodec ff_vorbis_decoder;
*/
import "C"

import (
	"log"
	"time"
	"unsafe"
)

type Format uint

const (
	Int16 = iota
	Float32
)

type Samples struct {
	Data      []byte
	Timecode  time.Duration
	Channels  uint
	Frequency uint
	Format    Format
}

func init() {
	C.avcodec_register(&C.ff_vorbis_decoder)
}

type Decoder struct {
	c  *C.AVCodec
	cc *C.AVCodecContext
}

func NewDecoder(data []byte) *Decoder {
	var d Decoder
	d.c = C.avcodec_find_decoder(C.AV_CODEC_ID_VORBIS)
	d.cc = C.avcodec_alloc_context3(d.c)
	d.cc.extradata = (*C.uint8_t)(&data[0])
	d.cc.extradata_size = C.int(len(data))
	C.avcodec_open2(d.cc, d.c, nil)
	return &d
}

func aligned(x uintptr) uintptr {
	return (x + 255) & 0xffffffffffffff00
}

func (d *Decoder) Decode(data []byte, timecode time.Duration) *Samples {
	var pkt C.AVPacket
	var fr C.AVFrame
	var got C.int
	C.avcodec_get_frame_defaults(&fr)
	C.av_init_packet(&pkt)
	dl := len(data)
	pkt.data = (*C.uint8_t)(&data[0])
	pkt.size = C.int(dl)
	dec := C.avcodec_decode_audio4(d.cc, &fr, &got, &pkt)
	if dec < 0 {
		log.Println("Unable to decode 1")
		return nil
	}
	if dec != pkt.size {
		log.Println("Partial decode")
	}
	if got == 0 {
		return nil
	}
	var afmt Format
	var bps int
	switch d.cc.sample_fmt {
	case C.AV_SAMPLE_FMT_S16:
		bps = 2
		afmt = Int16
	case C.AV_SAMPLE_FMT_FLT:
		bps = 4
		afmt = Float32
	default:
		log.Panic("Unsupported format")
	}
	sz := bps * int(d.cc.channels*fr.nb_samples)
	buf := make([]byte, sz)
	copy(buf, ((*[192000]byte)(unsafe.Pointer(fr.data[0])))[0:sz])
	if pkt.data != nil {
		C.av_free_packet(&pkt)
	}
	return &Samples{buf, timecode, uint(d.cc.channels),
		uint(d.cc.sample_rate), afmt}
}
