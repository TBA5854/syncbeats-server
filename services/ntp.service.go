package services

import "time"

type NTPResult struct {
	T1 int64
	T2 int64
	T3 int64
}

// HandlePing records T2 at call time.  The caller must set T3 immediately
// before writing the response to the wire.
//
// Client-side offset calculation:
//
//	RTT    = (T4 - T1) - (T3 - T2)
//	offset = ((T2 - T1) + (T3 - T4)) / 2
//
// where T4 is the time the client receives the pong.
func HandlePing(t1 int64) NTPResult {
	return NTPResult{
		T1: t1,
		T2: time.Now().UnixMilli(),
	}
}

func (r *NTPResult) Seal() {
	r.T3 = time.Now().UnixMilli()
}

func NowMS() int64 {
	return time.Now().UnixMilli()
}
