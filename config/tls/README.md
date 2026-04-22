Test TLS assets for local XMPP STARTTLS / direct-TLS development.

Files:
- `ca.crt`: local test CA certificate
- `ca.key`: local test CA private key
- `server.crt`: server certificate signed by `ca.crt`
- `server.key`: server private key

Server certificate SANs:
- `eag.example.com`
- `localhost`
- `127.0.0.1`

Example config:
- `xmpp_server.tls.cert: "config/tls/server.crt"`
- `xmpp_server.tls.key: "config/tls/server.key"`

These files are for development only.
