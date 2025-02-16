package apollofederatedtracingv1

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/99designs/gqlgen/graphql"
	"github.com/99designs/gqlgen/graphql/handler/apollofederatedtracingv1/generated"
	tracing_logger "github.com/99designs/gqlgen/graphql/handler/apollofederatedtracingv1/logger"
	"github.com/vektah/gqlparser/v2/gqlerror"
)

type TreeBuilder struct {
	Trace    *generated.Trace
	rootNode generated.Trace_Node
	nodes    map[string]NodeMap // nodes is used to store a pointer map using the node path (e.g. todo[0].id) to itself as well as it's parent

	startTime *time.Time
	stopped   bool
	mu        sync.Mutex

	errorOptions *ErrorOptions
	logger       tracing_logger.Logger
}

type NodeMap struct {
	self   *generated.Trace_Node
	parent *generated.Trace_Node
}

// NewTreeBuilder is used to start the node tree with a default root node, along with the related tree nodes map entry
func NewTreeBuilder(errorOptions *ErrorOptions, logger tracing_logger.Logger) *TreeBuilder {
	if errorOptions == nil {
		errorOptions = &ErrorOptions{
			ErrorOption:       ERROR_MASKED,
			TransformFunction: defaultErrorTransform,
		}
	}

	if logger == nil {
		// defaults to a noop logger
		logger = tracing_logger.NewNoopLogger()
	}

	switch errorOptions.ErrorOption {
	case ERROR_MASKED:
		errorOptions.TransformFunction = defaultErrorTransform
	case ERROR_UNMODIFIED:
		errorOptions.TransformFunction = nil
	case ERROR_TRANSFORM:
		if errorOptions.TransformFunction == nil {
			errorOptions.TransformFunction = defaultErrorTransform
		}
	default:
		errorOptions = &ErrorOptions{
			ErrorOption:       ERROR_MASKED,
			TransformFunction: defaultErrorTransform,
		}
	}

	tb := TreeBuilder{
		rootNode:     generated.Trace_Node{},
		errorOptions: errorOptions,
		logger:       logger,
	}

	t := generated.Trace{
		Root: &tb.rootNode,
	}
	tb.nodes = make(map[string]NodeMap)
	tb.nodes[""] = NodeMap{self: &tb.rootNode, parent: nil}

	tb.Trace = &t

	return &tb
}

// StartTimer marks the time using protobuf timestamp format for use in timing calculations
func (tb *TreeBuilder) StartTimer(ctx context.Context) {
	if tb.startTime != nil {
		tb.logger.Println(errors.New("StartTimer called twice"))
	}
	if tb.stopped {
		tb.logger.Println(errors.New("StartTimer called after StopTimer"))
	}

	opCtx := graphql.GetOperationContext(ctx)
	start := opCtx.Stats.OperationStart

	tb.Trace.StartTime = timestamppb.New(start)
	tb.startTime = &start
}

// StopTimer marks the end of the timer, along with setting the related fields in the protobuf representation
func (tb *TreeBuilder) StopTimer(ctx context.Context) {
	tb.logger.Print("StopTimer called")
	if tb.startTime == nil {
		tb.logger.Println(errors.New("StopTimer called before StartTimer"))
	}
	if tb.stopped {
		tb.logger.Println(errors.New("StopTimer called twice"))
	}

	ts := graphql.Now().UTC()
	tb.Trace.DurationNs = uint64(ts.Sub(*tb.startTime).Nanoseconds())
	tb.Trace.EndTime = timestamppb.New(ts)
	tb.stopped = true
}

// On each field, it calculates the time started at as now - tree.StartTime, as well as a deferred function upon full resolution of the
// field as now - tree.StartTime; these are used by Apollo to calculate how fields are being resolved in the AST
func (tb *TreeBuilder) WillResolveField(ctx context.Context) {
	if tb.startTime == nil {
		tb.logger.Println(errors.New("WillResolveField called before StartTimer"))
		return
	}
	if tb.stopped {
		tb.logger.Println(errors.New("WillResolveField called after StopTimer"))
		return
	}
	fc := graphql.GetFieldContext(ctx)

	node := tb.newNode(fc)
	node.StartTime = uint64(graphql.Now().Sub(*tb.startTime).Nanoseconds())
	defer func() {
		node.EndTime = uint64(graphql.Now().Sub(*tb.startTime).Nanoseconds())
	}()

	node.Type = fc.Field.Definition.Type.String()
	node.ParentType = fc.Object
}

func (tb *TreeBuilder) DidEncounterErrors(ctx context.Context, gqlErrors gqlerror.List) {
	if tb.startTime == nil {
		tb.logger.Println(errors.New("DidEncounterErrors called before StartTimer"))
		return
	}
	if tb.stopped {
		tb.logger.Println(errors.New("DidEncounterErrors called after StopTimer"))
		return
	}

	for _, err := range gqlErrors {
		if err != nil {
			tb.addProtobufError(err)
		}
	}
}

// newNode is called on each new node within the AST and sets related values such as the entry in the tree.node map and ID attribute
func (tb *TreeBuilder) newNode(path *graphql.FieldContext) *generated.Trace_Node {
	// if the path is empty, it is the root node of the operation
	if path.Path().String() == "" {
		return &tb.rootNode
	}

	self := &generated.Trace_Node{}
	pn := tb.ensureParentNode(path)

	if path.Index != nil {
		self.Id = &generated.Trace_Node_Index{Index: uint32(*path.Index)}
	} else {
		self.Id = &generated.Trace_Node_ResponseName{ResponseName: path.Field.Name}
	}

	// lock the map from being read/written concurrently to avoid panics
	tb.mu.Lock()
	nodeRef := tb.nodes[path.Path().String()]
	// set the values for the node references to help build the tree
	nodeRef.parent = pn
	nodeRef.self = self

	// since they are references, we point the parent to it's children nodes
	nodeRef.parent.Child = append(nodeRef.parent.Child, self)
	nodeRef.self = self
	tb.nodes[path.Path().String()] = nodeRef
	tb.mu.Unlock()

	return self
}

// ensureParentNode ensures the node isn't orphaned
func (tb *TreeBuilder) ensureParentNode(path *graphql.FieldContext) *generated.Trace_Node {
	// lock to read briefly, then unlock to avoid r/w issues
	tb.mu.Lock()
	nodeRef := tb.nodes[path.Parent.Path().String()]
	tb.mu.Unlock()

	if nodeRef.self != nil {
		return nodeRef.self
	}

	return tb.newNode(path.Parent)
}

func (tb *TreeBuilder) addProtobufError(
	gqlError *gqlerror.Error,
) {
	if tb.startTime == nil {
		tb.logger.Println(errors.New("addProtobufError called before StartTimer"))
		return
	}
	if tb.stopped {
		tb.logger.Println(errors.New("addProtobufError called after StopTimer"))
		return
	}
	tb.mu.Lock()
	var nodeRef *generated.Trace_Node

	if tb.nodes[gqlError.Path.String()].self != nil {
		nodeRef = tb.nodes[gqlError.Path.String()].self
	} else {
		tb.logger.Println("Error: Path not found in node map")
		tb.mu.Unlock()
		return
	}

	if tb.errorOptions.ErrorOption != ERROR_UNMODIFIED && tb.errorOptions.TransformFunction != nil {
		gqlError = tb.errorOptions.TransformFunction(gqlError)
	}

	errorLocations := make([]*generated.Trace_Location, len(gqlError.Locations))
	for i, loc := range gqlError.Locations {
		errorLocations[i] = &generated.Trace_Location{
			Line:   uint32(loc.Line),
			Column: uint32(loc.Column),
		}
	}

	gqlJson, err := json.Marshal(gqlError)
	if err != nil {
		tb.logger.Println(err)
		tb.mu.Unlock()
		return
	}

	nodeRef.Error = append(nodeRef.Error, &generated.Trace_Error{
		Message:  gqlError.Message,
		Location: errorLocations,
		Json:     string(gqlJson),
	})
	tb.mu.Unlock()
}

func defaultErrorTransform(_ *gqlerror.Error) *gqlerror.Error {
	return gqlerror.Errorf("<masked>")
}
