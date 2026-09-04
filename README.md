# 3D Job Desk Printer Bridge

A small, open-source program that runs on a shop computer and reports
Klipper (Moonraker) printer status to a [3D Job Desk](https://3djobdesk.com)
desk over outbound HTTPS. It never opens a port and refuses to contact public
IP addresses.

Download prebuilt binaries: https://app.3djobdesk.com/bridge

## Build from source

```bash
go test ./...
go build -o 3d-job-desk-bridge .
```

## Security

- Outbound HTTPS only, TLS 1.2+.
- Refuses public IPs, loopback-only metadata endpoints, and link-local addresses.
- Only reads Moonraker `/printer/objects/query`; it cannot send G-code.
- Pairing codes are single-use, hashed, and expire in 15 minutes.
- Device secrets are stored `0600` under the OS user config directory.

## License

MIT — see [LICENSE](LICENSE).
