package internal

import (
	"testing"

	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
)

type providerOpts struct {
	Number int
}

func (providerOpts) RequestOptionsForProvider() {}

type provider2Opts struct {
	Number int
}

func (provider2Opts) RequestOptionsForProvider() {}

func TestCastProviderOptions(t *testing.T) {
	assert.Zero(t, CastProviderOptions[providerOpts](nil))
	assert.Zero(t, CastProviderOptions[providerOpts](provider2Opts{10}))
	assert.Equal(t, 42, CastProviderOptions[providerOpts](providerOpts{42}).Number)
}

func TestMaybeF64ToF32(t *testing.T) {
	assert.Nil(t, MaybeF64ToF32(nil))
	assert.Equal(t, lo.ToPtr(float32(1.0)), MaybeF64ToF32(lo.ToPtr(float64(1.0))))
}
