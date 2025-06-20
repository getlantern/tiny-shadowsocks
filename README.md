# tiny-shadowsocks

This repository implements a tiny-go shadowsocks client so it can be used as a WATER dialer

## Requirements

- [TinyGo](https://tinygo.org/getting-started/install/) - It must be on version 0.31.2

## Building

For generating a debug version, run:

```bash
make debug
```

For generating a release version, run:

```bash
make release
```
## Config

If you need to add a new field that should be loaded from the config file, you can add it to the `Config` struct in `config/config.go` and generate the tinyjson file with:

```bash
make config
```

## Testing

```bash
make test
```

## Integration tests

For testing the generated WASM with the integration tests, you must start a sing-box shadowsocks inbound so we can try to make a request
<details>
<summary>sing-box shadowsocks inbound example</summary>

```json
{
  "log": {
    "level": "info",
    "output": "stdout"
  },
  "inbounds": [
    {
      "type": "shadowsocks",
      "tag": "ss-in",
      "listen": "127.0.0.1",
      "listen_port": 8388,
      "method": "chacha20-ietf-poly1305",
      "password": "8JCsPssfgS8tiRwiMlhARg==",
      "network": "tcp"
    }
  ],
  "outbounds": [
    {
      "type": "direct",
      "tag": "direct"
    }
  ]
}
```

After the inbound server is listening, you should be able to run the integration tests with `make integration-test`
