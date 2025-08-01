/*
 * Copyright 2025 coze-dev Authors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package compose

import (
	"context"
	"errors"
	"fmt"
	"runtime/debug"
	"strings"
	"time"

	"github.com/cloudwego/eino/callbacks"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
	"golang.org/x/exp/maps"

	"github.com/coze-dev/coze-studio/backend/domain/workflow/entity"
	"github.com/coze-dev/coze-studio/backend/domain/workflow/entity/vo"
	"github.com/coze-dev/coze-studio/backend/domain/workflow/internal/execute"
	"github.com/coze-dev/coze-studio/backend/domain/workflow/internal/nodes"
	"github.com/coze-dev/coze-studio/backend/pkg/errorx"
	"github.com/coze-dev/coze-studio/backend/pkg/logs"
	"github.com/coze-dev/coze-studio/backend/pkg/safego"
	"github.com/coze-dev/coze-studio/backend/pkg/sonic"
	"github.com/coze-dev/coze-studio/backend/types/errno"
)

type nodeRunConfig[O any] struct {
	nodeKey             vo.NodeKey
	nodeName            string
	nodeType            entity.NodeType
	timeoutMS           int64
	maxRetry            int64
	errProcessType      vo.ErrorProcessType
	dataOnErr           func(ctx context.Context) map[string]any
	callbackEnabled     bool
	preProcessors       []func(ctx context.Context, input map[string]any) (map[string]any, error)
	postProcessors      []func(ctx context.Context, input map[string]any) (map[string]any, error)
	streamPreProcessors []func(ctx context.Context,
		input *schema.StreamReader[map[string]any]) *schema.StreamReader[map[string]any]
	callbackInputConverter  func(context.Context, map[string]any) (map[string]any, error)
	callbackOutputConverter func(context.Context, map[string]any) (*nodes.StructuredCallbackOutput, error)
	init                    []func(context.Context) (context.Context, error)
	i                       compose.Invoke[map[string]any, map[string]any, O]
	s                       compose.Stream[map[string]any, map[string]any, O]
	t                       compose.Transform[map[string]any, map[string]any, O]
}

func newNodeRunConfig[O any](ns *NodeSchema,
	i compose.Invoke[map[string]any, map[string]any, O],
	s compose.Stream[map[string]any, map[string]any, O],
	t compose.Transform[map[string]any, map[string]any, O],
	opts *newNodeOptions) *nodeRunConfig[O] {
	meta := entity.NodeMetaByNodeType(ns.Type)

	var (
		timeoutMS      = meta.DefaultTimeoutMS
		maxRetry       int64
		errProcessType = vo.ErrorProcessTypeThrow
		dataOnErr      func(ctx context.Context) map[string]any
	)
	if ns.ExceptionConfigs != nil {
		timeoutMS = ns.ExceptionConfigs.TimeoutMS
		maxRetry = ns.ExceptionConfigs.MaxRetry
		if ns.ExceptionConfigs.ProcessType != nil {
			errProcessType = *ns.ExceptionConfigs.ProcessType
		}
		if len(ns.ExceptionConfigs.DataOnErr) > 0 {
			dataOnErr = func(ctx context.Context) map[string]any {
				return parseDefaultOutputOrFallback(ctx, ns.ExceptionConfigs.DataOnErr, ns.OutputTypes)
			}
		}
	}

	preProcessors := []func(ctx context.Context, input map[string]any) (map[string]any, error){
		preTypeConverter(ns.InputTypes),
		keyFinishedMarkerTrimmer(),
	}
	if meta.PreFillZero {
		preProcessors = append(preProcessors, ns.inputValueFiller())
	}

	var postProcessors []func(ctx context.Context, input map[string]any) (map[string]any, error)
	if meta.PostFillNil {
		postProcessors = append(postProcessors, ns.outputValueFiller())
	}

	streamPreProcessors := []func(ctx context.Context,
		input *schema.StreamReader[map[string]any]) *schema.StreamReader[map[string]any]{
		func(ctx context.Context, input *schema.StreamReader[map[string]any]) *schema.StreamReader[map[string]any] {
			f := func(in map[string]any) (map[string]any, error) {
				return preTypeConverter(ns.InputTypes)(ctx, in)
			}
			return schema.StreamReaderWithConvert(input, f)
		},
	}
	if meta.PreFillZero {
		streamPreProcessors = append(streamPreProcessors, ns.streamInputValueFiller())
	}

	opts.init = append(opts.init, func(ctx context.Context) (context.Context, error) {
		current, exceeded := execute.IncrAndCheckExecutedNodes(ctx)
		if exceeded {
			return nil, fmt.Errorf("exceeded max executed node count: %d, current: %d", execute.GetStaticConfig().MaxNodeCountPerExecution, current)
		}
		return ctx, nil
	})

	return &nodeRunConfig[O]{
		nodeKey:                 ns.Key,
		nodeName:                ns.Name,
		nodeType:                ns.Type,
		timeoutMS:               timeoutMS,
		maxRetry:                maxRetry,
		errProcessType:          errProcessType,
		dataOnErr:               dataOnErr,
		callbackEnabled:         meta.CallbackEnabled,
		preProcessors:           preProcessors,
		postProcessors:          postProcessors,
		streamPreProcessors:     streamPreProcessors,
		callbackInputConverter:  opts.callbackInputConverter,
		callbackOutputConverter: opts.callbackOutputConverter,
		init:                    opts.init,
		i:                       i,
		s:                       s,
		t:                       t,
	}
}

func newNodeRunConfigWOOpt(ns *NodeSchema,
	i compose.InvokeWOOpt[map[string]any, map[string]any],
	s compose.StreamWOOpt[map[string]any, map[string]any],
	t compose.TransformWOOpts[map[string]any, map[string]any],
	opts *newNodeOptions) *nodeRunConfig[any] {
	var (
		iWO compose.Invoke[map[string]any, map[string]any, any]
		sWO compose.Stream[map[string]any, map[string]any, any]
		tWO compose.Transform[map[string]any, map[string]any, any]
	)

	if i != nil {
		iWO = func(ctx context.Context, in map[string]any, _ ...any) (out map[string]any, err error) {
			return i(ctx, in)
		}
	}

	if s != nil {
		sWO = func(ctx context.Context, in map[string]any, _ ...any) (out *schema.StreamReader[map[string]any], err error) {
			return s(ctx, in)
		}
	}

	if t != nil {
		tWO = func(ctx context.Context, input *schema.StreamReader[map[string]any], opts ...any) (output *schema.StreamReader[map[string]any], err error) {
			return t(ctx, input)
		}
	}

	return newNodeRunConfig[any](ns, iWO, sWO, tWO, opts)
}

type newNodeOptions struct {
	callbackInputConverter  func(context.Context, map[string]any) (map[string]any, error)
	callbackOutputConverter func(context.Context, map[string]any) (*nodes.StructuredCallbackOutput, error)
	init                    []func(context.Context) (context.Context, error)
}

type newNodeOption func(*newNodeOptions)

func withCallbackInputConverter(f func(context.Context, map[string]any) (map[string]any, error)) newNodeOption {
	return func(opts *newNodeOptions) {
		opts.callbackInputConverter = f
	}
}
func withCallbackOutputConverter(f func(context.Context, map[string]any) (*nodes.StructuredCallbackOutput, error)) newNodeOption {
	return func(opts *newNodeOptions) {
		opts.callbackOutputConverter = f
	}
}
func withInit(f func(context.Context) (context.Context, error)) newNodeOption {
	return func(opts *newNodeOptions) {
		opts.init = append(opts.init, f)
	}
}

func invokableNode(ns *NodeSchema, i compose.InvokeWOOpt[map[string]any, map[string]any], opts ...newNodeOption) *Node {
	options := &newNodeOptions{}
	for _, opt := range opts {
		opt(options)
	}

	return newNodeRunConfigWOOpt(ns, i, nil, nil, options).toNode()
}

func invokableNodeWO[O any](ns *NodeSchema, i compose.Invoke[map[string]any, map[string]any, O], opts ...newNodeOption) *Node {
	options := &newNodeOptions{}
	for _, opt := range opts {
		opt(options)
	}

	return newNodeRunConfig(ns, i, nil, nil, options).toNode()
}

func invokableTransformableNode(ns *NodeSchema, i compose.InvokeWOOpt[map[string]any, map[string]any],
	t compose.TransformWOOpts[map[string]any, map[string]any], opts ...newNodeOption) *Node {
	options := &newNodeOptions{}
	for _, opt := range opts {
		opt(options)
	}
	return newNodeRunConfigWOOpt(ns, i, nil, t, options).toNode()
}

func invokableStreamableNodeWO[O any](ns *NodeSchema, i compose.Invoke[map[string]any, map[string]any, O], s compose.Stream[map[string]any, map[string]any, O], opts ...newNodeOption) *Node {
	options := &newNodeOptions{}
	for _, opt := range opts {
		opt(options)
	}
	return newNodeRunConfig(ns, i, s, nil, options).toNode()
}

func (nc *nodeRunConfig[O]) invoke() func(ctx context.Context, input map[string]any, opts ...O) (output map[string]any, err error) {
	if nc.i == nil {
		return nil
	}

	return func(ctx context.Context, input map[string]any, opts ...O) (output map[string]any, err error) {
		ctx, runner := newNodeRunner(ctx, nc)

		defer func() {
			if panicErr := recover(); panicErr != nil {
				err = safego.NewPanicErr(panicErr, debug.Stack())
			}

			if err == nil {
				err = runner.onEnd(ctx, output)
			}

			if err != nil {
				errOutput, hasErrOutput := runner.onError(ctx, err)
				if hasErrOutput {
					output = errOutput
					err = nil
					if output, err = runner.postProcess(ctx, output); err != nil {
						logs.CtxErrorf(ctx, "postProcess failed after returning error output: %v", err)
					}
				}
			}
		}()

		for _, i := range runner.init {
			if ctx, err = i(ctx); err != nil {
				return nil, err
			}
		}

		if input, err = runner.preProcess(ctx, input); err != nil {
			return nil, err
		}

		if ctx, err = runner.onStart(ctx, input); err != nil {
			return nil, err
		}

		if output, err = runner.invoke(ctx, input, opts...); err != nil {
			return nil, err
		}

		return runner.postProcess(ctx, output)
	}
}

func (nc *nodeRunConfig[O]) stream() func(ctx context.Context, input map[string]any, opts ...O) (output *schema.StreamReader[map[string]any], err error) {
	if nc.s == nil {
		return nil
	}

	return func(ctx context.Context, input map[string]any, opts ...O) (output *schema.StreamReader[map[string]any], err error) {
		ctx, runner := newNodeRunner(ctx, nc)

		defer func() {
			if panicErr := recover(); panicErr != nil {
				err = safego.NewPanicErr(panicErr, debug.Stack())
			}

			if err == nil {
				output, err = runner.onEndStream(ctx, output)
			}

			if err != nil {
				errOutput, hasErrOutput := runner.onError(ctx, err)
				if hasErrOutput {
					output = schema.StreamReaderFromArray([]map[string]any{errOutput})
					err = nil
				}
			}
		}()

		for _, i := range runner.init {
			if ctx, err = i(ctx); err != nil {
				return nil, err
			}
		}

		if input, err = runner.preProcess(ctx, input); err != nil {
			return nil, err
		}

		if ctx, err = runner.onStart(ctx, input); err != nil {
			return nil, err
		}

		return runner.stream(ctx, input, opts...)
	}
}

func (nc *nodeRunConfig[O]) transform() func(ctx context.Context, input *schema.StreamReader[map[string]any], opts ...O) (output *schema.StreamReader[map[string]any], err error) {
	if nc.t == nil {
		return nil
	}

	return func(ctx context.Context, input *schema.StreamReader[map[string]any], opts ...O) (output *schema.StreamReader[map[string]any], err error) {
		ctx, runner := newNodeRunner(ctx, nc)

		defer func() {
			if panicErr := recover(); panicErr != nil {
				err = safego.NewPanicErr(panicErr, debug.Stack())
			}

			if err == nil {
				output, err = runner.onEndStream(ctx, output)
			}

			if err != nil {
				errOutput, hasErrOutput := runner.onError(ctx, err)
				if hasErrOutput {
					output = schema.StreamReaderFromArray([]map[string]any{errOutput})
					err = nil
				}
			}
		}()

		for _, i := range runner.init {
			if ctx, err = i(ctx); err != nil {
				return nil, err
			}
		}

		for _, p := range runner.streamPreProcessors {
			input = p(ctx, input)
		}

		if ctx, input, err = runner.onStartStream(ctx, input); err != nil {
			return nil, err
		}

		return runner.transform(ctx, input, opts...)
	}
}

func (nc *nodeRunConfig[O]) toNode() *Node {
	var opts []compose.LambdaOpt
	opts = append(opts, compose.WithLambdaType(string(nc.nodeType)))

	if nc.callbackEnabled {
		opts = append(opts, compose.WithLambdaCallbackEnable(true))
	}
	l, err := compose.AnyLambda(nc.invoke(), nc.stream(), nil, nc.transform(), opts...)
	if err != nil {
		panic(fmt.Sprintf("failed to create lambda for node %s, err: %v", nc.nodeName, err))
	}

	return &Node{Lambda: l}
}

type nodeRunner[O any] struct {
	*nodeRunConfig[O]
	interrupted bool
	cancelFn    context.CancelFunc
}

func newNodeRunner[O any](ctx context.Context, cfg *nodeRunConfig[O]) (context.Context, *nodeRunner[O]) {
	runner := &nodeRunner[O]{
		nodeRunConfig: cfg,
	}

	if cfg.timeoutMS > 0 {
		ctx, runner.cancelFn = context.WithTimeout(ctx, time.Duration(cfg.timeoutMS)*time.Millisecond)
	}

	return ctx, runner
}

func (r *nodeRunner[O]) onStart(ctx context.Context, input map[string]any) (context.Context, error) {
	if !r.callbackEnabled {
		return ctx, nil
	}
	if r.callbackInputConverter != nil {
		convertedInput, err := r.callbackInputConverter(ctx, input)
		if err != nil {
			ctx = callbacks.OnStart(ctx, input)
			return ctx, err
		}
		ctx = callbacks.OnStart(ctx, convertedInput)
	} else {
		ctx = callbacks.OnStart(ctx, input)
	}

	return ctx, nil
}

func (r *nodeRunner[O]) onStartStream(ctx context.Context, input *schema.StreamReader[map[string]any]) (
	context.Context, *schema.StreamReader[map[string]any], error) {
	if !r.callbackEnabled {
		return ctx, input, nil
	}

	if r.callbackInputConverter != nil {
		copied := input.Copy(2)
		realConverter := func(ctx context.Context) func(map[string]any) (map[string]any, error) {
			return func(in map[string]any) (map[string]any, error) {
				return r.callbackInputConverter(ctx, in)
			}
		}
		callbackS := schema.StreamReaderWithConvert(copied[0], realConverter(ctx))
		newCtx, unused := callbacks.OnStartWithStreamInput(ctx, callbackS)
		unused.Close()
		return newCtx, copied[1], nil
	}

	newCtx, newInput := callbacks.OnStartWithStreamInput(ctx, input)
	return newCtx, newInput, nil
}

func (r *nodeRunner[O]) preProcess(ctx context.Context, input map[string]any) (_ map[string]any, err error) {
	for _, preProcessor := range r.preProcessors {
		if preProcessor == nil {
			continue
		}

		input, err = preProcessor(ctx, input)
		if err != nil {
			return nil, err
		}
	}
	return input, nil
}

func (r *nodeRunner[O]) postProcess(ctx context.Context, output map[string]any) (_ map[string]any, err error) {
	for _, postProcessor := range r.postProcessors {
		if postProcessor == nil {
			continue
		}

		output, err = postProcessor(ctx, output)
		if err != nil {
			return nil, err
		}
	}
	return output, nil
}

func (r *nodeRunner[O]) invoke(ctx context.Context, input map[string]any, opts ...O) (output map[string]any, err error) {
	var n int64
	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		output, err = r.i(ctx, input, opts...)
		if err != nil {
			if _, ok := compose.IsInterruptRerunError(err); ok { // interrupt, won't retry
				r.interrupted = true
				return nil, err
			}

			logs.CtxErrorf(ctx, "[invoke] node %s ID %s failed on %d attempt, err: %v", r.nodeName, r.nodeKey, n, err)
			if r.maxRetry > n {
				n++
				if exeCtx := execute.GetExeCtx(ctx); exeCtx != nil && exeCtx.NodeCtx != nil {
					exeCtx.CurrentRetryCount++
				}
				continue
			}
			return nil, err
		}

		return output, nil
	}
}

func (r *nodeRunner[O]) stream(ctx context.Context, input map[string]any, opts ...O) (output *schema.StreamReader[map[string]any], err error) {
	var n int64
	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		output, err = r.s(ctx, input, opts...)
		if err != nil {
			if _, ok := compose.IsInterruptRerunError(err); ok { // interrupt, won't retry
				r.interrupted = true
				return nil, err
			}

			logs.CtxErrorf(ctx, "[invoke] node %s ID %s failed on %d attempt, err: %v", r.nodeName, r.nodeKey, n, err)
			if r.maxRetry > n {
				n++
				if exeCtx := execute.GetExeCtx(ctx); exeCtx != nil && exeCtx.NodeCtx != nil {
					exeCtx.CurrentRetryCount++
				}
				continue
			}
			return nil, err
		}

		return output, nil
	}
}

func (r *nodeRunner[O]) transform(ctx context.Context, input *schema.StreamReader[map[string]any], opts ...O) (output *schema.StreamReader[map[string]any], err error) {
	if r.maxRetry == 0 {
		return r.t(ctx, input, opts...)
	}

	copied := input.Copy(int(r.maxRetry))

	var n int64
	defer func() {
		for i := n + 1; i < r.maxRetry; i++ {
			copied[i].Close()
		}
	}()

	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		output, err = r.t(ctx, copied[n], opts...)
		if err != nil {
			if _, ok := compose.IsInterruptRerunError(err); ok { // interrupt, won't retry
				r.interrupted = true
				return nil, err
			}

			logs.CtxErrorf(ctx, "[invoke] node %s ID %s failed on %d attempt, err: %v", r.nodeName, r.nodeKey, n, err)
			if r.maxRetry > n {
				n++
				if exeCtx := execute.GetExeCtx(ctx); exeCtx != nil && exeCtx.NodeCtx != nil {
					exeCtx.CurrentRetryCount++
				}
				continue
			}
			return nil, err
		}

		return output, nil
	}
}

func (r *nodeRunner[O]) onEnd(ctx context.Context, output map[string]any) error {
	if r.errProcessType == vo.ErrorProcessTypeExceptionBranch || r.errProcessType == vo.ErrorProcessTypeDefault {
		output["isSuccess"] = true
	}

	if !r.callbackEnabled {
		return nil
	}

	if r.callbackOutputConverter != nil {
		convertedOutput, err := r.callbackOutputConverter(ctx, output)
		if err != nil {
			return err
		}
		_ = callbacks.OnEnd(ctx, convertedOutput)
	} else {
		_ = callbacks.OnEnd(ctx, output)
	}

	return nil
}

func (r *nodeRunner[O]) onEndStream(ctx context.Context, output *schema.StreamReader[map[string]any]) (
	*schema.StreamReader[map[string]any], error) {
	if r.errProcessType == vo.ErrorProcessTypeExceptionBranch || r.errProcessType == vo.ErrorProcessTypeDefault {
		flag := schema.StreamReaderFromArray([]map[string]any{{"isSuccess": true}})
		output = schema.MergeStreamReaders([]*schema.StreamReader[map[string]any]{flag, output})
	}

	if !r.callbackEnabled {
		return output, nil
	}

	if r.callbackOutputConverter != nil {
		copied := output.Copy(2)
		realConverter := func(ctx context.Context) func(map[string]any) (*nodes.StructuredCallbackOutput, error) {
			return func(in map[string]any) (*nodes.StructuredCallbackOutput, error) {
				return r.callbackOutputConverter(ctx, in)
			}
		}
		callbackS := schema.StreamReaderWithConvert(copied[0], realConverter(ctx))
		_, unused := callbacks.OnEndWithStreamOutput(ctx, callbackS)
		unused.Close()

		return copied[1], nil
	}

	_, newOutput := callbacks.OnEndWithStreamOutput(ctx, output)
	return newOutput, nil
}

func (r *nodeRunner[O]) onError(ctx context.Context, err error) (map[string]any, bool) {
	if r.interrupted {
		if r.callbackEnabled {
			_ = callbacks.OnError(ctx, err)
		}
		return nil, false
	}

	var sErr vo.WorkflowError
	if !errors.As(err, &sErr) {
		if errors.Is(err, context.DeadlineExceeded) {
			sErr = vo.NodeTimeoutErr
		} else if errors.Is(err, context.Canceled) {
			sErr = vo.CancelErr
		} else {
			sErr = vo.WrapError(errno.ErrWorkflowExecuteFail, err, errorx.KV("cause", vo.UnwrapRootErr(err).Error()))
		}
	}

	code := int(sErr.Code())
	msg := sErr.Msg()

	switch r.errProcessType {
	case vo.ErrorProcessTypeDefault:
		d := r.dataOnErr(ctx)
		d["errorBody"] = map[string]any{
			"errorMessage": msg,
			"errorCode":    code,
		}
		d["isSuccess"] = false
		if r.callbackEnabled {
			sErr = sErr.ChangeErrLevel(vo.LevelWarn)
			sOutput := &nodes.StructuredCallbackOutput{
				Output:    d,
				RawOutput: d,
				Error:     sErr,
			}
			_ = callbacks.OnEnd(ctx, sOutput)
		}
		return d, true
	case vo.ErrorProcessTypeExceptionBranch:
		s := make(map[string]any)
		s["errorBody"] = map[string]any{
			"errorMessage": msg,
			"errorCode":    code,
		}
		s["isSuccess"] = false
		if r.callbackEnabled {
			sErr = sErr.ChangeErrLevel(vo.LevelWarn)
			sOutput := &nodes.StructuredCallbackOutput{
				Output:    s,
				RawOutput: s,
				Error:     sErr,
			}
			_ = callbacks.OnEnd(ctx, sOutput)
		}
		return s, true
	default:
		if r.callbackEnabled {
			_ = callbacks.OnError(ctx, sErr)
		}
		return nil, false
	}
}

func parseDefaultOutput(ctx context.Context, data string, schema_ map[string]*vo.TypeInfo) (map[string]any, error) {
	var result map[string]any

	err := sonic.UnmarshalString(data, &result)
	if err != nil {
		return nil, err
	}

	r, ws, e := nodes.ConvertInputs(ctx, result, schema_)
	if e != nil {
		return nil, e
	}

	if ws != nil {
		logs.CtxWarnf(ctx, "convert output warnings: %v", *ws)
	}

	return r, nil
}

func parseDefaultOutputOrFallback(ctx context.Context, data string, schema_ map[string]*vo.TypeInfo) map[string]any {
	result, err := parseDefaultOutput(ctx, data, schema_)
	if err != nil {
		fallback := make(map[string]any, len(schema_))
		for k, v := range schema_ {
			if v.Type == vo.DataTypeString {
				fallback[k] = data
				continue
			}
			fallback[k] = v.Zero()
		}
		return fallback
	}
	return result
}

func preTypeConverter(inTypes map[string]*vo.TypeInfo) func(ctx context.Context, in map[string]any) (map[string]any, error) {
	return func(ctx context.Context, in map[string]any) (map[string]any, error) {
		out, ws, err := nodes.ConvertInputs(ctx, in, inTypes)
		if err != nil {
			return nil, err
		}

		if ws != nil {
			logs.CtxWarnf(ctx, "convert inputs warnings: %v", *ws)
		}

		return out, err
	}
}

func trimKeyFinishedMarker(ctx context.Context, in map[string]any) (map[string]any, bool, error) {
	var (
		newIn   map[string]any
		trimmed bool
	)
	for k, v := range in {
		if vStr, ok := v.(string); ok {
			if strings.HasSuffix(vStr, nodes.KeyIsFinished) {
				if newIn == nil {
					newIn = maps.Clone(in)
				}
				vStr = strings.TrimSuffix(vStr, nodes.KeyIsFinished)
				newIn[k] = vStr
				trimmed = true
			}
		} else if vMap, ok := v.(map[string]any); ok {
			newMap, subTrimmed, err := trimKeyFinishedMarker(ctx, vMap)
			if err != nil {
				return nil, false, err
			}
			if subTrimmed {
				if newIn == nil {
					newIn = maps.Clone(in)
				}
				newIn[k] = newMap
				trimmed = true
			}
		}
	}

	if trimmed {
		return newIn, true, nil
	}

	return in, false, nil
}

func keyFinishedMarkerTrimmer() func(ctx context.Context, in map[string]any) (map[string]any, error) {
	return func(ctx context.Context, in map[string]any) (map[string]any, error) {
		out, _, err := trimKeyFinishedMarker(ctx, in)
		return out, err
	}
}
