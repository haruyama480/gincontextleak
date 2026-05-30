package gin

import "time"

type Context struct{}

func (*Context) Deadline() (time.Time, bool) {
	return time.Time{}, false
}

func (*Context) Done() <-chan struct{} {
	return nil
}

func (*Context) Err() error {
	return nil
}

func (*Context) Value(key any) any {
	return nil
}
