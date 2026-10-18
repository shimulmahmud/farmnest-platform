package logger

import (
        "context"
        "fmt"
        "os"

        "go.opentelemetry.io/otel/trace"
        "go.uber.org/zap"
        "go.uber.org/zap/zapcore"
)

// Logger wraps a zap.SugaredLogger for structured logging.
type Logger struct {
        sugar *zap.SugaredLogger
        zap   *zap.Logger
}

// New creates a new Logger instance based on the provided level and encoding.
func New(level, encoding string) (*Logger, error) {
        var zapLevel zapcore.Level
        if err := zapLevel.UnmarshalText([]byte(level)); err != nil {
                return nil, fmt.Errorf("invalid log level %q: %w", level, err)
        }

        encoderCfg := zap.NewProductionEncoderConfig()
        encoderCfg.TimeKey = "ts"
        encoderCfg.EncodeTime = zapcore.ISO8601TimeEncoder
        encoderCfg.EncodeLevel = zapcore.CapitalLevelEncoder

        var encoder zapcore.Encoder
        switch encoding {
        case "json":
                encoder = zapcore.NewJSONEncoder(encoderCfg)
        case "console":
                encoder = zapcore.NewConsoleEncoder(encoderCfg)
        default:
                return nil, fmt.Errorf("unsupported log encoding %q", encoding)
        }

        core := zapcore.NewCore(
                encoder,
                zapcore.AddSync(os.Stdout),
                zapLevel,
        )

        zapLogger := zap.New(core, zap.AddCaller(), zap.AddStacktrace(zapcore.ErrorLevel))
        sugar := zapLogger.Sugar()

        return &Logger{sugar: sugar, zap: zapLogger}, nil
}

// With returns a logger with additional structured fields.
func (l *Logger) With(keysAndValues ...interface{}) *Logger {
        return &Logger{sugar: l.sugar.With(keysAndValues...), zap: l.zap}
}

// Named returns a logger with a sub-logger name.
func (l *Logger) Named(name string) *Logger {
        return &Logger{sugar: l.sugar.Named(name), zap: l.zap.Named(name)}
}

// Sync flushes any buffered log entries.
func (l *Logger) Sync() {
        _ = l.sugar.Sync()
}

// --- Sugar-level logging ---

func (l *Logger) Debug(template string, args ...interface{}) {
        l.sugar.Debugf(template, args...)
}

func (l *Logger) Info(template string, args ...interface{}) {
        l.sugar.Infof(template, args...)
}

func (l *Logger) Warn(template string, args ...interface{}) {
        l.sugar.Warnf(template, args...)
}

func (l *Logger) Error(template string, args ...interface{}) {
        l.sugar.Errorf(template, args...)
}

func (l *Logger) Fatal(template string, args ...interface{}) {
        l.sugar.Fatalf(template, args...)
}

// --- With-context logging ---

func (l *Logger) Debugw(msg string, keysAndValues ...interface{}) {
        l.sugar.Debugw(msg, keysAndValues...)
}

func (l *Logger) Infow(msg string, keysAndValues ...interface{}) {
        l.sugar.Infow(msg, keysAndValues...)
}

func (l *Logger) Warnw(msg string, keysAndValues ...interface{}) {
        l.sugar.Warnw(msg, keysAndValues...)
}

func (l *Logger) Errorw(msg string, keysAndValues ...interface{}) {
        l.sugar.Errorw(msg, keysAndValues...)
}

func (l *Logger) Fatalw(msg string, keysAndValues ...interface{}) {
        l.sugar.Fatalw(msg, keysAndValues...)
}

// CtxInfow logs at Info level with trace context extracted from ctx.
// It automatically injects trace_id, span_id, and request_id (if present)
// into the structured log output for correlation with distributed traces.
func (l *Logger) CtxInfow(ctx context.Context, msg string, keysAndValues ...interface{}) {
        fields := traceFields(ctx)
        fields = append(fields, keysAndValues...)
        l.sugar.Infow(msg, fields...)
}

// CtxErrorw logs at Error level with trace context extracted from ctx.
func (l *Logger) CtxErrorw(ctx context.Context, msg string, keysAndValues ...interface{}) {
        fields := traceFields(ctx)
        fields = append(fields, keysAndValues...)
        l.sugar.Errorw(msg, fields...)
}

// CtxWarnw logs at Warn level with trace context extracted from ctx.
func (l *Logger) CtxWarnw(ctx context.Context, msg string, keysAndValues ...interface{}) {
        fields := traceFields(ctx)
        fields = append(fields, keysAndValues...)
        l.sugar.Warnw(msg, fields...)
}

// CtxDebugw logs at Debug level with trace context extracted from ctx.
func (l *Logger) CtxDebugw(ctx context.Context, msg string, keysAndValues ...interface{}) {
        fields := traceFields(ctx)
        fields = append(fields, keysAndValues...)
        l.sugar.Debugw(msg, fields...)
}

// traceFields extracts trace_id and span_id from the OTel context.
// No secrets are exposed — only trace/span hex strings.
func traceFields(ctx context.Context) []interface{} {
        sc := trace.SpanContextFromContext(ctx)
        if !sc.IsValid() {
                return nil
        }
        return []interface{}{
                "trace_id", sc.TraceID().String(),
                "span_id", sc.SpanID().String(),
                "trace_sampled", sc.IsSampled(),
        }
}

// Zap returns the underlying *zap.Logger for use with OpenTelemetry or other
// libraries that accept zap's native interface.
func (l *Logger) Zap() *zap.Logger {
        return l.zap
}
