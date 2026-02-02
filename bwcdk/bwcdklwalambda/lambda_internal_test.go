package bwcdklwalambda

import (
	"testing"
)

func TestParsePassThroughPath(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		path       string
		wantSuffix string
		wantErr    bool
	}{
		{
			name:       "valid simple handler",
			path:       "/l/authorize",
			wantSuffix: "Authorize",
		},
		{
			name:       "valid kebab-case handler",
			path:       "/l/some-handler",
			wantSuffix: "SomeHandler",
		},
		{
			name:       "valid multi-part kebab-case",
			path:       "/l/my-long-handler-name",
			wantSuffix: "MyLongHandlerName",
		},
		{
			name:    "missing l prefix",
			path:    "/authorize",
			wantErr: true,
		},
		{
			name:    "wrong prefix",
			path:    "/api/authorize",
			wantErr: true,
		},
		{
			name:    "empty handler",
			path:    "/l/",
			wantErr: true,
		},
		{
			name:    "too many segments",
			path:    "/l/authorize/extra",
			wantErr: true,
		},
		{
			name:    "not kebab-case - camelCase",
			path:    "/l/someHandler",
			wantErr: true,
		},
		{
			name:    "not kebab-case - PascalCase",
			path:    "/l/SomeHandler",
			wantErr: true,
		},
		{
			name:    "not kebab-case - snake_case",
			path:    "/l/some_handler",
			wantErr: true,
		},
		{
			name:    "empty path",
			path:    "",
			wantErr: true,
		},
		{
			name:    "just slash",
			path:    "/",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			suffix, err := parsePassThroughPath(tt.path)

			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error but got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if suffix != tt.wantSuffix {
				t.Errorf("suffix = %q, want %q", suffix, tt.wantSuffix)
			}
		})
	}
}
