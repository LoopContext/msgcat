package test

import (
	"context"
	"time"
)

type MockContext struct {
	Ctx context.Context
}

func (ctx *MockContext) Context() context.Context {
	return ctx.Ctx
}

func (ctx *MockContext) SetValue(key interface{}, value interface{}) {
	ctx.Ctx = context.WithValue(ctx.Ctx, key, value)
}

func (ctx *MockContext) GetValue(key interface{}) interface{} {
	return ctx.Ctx.Value(key)
}

func (ctx *MockContext) Deadline() (time.Time, bool) {
	return ctx.Ctx.Deadline()
}

func (ctx *MockContext) Done() <-chan struct{} {
	return ctx.Ctx.Done()
}

func (ctx *MockContext) Err() error {
	return ctx.Ctx.Err()
}


func (ctx *MockContext) Value(key interface{}) interface{} {
	return ctx.Ctx.Value(key)
}
