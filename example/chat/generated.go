package chat

import (
	"bytes"
	"context"
	"errors"
	"io"
	"strconv"
	"sync"
	"time"

	"github.com/99designs/gqlgen/graphql"
	"github.com/99designs/gqlgen/graphql/introspection"
	"github.com/vektah/gqlparser"
	"github.com/vektah/gqlparser/ast"
)

// region    ************************** generated!.gotpl **************************

// NewExecutableSchema creates an ExecutableSchema from the ResolverRoot interface.
func NewExecutableSchema(cfg Config) graphql.ExecutableSchema {
	return &executableSchema{
		resolvers:  cfg.Resolvers,
		directives: cfg.Directives,
		complexity: cfg.Complexity,
	}
}

type Config struct {
	Resolvers  ResolverRoot
	Directives DirectiveRoot
	Complexity ComplexityRoot
}

type ResolverRoot interface {
	Mutation() MutationResolver
	Query() QueryResolver
	Subscription() SubscriptionResolver
}

type DirectiveRoot struct {
}

type ComplexityRoot struct {
	Chatroom struct {
		Name     func(childComplexity int) int
		Messages func(childComplexity int) int
	}

	Message struct {
		Id        func(childComplexity int) int
		Text      func(childComplexity int) int
		CreatedBy func(childComplexity int) int
		CreatedAt func(childComplexity int) int
	}

	Mutation struct {
		Post func(childComplexity int, text string, username string, roomName string) int
	}

	Query struct {
		Room func(childComplexity int, name string) int
	}

	Subscription struct {
		MessageAdded func(childComplexity int, roomName string) int
	}
}

type MutationResolver interface {
	Post(ctx context.Context, text string, username string, roomName string) (Message, error)
}
type QueryResolver interface {
	Room(ctx context.Context, name string) (*Chatroom, error)
}
type SubscriptionResolver interface {
	MessageAdded(ctx context.Context, roomName string) (<-chan Message, error)
}

type executableSchema struct {
	resolvers  ResolverRoot
	directives DirectiveRoot
	complexity ComplexityRoot
}

func (e *executableSchema) Schema() *ast.Schema {
	return parsedSchema
}

func (e *executableSchema) Complexity(typeName, field string, childComplexity int, rawArgs map[string]interface{}) (int, bool) {
	switch typeName + "." + field {

	case "Chatroom.name":
		if e.complexity.Chatroom.Name == nil {
			break
		}

		return e.complexity.Chatroom.Name(childComplexity), true

	case "Chatroom.messages":
		if e.complexity.Chatroom.Messages == nil {
			break
		}

		return e.complexity.Chatroom.Messages(childComplexity), true

	case "Message.id":
		if e.complexity.Message.Id == nil {
			break
		}

		return e.complexity.Message.Id(childComplexity), true

	case "Message.text":
		if e.complexity.Message.Text == nil {
			break
		}

		return e.complexity.Message.Text(childComplexity), true

	case "Message.createdBy":
		if e.complexity.Message.CreatedBy == nil {
			break
		}

		return e.complexity.Message.CreatedBy(childComplexity), true

	case "Message.createdAt":
		if e.complexity.Message.CreatedAt == nil {
			break
		}

		return e.complexity.Message.CreatedAt(childComplexity), true

	case "Mutation.post":
		if e.complexity.Mutation.Post == nil {
			break
		}

		args, err := e.field_Mutation_post_args(context.TODO(), rawArgs)
		if err != nil {
			return 0, false
		}

		return e.complexity.Mutation.Post(childComplexity, args["text"].(string), args["username"].(string), args["roomName"].(string)), true

	case "Query.room":
		if e.complexity.Query.Room == nil {
			break
		}

		args, err := e.field_Query_room_args(context.TODO(), rawArgs)
		if err != nil {
			return 0, false
		}

		return e.complexity.Query.Room(childComplexity, args["name"].(string)), true

	case "Subscription.messageAdded":
		if e.complexity.Subscription.MessageAdded == nil {
			break
		}

		args, err := e.field_Subscription_messageAdded_args(context.TODO(), rawArgs)
		if err != nil {
			return 0, false
		}

		return e.complexity.Subscription.MessageAdded(childComplexity, args["roomName"].(string)), true

	}
	return 0, false
}

func (e *executableSchema) Query(ctx context.Context, op *ast.OperationDefinition) *graphql.Response {
	ec := executionContext{graphql.GetRequestContext(ctx), e}

	buf := ec.RequestMiddleware(ctx, func(ctx context.Context) []byte {
		data := ec._Query(ctx, op.SelectionSet)
		var buf bytes.Buffer
		data.MarshalGQL(&buf)
		return buf.Bytes()
	})

	return &graphql.Response{
		Data:       buf,
		Errors:     ec.Errors,
		Extensions: ec.Extensions,
	}
}

func (e *executableSchema) Mutation(ctx context.Context, op *ast.OperationDefinition) *graphql.Response {
	ec := executionContext{graphql.GetRequestContext(ctx), e}

	buf := ec.RequestMiddleware(ctx, func(ctx context.Context) []byte {
		data := ec._Mutation(ctx, op.SelectionSet)
		var buf bytes.Buffer
		data.MarshalGQL(&buf)
		return buf.Bytes()
	})

	return &graphql.Response{
		Data:       buf,
		Errors:     ec.Errors,
		Extensions: ec.Extensions,
	}
}

func (e *executableSchema) Subscription(ctx context.Context, op *ast.OperationDefinition) func() *graphql.Response {
	ec := executionContext{graphql.GetRequestContext(ctx), e}

	next := ec._Subscription(ctx, op.SelectionSet)
	if ec.Errors != nil {
		return graphql.OneShot(&graphql.Response{Data: []byte("null"), Errors: ec.Errors})
	}

	var buf bytes.Buffer
	return func() *graphql.Response {
		buf := ec.RequestMiddleware(ctx, func(ctx context.Context) []byte {
			buf.Reset()
			data := next()

			if data == nil {
				return nil
			}
			data.MarshalGQL(&buf)
			return buf.Bytes()
		})

		if buf == nil {
			return nil
		}

		return &graphql.Response{
			Data:       buf,
			Errors:     ec.Errors,
			Extensions: ec.Extensions,
		}
	}
}

type executionContext struct {
	*graphql.RequestContext
	*executableSchema
}

func (ec *executionContext) FieldMiddleware(ctx context.Context, obj interface{}, next graphql.Resolver) (ret interface{}) {
	defer func() {
		if r := recover(); r != nil {
			ec.Error(ctx, ec.Recover(ctx, r))
			ret = nil
		}
	}()
	res, err := ec.ResolverMiddleware(ctx, next)
	if err != nil {
		ec.Error(ctx, err)
		return nil
	}
	return res
}

func (ec *executionContext) introspectSchema() (*introspection.Schema, error) {
	if ec.DisableIntrospection {
		return nil, errors.New("introspection disabled")
	}
	return introspection.WrapSchema(parsedSchema), nil
}

func (ec *executionContext) introspectType(name string) (*introspection.Type, error) {
	if ec.DisableIntrospection {
		return nil, errors.New("introspection disabled")
	}
	return introspection.WrapTypeFromDef(parsedSchema, parsedSchema.Types[name]), nil
}

var parsedSchema = gqlparser.MustLoadSchema(
	&ast.Source{Name: "schema.graphql", Input: `type Chatroom {
    name: String!
    messages: [Message!]!
}

type Message {
    id: ID!
    text: String!
    createdBy: String!
    createdAt: Time!
}

type Query {
    room(name:String!): Chatroom
}

type Mutation {
    post(text: String!, username: String!, roomName: String!): Message!
}

type Subscription {
    messageAdded(roomName: String!): Message!
}

scalar Time
`},
)

// ChainFieldMiddleware add chain by FieldMiddleware
// nolint: deadcode
func chainFieldMiddleware(handleFunc ...graphql.FieldMiddleware) graphql.FieldMiddleware {
	n := len(handleFunc)

	if n > 1 {
		lastI := n - 1
		return func(ctx context.Context, next graphql.Resolver) (interface{}, error) {
			var (
				chainHandler graphql.Resolver
				curI         int
			)
			chainHandler = func(currentCtx context.Context) (interface{}, error) {
				if curI == lastI {
					return next(currentCtx)
				}
				curI++
				res, err := handleFunc[curI](currentCtx, chainHandler)
				curI--
				return res, err

			}
			return handleFunc[0](ctx, chainHandler)
		}
	}

	if n == 1 {
		return handleFunc[0]
	}

	return func(ctx context.Context, next graphql.Resolver) (interface{}, error) {
		return next(ctx)
	}
}

// endregion ************************** generated!.gotpl **************************

// region    ***************************** args.gotpl *****************************

func (e *executableSchema) field_Mutation_post_args(ctx context.Context, rawArgs map[string]interface{}) (map[string]interface{}, error) {
	var err error
	args := map[string]interface{}{}
	var arg0 string
	if tmp, ok := rawArgs["text"]; ok {
		arg0, err = e.unmarshalString2string(tmp)
		if err != nil {
			return nil, err
		}
	}
	args["text"] = arg0
	var arg1 string
	if tmp, ok := rawArgs["username"]; ok {
		arg1, err = e.unmarshalString2string(tmp)
		if err != nil {
			return nil, err
		}
	}
	args["username"] = arg1
	var arg2 string
	if tmp, ok := rawArgs["roomName"]; ok {
		arg2, err = e.unmarshalString2string(tmp)
		if err != nil {
			return nil, err
		}
	}
	args["roomName"] = arg2
	return args, nil
}

func (e *executableSchema) field_Query___type_args(ctx context.Context, rawArgs map[string]interface{}) (map[string]interface{}, error) {
	var err error
	args := map[string]interface{}{}
	var arg0 string
	if tmp, ok := rawArgs["name"]; ok {
		arg0, err = e.unmarshalString2string(tmp)
		if err != nil {
			return nil, err
		}
	}
	args["name"] = arg0
	return args, nil
}

func (e *executableSchema) field_Query_room_args(ctx context.Context, rawArgs map[string]interface{}) (map[string]interface{}, error) {
	var err error
	args := map[string]interface{}{}
	var arg0 string
	if tmp, ok := rawArgs["name"]; ok {
		arg0, err = e.unmarshalString2string(tmp)
		if err != nil {
			return nil, err
		}
	}
	args["name"] = arg0
	return args, nil
}

func (e *executableSchema) field_Subscription_messageAdded_args(ctx context.Context, rawArgs map[string]interface{}) (map[string]interface{}, error) {
	var err error
	args := map[string]interface{}{}
	var arg0 string
	if tmp, ok := rawArgs["roomName"]; ok {
		arg0, err = e.unmarshalString2string(tmp)
		if err != nil {
			return nil, err
		}
	}
	args["roomName"] = arg0
	return args, nil
}

func (e *executableSchema) field___Type_enumValues_args(ctx context.Context, rawArgs map[string]interface{}) (map[string]interface{}, error) {
	var err error
	args := map[string]interface{}{}
	var arg0 bool
	if tmp, ok := rawArgs["includeDeprecated"]; ok {
		arg0, err = e.unmarshalBoolean2bool(tmp)
		if err != nil {
			return nil, err
		}
	}
	args["includeDeprecated"] = arg0
	return args, nil
}

func (e *executableSchema) field___Type_fields_args(ctx context.Context, rawArgs map[string]interface{}) (map[string]interface{}, error) {
	var err error
	args := map[string]interface{}{}
	var arg0 bool
	if tmp, ok := rawArgs["includeDeprecated"]; ok {
		arg0, err = e.unmarshalBoolean2bool(tmp)
		if err != nil {
			return nil, err
		}
	}
	args["includeDeprecated"] = arg0
	return args, nil
}

// endregion ***************************** args.gotpl *****************************

// region    **************************** field.gotpl *****************************

// nolint: vetshadow
func (ec *executionContext) _Chatroom_name(ctx context.Context, field graphql.CollectedField, obj *Chatroom) graphql.Marshaler {
	ctx = ec.Tracer.StartFieldExecution(ctx, field)
	defer func() { ec.Tracer.EndFieldExecution(ctx) }()
	rctx := &graphql.ResolverContext{
		Object: "Chatroom",
		Field:  field,
		Args:   nil,
	}
	ctx = graphql.WithResolverContext(ctx, rctx)
	ctx = ec.Tracer.StartFieldResolverExecution(ctx, rctx)
	resTmp := ec.FieldMiddleware(ctx, obj, func(rctx context.Context) (interface{}, error) {
		ctx = rctx // use context from middleware stack in children
		return obj.Name, nil
	})
	if resTmp == nil {
		if !ec.HasError(rctx) {
			ec.Errorf(ctx, "must not be null")
		}
		return graphql.Null
	}
	res := resTmp.(string)
	rctx.Result = res
	ctx = ec.Tracer.StartFieldChildExecution(ctx)
	return graphql.MarshalString(res)
}

// nolint: vetshadow
func (ec *executionContext) _Chatroom_messages(ctx context.Context, field graphql.CollectedField, obj *Chatroom) graphql.Marshaler {
	ctx = ec.Tracer.StartFieldExecution(ctx, field)
	defer func() { ec.Tracer.EndFieldExecution(ctx) }()
	rctx := &graphql.ResolverContext{
		Object: "Chatroom",
		Field:  field,
		Args:   nil,
	}
	ctx = graphql.WithResolverContext(ctx, rctx)
	ctx = ec.Tracer.StartFieldResolverExecution(ctx, rctx)
	resTmp := ec.FieldMiddleware(ctx, obj, func(rctx context.Context) (interface{}, error) {
		ctx = rctx // use context from middleware stack in children
		return obj.Messages, nil
	})
	if resTmp == nil {
		if !ec.HasError(rctx) {
			ec.Errorf(ctx, "must not be null")
		}
		return graphql.Null
	}
	res := resTmp.([]Message)
	rctx.Result = res
	ctx = ec.Tracer.StartFieldChildExecution(ctx)

	arr1 := make(graphql.Array, len(res))
	var wg sync.WaitGroup

	isLen1 := len(res) == 1
	if !isLen1 {
		wg.Add(len(res))
	}

	for idx1 := range res {
		idx1 := idx1
		rctx := &graphql.ResolverContext{
			Index:  &idx1,
			Result: &res[idx1],
		}
		ctx := graphql.WithResolverContext(ctx, rctx)
		f := func(idx1 int) {
			if !isLen1 {
				defer wg.Done()
			}
			arr1[idx1] = func() graphql.Marshaler {

				return ec._Message(ctx, field.Selections, &res[idx1])
			}()
		}
		if isLen1 {
			f(idx1)
		} else {
			go f(idx1)
		}

	}
	wg.Wait()
	return arr1
}

// nolint: vetshadow
func (ec *executionContext) _Message_id(ctx context.Context, field graphql.CollectedField, obj *Message) graphql.Marshaler {
	ctx = ec.Tracer.StartFieldExecution(ctx, field)
	defer func() { ec.Tracer.EndFieldExecution(ctx) }()
	rctx := &graphql.ResolverContext{
		Object: "Message",
		Field:  field,
		Args:   nil,
	}
	ctx = graphql.WithResolverContext(ctx, rctx)
	ctx = ec.Tracer.StartFieldResolverExecution(ctx, rctx)
	resTmp := ec.FieldMiddleware(ctx, obj, func(rctx context.Context) (interface{}, error) {
		ctx = rctx // use context from middleware stack in children
		return obj.ID, nil
	})
	if resTmp == nil {
		if !ec.HasError(rctx) {
			ec.Errorf(ctx, "must not be null")
		}
		return graphql.Null
	}
	res := resTmp.(string)
	rctx.Result = res
	ctx = ec.Tracer.StartFieldChildExecution(ctx)
	return graphql.MarshalID(res)
}

// nolint: vetshadow
func (ec *executionContext) _Message_text(ctx context.Context, field graphql.CollectedField, obj *Message) graphql.Marshaler {
	ctx = ec.Tracer.StartFieldExecution(ctx, field)
	defer func() { ec.Tracer.EndFieldExecution(ctx) }()
	rctx := &graphql.ResolverContext{
		Object: "Message",
		Field:  field,
		Args:   nil,
	}
	ctx = graphql.WithResolverContext(ctx, rctx)
	ctx = ec.Tracer.StartFieldResolverExecution(ctx, rctx)
	resTmp := ec.FieldMiddleware(ctx, obj, func(rctx context.Context) (interface{}, error) {
		ctx = rctx // use context from middleware stack in children
		return obj.Text, nil
	})
	if resTmp == nil {
		if !ec.HasError(rctx) {
			ec.Errorf(ctx, "must not be null")
		}
		return graphql.Null
	}
	res := resTmp.(string)
	rctx.Result = res
	ctx = ec.Tracer.StartFieldChildExecution(ctx)
	return graphql.MarshalString(res)
}

// nolint: vetshadow
func (ec *executionContext) _Message_createdBy(ctx context.Context, field graphql.CollectedField, obj *Message) graphql.Marshaler {
	ctx = ec.Tracer.StartFieldExecution(ctx, field)
	defer func() { ec.Tracer.EndFieldExecution(ctx) }()
	rctx := &graphql.ResolverContext{
		Object: "Message",
		Field:  field,
		Args:   nil,
	}
	ctx = graphql.WithResolverContext(ctx, rctx)
	ctx = ec.Tracer.StartFieldResolverExecution(ctx, rctx)
	resTmp := ec.FieldMiddleware(ctx, obj, func(rctx context.Context) (interface{}, error) {
		ctx = rctx // use context from middleware stack in children
		return obj.CreatedBy, nil
	})
	if resTmp == nil {
		if !ec.HasError(rctx) {
			ec.Errorf(ctx, "must not be null")
		}
		return graphql.Null
	}
	res := resTmp.(string)
	rctx.Result = res
	ctx = ec.Tracer.StartFieldChildExecution(ctx)
	return graphql.MarshalString(res)
}

// nolint: vetshadow
func (ec *executionContext) _Message_createdAt(ctx context.Context, field graphql.CollectedField, obj *Message) graphql.Marshaler {
	ctx = ec.Tracer.StartFieldExecution(ctx, field)
	defer func() { ec.Tracer.EndFieldExecution(ctx) }()
	rctx := &graphql.ResolverContext{
		Object: "Message",
		Field:  field,
		Args:   nil,
	}
	ctx = graphql.WithResolverContext(ctx, rctx)
	ctx = ec.Tracer.StartFieldResolverExecution(ctx, rctx)
	resTmp := ec.FieldMiddleware(ctx, obj, func(rctx context.Context) (interface{}, error) {
		ctx = rctx // use context from middleware stack in children
		return obj.CreatedAt, nil
	})
	if resTmp == nil {
		if !ec.HasError(rctx) {
			ec.Errorf(ctx, "must not be null")
		}
		return graphql.Null
	}
	res := resTmp.(time.Time)
	rctx.Result = res
	ctx = ec.Tracer.StartFieldChildExecution(ctx)
	return graphql.MarshalTime(res)
}

// nolint: vetshadow
func (ec *executionContext) _Mutation_post(ctx context.Context, field graphql.CollectedField) graphql.Marshaler {
	ctx = ec.Tracer.StartFieldExecution(ctx, field)
	defer func() { ec.Tracer.EndFieldExecution(ctx) }()
	rctx := &graphql.ResolverContext{
		Object: "Mutation",
		Field:  field,
		Args:   nil,
	}
	ctx = graphql.WithResolverContext(ctx, rctx)
	rawArgs := field.ArgumentMap(ec.Variables)
	args, err := ec.field_Mutation_post_args(ctx, rawArgs)
	if err != nil {
		ec.Error(ctx, err)
		return graphql.Null
	}
	rctx.Args = args
	ctx = ec.Tracer.StartFieldResolverExecution(ctx, rctx)
	resTmp := ec.FieldMiddleware(ctx, nil, func(rctx context.Context) (interface{}, error) {
		ctx = rctx // use context from middleware stack in children
		return ec.resolvers.Mutation().Post(rctx, args["text"].(string), args["username"].(string), args["roomName"].(string))
	})
	if resTmp == nil {
		if !ec.HasError(rctx) {
			ec.Errorf(ctx, "must not be null")
		}
		return graphql.Null
	}
	res := resTmp.(Message)
	rctx.Result = res
	ctx = ec.Tracer.StartFieldChildExecution(ctx)

	return ec._Message(ctx, field.Selections, &res)
}

// nolint: vetshadow
func (ec *executionContext) _Query_room(ctx context.Context, field graphql.CollectedField) graphql.Marshaler {
	ctx = ec.Tracer.StartFieldExecution(ctx, field)
	defer func() { ec.Tracer.EndFieldExecution(ctx) }()
	rctx := &graphql.ResolverContext{
		Object: "Query",
		Field:  field,
		Args:   nil,
	}
	ctx = graphql.WithResolverContext(ctx, rctx)
	rawArgs := field.ArgumentMap(ec.Variables)
	args, err := ec.field_Query_room_args(ctx, rawArgs)
	if err != nil {
		ec.Error(ctx, err)
		return graphql.Null
	}
	rctx.Args = args
	ctx = ec.Tracer.StartFieldResolverExecution(ctx, rctx)
	resTmp := ec.FieldMiddleware(ctx, nil, func(rctx context.Context) (interface{}, error) {
		ctx = rctx // use context from middleware stack in children
		return ec.resolvers.Query().Room(rctx, args["name"].(string))
	})
	if resTmp == nil {
		return graphql.Null
	}
	res := resTmp.(*Chatroom)
	rctx.Result = res
	ctx = ec.Tracer.StartFieldChildExecution(ctx)

	if res == nil {
		return graphql.Null
	}

	return ec._Chatroom(ctx, field.Selections, res)
}

// nolint: vetshadow
func (ec *executionContext) _Query___type(ctx context.Context, field graphql.CollectedField) graphql.Marshaler {
	ctx = ec.Tracer.StartFieldExecution(ctx, field)
	defer func() { ec.Tracer.EndFieldExecution(ctx) }()
	rctx := &graphql.ResolverContext{
		Object: "Query",
		Field:  field,
		Args:   nil,
	}
	ctx = graphql.WithResolverContext(ctx, rctx)
	rawArgs := field.ArgumentMap(ec.Variables)
	args, err := ec.field_Query___type_args(ctx, rawArgs)
	if err != nil {
		ec.Error(ctx, err)
		return graphql.Null
	}
	rctx.Args = args
	ctx = ec.Tracer.StartFieldResolverExecution(ctx, rctx)
	resTmp := ec.FieldMiddleware(ctx, nil, func(rctx context.Context) (interface{}, error) {
		ctx = rctx // use context from middleware stack in children
		return ec.introspectType(args["name"].(string))
	})
	if resTmp == nil {
		return graphql.Null
	}
	res := resTmp.(*introspection.Type)
	rctx.Result = res
	ctx = ec.Tracer.StartFieldChildExecution(ctx)

	if res == nil {
		return graphql.Null
	}

	return ec.___Type(ctx, field.Selections, res)
}

// nolint: vetshadow
func (ec *executionContext) _Query___schema(ctx context.Context, field graphql.CollectedField) graphql.Marshaler {
	ctx = ec.Tracer.StartFieldExecution(ctx, field)
	defer func() { ec.Tracer.EndFieldExecution(ctx) }()
	rctx := &graphql.ResolverContext{
		Object: "Query",
		Field:  field,
		Args:   nil,
	}
	ctx = graphql.WithResolverContext(ctx, rctx)
	ctx = ec.Tracer.StartFieldResolverExecution(ctx, rctx)
	resTmp := ec.FieldMiddleware(ctx, nil, func(rctx context.Context) (interface{}, error) {
		ctx = rctx // use context from middleware stack in children
		return ec.introspectSchema()
	})
	if resTmp == nil {
		return graphql.Null
	}
	res := resTmp.(*introspection.Schema)
	rctx.Result = res
	ctx = ec.Tracer.StartFieldChildExecution(ctx)

	if res == nil {
		return graphql.Null
	}

	return ec.___Schema(ctx, field.Selections, res)
}

func (ec *executionContext) _Subscription_messageAdded(ctx context.Context, field graphql.CollectedField) func() graphql.Marshaler {
	ctx = graphql.WithResolverContext(ctx, &graphql.ResolverContext{
		Field: field,
		Args:  nil,
	})
	rawArgs := field.ArgumentMap(ec.Variables)
	args, err := ec.field_Subscription_messageAdded_args(ctx, rawArgs)
	if err != nil {
		ec.Error(ctx, err)
		return nil
	}
	// FIXME: subscriptions are missing request middleware stack https://github.com/99designs/gqlgen/issues/259
	//          and Tracer stack
	rctx := ctx
	results, err := ec.resolvers.Subscription().MessageAdded(rctx, args["roomName"].(string))
	if err != nil {
		ec.Error(ctx, err)
		return nil
	}
	return func() graphql.Marshaler {
		res, ok := <-results
		if !ok {
			return nil
		}
		return graphql.WriterFunc(func(w io.Writer) {
			w.Write([]byte{'{'})
			graphql.MarshalString(field.Alias).MarshalGQL(w)
			w.Write([]byte{':'})
			func() graphql.Marshaler {

				return ec._Message(ctx, field.Selections, &res)
			}().MarshalGQL(w)
			w.Write([]byte{'}'})
		})
	}
}

// nolint: vetshadow
func (ec *executionContext) ___Directive_name(ctx context.Context, field graphql.CollectedField, obj *introspection.Directive) graphql.Marshaler {
	ctx = ec.Tracer.StartFieldExecution(ctx, field)
	defer func() { ec.Tracer.EndFieldExecution(ctx) }()
	rctx := &graphql.ResolverContext{
		Object: "__Directive",
		Field:  field,
		Args:   nil,
	}
	ctx = graphql.WithResolverContext(ctx, rctx)
	ctx = ec.Tracer.StartFieldResolverExecution(ctx, rctx)
	resTmp := ec.FieldMiddleware(ctx, obj, func(rctx context.Context) (interface{}, error) {
		ctx = rctx // use context from middleware stack in children
		return obj.Name, nil
	})
	if resTmp == nil {
		if !ec.HasError(rctx) {
			ec.Errorf(ctx, "must not be null")
		}
		return graphql.Null
	}
	res := resTmp.(string)
	rctx.Result = res
	ctx = ec.Tracer.StartFieldChildExecution(ctx)
	return graphql.MarshalString(res)
}

// nolint: vetshadow
func (ec *executionContext) ___Directive_description(ctx context.Context, field graphql.CollectedField, obj *introspection.Directive) graphql.Marshaler {
	ctx = ec.Tracer.StartFieldExecution(ctx, field)
	defer func() { ec.Tracer.EndFieldExecution(ctx) }()
	rctx := &graphql.ResolverContext{
		Object: "__Directive",
		Field:  field,
		Args:   nil,
	}
	ctx = graphql.WithResolverContext(ctx, rctx)
	ctx = ec.Tracer.StartFieldResolverExecution(ctx, rctx)
	resTmp := ec.FieldMiddleware(ctx, obj, func(rctx context.Context) (interface{}, error) {
		ctx = rctx // use context from middleware stack in children
		return obj.Description, nil
	})
	if resTmp == nil {
		return graphql.Null
	}
	res := resTmp.(string)
	rctx.Result = res
	ctx = ec.Tracer.StartFieldChildExecution(ctx)
	return graphql.MarshalString(res)
}

// nolint: vetshadow
func (ec *executionContext) ___Directive_locations(ctx context.Context, field graphql.CollectedField, obj *introspection.Directive) graphql.Marshaler {
	ctx = ec.Tracer.StartFieldExecution(ctx, field)
	defer func() { ec.Tracer.EndFieldExecution(ctx) }()
	rctx := &graphql.ResolverContext{
		Object: "__Directive",
		Field:  field,
		Args:   nil,
	}
	ctx = graphql.WithResolverContext(ctx, rctx)
	ctx = ec.Tracer.StartFieldResolverExecution(ctx, rctx)
	resTmp := ec.FieldMiddleware(ctx, obj, func(rctx context.Context) (interface{}, error) {
		ctx = rctx // use context from middleware stack in children
		return obj.Locations, nil
	})
	if resTmp == nil {
		if !ec.HasError(rctx) {
			ec.Errorf(ctx, "must not be null")
		}
		return graphql.Null
	}
	res := resTmp.([]string)
	rctx.Result = res
	ctx = ec.Tracer.StartFieldChildExecution(ctx)

	arr1 := make(graphql.Array, len(res))

	for idx1 := range res {
		arr1[idx1] = func() graphql.Marshaler {
			return graphql.MarshalString(res[idx1])
		}()
	}

	return arr1
}

// nolint: vetshadow
func (ec *executionContext) ___Directive_args(ctx context.Context, field graphql.CollectedField, obj *introspection.Directive) graphql.Marshaler {
	ctx = ec.Tracer.StartFieldExecution(ctx, field)
	defer func() { ec.Tracer.EndFieldExecution(ctx) }()
	rctx := &graphql.ResolverContext{
		Object: "__Directive",
		Field:  field,
		Args:   nil,
	}
	ctx = graphql.WithResolverContext(ctx, rctx)
	ctx = ec.Tracer.StartFieldResolverExecution(ctx, rctx)
	resTmp := ec.FieldMiddleware(ctx, obj, func(rctx context.Context) (interface{}, error) {
		ctx = rctx // use context from middleware stack in children
		return obj.Args, nil
	})
	if resTmp == nil {
		if !ec.HasError(rctx) {
			ec.Errorf(ctx, "must not be null")
		}
		return graphql.Null
	}
	res := resTmp.([]introspection.InputValue)
	rctx.Result = res
	ctx = ec.Tracer.StartFieldChildExecution(ctx)

	arr1 := make(graphql.Array, len(res))
	var wg sync.WaitGroup

	isLen1 := len(res) == 1
	if !isLen1 {
		wg.Add(len(res))
	}

	for idx1 := range res {
		idx1 := idx1
		rctx := &graphql.ResolverContext{
			Index:  &idx1,
			Result: &res[idx1],
		}
		ctx := graphql.WithResolverContext(ctx, rctx)
		f := func(idx1 int) {
			if !isLen1 {
				defer wg.Done()
			}
			arr1[idx1] = func() graphql.Marshaler {

				return ec.___InputValue(ctx, field.Selections, &res[idx1])
			}()
		}
		if isLen1 {
			f(idx1)
		} else {
			go f(idx1)
		}

	}
	wg.Wait()
	return arr1
}

// nolint: vetshadow
func (ec *executionContext) ___EnumValue_name(ctx context.Context, field graphql.CollectedField, obj *introspection.EnumValue) graphql.Marshaler {
	ctx = ec.Tracer.StartFieldExecution(ctx, field)
	defer func() { ec.Tracer.EndFieldExecution(ctx) }()
	rctx := &graphql.ResolverContext{
		Object: "__EnumValue",
		Field:  field,
		Args:   nil,
	}
	ctx = graphql.WithResolverContext(ctx, rctx)
	ctx = ec.Tracer.StartFieldResolverExecution(ctx, rctx)
	resTmp := ec.FieldMiddleware(ctx, obj, func(rctx context.Context) (interface{}, error) {
		ctx = rctx // use context from middleware stack in children
		return obj.Name, nil
	})
	if resTmp == nil {
		if !ec.HasError(rctx) {
			ec.Errorf(ctx, "must not be null")
		}
		return graphql.Null
	}
	res := resTmp.(string)
	rctx.Result = res
	ctx = ec.Tracer.StartFieldChildExecution(ctx)
	return graphql.MarshalString(res)
}

// nolint: vetshadow
func (ec *executionContext) ___EnumValue_description(ctx context.Context, field graphql.CollectedField, obj *introspection.EnumValue) graphql.Marshaler {
	ctx = ec.Tracer.StartFieldExecution(ctx, field)
	defer func() { ec.Tracer.EndFieldExecution(ctx) }()
	rctx := &graphql.ResolverContext{
		Object: "__EnumValue",
		Field:  field,
		Args:   nil,
	}
	ctx = graphql.WithResolverContext(ctx, rctx)
	ctx = ec.Tracer.StartFieldResolverExecution(ctx, rctx)
	resTmp := ec.FieldMiddleware(ctx, obj, func(rctx context.Context) (interface{}, error) {
		ctx = rctx // use context from middleware stack in children
		return obj.Description, nil
	})
	if resTmp == nil {
		return graphql.Null
	}
	res := resTmp.(string)
	rctx.Result = res
	ctx = ec.Tracer.StartFieldChildExecution(ctx)
	return graphql.MarshalString(res)
}

// nolint: vetshadow
func (ec *executionContext) ___EnumValue_isDeprecated(ctx context.Context, field graphql.CollectedField, obj *introspection.EnumValue) graphql.Marshaler {
	ctx = ec.Tracer.StartFieldExecution(ctx, field)
	defer func() { ec.Tracer.EndFieldExecution(ctx) }()
	rctx := &graphql.ResolverContext{
		Object: "__EnumValue",
		Field:  field,
		Args:   nil,
	}
	ctx = graphql.WithResolverContext(ctx, rctx)
	ctx = ec.Tracer.StartFieldResolverExecution(ctx, rctx)
	resTmp := ec.FieldMiddleware(ctx, obj, func(rctx context.Context) (interface{}, error) {
		ctx = rctx // use context from middleware stack in children
		return obj.IsDeprecated(), nil
	})
	if resTmp == nil {
		if !ec.HasError(rctx) {
			ec.Errorf(ctx, "must not be null")
		}
		return graphql.Null
	}
	res := resTmp.(bool)
	rctx.Result = res
	ctx = ec.Tracer.StartFieldChildExecution(ctx)
	return graphql.MarshalBoolean(res)
}

// nolint: vetshadow
func (ec *executionContext) ___EnumValue_deprecationReason(ctx context.Context, field graphql.CollectedField, obj *introspection.EnumValue) graphql.Marshaler {
	ctx = ec.Tracer.StartFieldExecution(ctx, field)
	defer func() { ec.Tracer.EndFieldExecution(ctx) }()
	rctx := &graphql.ResolverContext{
		Object: "__EnumValue",
		Field:  field,
		Args:   nil,
	}
	ctx = graphql.WithResolverContext(ctx, rctx)
	ctx = ec.Tracer.StartFieldResolverExecution(ctx, rctx)
	resTmp := ec.FieldMiddleware(ctx, obj, func(rctx context.Context) (interface{}, error) {
		ctx = rctx // use context from middleware stack in children
		return obj.DeprecationReason(), nil
	})
	if resTmp == nil {
		return graphql.Null
	}
	res := resTmp.(*string)
	rctx.Result = res
	ctx = ec.Tracer.StartFieldChildExecution(ctx)

	if res == nil {
		return graphql.Null
	}
	return graphql.MarshalString(*res)
}

// nolint: vetshadow
func (ec *executionContext) ___Field_name(ctx context.Context, field graphql.CollectedField, obj *introspection.Field) graphql.Marshaler {
	ctx = ec.Tracer.StartFieldExecution(ctx, field)
	defer func() { ec.Tracer.EndFieldExecution(ctx) }()
	rctx := &graphql.ResolverContext{
		Object: "__Field",
		Field:  field,
		Args:   nil,
	}
	ctx = graphql.WithResolverContext(ctx, rctx)
	ctx = ec.Tracer.StartFieldResolverExecution(ctx, rctx)
	resTmp := ec.FieldMiddleware(ctx, obj, func(rctx context.Context) (interface{}, error) {
		ctx = rctx // use context from middleware stack in children
		return obj.Name, nil
	})
	if resTmp == nil {
		if !ec.HasError(rctx) {
			ec.Errorf(ctx, "must not be null")
		}
		return graphql.Null
	}
	res := resTmp.(string)
	rctx.Result = res
	ctx = ec.Tracer.StartFieldChildExecution(ctx)
	return graphql.MarshalString(res)
}

// nolint: vetshadow
func (ec *executionContext) ___Field_description(ctx context.Context, field graphql.CollectedField, obj *introspection.Field) graphql.Marshaler {
	ctx = ec.Tracer.StartFieldExecution(ctx, field)
	defer func() { ec.Tracer.EndFieldExecution(ctx) }()
	rctx := &graphql.ResolverContext{
		Object: "__Field",
		Field:  field,
		Args:   nil,
	}
	ctx = graphql.WithResolverContext(ctx, rctx)
	ctx = ec.Tracer.StartFieldResolverExecution(ctx, rctx)
	resTmp := ec.FieldMiddleware(ctx, obj, func(rctx context.Context) (interface{}, error) {
		ctx = rctx // use context from middleware stack in children
		return obj.Description, nil
	})
	if resTmp == nil {
		return graphql.Null
	}
	res := resTmp.(string)
	rctx.Result = res
	ctx = ec.Tracer.StartFieldChildExecution(ctx)
	return graphql.MarshalString(res)
}

// nolint: vetshadow
func (ec *executionContext) ___Field_args(ctx context.Context, field graphql.CollectedField, obj *introspection.Field) graphql.Marshaler {
	ctx = ec.Tracer.StartFieldExecution(ctx, field)
	defer func() { ec.Tracer.EndFieldExecution(ctx) }()
	rctx := &graphql.ResolverContext{
		Object: "__Field",
		Field:  field,
		Args:   nil,
	}
	ctx = graphql.WithResolverContext(ctx, rctx)
	ctx = ec.Tracer.StartFieldResolverExecution(ctx, rctx)
	resTmp := ec.FieldMiddleware(ctx, obj, func(rctx context.Context) (interface{}, error) {
		ctx = rctx // use context from middleware stack in children
		return obj.Args, nil
	})
	if resTmp == nil {
		if !ec.HasError(rctx) {
			ec.Errorf(ctx, "must not be null")
		}
		return graphql.Null
	}
	res := resTmp.([]introspection.InputValue)
	rctx.Result = res
	ctx = ec.Tracer.StartFieldChildExecution(ctx)

	arr1 := make(graphql.Array, len(res))
	var wg sync.WaitGroup

	isLen1 := len(res) == 1
	if !isLen1 {
		wg.Add(len(res))
	}

	for idx1 := range res {
		idx1 := idx1
		rctx := &graphql.ResolverContext{
			Index:  &idx1,
			Result: &res[idx1],
		}
		ctx := graphql.WithResolverContext(ctx, rctx)
		f := func(idx1 int) {
			if !isLen1 {
				defer wg.Done()
			}
			arr1[idx1] = func() graphql.Marshaler {

				return ec.___InputValue(ctx, field.Selections, &res[idx1])
			}()
		}
		if isLen1 {
			f(idx1)
		} else {
			go f(idx1)
		}

	}
	wg.Wait()
	return arr1
}

// nolint: vetshadow
func (ec *executionContext) ___Field_type(ctx context.Context, field graphql.CollectedField, obj *introspection.Field) graphql.Marshaler {
	ctx = ec.Tracer.StartFieldExecution(ctx, field)
	defer func() { ec.Tracer.EndFieldExecution(ctx) }()
	rctx := &graphql.ResolverContext{
		Object: "__Field",
		Field:  field,
		Args:   nil,
	}
	ctx = graphql.WithResolverContext(ctx, rctx)
	ctx = ec.Tracer.StartFieldResolverExecution(ctx, rctx)
	resTmp := ec.FieldMiddleware(ctx, obj, func(rctx context.Context) (interface{}, error) {
		ctx = rctx // use context from middleware stack in children
		return obj.Type, nil
	})
	if resTmp == nil {
		if !ec.HasError(rctx) {
			ec.Errorf(ctx, "must not be null")
		}
		return graphql.Null
	}
	res := resTmp.(*introspection.Type)
	rctx.Result = res
	ctx = ec.Tracer.StartFieldChildExecution(ctx)

	if res == nil {
		if !ec.HasError(rctx) {
			ec.Errorf(ctx, "must not be null")
		}
		return graphql.Null
	}

	return ec.___Type(ctx, field.Selections, res)
}

// nolint: vetshadow
func (ec *executionContext) ___Field_isDeprecated(ctx context.Context, field graphql.CollectedField, obj *introspection.Field) graphql.Marshaler {
	ctx = ec.Tracer.StartFieldExecution(ctx, field)
	defer func() { ec.Tracer.EndFieldExecution(ctx) }()
	rctx := &graphql.ResolverContext{
		Object: "__Field",
		Field:  field,
		Args:   nil,
	}
	ctx = graphql.WithResolverContext(ctx, rctx)
	ctx = ec.Tracer.StartFieldResolverExecution(ctx, rctx)
	resTmp := ec.FieldMiddleware(ctx, obj, func(rctx context.Context) (interface{}, error) {
		ctx = rctx // use context from middleware stack in children
		return obj.IsDeprecated(), nil
	})
	if resTmp == nil {
		if !ec.HasError(rctx) {
			ec.Errorf(ctx, "must not be null")
		}
		return graphql.Null
	}
	res := resTmp.(bool)
	rctx.Result = res
	ctx = ec.Tracer.StartFieldChildExecution(ctx)
	return graphql.MarshalBoolean(res)
}

// nolint: vetshadow
func (ec *executionContext) ___Field_deprecationReason(ctx context.Context, field graphql.CollectedField, obj *introspection.Field) graphql.Marshaler {
	ctx = ec.Tracer.StartFieldExecution(ctx, field)
	defer func() { ec.Tracer.EndFieldExecution(ctx) }()
	rctx := &graphql.ResolverContext{
		Object: "__Field",
		Field:  field,
		Args:   nil,
	}
	ctx = graphql.WithResolverContext(ctx, rctx)
	ctx = ec.Tracer.StartFieldResolverExecution(ctx, rctx)
	resTmp := ec.FieldMiddleware(ctx, obj, func(rctx context.Context) (interface{}, error) {
		ctx = rctx // use context from middleware stack in children
		return obj.DeprecationReason(), nil
	})
	if resTmp == nil {
		return graphql.Null
	}
	res := resTmp.(*string)
	rctx.Result = res
	ctx = ec.Tracer.StartFieldChildExecution(ctx)

	if res == nil {
		return graphql.Null
	}
	return graphql.MarshalString(*res)
}

// nolint: vetshadow
func (ec *executionContext) ___InputValue_name(ctx context.Context, field graphql.CollectedField, obj *introspection.InputValue) graphql.Marshaler {
	ctx = ec.Tracer.StartFieldExecution(ctx, field)
	defer func() { ec.Tracer.EndFieldExecution(ctx) }()
	rctx := &graphql.ResolverContext{
		Object: "__InputValue",
		Field:  field,
		Args:   nil,
	}
	ctx = graphql.WithResolverContext(ctx, rctx)
	ctx = ec.Tracer.StartFieldResolverExecution(ctx, rctx)
	resTmp := ec.FieldMiddleware(ctx, obj, func(rctx context.Context) (interface{}, error) {
		ctx = rctx // use context from middleware stack in children
		return obj.Name, nil
	})
	if resTmp == nil {
		if !ec.HasError(rctx) {
			ec.Errorf(ctx, "must not be null")
		}
		return graphql.Null
	}
	res := resTmp.(string)
	rctx.Result = res
	ctx = ec.Tracer.StartFieldChildExecution(ctx)
	return graphql.MarshalString(res)
}

// nolint: vetshadow
func (ec *executionContext) ___InputValue_description(ctx context.Context, field graphql.CollectedField, obj *introspection.InputValue) graphql.Marshaler {
	ctx = ec.Tracer.StartFieldExecution(ctx, field)
	defer func() { ec.Tracer.EndFieldExecution(ctx) }()
	rctx := &graphql.ResolverContext{
		Object: "__InputValue",
		Field:  field,
		Args:   nil,
	}
	ctx = graphql.WithResolverContext(ctx, rctx)
	ctx = ec.Tracer.StartFieldResolverExecution(ctx, rctx)
	resTmp := ec.FieldMiddleware(ctx, obj, func(rctx context.Context) (interface{}, error) {
		ctx = rctx // use context from middleware stack in children
		return obj.Description, nil
	})
	if resTmp == nil {
		return graphql.Null
	}
	res := resTmp.(string)
	rctx.Result = res
	ctx = ec.Tracer.StartFieldChildExecution(ctx)
	return graphql.MarshalString(res)
}

// nolint: vetshadow
func (ec *executionContext) ___InputValue_type(ctx context.Context, field graphql.CollectedField, obj *introspection.InputValue) graphql.Marshaler {
	ctx = ec.Tracer.StartFieldExecution(ctx, field)
	defer func() { ec.Tracer.EndFieldExecution(ctx) }()
	rctx := &graphql.ResolverContext{
		Object: "__InputValue",
		Field:  field,
		Args:   nil,
	}
	ctx = graphql.WithResolverContext(ctx, rctx)
	ctx = ec.Tracer.StartFieldResolverExecution(ctx, rctx)
	resTmp := ec.FieldMiddleware(ctx, obj, func(rctx context.Context) (interface{}, error) {
		ctx = rctx // use context from middleware stack in children
		return obj.Type, nil
	})
	if resTmp == nil {
		if !ec.HasError(rctx) {
			ec.Errorf(ctx, "must not be null")
		}
		return graphql.Null
	}
	res := resTmp.(*introspection.Type)
	rctx.Result = res
	ctx = ec.Tracer.StartFieldChildExecution(ctx)

	if res == nil {
		if !ec.HasError(rctx) {
			ec.Errorf(ctx, "must not be null")
		}
		return graphql.Null
	}

	return ec.___Type(ctx, field.Selections, res)
}

// nolint: vetshadow
func (ec *executionContext) ___InputValue_defaultValue(ctx context.Context, field graphql.CollectedField, obj *introspection.InputValue) graphql.Marshaler {
	ctx = ec.Tracer.StartFieldExecution(ctx, field)
	defer func() { ec.Tracer.EndFieldExecution(ctx) }()
	rctx := &graphql.ResolverContext{
		Object: "__InputValue",
		Field:  field,
		Args:   nil,
	}
	ctx = graphql.WithResolverContext(ctx, rctx)
	ctx = ec.Tracer.StartFieldResolverExecution(ctx, rctx)
	resTmp := ec.FieldMiddleware(ctx, obj, func(rctx context.Context) (interface{}, error) {
		ctx = rctx // use context from middleware stack in children
		return obj.DefaultValue, nil
	})
	if resTmp == nil {
		return graphql.Null
	}
	res := resTmp.(*string)
	rctx.Result = res
	ctx = ec.Tracer.StartFieldChildExecution(ctx)

	if res == nil {
		return graphql.Null
	}
	return graphql.MarshalString(*res)
}

// nolint: vetshadow
func (ec *executionContext) ___Schema_types(ctx context.Context, field graphql.CollectedField, obj *introspection.Schema) graphql.Marshaler {
	ctx = ec.Tracer.StartFieldExecution(ctx, field)
	defer func() { ec.Tracer.EndFieldExecution(ctx) }()
	rctx := &graphql.ResolverContext{
		Object: "__Schema",
		Field:  field,
		Args:   nil,
	}
	ctx = graphql.WithResolverContext(ctx, rctx)
	ctx = ec.Tracer.StartFieldResolverExecution(ctx, rctx)
	resTmp := ec.FieldMiddleware(ctx, obj, func(rctx context.Context) (interface{}, error) {
		ctx = rctx // use context from middleware stack in children
		return obj.Types(), nil
	})
	if resTmp == nil {
		if !ec.HasError(rctx) {
			ec.Errorf(ctx, "must not be null")
		}
		return graphql.Null
	}
	res := resTmp.([]introspection.Type)
	rctx.Result = res
	ctx = ec.Tracer.StartFieldChildExecution(ctx)

	arr1 := make(graphql.Array, len(res))
	var wg sync.WaitGroup

	isLen1 := len(res) == 1
	if !isLen1 {
		wg.Add(len(res))
	}

	for idx1 := range res {
		idx1 := idx1
		rctx := &graphql.ResolverContext{
			Index:  &idx1,
			Result: &res[idx1],
		}
		ctx := graphql.WithResolverContext(ctx, rctx)
		f := func(idx1 int) {
			if !isLen1 {
				defer wg.Done()
			}
			arr1[idx1] = func() graphql.Marshaler {

				return ec.___Type(ctx, field.Selections, &res[idx1])
			}()
		}
		if isLen1 {
			f(idx1)
		} else {
			go f(idx1)
		}

	}
	wg.Wait()
	return arr1
}

// nolint: vetshadow
func (ec *executionContext) ___Schema_queryType(ctx context.Context, field graphql.CollectedField, obj *introspection.Schema) graphql.Marshaler {
	ctx = ec.Tracer.StartFieldExecution(ctx, field)
	defer func() { ec.Tracer.EndFieldExecution(ctx) }()
	rctx := &graphql.ResolverContext{
		Object: "__Schema",
		Field:  field,
		Args:   nil,
	}
	ctx = graphql.WithResolverContext(ctx, rctx)
	ctx = ec.Tracer.StartFieldResolverExecution(ctx, rctx)
	resTmp := ec.FieldMiddleware(ctx, obj, func(rctx context.Context) (interface{}, error) {
		ctx = rctx // use context from middleware stack in children
		return obj.QueryType(), nil
	})
	if resTmp == nil {
		if !ec.HasError(rctx) {
			ec.Errorf(ctx, "must not be null")
		}
		return graphql.Null
	}
	res := resTmp.(*introspection.Type)
	rctx.Result = res
	ctx = ec.Tracer.StartFieldChildExecution(ctx)

	if res == nil {
		if !ec.HasError(rctx) {
			ec.Errorf(ctx, "must not be null")
		}
		return graphql.Null
	}

	return ec.___Type(ctx, field.Selections, res)
}

// nolint: vetshadow
func (ec *executionContext) ___Schema_mutationType(ctx context.Context, field graphql.CollectedField, obj *introspection.Schema) graphql.Marshaler {
	ctx = ec.Tracer.StartFieldExecution(ctx, field)
	defer func() { ec.Tracer.EndFieldExecution(ctx) }()
	rctx := &graphql.ResolverContext{
		Object: "__Schema",
		Field:  field,
		Args:   nil,
	}
	ctx = graphql.WithResolverContext(ctx, rctx)
	ctx = ec.Tracer.StartFieldResolverExecution(ctx, rctx)
	resTmp := ec.FieldMiddleware(ctx, obj, func(rctx context.Context) (interface{}, error) {
		ctx = rctx // use context from middleware stack in children
		return obj.MutationType(), nil
	})
	if resTmp == nil {
		return graphql.Null
	}
	res := resTmp.(*introspection.Type)
	rctx.Result = res
	ctx = ec.Tracer.StartFieldChildExecution(ctx)

	if res == nil {
		return graphql.Null
	}

	return ec.___Type(ctx, field.Selections, res)
}

// nolint: vetshadow
func (ec *executionContext) ___Schema_subscriptionType(ctx context.Context, field graphql.CollectedField, obj *introspection.Schema) graphql.Marshaler {
	ctx = ec.Tracer.StartFieldExecution(ctx, field)
	defer func() { ec.Tracer.EndFieldExecution(ctx) }()
	rctx := &graphql.ResolverContext{
		Object: "__Schema",
		Field:  field,
		Args:   nil,
	}
	ctx = graphql.WithResolverContext(ctx, rctx)
	ctx = ec.Tracer.StartFieldResolverExecution(ctx, rctx)
	resTmp := ec.FieldMiddleware(ctx, obj, func(rctx context.Context) (interface{}, error) {
		ctx = rctx // use context from middleware stack in children
		return obj.SubscriptionType(), nil
	})
	if resTmp == nil {
		return graphql.Null
	}
	res := resTmp.(*introspection.Type)
	rctx.Result = res
	ctx = ec.Tracer.StartFieldChildExecution(ctx)

	if res == nil {
		return graphql.Null
	}

	return ec.___Type(ctx, field.Selections, res)
}

// nolint: vetshadow
func (ec *executionContext) ___Schema_directives(ctx context.Context, field graphql.CollectedField, obj *introspection.Schema) graphql.Marshaler {
	ctx = ec.Tracer.StartFieldExecution(ctx, field)
	defer func() { ec.Tracer.EndFieldExecution(ctx) }()
	rctx := &graphql.ResolverContext{
		Object: "__Schema",
		Field:  field,
		Args:   nil,
	}
	ctx = graphql.WithResolverContext(ctx, rctx)
	ctx = ec.Tracer.StartFieldResolverExecution(ctx, rctx)
	resTmp := ec.FieldMiddleware(ctx, obj, func(rctx context.Context) (interface{}, error) {
		ctx = rctx // use context from middleware stack in children
		return obj.Directives(), nil
	})
	if resTmp == nil {
		if !ec.HasError(rctx) {
			ec.Errorf(ctx, "must not be null")
		}
		return graphql.Null
	}
	res := resTmp.([]introspection.Directive)
	rctx.Result = res
	ctx = ec.Tracer.StartFieldChildExecution(ctx)

	arr1 := make(graphql.Array, len(res))
	var wg sync.WaitGroup

	isLen1 := len(res) == 1
	if !isLen1 {
		wg.Add(len(res))
	}

	for idx1 := range res {
		idx1 := idx1
		rctx := &graphql.ResolverContext{
			Index:  &idx1,
			Result: &res[idx1],
		}
		ctx := graphql.WithResolverContext(ctx, rctx)
		f := func(idx1 int) {
			if !isLen1 {
				defer wg.Done()
			}
			arr1[idx1] = func() graphql.Marshaler {

				return ec.___Directive(ctx, field.Selections, &res[idx1])
			}()
		}
		if isLen1 {
			f(idx1)
		} else {
			go f(idx1)
		}

	}
	wg.Wait()
	return arr1
}

// nolint: vetshadow
func (ec *executionContext) ___Type_kind(ctx context.Context, field graphql.CollectedField, obj *introspection.Type) graphql.Marshaler {
	ctx = ec.Tracer.StartFieldExecution(ctx, field)
	defer func() { ec.Tracer.EndFieldExecution(ctx) }()
	rctx := &graphql.ResolverContext{
		Object: "__Type",
		Field:  field,
		Args:   nil,
	}
	ctx = graphql.WithResolverContext(ctx, rctx)
	ctx = ec.Tracer.StartFieldResolverExecution(ctx, rctx)
	resTmp := ec.FieldMiddleware(ctx, obj, func(rctx context.Context) (interface{}, error) {
		ctx = rctx // use context from middleware stack in children
		return obj.Kind(), nil
	})
	if resTmp == nil {
		if !ec.HasError(rctx) {
			ec.Errorf(ctx, "must not be null")
		}
		return graphql.Null
	}
	res := resTmp.(string)
	rctx.Result = res
	ctx = ec.Tracer.StartFieldChildExecution(ctx)
	return graphql.MarshalString(res)
}

// nolint: vetshadow
func (ec *executionContext) ___Type_name(ctx context.Context, field graphql.CollectedField, obj *introspection.Type) graphql.Marshaler {
	ctx = ec.Tracer.StartFieldExecution(ctx, field)
	defer func() { ec.Tracer.EndFieldExecution(ctx) }()
	rctx := &graphql.ResolverContext{
		Object: "__Type",
		Field:  field,
		Args:   nil,
	}
	ctx = graphql.WithResolverContext(ctx, rctx)
	ctx = ec.Tracer.StartFieldResolverExecution(ctx, rctx)
	resTmp := ec.FieldMiddleware(ctx, obj, func(rctx context.Context) (interface{}, error) {
		ctx = rctx // use context from middleware stack in children
		return obj.Name(), nil
	})
	if resTmp == nil {
		return graphql.Null
	}
	res := resTmp.(*string)
	rctx.Result = res
	ctx = ec.Tracer.StartFieldChildExecution(ctx)

	if res == nil {
		return graphql.Null
	}
	return graphql.MarshalString(*res)
}

// nolint: vetshadow
func (ec *executionContext) ___Type_description(ctx context.Context, field graphql.CollectedField, obj *introspection.Type) graphql.Marshaler {
	ctx = ec.Tracer.StartFieldExecution(ctx, field)
	defer func() { ec.Tracer.EndFieldExecution(ctx) }()
	rctx := &graphql.ResolverContext{
		Object: "__Type",
		Field:  field,
		Args:   nil,
	}
	ctx = graphql.WithResolverContext(ctx, rctx)
	ctx = ec.Tracer.StartFieldResolverExecution(ctx, rctx)
	resTmp := ec.FieldMiddleware(ctx, obj, func(rctx context.Context) (interface{}, error) {
		ctx = rctx // use context from middleware stack in children
		return obj.Description(), nil
	})
	if resTmp == nil {
		return graphql.Null
	}
	res := resTmp.(string)
	rctx.Result = res
	ctx = ec.Tracer.StartFieldChildExecution(ctx)
	return graphql.MarshalString(res)
}

// nolint: vetshadow
func (ec *executionContext) ___Type_fields(ctx context.Context, field graphql.CollectedField, obj *introspection.Type) graphql.Marshaler {
	ctx = ec.Tracer.StartFieldExecution(ctx, field)
	defer func() { ec.Tracer.EndFieldExecution(ctx) }()
	rctx := &graphql.ResolverContext{
		Object: "__Type",
		Field:  field,
		Args:   nil,
	}
	ctx = graphql.WithResolverContext(ctx, rctx)
	rawArgs := field.ArgumentMap(ec.Variables)
	args, err := ec.field___Type_fields_args(ctx, rawArgs)
	if err != nil {
		ec.Error(ctx, err)
		return graphql.Null
	}
	rctx.Args = args
	ctx = ec.Tracer.StartFieldResolverExecution(ctx, rctx)
	resTmp := ec.FieldMiddleware(ctx, obj, func(rctx context.Context) (interface{}, error) {
		ctx = rctx // use context from middleware stack in children
		return obj.Fields(args["includeDeprecated"].(bool)), nil
	})
	if resTmp == nil {
		return graphql.Null
	}
	res := resTmp.([]introspection.Field)
	rctx.Result = res
	ctx = ec.Tracer.StartFieldChildExecution(ctx)

	arr1 := make(graphql.Array, len(res))
	var wg sync.WaitGroup

	isLen1 := len(res) == 1
	if !isLen1 {
		wg.Add(len(res))
	}

	for idx1 := range res {
		idx1 := idx1
		rctx := &graphql.ResolverContext{
			Index:  &idx1,
			Result: &res[idx1],
		}
		ctx := graphql.WithResolverContext(ctx, rctx)
		f := func(idx1 int) {
			if !isLen1 {
				defer wg.Done()
			}
			arr1[idx1] = func() graphql.Marshaler {

				return ec.___Field(ctx, field.Selections, &res[idx1])
			}()
		}
		if isLen1 {
			f(idx1)
		} else {
			go f(idx1)
		}

	}
	wg.Wait()
	return arr1
}

// nolint: vetshadow
func (ec *executionContext) ___Type_interfaces(ctx context.Context, field graphql.CollectedField, obj *introspection.Type) graphql.Marshaler {
	ctx = ec.Tracer.StartFieldExecution(ctx, field)
	defer func() { ec.Tracer.EndFieldExecution(ctx) }()
	rctx := &graphql.ResolverContext{
		Object: "__Type",
		Field:  field,
		Args:   nil,
	}
	ctx = graphql.WithResolverContext(ctx, rctx)
	ctx = ec.Tracer.StartFieldResolverExecution(ctx, rctx)
	resTmp := ec.FieldMiddleware(ctx, obj, func(rctx context.Context) (interface{}, error) {
		ctx = rctx // use context from middleware stack in children
		return obj.Interfaces(), nil
	})
	if resTmp == nil {
		return graphql.Null
	}
	res := resTmp.([]introspection.Type)
	rctx.Result = res
	ctx = ec.Tracer.StartFieldChildExecution(ctx)

	arr1 := make(graphql.Array, len(res))
	var wg sync.WaitGroup

	isLen1 := len(res) == 1
	if !isLen1 {
		wg.Add(len(res))
	}

	for idx1 := range res {
		idx1 := idx1
		rctx := &graphql.ResolverContext{
			Index:  &idx1,
			Result: &res[idx1],
		}
		ctx := graphql.WithResolverContext(ctx, rctx)
		f := func(idx1 int) {
			if !isLen1 {
				defer wg.Done()
			}
			arr1[idx1] = func() graphql.Marshaler {

				return ec.___Type(ctx, field.Selections, &res[idx1])
			}()
		}
		if isLen1 {
			f(idx1)
		} else {
			go f(idx1)
		}

	}
	wg.Wait()
	return arr1
}

// nolint: vetshadow
func (ec *executionContext) ___Type_possibleTypes(ctx context.Context, field graphql.CollectedField, obj *introspection.Type) graphql.Marshaler {
	ctx = ec.Tracer.StartFieldExecution(ctx, field)
	defer func() { ec.Tracer.EndFieldExecution(ctx) }()
	rctx := &graphql.ResolverContext{
		Object: "__Type",
		Field:  field,
		Args:   nil,
	}
	ctx = graphql.WithResolverContext(ctx, rctx)
	ctx = ec.Tracer.StartFieldResolverExecution(ctx, rctx)
	resTmp := ec.FieldMiddleware(ctx, obj, func(rctx context.Context) (interface{}, error) {
		ctx = rctx // use context from middleware stack in children
		return obj.PossibleTypes(), nil
	})
	if resTmp == nil {
		return graphql.Null
	}
	res := resTmp.([]introspection.Type)
	rctx.Result = res
	ctx = ec.Tracer.StartFieldChildExecution(ctx)

	arr1 := make(graphql.Array, len(res))
	var wg sync.WaitGroup

	isLen1 := len(res) == 1
	if !isLen1 {
		wg.Add(len(res))
	}

	for idx1 := range res {
		idx1 := idx1
		rctx := &graphql.ResolverContext{
			Index:  &idx1,
			Result: &res[idx1],
		}
		ctx := graphql.WithResolverContext(ctx, rctx)
		f := func(idx1 int) {
			if !isLen1 {
				defer wg.Done()
			}
			arr1[idx1] = func() graphql.Marshaler {

				return ec.___Type(ctx, field.Selections, &res[idx1])
			}()
		}
		if isLen1 {
			f(idx1)
		} else {
			go f(idx1)
		}

	}
	wg.Wait()
	return arr1
}

// nolint: vetshadow
func (ec *executionContext) ___Type_enumValues(ctx context.Context, field graphql.CollectedField, obj *introspection.Type) graphql.Marshaler {
	ctx = ec.Tracer.StartFieldExecution(ctx, field)
	defer func() { ec.Tracer.EndFieldExecution(ctx) }()
	rctx := &graphql.ResolverContext{
		Object: "__Type",
		Field:  field,
		Args:   nil,
	}
	ctx = graphql.WithResolverContext(ctx, rctx)
	rawArgs := field.ArgumentMap(ec.Variables)
	args, err := ec.field___Type_enumValues_args(ctx, rawArgs)
	if err != nil {
		ec.Error(ctx, err)
		return graphql.Null
	}
	rctx.Args = args
	ctx = ec.Tracer.StartFieldResolverExecution(ctx, rctx)
	resTmp := ec.FieldMiddleware(ctx, obj, func(rctx context.Context) (interface{}, error) {
		ctx = rctx // use context from middleware stack in children
		return obj.EnumValues(args["includeDeprecated"].(bool)), nil
	})
	if resTmp == nil {
		return graphql.Null
	}
	res := resTmp.([]introspection.EnumValue)
	rctx.Result = res
	ctx = ec.Tracer.StartFieldChildExecution(ctx)

	arr1 := make(graphql.Array, len(res))
	var wg sync.WaitGroup

	isLen1 := len(res) == 1
	if !isLen1 {
		wg.Add(len(res))
	}

	for idx1 := range res {
		idx1 := idx1
		rctx := &graphql.ResolverContext{
			Index:  &idx1,
			Result: &res[idx1],
		}
		ctx := graphql.WithResolverContext(ctx, rctx)
		f := func(idx1 int) {
			if !isLen1 {
				defer wg.Done()
			}
			arr1[idx1] = func() graphql.Marshaler {

				return ec.___EnumValue(ctx, field.Selections, &res[idx1])
			}()
		}
		if isLen1 {
			f(idx1)
		} else {
			go f(idx1)
		}

	}
	wg.Wait()
	return arr1
}

// nolint: vetshadow
func (ec *executionContext) ___Type_inputFields(ctx context.Context, field graphql.CollectedField, obj *introspection.Type) graphql.Marshaler {
	ctx = ec.Tracer.StartFieldExecution(ctx, field)
	defer func() { ec.Tracer.EndFieldExecution(ctx) }()
	rctx := &graphql.ResolverContext{
		Object: "__Type",
		Field:  field,
		Args:   nil,
	}
	ctx = graphql.WithResolverContext(ctx, rctx)
	ctx = ec.Tracer.StartFieldResolverExecution(ctx, rctx)
	resTmp := ec.FieldMiddleware(ctx, obj, func(rctx context.Context) (interface{}, error) {
		ctx = rctx // use context from middleware stack in children
		return obj.InputFields(), nil
	})
	if resTmp == nil {
		return graphql.Null
	}
	res := resTmp.([]introspection.InputValue)
	rctx.Result = res
	ctx = ec.Tracer.StartFieldChildExecution(ctx)

	arr1 := make(graphql.Array, len(res))
	var wg sync.WaitGroup

	isLen1 := len(res) == 1
	if !isLen1 {
		wg.Add(len(res))
	}

	for idx1 := range res {
		idx1 := idx1
		rctx := &graphql.ResolverContext{
			Index:  &idx1,
			Result: &res[idx1],
		}
		ctx := graphql.WithResolverContext(ctx, rctx)
		f := func(idx1 int) {
			if !isLen1 {
				defer wg.Done()
			}
			arr1[idx1] = func() graphql.Marshaler {

				return ec.___InputValue(ctx, field.Selections, &res[idx1])
			}()
		}
		if isLen1 {
			f(idx1)
		} else {
			go f(idx1)
		}

	}
	wg.Wait()
	return arr1
}

// nolint: vetshadow
func (ec *executionContext) ___Type_ofType(ctx context.Context, field graphql.CollectedField, obj *introspection.Type) graphql.Marshaler {
	ctx = ec.Tracer.StartFieldExecution(ctx, field)
	defer func() { ec.Tracer.EndFieldExecution(ctx) }()
	rctx := &graphql.ResolverContext{
		Object: "__Type",
		Field:  field,
		Args:   nil,
	}
	ctx = graphql.WithResolverContext(ctx, rctx)
	ctx = ec.Tracer.StartFieldResolverExecution(ctx, rctx)
	resTmp := ec.FieldMiddleware(ctx, obj, func(rctx context.Context) (interface{}, error) {
		ctx = rctx // use context from middleware stack in children
		return obj.OfType(), nil
	})
	if resTmp == nil {
		return graphql.Null
	}
	res := resTmp.(*introspection.Type)
	rctx.Result = res
	ctx = ec.Tracer.StartFieldChildExecution(ctx)

	if res == nil {
		return graphql.Null
	}

	return ec.___Type(ctx, field.Selections, res)
}

// endregion **************************** field.gotpl *****************************

// region    **************************** input.gotpl *****************************

// endregion **************************** input.gotpl *****************************

// region    ************************** interface.gotpl ***************************

// endregion ************************** interface.gotpl ***************************

// region    **************************** object.gotpl ****************************

var chatroomImplementors = []string{"Chatroom"}

// nolint: gocyclo, errcheck, gas, goconst
func (ec *executionContext) _Chatroom(ctx context.Context, sel ast.SelectionSet, obj *Chatroom) graphql.Marshaler {
	fields := graphql.CollectFields(ctx, sel, chatroomImplementors)

	out := graphql.NewFieldSet(fields)
	invalid := false
	for i, field := range fields {
		switch field.Name {
		case "__typename":
			out.Values[i] = graphql.MarshalString("Chatroom")
		case "name":
			out.Values[i] = ec._Chatroom_name(ctx, field, obj)
			if out.Values[i] == graphql.Null {
				invalid = true
			}
		case "messages":
			out.Values[i] = ec._Chatroom_messages(ctx, field, obj)
			if out.Values[i] == graphql.Null {
				invalid = true
			}
		default:
			panic("unknown field " + strconv.Quote(field.Name))
		}
	}
	out.Dispatch()
	if invalid {
		return graphql.Null
	}
	return out
}

var messageImplementors = []string{"Message"}

// nolint: gocyclo, errcheck, gas, goconst
func (ec *executionContext) _Message(ctx context.Context, sel ast.SelectionSet, obj *Message) graphql.Marshaler {
	fields := graphql.CollectFields(ctx, sel, messageImplementors)

	out := graphql.NewFieldSet(fields)
	invalid := false
	for i, field := range fields {
		switch field.Name {
		case "__typename":
			out.Values[i] = graphql.MarshalString("Message")
		case "id":
			out.Values[i] = ec._Message_id(ctx, field, obj)
			if out.Values[i] == graphql.Null {
				invalid = true
			}
		case "text":
			out.Values[i] = ec._Message_text(ctx, field, obj)
			if out.Values[i] == graphql.Null {
				invalid = true
			}
		case "createdBy":
			out.Values[i] = ec._Message_createdBy(ctx, field, obj)
			if out.Values[i] == graphql.Null {
				invalid = true
			}
		case "createdAt":
			out.Values[i] = ec._Message_createdAt(ctx, field, obj)
			if out.Values[i] == graphql.Null {
				invalid = true
			}
		default:
			panic("unknown field " + strconv.Quote(field.Name))
		}
	}
	out.Dispatch()
	if invalid {
		return graphql.Null
	}
	return out
}

var mutationImplementors = []string{"Mutation"}

// nolint: gocyclo, errcheck, gas, goconst
func (ec *executionContext) _Mutation(ctx context.Context, sel ast.SelectionSet) graphql.Marshaler {
	fields := graphql.CollectFields(ctx, sel, mutationImplementors)

	ctx = graphql.WithResolverContext(ctx, &graphql.ResolverContext{
		Object: "Mutation",
	})

	out := graphql.NewFieldSet(fields)
	invalid := false
	for i, field := range fields {
		switch field.Name {
		case "__typename":
			out.Values[i] = graphql.MarshalString("Mutation")
		case "post":
			out.Values[i] = ec._Mutation_post(ctx, field)
			if out.Values[i] == graphql.Null {
				invalid = true
			}
		default:
			panic("unknown field " + strconv.Quote(field.Name))
		}
	}
	out.Dispatch()
	if invalid {
		return graphql.Null
	}
	return out
}

var queryImplementors = []string{"Query"}

// nolint: gocyclo, errcheck, gas, goconst
func (ec *executionContext) _Query(ctx context.Context, sel ast.SelectionSet) graphql.Marshaler {
	fields := graphql.CollectFields(ctx, sel, queryImplementors)

	ctx = graphql.WithResolverContext(ctx, &graphql.ResolverContext{
		Object: "Query",
	})

	out := graphql.NewFieldSet(fields)
	invalid := false
	for i, field := range fields {
		switch field.Name {
		case "__typename":
			out.Values[i] = graphql.MarshalString("Query")
		case "room":
			field := field
			out.Concurrently(i, func() (res graphql.Marshaler) {
				res = ec._Query_room(ctx, field)
				return res
			})
		case "__type":
			out.Values[i] = ec._Query___type(ctx, field)
		case "__schema":
			out.Values[i] = ec._Query___schema(ctx, field)
		default:
			panic("unknown field " + strconv.Quote(field.Name))
		}
	}
	out.Dispatch()
	if invalid {
		return graphql.Null
	}
	return out
}

var subscriptionImplementors = []string{"Subscription"}

// nolint: gocyclo, errcheck, gas, goconst
func (ec *executionContext) _Subscription(ctx context.Context, sel ast.SelectionSet) func() graphql.Marshaler {
	fields := graphql.CollectFields(ctx, sel, subscriptionImplementors)
	ctx = graphql.WithResolverContext(ctx, &graphql.ResolverContext{
		Object: "Subscription",
	})
	if len(fields) != 1 {
		ec.Errorf(ctx, "must subscribe to exactly one stream")
		return nil
	}

	switch fields[0].Name {
	case "messageAdded":
		return ec._Subscription_messageAdded(ctx, fields[0])
	default:
		panic("unknown field " + strconv.Quote(fields[0].Name))
	}
}

var __DirectiveImplementors = []string{"__Directive"}

// nolint: gocyclo, errcheck, gas, goconst
func (ec *executionContext) ___Directive(ctx context.Context, sel ast.SelectionSet, obj *introspection.Directive) graphql.Marshaler {
	fields := graphql.CollectFields(ctx, sel, __DirectiveImplementors)

	out := graphql.NewFieldSet(fields)
	invalid := false
	for i, field := range fields {
		switch field.Name {
		case "__typename":
			out.Values[i] = graphql.MarshalString("__Directive")
		case "name":
			out.Values[i] = ec.___Directive_name(ctx, field, obj)
			if out.Values[i] == graphql.Null {
				invalid = true
			}
		case "description":
			out.Values[i] = ec.___Directive_description(ctx, field, obj)
		case "locations":
			out.Values[i] = ec.___Directive_locations(ctx, field, obj)
			if out.Values[i] == graphql.Null {
				invalid = true
			}
		case "args":
			out.Values[i] = ec.___Directive_args(ctx, field, obj)
			if out.Values[i] == graphql.Null {
				invalid = true
			}
		default:
			panic("unknown field " + strconv.Quote(field.Name))
		}
	}
	out.Dispatch()
	if invalid {
		return graphql.Null
	}
	return out
}

var __EnumValueImplementors = []string{"__EnumValue"}

// nolint: gocyclo, errcheck, gas, goconst
func (ec *executionContext) ___EnumValue(ctx context.Context, sel ast.SelectionSet, obj *introspection.EnumValue) graphql.Marshaler {
	fields := graphql.CollectFields(ctx, sel, __EnumValueImplementors)

	out := graphql.NewFieldSet(fields)
	invalid := false
	for i, field := range fields {
		switch field.Name {
		case "__typename":
			out.Values[i] = graphql.MarshalString("__EnumValue")
		case "name":
			out.Values[i] = ec.___EnumValue_name(ctx, field, obj)
			if out.Values[i] == graphql.Null {
				invalid = true
			}
		case "description":
			out.Values[i] = ec.___EnumValue_description(ctx, field, obj)
		case "isDeprecated":
			out.Values[i] = ec.___EnumValue_isDeprecated(ctx, field, obj)
			if out.Values[i] == graphql.Null {
				invalid = true
			}
		case "deprecationReason":
			out.Values[i] = ec.___EnumValue_deprecationReason(ctx, field, obj)
		default:
			panic("unknown field " + strconv.Quote(field.Name))
		}
	}
	out.Dispatch()
	if invalid {
		return graphql.Null
	}
	return out
}

var __FieldImplementors = []string{"__Field"}

// nolint: gocyclo, errcheck, gas, goconst
func (ec *executionContext) ___Field(ctx context.Context, sel ast.SelectionSet, obj *introspection.Field) graphql.Marshaler {
	fields := graphql.CollectFields(ctx, sel, __FieldImplementors)

	out := graphql.NewFieldSet(fields)
	invalid := false
	for i, field := range fields {
		switch field.Name {
		case "__typename":
			out.Values[i] = graphql.MarshalString("__Field")
		case "name":
			out.Values[i] = ec.___Field_name(ctx, field, obj)
			if out.Values[i] == graphql.Null {
				invalid = true
			}
		case "description":
			out.Values[i] = ec.___Field_description(ctx, field, obj)
		case "args":
			out.Values[i] = ec.___Field_args(ctx, field, obj)
			if out.Values[i] == graphql.Null {
				invalid = true
			}
		case "type":
			out.Values[i] = ec.___Field_type(ctx, field, obj)
			if out.Values[i] == graphql.Null {
				invalid = true
			}
		case "isDeprecated":
			out.Values[i] = ec.___Field_isDeprecated(ctx, field, obj)
			if out.Values[i] == graphql.Null {
				invalid = true
			}
		case "deprecationReason":
			out.Values[i] = ec.___Field_deprecationReason(ctx, field, obj)
		default:
			panic("unknown field " + strconv.Quote(field.Name))
		}
	}
	out.Dispatch()
	if invalid {
		return graphql.Null
	}
	return out
}

var __InputValueImplementors = []string{"__InputValue"}

// nolint: gocyclo, errcheck, gas, goconst
func (ec *executionContext) ___InputValue(ctx context.Context, sel ast.SelectionSet, obj *introspection.InputValue) graphql.Marshaler {
	fields := graphql.CollectFields(ctx, sel, __InputValueImplementors)

	out := graphql.NewFieldSet(fields)
	invalid := false
	for i, field := range fields {
		switch field.Name {
		case "__typename":
			out.Values[i] = graphql.MarshalString("__InputValue")
		case "name":
			out.Values[i] = ec.___InputValue_name(ctx, field, obj)
			if out.Values[i] == graphql.Null {
				invalid = true
			}
		case "description":
			out.Values[i] = ec.___InputValue_description(ctx, field, obj)
		case "type":
			out.Values[i] = ec.___InputValue_type(ctx, field, obj)
			if out.Values[i] == graphql.Null {
				invalid = true
			}
		case "defaultValue":
			out.Values[i] = ec.___InputValue_defaultValue(ctx, field, obj)
		default:
			panic("unknown field " + strconv.Quote(field.Name))
		}
	}
	out.Dispatch()
	if invalid {
		return graphql.Null
	}
	return out
}

var __SchemaImplementors = []string{"__Schema"}

// nolint: gocyclo, errcheck, gas, goconst
func (ec *executionContext) ___Schema(ctx context.Context, sel ast.SelectionSet, obj *introspection.Schema) graphql.Marshaler {
	fields := graphql.CollectFields(ctx, sel, __SchemaImplementors)

	out := graphql.NewFieldSet(fields)
	invalid := false
	for i, field := range fields {
		switch field.Name {
		case "__typename":
			out.Values[i] = graphql.MarshalString("__Schema")
		case "types":
			out.Values[i] = ec.___Schema_types(ctx, field, obj)
			if out.Values[i] == graphql.Null {
				invalid = true
			}
		case "queryType":
			out.Values[i] = ec.___Schema_queryType(ctx, field, obj)
			if out.Values[i] == graphql.Null {
				invalid = true
			}
		case "mutationType":
			out.Values[i] = ec.___Schema_mutationType(ctx, field, obj)
		case "subscriptionType":
			out.Values[i] = ec.___Schema_subscriptionType(ctx, field, obj)
		case "directives":
			out.Values[i] = ec.___Schema_directives(ctx, field, obj)
			if out.Values[i] == graphql.Null {
				invalid = true
			}
		default:
			panic("unknown field " + strconv.Quote(field.Name))
		}
	}
	out.Dispatch()
	if invalid {
		return graphql.Null
	}
	return out
}

var __TypeImplementors = []string{"__Type"}

// nolint: gocyclo, errcheck, gas, goconst
func (ec *executionContext) ___Type(ctx context.Context, sel ast.SelectionSet, obj *introspection.Type) graphql.Marshaler {
	fields := graphql.CollectFields(ctx, sel, __TypeImplementors)

	out := graphql.NewFieldSet(fields)
	invalid := false
	for i, field := range fields {
		switch field.Name {
		case "__typename":
			out.Values[i] = graphql.MarshalString("__Type")
		case "kind":
			out.Values[i] = ec.___Type_kind(ctx, field, obj)
			if out.Values[i] == graphql.Null {
				invalid = true
			}
		case "name":
			out.Values[i] = ec.___Type_name(ctx, field, obj)
		case "description":
			out.Values[i] = ec.___Type_description(ctx, field, obj)
		case "fields":
			out.Values[i] = ec.___Type_fields(ctx, field, obj)
		case "interfaces":
			out.Values[i] = ec.___Type_interfaces(ctx, field, obj)
		case "possibleTypes":
			out.Values[i] = ec.___Type_possibleTypes(ctx, field, obj)
		case "enumValues":
			out.Values[i] = ec.___Type_enumValues(ctx, field, obj)
		case "inputFields":
			out.Values[i] = ec.___Type_inputFields(ctx, field, obj)
		case "ofType":
			out.Values[i] = ec.___Type_ofType(ctx, field, obj)
		default:
			panic("unknown field " + strconv.Quote(field.Name))
		}
	}
	out.Dispatch()
	if invalid {
		return graphql.Null
	}
	return out
}

// endregion **************************** object.gotpl ****************************

// region    ***************************** type.gotpl *****************************

func (e *executableSchema) unmarshalBoolean2bool(v interface{}) (bool, error) {
	return graphql.UnmarshalBoolean(v)
}
func (e *executableSchema) unmarshalString2string(v interface{}) (string, error) {
	return graphql.UnmarshalString(v)
}

// endregion ***************************** type.gotpl *****************************
