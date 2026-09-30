package services

import "testing"

func TestValidateURL(t *testing.T) {

	tests := []struct {
		name    string
		url     string
		wantErr bool
	}{
		{
			name:    "valid https URL",
			url:     "https://google.com",
			wantErr: false,
		},
		{
			name:    "valid http URL",
			url:     "http://example.com",
			wantErr: false,
		},
		{
			name:    "missing scheme",
			url:     "google.com",
			wantErr: true,
		},
		{
			name:    "invalid scheme",
			url:     "ftp://example.com",
			wantErr: true,
		},
		{
			name:    "invalid URL",
			url:     "hello",
			wantErr: true,
		},
		{
			name:    "missing host",
			url:     "https://",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			err := ValidateURL(tt.url)

			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateURL() error = %v, wantErr = %v", err, tt.wantErr)
			}
		})
	}
}
