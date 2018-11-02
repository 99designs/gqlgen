package gqlapollotracing

import "time"

func SetTimeNowFunc(f func() time.Time) {
	timeNowFunc = f
}
