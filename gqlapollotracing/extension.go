package gqlapollotracing

import (
	"sync"
	"time"
)

type tracingData struct {
	mu sync.Mutex `json:"-"`

	StartTime  time.Time     `json:"startTime"`
	EndTime    time.Time     `json:"endTime"`
	Duration   time.Duration `json:"duration"`
	Parsing    *startOffset  `json:"parsing"`
	Validation *startOffset  `json:"validation"`
	Execution  *execution    `json:"execution"`
}

func (td *tracingData) prepare() {
	td.Duration = time.Time(td.EndTime).Sub(time.Time(td.StartTime))
	if td.Parsing != nil {
		td.Parsing.prepare(td)
	}
	if td.Validation != nil {
		td.Validation.prepare(td)
	}
	if td.Execution != nil {
		td.Execution.prepare(td)
	}
}

type startOffset struct {
	StartTime time.Time `json:"-"`
	EndTime   time.Time `json:"-"`

	StartOffset time.Duration `json:"startOffset"`
	Duration    time.Duration `json:"duration"`
}

func (so *startOffset) prepare(td *tracingData) {
	so.StartOffset = time.Time(so.StartTime).Sub(time.Time(td.StartTime))
	so.Duration = so.EndTime.Sub(so.StartTime)
}

type execution struct {
	Resolvers []*executionSpan `json:"resolvers"`
}

func (e *execution) prepare(td *tracingData) {
	for _, es := range e.Resolvers {
		es.prepare(td)
	}
}

type executionSpan struct {
	startOffset

	Path       []interface{} `json:"path"`
	ParentType string        `json:"parentType"`
	FieldName  string        `json:"fieldName"`
	ReturnType string        `json:"returnType"`
}
