// Lifted from https://golang.org/src/hash/crc32/crc32_amd64.s

// Copyright 2011 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

#include "textflag.h"

// CRC32 polynomial data
//
// These constants are lifted from the
// Linux kernel, since they avoid the costly
// PSHUFB 16 byte reversal proposed in the
// original Intel paper.
DATA r2r1<>+0(SB)/8, $0x154442bd4
DATA r2r1<>+8(SB)/8, $0x1c6e41596
DATA r4r3<>+0(SB)/8, $0x1751997d0
DATA r4r3<>+8(SB)/8, $0x0ccaa009e
DATA rupoly<>+0(SB)/8, $0x1db710641
DATA rupoly<>+8(SB)/8, $0x1f7011641
DATA r5<>+0(SB)/8, $0x163cd6124

GLOBL r2r1<>(SB),RODATA,$16
GLOBL r4r3<>(SB),RODATA,$16
GLOBL rupoly<>(SB),RODATA,$16
GLOBL r5<>(SB),RODATA,$8

// Based on https://www.intel.com/content/dam/www/public/us/en/documents/white-papers/fast-crc-computation-generic-polynomials-pclmulqdq-paper.pdf
// len(p) must be at least 64, and must be a multiple of 16.

// func ieeeCLMUL(sum uint32, p []byte) uint32
TEXT ·ieeeCLMUL(SB),NOSPLIT,$0
  MOVL   sum+0(FP), X0            // Initial CRC value
  MOVQ   p+8(FP), SI              // data pointer
  MOVQ   p_len+16(FP), CX         // len(p)

  MOVOU  (SI), X1
  MOVOU  16(SI), X2
  MOVOU  32(SI), X3
  MOVOU  48(SI), X4
  PXOR   X0, X1
  ADDQ   $64, SI                  // buf+=64
  SUBQ   $64, CX                  // len-=64
  CMPQ   CX, $64                  // Less than 64 bytes left
  JB     remain64

  MOVOA  r2r1<>+0(SB), X0
loopback64:
  MOVOA  X1, X5
  MOVOA  X2, X6
  MOVOA  X3, X7
  MOVOA  X4, X8

  PCLMULQDQ $0, X0, X1
  PCLMULQDQ $0, X0, X2
  PCLMULQDQ $0, X0, X3
  PCLMULQDQ $0, X0, X4

  /* Load next early */
  MOVOU    (SI), X11
  MOVOU    16(SI), X12
  MOVOU    32(SI), X13
  MOVOU    48(SI), X14

  PCLMULQDQ $0x11, X0, X5
  PCLMULQDQ $0x11, X0, X6
  PCLMULQDQ $0x11, X0, X7
  PCLMULQDQ $0x11, X0, X8

  PXOR     X5, X1
  PXOR     X6, X2
  PXOR     X7, X3
  PXOR     X8, X4

  PXOR     X11, X1
  PXOR     X12, X2
  PXOR     X13, X3
  PXOR     X14, X4

  ADDQ    $0x40, DI
  ADDQ    $64, SI      // buf+=64
  SUBQ    $64, CX      // len-=64
  CMPQ    CX, $64      // Less than 64 bytes left?
  JGE     loopback64

  /* Fold result into a single register (X1) */
remain64:
  MOVOA       r4r3<>+0(SB), X0

  MOVOA       X1, X5
  PCLMULQDQ   $0, X0, X1
  PCLMULQDQ   $0x11, X0, X5
  PXOR        X5, X1
  PXOR        X2, X1

  MOVOA       X1, X5
  PCLMULQDQ   $0, X0, X1
  PCLMULQDQ   $0x11, X0, X5
  PXOR        X5, X1
  PXOR        X3, X1

  MOVOA       X1, X5
  PCLMULQDQ   $0, X0, X1
  PCLMULQDQ   $0x11, X0, X5
  PXOR        X5, X1
  PXOR        X4, X1

  /* If there is less than 16 bytes left we are done */
  CMPQ        CX, $16
  JB          finish

  /* Encode 16 bytes */
remain16:
  MOVOU       (SI), X10
  MOVOA       X1, X5
  PCLMULQDQ   $0, X0, X1
  PCLMULQDQ   $0x11, X0, X5
  PXOR        X5, X1
  PXOR        X10, X1
  SUBQ        $16, CX
  ADDQ        $16, SI
  CMPQ        CX, $16
  JGE         remain16

finish:
  /* Fold final result into 32 bits and return it */
  PCMPEQB     X3, X3
  PCLMULQDQ   $1, X1, X0
  PSRLDQ      $8, X1
  PXOR        X0, X1

  MOVOA       X1, X2
  MOVQ        r5<>+0(SB), X0

  /* Creates 32 bit mask. Note that we don't care about upper half. */
  PSRLQ       $32, X3

  PSRLDQ      $4, X2
  PAND        X3, X1
  PCLMULQDQ   $0, X0, X1
  PXOR        X2, X1

  MOVOA       rupoly<>+0(SB), X0

  MOVOA       X1, X2
  PAND        X3, X1
  PCLMULQDQ   $0x10, X0, X1
  PAND        X3, X1
  PCLMULQDQ   $0, X0, X1
  PXOR        X2, X1

  PEXTRD  $1, X1, AX
  MOVL        AX, ret+32(FP)

  RET

