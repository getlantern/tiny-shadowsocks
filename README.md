# tiny-shadowsocks

This repository implements a tiny-go shadowsocks client so it can be used as a WATER dialer

## Requirements

- [TinyGo](https://tinygo.org/getting-started/install/)

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
