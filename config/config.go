// Package config provides the configuration for the shadowsocks dialer
package config

// Config contains the configuration for the shadowsocks dialer.
// Currently it only support the following encryption methods: "aes-128-ctr",
// "aes-192-ctr", "aes-256-ctr", "rc4-md5", "chacha20-ietf", "xchacha20"
//
//tinyjson:json
type Config struct {
	RemoteAddr string `json:"remote_addr"`
	RemotePort string `json:"remote_port"`
	Password   string `json:"password"`
	Method     string `json:"method"`
}
