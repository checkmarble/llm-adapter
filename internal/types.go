package internal

import (
	"github.com/samber/lo"
)

func CastProviderOptions[T ProviderRequestOptions](opts ProviderRequestOptions) T {
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

func MaybeIntToInt32(i *int) *int32 {
	if i == nil {
		return nil
	}

	return lo.ToPtr(int32(*i))
}
