// Copyright (c) 2012, Jorge Acereda Maciá. All rights reserved.  
//
// Use of this source code is governed by a BSD-style license that can
// be found in the LICENSE file.

// Package ffvorbis provides a wrapper around the vorbis codec in ffmpeg.
package ffvorbis

// #cgo LDFLAGS: -lavcodec -lavutil
/*
#include "libavcodec/avcodec.h"
#include "libavutil/frame.h"
#if LIBAVCODEC_VERSION_MAJOR == 53
#define AV_CODEC_ID_VORBIS CODEC_ID_VORBIS
#endif
#include <string.h>

static void convertS16(void * vd, const void * vs, int n) {
 const int16_t * s = (const int16_t*)vs;
 float * d = (float*)vd;
 float scale = 1 / 65536.0; // 32768.0f;
 int i;
 for (i = 0; i < n; i++) d[i] = scale * s[i];
}


static void convertFLTP(void * vd, const void * vs, int n, int nch) {
 const float ** s = (const float**)vs;
 float * d = (float*)vd;
 int i, ch;
 for (ch = 0; ch < nch; ch++)
  for (i = 0; i < n; i++) d[i*nch+ch] = s[ch][i];
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
	log.Println("channels ", d.cc.channels, "rate ", d.cc.sample_rate)
	d.cc.extradata = (*C.uint8_t)(&data[0])
	d.cc.extradata_size = C.int(len(data))
	C.avcodec_open2(d.cc, d.c, nil)
	return &d
}

func (d *Decoder) Decode(data []byte) []float32 {
	var pkt C.AVPacket
	var fr *C.AVFrame
	var got C.int
	fr = C.av_frame_alloc()
	defer C.av_frame_free(&fr)
	C.av_init_packet(&pkt)
	defer C.av_packet_unref(&pkt)
	pkt.data = (*C.uint8_t)(&data[0])
	pkt.size = C.int(len(data))
	dec := C.avcodec_decode_audio4(d.cc, fr, &got, &pkt)
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
	case C.AV_SAMPLE_FMT_FLTP:
		C.convertFLTP(dst, unsafe.Pointer(&fr.data[0]), nvals, d.cc.channels)
	default:
		log.Panic("Unsupported format ", d.cc.sample_fmt)
	}
	return buf
}
