// The MIT License
//
// Copyright (c) 2020 Temporal Technologies Inc.  All rights reserved.
//
// Copyright (c) 2020 Uber Technologies, Inc.
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

// Code generated by gowrap. DO NOT EDIT.
// template: gowrap_template
// gowrap: http://github.com/hexdigest/gowrap

package telemetry

//go:generate gowrap gen -p go.temporal.io/server/common/persistence -i MetadataStore -t gowrap_template -o shard_store_gen.go -l ""

import (
	"context"
	"encoding/json"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"go.temporal.io/server/common/log"
	"go.temporal.io/server/common/log/tag"
	_sourcePersistence "go.temporal.io/server/common/persistence"
	"go.temporal.io/server/common/telemetry"
)

// telemetryMetadataStore implements MetadataStore interface instrumented with OpenTelemetry.
type telemetryMetadataStore struct {
	_sourcePersistence.MetadataStore
	tracer    trace.Tracer
	logger    log.Logger
	debugMode bool
}

// newTelemetryMetadataStore returns telemetryMetadataStore.
func newTelemetryMetadataStore(
	base _sourcePersistence.MetadataStore,
	logger log.Logger,
	tracer trace.Tracer,
) telemetryMetadataStore {
	return telemetryMetadataStore{
		MetadataStore: base,
		tracer:        tracer,
		debugMode:     telemetry.DebugMode(),
	}
}

// CreateNamespace wraps MetadataStore.CreateNamespace.
func (d telemetryMetadataStore) CreateNamespace(ctx context.Context, request *_sourcePersistence.InternalCreateNamespaceRequest) (cp1 *_sourcePersistence.CreateNamespaceResponse, err error) {
	ctx, span := d.tracer.Start(
		ctx,
		"persistence.MetadataStore/CreateNamespace",
		trace.WithAttributes(
			attribute.Key("persistence.store").String("MetadataStore"),
			attribute.Key("persistence.method").String("CreateNamespace"),
		))
	defer span.End()

	if deadline, ok := ctx.Deadline(); ok {
		span.SetAttributes(attribute.String("deadline", deadline.Format(time.RFC3339Nano)))
		span.SetAttributes(attribute.String("timeout", time.Until(deadline).String()))
	}

	cp1, err = d.MetadataStore.CreateNamespace(ctx, request)
	if err != nil {
		span.RecordError(err)
	}

	if d.debugMode {

		requestPayload, err := json.MarshalIndent(request, "", "    ")
		if err != nil {
			d.logger.Error("failed to serialize *_sourcePersistence.InternalCreateNamespaceRequest for OTEL span", tag.Error(err))
		} else {
			span.SetAttributes(attribute.Key("persistence.request.payload").String(string(requestPayload)))
		}

		responsePayload, err := json.MarshalIndent(cp1, "", "    ")
		if err != nil {
			d.logger.Error("failed to serialize *_sourcePersistence.CreateNamespaceResponse for OTEL span", tag.Error(err))
		} else {
			span.SetAttributes(attribute.Key("persistence.response.payload").String(string(responsePayload)))
		}

	}

	return
}

// DeleteNamespace wraps MetadataStore.DeleteNamespace.
func (d telemetryMetadataStore) DeleteNamespace(ctx context.Context, request *_sourcePersistence.DeleteNamespaceRequest) (err error) {
	ctx, span := d.tracer.Start(
		ctx,
		"persistence.MetadataStore/DeleteNamespace",
		trace.WithAttributes(
			attribute.Key("persistence.store").String("MetadataStore"),
			attribute.Key("persistence.method").String("DeleteNamespace"),
		))
	defer span.End()

	if deadline, ok := ctx.Deadline(); ok {
		span.SetAttributes(attribute.String("deadline", deadline.Format(time.RFC3339Nano)))
		span.SetAttributes(attribute.String("timeout", time.Until(deadline).String()))
	}

	err = d.MetadataStore.DeleteNamespace(ctx, request)
	if err != nil {
		span.RecordError(err)
	}

	if d.debugMode {

		requestPayload, err := json.MarshalIndent(request, "", "    ")
		if err != nil {
			d.logger.Error("failed to serialize *_sourcePersistence.DeleteNamespaceRequest for OTEL span", tag.Error(err))
		} else {
			span.SetAttributes(attribute.Key("persistence.request.payload").String(string(requestPayload)))
		}

	}

	return
}

// DeleteNamespaceByName wraps MetadataStore.DeleteNamespaceByName.
func (d telemetryMetadataStore) DeleteNamespaceByName(ctx context.Context, request *_sourcePersistence.DeleteNamespaceByNameRequest) (err error) {
	ctx, span := d.tracer.Start(
		ctx,
		"persistence.MetadataStore/DeleteNamespaceByName",
		trace.WithAttributes(
			attribute.Key("persistence.store").String("MetadataStore"),
			attribute.Key("persistence.method").String("DeleteNamespaceByName"),
		))
	defer span.End()

	if deadline, ok := ctx.Deadline(); ok {
		span.SetAttributes(attribute.String("deadline", deadline.Format(time.RFC3339Nano)))
		span.SetAttributes(attribute.String("timeout", time.Until(deadline).String()))
	}

	err = d.MetadataStore.DeleteNamespaceByName(ctx, request)
	if err != nil {
		span.RecordError(err)
	}

	if d.debugMode {

		requestPayload, err := json.MarshalIndent(request, "", "    ")
		if err != nil {
			d.logger.Error("failed to serialize *_sourcePersistence.DeleteNamespaceByNameRequest for OTEL span", tag.Error(err))
		} else {
			span.SetAttributes(attribute.Key("persistence.request.payload").String(string(requestPayload)))
		}

	}

	return
}

// GetMetadata wraps MetadataStore.GetMetadata.
func (d telemetryMetadataStore) GetMetadata(ctx context.Context) (gp1 *_sourcePersistence.GetMetadataResponse, err error) {
	ctx, span := d.tracer.Start(
		ctx,
		"persistence.MetadataStore/GetMetadata",
		trace.WithAttributes(
			attribute.Key("persistence.store").String("MetadataStore"),
			attribute.Key("persistence.method").String("GetMetadata"),
		))
	defer span.End()

	if deadline, ok := ctx.Deadline(); ok {
		span.SetAttributes(attribute.String("deadline", deadline.Format(time.RFC3339Nano)))
		span.SetAttributes(attribute.String("timeout", time.Until(deadline).String()))
	}

	gp1, err = d.MetadataStore.GetMetadata(ctx)
	if err != nil {
		span.RecordError(err)
	}

	if d.debugMode {

		responsePayload, err := json.MarshalIndent(gp1, "", "    ")
		if err != nil {
			d.logger.Error("failed to serialize *_sourcePersistence.GetMetadataResponse for OTEL span", tag.Error(err))
		} else {
			span.SetAttributes(attribute.Key("persistence.response.payload").String(string(responsePayload)))
		}

	}

	return
}

// GetNamespace wraps MetadataStore.GetNamespace.
func (d telemetryMetadataStore) GetNamespace(ctx context.Context, request *_sourcePersistence.GetNamespaceRequest) (ip1 *_sourcePersistence.InternalGetNamespaceResponse, err error) {
	ctx, span := d.tracer.Start(
		ctx,
		"persistence.MetadataStore/GetNamespace",
		trace.WithAttributes(
			attribute.Key("persistence.store").String("MetadataStore"),
			attribute.Key("persistence.method").String("GetNamespace"),
		))
	defer span.End()

	if deadline, ok := ctx.Deadline(); ok {
		span.SetAttributes(attribute.String("deadline", deadline.Format(time.RFC3339Nano)))
		span.SetAttributes(attribute.String("timeout", time.Until(deadline).String()))
	}

	ip1, err = d.MetadataStore.GetNamespace(ctx, request)
	if err != nil {
		span.RecordError(err)
	}

	if d.debugMode {

		requestPayload, err := json.MarshalIndent(request, "", "    ")
		if err != nil {
			d.logger.Error("failed to serialize *_sourcePersistence.GetNamespaceRequest for OTEL span", tag.Error(err))
		} else {
			span.SetAttributes(attribute.Key("persistence.request.payload").String(string(requestPayload)))
		}

		responsePayload, err := json.MarshalIndent(ip1, "", "    ")
		if err != nil {
			d.logger.Error("failed to serialize *_sourcePersistence.InternalGetNamespaceResponse for OTEL span", tag.Error(err))
		} else {
			span.SetAttributes(attribute.Key("persistence.response.payload").String(string(responsePayload)))
		}

	}

	return
}

// ListNamespaces wraps MetadataStore.ListNamespaces.
func (d telemetryMetadataStore) ListNamespaces(ctx context.Context, request *_sourcePersistence.InternalListNamespacesRequest) (ip1 *_sourcePersistence.InternalListNamespacesResponse, err error) {
	ctx, span := d.tracer.Start(
		ctx,
		"persistence.MetadataStore/ListNamespaces",
		trace.WithAttributes(
			attribute.Key("persistence.store").String("MetadataStore"),
			attribute.Key("persistence.method").String("ListNamespaces"),
		))
	defer span.End()

	if deadline, ok := ctx.Deadline(); ok {
		span.SetAttributes(attribute.String("deadline", deadline.Format(time.RFC3339Nano)))
		span.SetAttributes(attribute.String("timeout", time.Until(deadline).String()))
	}

	ip1, err = d.MetadataStore.ListNamespaces(ctx, request)
	if err != nil {
		span.RecordError(err)
	}

	if d.debugMode {

		requestPayload, err := json.MarshalIndent(request, "", "    ")
		if err != nil {
			d.logger.Error("failed to serialize *_sourcePersistence.InternalListNamespacesRequest for OTEL span", tag.Error(err))
		} else {
			span.SetAttributes(attribute.Key("persistence.request.payload").String(string(requestPayload)))
		}

		responsePayload, err := json.MarshalIndent(ip1, "", "    ")
		if err != nil {
			d.logger.Error("failed to serialize *_sourcePersistence.InternalListNamespacesResponse for OTEL span", tag.Error(err))
		} else {
			span.SetAttributes(attribute.Key("persistence.response.payload").String(string(responsePayload)))
		}

	}

	return
}

// RenameNamespace wraps MetadataStore.RenameNamespace.
func (d telemetryMetadataStore) RenameNamespace(ctx context.Context, request *_sourcePersistence.InternalRenameNamespaceRequest) (err error) {
	ctx, span := d.tracer.Start(
		ctx,
		"persistence.MetadataStore/RenameNamespace",
		trace.WithAttributes(
			attribute.Key("persistence.store").String("MetadataStore"),
			attribute.Key("persistence.method").String("RenameNamespace"),
		))
	defer span.End()

	if deadline, ok := ctx.Deadline(); ok {
		span.SetAttributes(attribute.String("deadline", deadline.Format(time.RFC3339Nano)))
		span.SetAttributes(attribute.String("timeout", time.Until(deadline).String()))
	}

	err = d.MetadataStore.RenameNamespace(ctx, request)
	if err != nil {
		span.RecordError(err)
	}

	if d.debugMode {

		requestPayload, err := json.MarshalIndent(request, "", "    ")
		if err != nil {
			d.logger.Error("failed to serialize *_sourcePersistence.InternalRenameNamespaceRequest for OTEL span", tag.Error(err))
		} else {
			span.SetAttributes(attribute.Key("persistence.request.payload").String(string(requestPayload)))
		}

	}

	return
}

// UpdateNamespace wraps MetadataStore.UpdateNamespace.
func (d telemetryMetadataStore) UpdateNamespace(ctx context.Context, request *_sourcePersistence.InternalUpdateNamespaceRequest) (err error) {
	ctx, span := d.tracer.Start(
		ctx,
		"persistence.MetadataStore/UpdateNamespace",
		trace.WithAttributes(
			attribute.Key("persistence.store").String("MetadataStore"),
			attribute.Key("persistence.method").String("UpdateNamespace"),
		))
	defer span.End()

	if deadline, ok := ctx.Deadline(); ok {
		span.SetAttributes(attribute.String("deadline", deadline.Format(time.RFC3339Nano)))
		span.SetAttributes(attribute.String("timeout", time.Until(deadline).String()))
	}

	err = d.MetadataStore.UpdateNamespace(ctx, request)
	if err != nil {
		span.RecordError(err)
	}

	if d.debugMode {

		requestPayload, err := json.MarshalIndent(request, "", "    ")
		if err != nil {
			d.logger.Error("failed to serialize *_sourcePersistence.InternalUpdateNamespaceRequest for OTEL span", tag.Error(err))
		} else {
			span.SetAttributes(attribute.Key("persistence.request.payload").String(string(requestPayload)))
		}

	}

	return
}
