package browser

import (
	"testing"
)

func TestParseProxyURL(t *testing.T) {
	tests := []struct {
		name             string
		input            string
		wantProxyAddr    string
		wantUsername     string
		wantPassword     string
		wantErr          bool
	}{
		{
			name:          "HTTP proxy with auth",
			input:         "http://saf6fqgy:TcNETwYZCRPk@162.141.95.71:5782",
			wantProxyAddr: "http://162.141.95.71:5782",
			wantUsername:  "saf6fqgy",
			wantPassword:  "TcNETwYZCRPk",
			wantErr:       false,
		},
		{
			name:          "SOCKS5 proxy with auth",
			input:         "socks5://user:pass@192.168.1.1:1080",
			wantProxyAddr: "socks5://192.168.1.1:1080",
			wantUsername:  "user",
			wantPassword:  "pass",
			wantErr:       false,
		},
		{
			name:          "HTTP proxy without auth",
			input:         "http://162.141.95.71:5782",
			wantProxyAddr: "http://162.141.95.71:5782",
			wantUsername:  "",
			wantPassword:  "",
			wantErr:       false,
		},
		{
			name:          "Empty proxy",
			input:         "",
			wantProxyAddr: "",
			wantUsername:  "",
			wantPassword:  "",
			wantErr:       false,
		},
		{
			name:          "HTTPS proxy with auth",
			input:         "https://admin:secret123@proxy.example.com:8080",
			wantProxyAddr: "https://proxy.example.com:8080",
			wantUsername:  "admin",
			wantPassword:  "secret123",
			wantErr:       false,
		},
		{
			name:          "Proxy with special chars in password",
			input:         "http://user:p@ss:w0rd!@proxy.local:3128",
			wantProxyAddr: "http://proxy.local:3128",
			wantUsername:  "user",
			wantPassword:  "p@ss:w0rd!",
			wantErr:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			proxyAddr, username, password, err := parseProxyURL(tt.input)
			
			if (err != nil) != tt.wantErr {
				t.Errorf("parseProxyURL() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			
			if proxyAddr != tt.wantProxyAddr {
				t.Errorf("parseProxyURL() proxyAddr = %v, want %v", proxyAddr, tt.wantProxyAddr)
			}
			
			if username != tt.wantUsername {
				t.Errorf("parseProxyURL() username = %v, want %v", username, tt.wantUsername)
			}
			
			if password != tt.wantPassword {
				t.Errorf("parseProxyURL() password = %v, want %v", password, tt.wantPassword)
			}
		})
	}
}
