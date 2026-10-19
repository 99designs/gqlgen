package tableimpl

import (
	"context"
	"fmt"
	"io"
	"strconv"

	"github.com/vektah/gqlparser/v2/ast"

	"github.com/99designs/gqlgen/graphql"
)

// Stream receives the events of a subscription field from the channel its resolver
// returns. Generated code declares one per subscription field, with StreamOf or
// EventStreamOf.
type Stream struct {
	// recv returns the function that receives the next event from res, or false when
	// res is not the channel of the field.
	recv func(res any) (func(ctx context.Context) (eventCtx context.Context, v any, ok bool), bool)
	// event is set for fields whose events carry their own context.
	event bool
	name  string
}

// StreamOf returns the Stream of a subscription field whose resolver returns a
// <-chan T.
//
//go:noinline
func StreamOf[T any]() *Stream {
	return &Stream{
		recv: func(res any) (func(context.Context) (context.Context, any, bool), bool) {
			ch, ok := res.(<-chan T)
			if !ok {
				return nil, false
			}
			return func(ctx context.Context) (context.Context, any, bool) {
				select {
				case v, ok := <-ch:
					return ctx, v, ok
				case <-ctx.Done():
					return ctx, nil, false
				}
			}, true
		},
		name: fmt.Sprintf("%T", (<-chan T)(nil)),
	}
}

// EventStreamOf returns the Stream of a subscription field marked
// @subscriptionContext, whose resolver returns a <-chan graphql.Event[T].
//
//go:noinline
func EventStreamOf[T any]() *Stream {
	return &Stream{
		recv: func(res any) (func(context.Context) (context.Context, any, bool), bool) {
			ch, ok := res.(<-chan graphql.Event[T])
			if !ok {
				return nil, false
			}
			return func(ctx context.Context) (context.Context, any, bool) {
				select {
				case ev, ok := <-ch:
					if !ok {
						return ctx, nil, false
					}
					if ev.Context != nil {
						return ev.Context, ev.Value, true
					}
					return ctx, ev.Value, true
				case <-ctx.Done():
					return ctx, nil, false
				}
			}, true
		},
		event: true,
		name:  fmt.Sprintf("%T", (<-chan graphql.Event[T])(nil)),
	}
}

// Subscribe resolves the subscription field selected by sel, and returns the function
// that marshals its next event. The function returns nil when the stream ends. Subscribe
// returns nil when the field cannot be resolved; the error is added to the response.
func (o *Object[EC]) Subscribe(
	ctx context.Context,
	ec EC,
	sel ast.SelectionSet,
) func(context.Context) graphql.Marshaler {
	ctx, f, field, ok := o.subscribedField(ctx, ec, sel)
	if !ok {
		return nil
	}
	next := o.resolveStream(ctx, ec, f, field)
	if next == nil {
		return nil
	}
	return func(ctx context.Context) graphql.Marshaler {
		_, m := next(ctx)
		return m
	}
}

// SubscribeWithEventContext is Subscribe for subscription types with fields marked
// @subscriptionContext. The returned function also returns the context of the event,
// which is the context it was given for the fields without one.
func (o *Object[EC]) SubscribeWithEventContext(
	ctx context.Context,
	ec EC,
	sel ast.SelectionSet,
) func(context.Context) (context.Context, graphql.Marshaler) {
	ctx, f, field, ok := o.subscribedField(ctx, ec, sel)
	if !ok {
		return nil
	}
	next := o.resolveStream(ctx, ec, f, field)
	if next == nil && !f.Stream.event {
		// The functions mode wraps the nil stream of an unmarked field, as
		// graphql.StreamWithoutEventContext.
		return graphql.StreamWithoutEventContext(nil)
	}
	return next
}

// subscribedField returns the one field that sel selects on the subscription type.
func (o *Object[EC]) subscribedField(
	ctx context.Context,
	ec EC,
	sel ast.SelectionSet,
) (context.Context, *Field[EC], graphql.CollectedField, bool) {
	fields := graphql.CollectFields(ec.OpCtx(), sel, o.implementors)
	ctx = graphql.WithFieldContext(ctx, &graphql.FieldContext{Object: o.Name})
	if len(fields) != 1 {
		graphql.AddErrorf(ctx, "must subscribe to exactly one stream")
		return ctx, nil, graphql.CollectedField{}, false
	}
	f := o.index[fields[0].Name]
	if f == nil {
		panic("unknown field " + strconv.Quote(fields[0].Name))
	}
	return ctx, f, fields[0], true
}

// resolveStream resolves the subscription field f, as graphql.ResolveFieldStream does
// for the functions mode. It returns nil when the field resolves to an error or null.
func (o *Object[EC]) resolveStream(
	ctx context.Context,
	ec EC,
	f *Field[EC],
	field graphql.CollectedField,
) (ret func(context.Context) (context.Context, graphql.Marshaler)) {
	oc := ec.OpCtx()
	fc, err := o.fieldContext(ctx, ec, f, field)
	if err != nil {
		return nil
	}
	ctx = graphql.WithFieldContext(ctx, fc)

	if !o.omitPanicHandler {
		defer func() {
			if r := recover(); r != nil {
				oc.Error(ctx, oc.Recover(ctx, r))
				ret = nil
			}
		}()
	}

	res, err := o.resolve(&ctx, ec, f, field, nil)
	if err != nil {
		oc.Error(ctx, graphql.AddFieldLocationToError(ctx, err))
		return nil
	}
	if res == nil {
		if f.nonNull || fc.NonNull {
			graphql.AddInvalidNullError(ctx, "")
		}
		return nil
	}

	if recv, ok := f.Stream.recv(res); ok {
		fc.Result = res
		return func(ctx context.Context) (context.Context, graphql.Marshaler) {
			eventCtx, v, ok := recv(ctx)
			if !ok {
				return eventCtx, nil
			}
			return eventCtx, graphql.WriterFunc(func(w io.Writer) {
				w.Write([]byte{'{'})
				graphql.MarshalString(field.Alias).MarshalGQL(w)
				w.Write([]byte{':'})
				f.Out.marshal(ctx, ec, field.Selections, v).MarshalGQL(w)
				w.Write([]byte{'}'})
			})
		}
	}

	// A directive may return the stream function itself.
	if f.Stream.event {
		if next, ok := res.(func(context.Context) (context.Context, graphql.Marshaler)); ok {
			fc.Result = next
			return next
		}
	} else if next, ok := res.(func(context.Context) graphql.Marshaler); ok {
		fc.Result = next
		return graphql.StreamWithoutEventContext(next)
	}
	graphql.AddErrorf(
		ctx,
		`unexpected type %T from middleware/directive chain, should be %s`,
		res,
		f.Stream.name,
	)
	return nil
}
