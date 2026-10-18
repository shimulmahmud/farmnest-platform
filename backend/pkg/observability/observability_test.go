package observability

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewProvider(t *testing.T) {
	p, err := NewProvider("test-service", 9091)
	assert.NoError(t, err)
	assert.NotNil(t, p)
	assert.NotNil(t, p.TracerProvider)
	assert.NotNil(t, p.MeterProvider)

	tracer := p.Tracer("test")
	assert.NotNil(t, tracer)

	meter := p.Meter("test")
	assert.NotNil(t, meter)
}

func TestProvider_Shutdown(t *testing.T) {
	p, err := NewProvider("test-service", 9092)
	assert.NoError(t, err)
	err = p.Shutdown(context.Background())
	assert.NoError(t, err)
}
