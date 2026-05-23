package database

import (
	"context"
	"strings"
	"unicode"

	"github.com/jackc/pgx/v5"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

type tracerContextKey struct{}

type Tracer struct {
	tracer trace.Tracer
}

func NewTracer() *Tracer {
	return &Tracer{
		tracer: otel.Tracer("github.com/avijitnpm/modular-monolith/internal/database"),
	}
}

func (t *Tracer) TraceQueryStart(
	ctx context.Context,
	_ *pgx.Conn,
	data pgx.TraceQueryStartData,
) context.Context {

	ctx, span := t.tracer.Start(
		ctx,
		"postgres "+queryOperation(data.SQL),
		trace.WithSpanKind(trace.SpanKindClient),
		trace.WithAttributes(
			attribute.String("db.system.name", "postgresql"),
			attribute.String("db.operation.name", queryOperation(data.SQL)),
		),
	)

	return context.WithValue(ctx, tracerContextKey{}, span)
}

func (t *Tracer) TraceQueryEnd(
	ctx context.Context,
	_ *pgx.Conn,
	data pgx.TraceQueryEndData,
) {

	span, ok := ctx.Value(tracerContextKey{}).(trace.Span)
	if !ok {
		return
	}

	if data.CommandTag.String() != "" {
		span.SetAttributes(
			attribute.String("db.response.status_code", data.CommandTag.String()),
		)
	}

	if data.Err != nil {
		span.RecordError(data.Err)
		span.SetStatus(codes.Error, data.Err.Error())
	}

	span.End()
}

func queryOperation(sql string) string {
	sql = strings.TrimLeftFunc(sql, unicode.IsSpace)
	if sql == "" {
		return "query"
	}

	fields := strings.Fields(sql)
	if len(fields) == 0 {
		return "query"
	}

	return strings.ToUpper(fields[0])
}
