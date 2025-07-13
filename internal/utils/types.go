package utils

import (
	"github.com/checkmarble/marble-llm-adapter/internal"
	"github.com/samber/lo"
)

func CastProviderOptions[T internal.ProviderRequestOptions](opts internal.ProviderRequestOptions) T {
	if opts == nil {
		return lo.FromPtr[T](nil)
	}

	if cast, ok := opts.(T); ok {
		return cast
	}

	return lo.FromPtr[T](nil)
}

func MaybeF64ToF32(f *float64) *float32 {
	if f == nil {
		return nil
	}

	return lo.ToPtr(float32(*f))
}
