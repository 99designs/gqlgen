package handler

import (
	"context"
	"github.com/99designs/gqlgen/graphql"
	"github.com/99designs/gqlgen/graphql/errcode"
	"github.com/99designs/gqlgen/graphql/executor"
	"github.com/pkg/errors"
	"github.com/vektah/gqlparser/v2/gqlerror"
)

func (s *Server) ExecGraphCommand(ctx context.Context, params *graphql.RawParams) (*graphql.Response, error) {
	var response *graphql.Response
	defer func() {
		if err := recover(); err != nil {
			err := s.exec.PresentRecoveredError(ctx, err)
			_errs := make([]*gqlerror.Error, 0)
			_errs = append(_errs, &gqlerror.Error{Message: err.Error()})
			response = &graphql.Response{Errors: _errs}
		}
	}()

	// Deliberately assigning value to `response` so that we can also capture the value from `defer func() block`
	ctx = graphql.StartOperationTrace(ctx)
	rc, err := s.exec.CreateOperationContext(ctx, params)
	if err != nil {
		response = s.exec.DispatchError(graphql.WithOperationContext(ctx, rc), err)
		return response, nil
	}
	responses, responseContext := s.exec.DispatchOperation(ctx, rc)
	response = responses(responseContext)
	return response, nil
}

type SubscriptionHandler struct {
	operationContext *graphql.OperationContext
	exec             *executor.Executor
	ctx              context.Context
	Response         *graphql.Response
	PanicHandler     func() *gqlerror.Error
}

func (s SubscriptionHandler) Exec() (graphql.ResponseHandler, context.Context) {
	return s.exec.DispatchOperation(s.ctx, s.operationContext)
}

func (s *Server) ExecGraphSubscriptionsCommand(ctx context.Context, params *graphql.RawParams) (SubscriptionHandler, error) {
	ctx = graphql.StartOperationTrace(ctx)
	rc, err := s.exec.CreateOperationContext(ctx, params)
	if err != nil {
		resp := s.exec.DispatchError(graphql.WithOperationContext(ctx, rc), err)
		switch errcode.GetErrorKind(err) {
		case errcode.KindProtocol:
			return SubscriptionHandler{}, resp.Errors
		default:
			return SubscriptionHandler{Response: &graphql.Response{Errors: err}}, nil
		}
	}

	ctx = graphql.WithOperationContext(ctx, rc)

	ctx, cancel := context.WithCancel(ctx)
	//c.mu.Lock()
	//c.active[msg.id] = cancel
	//c.mu.Unlock()

	subHandler := SubscriptionHandler{}
	subHandler.operationContext = rc
	subHandler.ctx = ctx
	subHandler.exec = s.exec
	subHandler.PanicHandler = func() *gqlerror.Error {
		defer cancel()
		if r := recover(); r != nil {
			err := subHandler.operationContext.Recover(ctx, r)
			var gqlerr *gqlerror.Error
			if !errors.As(err, &gqlerr) {
				gqlerr = &gqlerror.Error{}
				if err != nil {
					gqlerr.Message = err.Error()
					return gqlerr
				}
			}
		}
		//c.complete(msg.id)
		//c.mu.Lock()
		//delete(c.active, msg.id)
		//c.mu.Unlock()
		return nil
	}
	return subHandler, nil
}
