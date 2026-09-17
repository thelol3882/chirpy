package auth

import (
	"net/http"
	"testing"
)

func authHeader(value string) http.Header {
	headers := http.Header{}
	headers.Set("Authorization", value)
	return headers
}

func TestGetBearerToken(t *testing.T) {
	tests := []struct {
		name    string
		headers http.Header
		want    string
		wantErr bool
	}{
		{
			name:    "valid bearer token",
			headers: authHeader("Bearer abc123"),
			want:    "abc123",
		},
		{
			name:    "scheme is case insensitive",
			headers: authHeader("bearer abc123"),
			want:    "abc123",
		},
		{
			name:    "scheme in upper case",
			headers: authHeader("BEARER abc123"),
			want:    "abc123",
		},
		{
			name:    "extra spaces between scheme and token",
			headers: authHeader("Bearer    abc123"),
			want:    "abc123",
		},
		{
			name:    "surrounding whitespace",
			headers: authHeader("  Bearer abc123  "),
			want:    "abc123",
		},
		{
			name:    "no authorization header",
			headers: http.Header{},
			wantErr: true,
		},
		{
			name:    "empty authorization header",
			headers: authHeader(""),
			wantErr: true,
		},
		{
			name:    "wrong scheme",
			headers: authHeader("Basic abc123"),
			wantErr: true,
		},
		{
			name:    "scheme with no token",
			headers: authHeader("Bearer"),
			wantErr: true,
		},
		{
			name:    "token with no scheme",
			headers: authHeader("abc123"),
			wantErr: true,
		},
		{
			name:    "bearer is not the scheme",
			headers: authHeader("Digest Bearer"),
			wantErr: true,
		},
		{
			name:    "too many fields",
			headers: authHeader("Bearer abc123 extra"),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetBearerToken(tt.headers)

			if tt.wantErr {
				if err == nil {
					t.Fatalf("GetBearerToken() = %q, want error", got)
				}
				return
			}

			if err != nil {
				t.Fatalf("GetBearerToken() unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("GetBearerToken() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestGetApiKey(t *testing.T) {
	tests := []struct {
		name    string
		headers http.Header
		want    string
		wantErr bool
	}{
		{
			name:    "valid bearer token",
			headers: authHeader("ApiKey abc123"),
			want:    "abc123",
		},
		{
			name:    "scheme is case insensitive",
			headers: authHeader("apikey abc123"),
			want:    "abc123",
		},
		{
			name:    "scheme in upper case",
			headers: authHeader("APIKEY abc123"),
			want:    "abc123",
		},
		{
			name:    "extra spaces between scheme and token",
			headers: authHeader("ApiKey    abc123"),
			want:    "abc123",
		},
		{
			name:    "surrounding whitespace",
			headers: authHeader("  ApiKey abc123  "),
			want:    "abc123",
		},
		{
			name:    "no authorization header",
			headers: http.Header{},
			wantErr: true,
		},
		{
			name:    "empty authorization header",
			headers: authHeader(""),
			wantErr: true,
		},
		{
			name:    "wrong scheme",
			headers: authHeader("Basic abc123"),
			wantErr: true,
		},
		{
			name:    "scheme with no token",
			headers: authHeader("ApiKey"),
			wantErr: true,
		},
		{
			name:    "token with no scheme",
			headers: authHeader("abc123"),
			wantErr: true,
		},
		{
			name:    "bearer is not the scheme",
			headers: authHeader("Digest ApiKey"),
			wantErr: true,
		},
		{
			name:    "too many fields",
			headers: authHeader("ApiKey abc123 extra"),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetAPIKey(tt.headers)

			if tt.wantErr {
				if err == nil {
					t.Fatalf("GetBearerToken() = %q, want error", got)
				}
				return
			}

			if err != nil {
				t.Fatalf("GetBearerToken() unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("GetBearerToken() = %q, want %q", got, tt.want)
			}
		})
	}
}
