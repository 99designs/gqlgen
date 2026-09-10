package disableconcurrency

import "context"

type Methods struct{}

func (Methods) NoContext() bool { return true }

func (Methods) WithContext(_ context.Context) bool { return true }

func (Methods) WithContextInline(_ context.Context) bool { return true }

type InlineObject struct{}

func (InlineObject) A(_ context.Context) bool { return true }

func (InlineObject) B(_ context.Context) bool { return true }
