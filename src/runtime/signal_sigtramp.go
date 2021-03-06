// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build dragonfly linux netbsd

package runtime

import "unsafe"

// Continuation of the (assembly) sigtramp() logic.
// This may be called with the world stopped.
//go:nosplit
//go:nowritebarrierrec
func sigtrampgo(sig uint32, info *siginfo, ctx unsafe.Pointer) {
	if sigfwdgo(sig, info, ctx) {
		return
	}
	g := getg()
	if g == nil {
		if sig == _SIGPROF {
			// Ignore profiling signals that arrive on
			// non-Go threads. On some systems they will
			// be handled directly by the signal handler,
			// by calling sigprofNonGo, in which case we won't
			// get here anyhow.
			return
		}
		badsignal(uintptr(sig), &sigctxt{info, ctx})
		return
	}

	// If some non-Go code called sigaltstack, adjust.
	sp := uintptr(unsafe.Pointer(&sig))
	if sp < g.m.gsignal.stack.lo || sp >= g.m.gsignal.stack.hi {
		var st stackt
		sigaltstack(nil, &st)
		if st.ss_flags&_SS_DISABLE != 0 {
			setg(nil)
			cgocallback(unsafe.Pointer(funcPC(noSignalStack)), noescape(unsafe.Pointer(&sig)), unsafe.Sizeof(sig), 0)
		}
		stsp := uintptr(unsafe.Pointer(st.ss_sp))
		if sp < stsp || sp >= stsp+st.ss_size {
			setg(nil)
			cgocallback(unsafe.Pointer(funcPC(sigNotOnStack)), noescape(unsafe.Pointer(&sig)), unsafe.Sizeof(sig), 0)
		}
		setGsignalStack(&st)
		g.m.gsignal.stktopsp = getcallersp(unsafe.Pointer(&sig))
	}

	setg(g.m.gsignal)
	sighandler(sig, info, ctx, g)
	setg(g)
}
