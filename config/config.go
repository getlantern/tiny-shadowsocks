// Package config provides the configuration for the shadowsocks dialer
package config

// Config contains the configuration for the shadowsocks dialer.
// Currently it only support the following encryption methods: "chacha20-ietf-poly1305"
// and "xchacha20-ietf-poly1305"
//
//tinyjson:json
type Config struct {
	RemoteAddr string `json:"remote_addr"`
	RemotePort string `json:"remote_port"`
	Password   string `json:"password"`
	Method     string `json:"method"`
}
