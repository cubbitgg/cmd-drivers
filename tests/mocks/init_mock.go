package mocks

import "context"

// MockFormatProvider is a test double for providers.FormatProvider.
// Set FormatFunc before use — leaving it nil will panic, forcing explicit test setup.
type MockFormatProvider struct {
	FormatFunc func(ctx context.Context, device, fsType string) error
}

func (m *MockFormatProvider) Format(ctx context.Context, device, fsType string) error {
	return m.FormatFunc(ctx, device, fsType)
}
