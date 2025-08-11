package remote

import "time"

func sleepMs(ms int) {
    if ms <= 0 { return }
    time.Sleep(time.Duration(ms) * time.Millisecond)
}


