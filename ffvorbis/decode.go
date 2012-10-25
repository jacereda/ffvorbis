// Copyright (c) 2012, Jorge Acereda Maciá. All rights reserved.  
//
// Use of this source code is governed by a BSD-style license that can
// be found in the LICENSE file.

// Package ffvorbis provides a wrapper around the vorbis codec in ffmpeg.
package ffvorbis

// #cgo LDFLAGS: -lavcodec
/*
#include "libavcodec/avcodec.h"
#if LIBAVCODEC_VERSION_MAJOR == 53
#define AV_CODEC_ID_VORBIS CODEC_ID_VORBIS
#endif
#include <string.h>
extern AVCodec ff_vorbis_decoder;

static void convertS16(void * vd, const void * vs, int n) {
 const int16_t * s = (const int16_t*)vs;
 float * d = (float*)vd;
 float scale = 1 / 65536.0; // 32768.0f;
 int i;
 for (i = 0; i < n; i++) d[i] = scale * s[i];
}
*/
import "C"

import (
	"log"
	"unsafe"
)

func init() {
	C.avcodec_register_all()
}

type Decoder struct {
	c  *C.AVCodec
	cc *C.AVCodecContext
}

func NewDecoder(data []byte, channels, rate int) *Decoder {
	var d Decoder
	d.c = C.avcodec_find_decoder(C.AV_CODEC_ID_VORBIS)
	d.cc = C.avcodec_alloc_context3(d.c)
	d.cc.codec_type = C.AVMEDIA_TYPE_AUDIO
	d.cc.sample_rate = C.int(rate)
	d.cc.channels = C.int(channels)
	d.cc.extradata = (*C.uint8_t)(&data[0])
	d.cc.extradata_size = C.int(len(data))
	d.cc.channels = 2
	C.avcodec_open2(d.cc, d.c, nil)
	return &d
}

func (d *Decoder) Decode(data []byte) []float32 {
	var pkt C.AVPacket
	var fr C.AVFrame
	var got C.int
	C.avcodec_get_frame_defaults(&fr)
	C.av_init_packet(&pkt)
	pkt.data = (*C.uint8_t)(&data[0])
	pkt.size = C.int(len(data))
	dec := C.avcodec_decode_audio4(d.cc, &fr, &got, &pkt)
	if dec < 0 {
		log.Println("Unable to decode")
		return nil
	}
	if dec != pkt.size {
		log.Println("Partial decode")
	}
	if got == 0 {
		return nil
	}
	nvals := d.cc.channels * fr.nb_samples
	buf := make([]float32, nvals)
	dst := unsafe.Pointer(&buf[0])
	src := unsafe.Pointer(fr.data[0])
	switch d.cc.sample_fmt {
	case C.AV_SAMPLE_FMT_FLT:
		C.memcpy(dst, src, C.size_t(nvals*4))
	case C.AV_SAMPLE_FMT_S16:
		C.convertS16(dst, src, nvals)
	default:
		log.Panic("Unsupported format")
	}
	if pkt.data != nil {
		C.av_free_packet(&pkt)
	}
	return buf
}
