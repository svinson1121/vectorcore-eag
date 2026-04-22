# VectorCore EAG — XMPP Server Interface

**Client Developer Reference**

---

## Overview

VectorCore EAG includes a built-in XMPP server. External clients connect inbound,
authenticate, and receive a continuous push stream of CAP 1.2 emergency alerts
formatted as XEP-0127 headline messages. No polling or subscription management
is required — once connected and authenticated, alerts are delivered immediately
as they are ingested.

This document covers everything needed to build a client that connects to the EAG
XMPP server.

---

## Standards Implemented

| Standard | Description |
|---|---|
| RFC 6120 | XMPP Core — stream negotiation, stanza framing |
| RFC 6121 | XMPP Instant Messaging — resource binding (subset) |
| RFC 4616 | SASL PLAIN authentication mechanism |
| RFC 5246 / RFC 8446 | TLS 1.2 / TLS 1.3 (for STARTTLS and direct TLS listeners) |
| XEP-0034 | SASL Integration (stream feature advertisement) |
| XEP-0127 | Common Alerting Protocol (CAP) over XMPP — alert stanza format |
| CAP 1.2 | OASIS Common Alerting Protocol — alert payload |

---

## Connection Endpoints

EAG exposes up to two independent TCP listeners. Both are configured in
`config.yaml` and may be enabled independently.

### c2s — Plain TCP with optional STARTTLS (default port 5222)

```yaml
xmpp_server:
  c2s:
    enabled: true
    port: 5222
    starttls: true   # false = plain TCP only, no TLS upgrade offered
```

- When `starttls: false` — plain TCP, no encryption. Suitable for trusted internal networks.
- When `starttls: true` — server offers STARTTLS in stream features, but clients may still authenticate over plain TCP if they do not request TLS. Requires `tls.cert` and `tls.key` to be configured.

### c2s_tls — Direct TLS (default port 5223)

```yaml
xmpp_server:
  c2s_tls:
    enabled: true
    port: 5223
```

TLS is active from the first byte. No negotiation step — the client connects
directly into a TLS session. Requires `tls.cert` and `tls.key`.

### TLS configuration

```yaml
xmpp_server:
  domain: "eag.example.com"
  tls:
    cert: "/etc/vectorcore/eag.crt"
    key:  "/etc/vectorcore/eag.key"
```

TLS 1.2 minimum is enforced. Self-signed certificates are valid — clients may
need to disable certificate verification or trust the server's CA explicitly.

---

## Connection Flow

### Option A — Plain TCP (c2s, starttls: false)

```
1. TCP connect to host:port
2. XMPP stream open
3. Server offers SASL features
4. SASL PLAIN authentication
5. Stream restart
6. Resource bind
7. [Session established — alerts begin arriving]
```

### Option B — c2s with STARTTLS (c2s, starttls: true)

```
1. TCP connect to host:port
2. XMPP stream open
3. Server offers STARTTLS and SASL features
4. Client either:
   - sends `<starttls>`, completes TLS, restarts the stream, then authenticates
   - or sends SASL PLAIN immediately over plain TCP
5. Stream restart
6. Resource bind
7. [Session established — alerts begin arriving]
```

### Option C — Direct TLS (c2s_tls)

```
1. TLS connect to host:port (TLS from byte 0)
2. XMPP stream open (already inside TLS)
3. Server offers SASL features
4. SASL PLAIN authentication
5. Stream restart
6. Resource bind
7. [Session established — alerts begin arriving]
```

---

## Detailed Protocol Exchange

### 1. Stream Open

The client sends an XML stream opening. The server responds with its own
stream opening and a `<stream:features>` element.

**Client → Server:**
```xml
<?xml version='1.0'?>
<stream:stream
  to="eag.example.com"
  xmlns="jabber:client"
  xmlns:stream="http://etherx.jabber.org/streams"
  version="1.0">
```

**Server → Client:**
```xml
<?xml version='1.0'?>
<stream:stream
  from="eag.example.com"
  id="a3f9c12b"
  xmlns="jabber:client"
  xmlns:stream="http://etherx.jabber.org/streams"
  version="1.0">
```

> The `id` attribute is a random hex string generated per connection. It has
> no functional significance for clients.

---

### 2a. STARTTLS Negotiation (Option B only)

When `starttls: true`, the server offers STARTTLS alongside SASL authentication.
Clients may upgrade to TLS first or authenticate immediately without TLS.

**Server → Client (features):**
```xml
<stream:features>
  <starttls xmlns="urn:ietf:params:xml:ns:xmpp-tls"/>
  <mechanisms xmlns="urn:ietf:params:xml:ns:xmpp-sasl">
    <mechanism>PLAIN</mechanism>
  </mechanisms>
</stream:features>
```

**Client → Server:**
```xml
<starttls xmlns="urn:ietf:params:xml:ns:xmpp-tls"/>
```

**Server → Client:**
```xml
<proceed xmlns="urn:ietf:params:xml:ns:xmpp-tls"/>
```

Both sides immediately perform a TLS handshake on the existing TCP connection.
After the handshake completes, the client **must** restart the XML stream
(step 1 again) — TLS wraps the same connection, there is no reconnect.

---

### 2b. SASL Feature Advertisement

After stream open (and after TLS if applicable), the server advertises the
available authentication mechanism.

**Server → Client (features):**
```xml
<stream:features>
  <mechanisms xmlns="urn:ietf:params:xml:ns:xmpp-sasl">
    <mechanism>PLAIN</mechanism>
  </mechanisms>
</stream:features>
```

Only `PLAIN` is supported. Anonymous connections are not permitted.

---

### 3. SASL PLAIN Authentication

**Mechanism:** RFC 4616 SASL PLAIN

The client encodes credentials as:

```
base64( authzid \x00 authcid \x00 passwd )
```

- `authzid` — authorization identity. Set to empty string (leave blank before the first `\x00`).
- `authcid` — the peer's **username** as configured in the EAG web UI.
- `passwd`  — the peer's **password** as configured in the EAG web UI.
- `\x00`    — ASCII NUL byte (0x00) separator.

**Example — username `ops-client`, password `s3cr3t`:**

```
\x00 ops-client \x00 s3cr3t
→ base64 → AG9wcy1jbGllbnQAczNjcjN0
```

**Client → Server:**
```xml
<auth xmlns="urn:ietf:params:xml:ns:xmpp-sasl" mechanism="PLAIN">
  AG9wcy1jbGllbnQAczNjcjN0
</auth>
```

**Server → Client (success):**
```xml
<success xmlns="urn:ietf:params:xml:ns:xmpp-sasl"/>
```

**Server → Client (failure):**
```xml
<failure xmlns="urn:ietf:params:xml:ns:xmpp-sasl">
  <not-authorized/>
</failure>
```

Failure reasons: peer username not found, peer disabled, incorrect password.
The connection is closed after a failure.

After `<success/>`, the client **must** restart the XML stream (step 1 again).

---

### 4. Resource Binding

After the post-authentication stream restart, the server advertises bind:

**Server → Client (features):**
```xml
<stream:features>
  <bind xmlns="urn:ietf:params:xml:ns:xmpp-bind"/>
  <session xmlns="urn:ietf:params:xml:ns:xmpp-session"/>
</stream:features>
```

**Client → Server:**
```xml
<iq type="set" id="bind_1">
  <bind xmlns="urn:ietf:params:xml:ns:xmpp-bind">
    <resource>my-client-name</resource>
  </bind>
</iq>
```

The resource string is arbitrary. If omitted, the server assigns `eag`.

**Server → Client:**
```xml
<iq type="result" id="bind_1">
  <bind xmlns="urn:ietf:params:xml:ns:xmpp-bind">
    <jid>ops-client@eag.example.com/my-client-name</jid>
  </bind>
</iq>
```

**Session IQ (optional legacy):**

Some clients send a session establishment IQ after bind. EAG handles it:

```xml
<!-- Client -->
<iq type="set" id="sess_1">
  <session xmlns="urn:ietf:params:xml:ns:xmpp-session"/>
</iq>

<!-- Server -->
<iq type="result" id="sess_1"/>
```

The session is now fully established.

---

## Receiving Alerts

Once the session is established, the server pushes CAP alerts as XMPP
`<message type="headline">` stanzas. No subscription or request is needed.

### Stanza format

```xml
<message to="ops-client@eag.example.com" type="headline">
  <alert xmlns="urn:oasis:names:tc:emergency:cap:1.2">
    <identifier>urn:oid:2.49.0.1.840.0.abc123def456</identifier>
    <sender>w-nws.webmaster@noaa.gov</sender>
    <sent>2026-04-14T12:00:00-05:00</sent>
    <status>Actual</status>
    <msgType>Alert</msgType>
    <scope>Public</scope>
    <info>
      <language>en-US</language>
      <category>Met</category>
      <event>Tornado Warning</event>
      <responseType>Shelter</responseType>
      <urgency>Immediate</urgency>
      <severity>Extreme</severity>
      <certainty>Observed</certainty>
      <effective>2026-04-14T12:00:00-05:00</effective>
      <onset>2026-04-14T12:00:00-05:00</onset>
      <expires>2026-04-14T12:45:00-05:00</expires>
      <senderName>NWS Birmingham AL</senderName>
      <headline>Tornado Warning issued April 14 at 12:00PM CDT</headline>
      <description>A tornado has been observed...</description>
      <instruction>Take shelter immediately in an interior room...</instruction>
      <parameter>
        <valueName>affectedZones</valueName>
        <value>https://api.weather.gov/zones/county/ALJ001</value>
      </parameter>
      <area>
        <areaDesc>Jefferson; Shelby</areaDesc>
        <geocode>
          <valueName>SAME</valueName>
          <value>001073</value>
        </geocode>
        <geocode>
          <valueName>UGC</valueName>
          <value>ALJ001</value>
        </geocode>
      </area>
    </info>
  </alert>
</message>
```

### Key points

- `type="headline"` per XEP-0127. No acknowledgement required. Not stored in offline queues.
- The `<alert>` element is a **direct child** of `<message>` with the CAP 1.2 namespace.
- No `<body>` element is present. Clients must parse the CAP XML child.
- All text values are XML-entity-escaped (`&amp;` `&lt;` `&gt;` `&quot;` `&apos;`).
- All datetimes are RFC 3339 with timezone offset. Do not assume UTC.
- `Cancel` alerts are **not forwarded** — they are processed internally.

---

## CAP Field Reference

### Alert level

| Element | Always present | Values / Notes |
|---|---|---|
| `identifier` | yes | Globally unique CAP ID — use as dedup key |
| `sender` | yes | Originating sender address |
| `sent` | yes | RFC 3339 issue time |
| `status` | yes | `Actual` `Exercise` `System` `Test` `Draft` |
| `msgType` | yes | `Alert` or `Update` (Cancel is never forwarded) |
| `scope` | yes | `Public` `Restricted` `Private` |
| `references` | no | Space-separated prior alert IDs superseded by an Update |

### Info level

| Element | Always present | Values / Notes |
|---|---|---|
| `language` | yes | Always `en-US` |
| `category` | yes | `Geo` `Met` `Safety` `Security` `Rescue` `Fire` `Health` `Env` `Transport` `Infra` `CBRNE` `Other` |
| `event` | yes | Free text event type, e.g. `Tornado Warning` |
| `urgency` | yes | `Immediate` `Expected` `Future` `Past` `Unknown` |
| `severity` | yes | `Extreme` `Severe` `Moderate` `Minor` `Unknown` |
| `certainty` | yes | `Observed` `Likely` `Possible` `Unlikely` `Unknown` |
| `effective` | no | RFC 3339 — when the alert becomes effective |
| `onset` | no | RFC 3339 — expected start of hazardous condition |
| `expires` | no | RFC 3339 — when the alert expires |
| `responseType` | no | `Shelter` `Evacuate` `Prepare` `Execute` `Avoid` `Monitor` `Assess` `AllClear` `None` |
| `senderName` | no | Human-readable name of the issuing organisation |
| `headline` | no | Short human-readable summary |
| `description` | no | Full plain-text description |
| `instruction` | no | Recommended public action |
| `parameter` | no | Repeatable — source-specific extensions; each has `<valueName>` and `<value>` |
| `area/areaDesc` | no | Human-readable affected area |
| `area/geocode` | no | Repeatable — `<valueName>` is `SAME` or `UGC`; `<value>` is the code |

---

## Per-Peer Filters

Each peer account in EAG has optional server-side filters. Alerts that do not
match **all** configured filters are not forwarded to that peer. Clients receive
only pre-filtered alerts — no client-side filtering is necessary.

| Filter | Match type | CAP field |
|---|---|---|
| Severity | Exact membership | `<severity>` |
| Event | Substring | `<event>` |
| Area | Substring | `<areaDesc>` |
| Status | Exact membership | `<status>` |

An empty filter list means "match all". Filters are configured per-peer in the
EAG web UI under **CAP XMPP Peers**.

---

## Keepalive and Disconnection

EAG holds the connection open indefinitely after session establishment. The
server does not send XMPP `<presence>` or ping stanzas.

Clients should:

- Implement TCP keepalive at the socket level, or
- Send periodic XMPP whitespace pings (`\n`) to detect dead connections, or
- Use XEP-0199 (XMPP Ping) — EAG will not respond to ping IQs but the TCP
  error on a dead connection will be detected by the write.

On disconnect, simply reconnect and re-authenticate. EAG will sweep all
unforwarded alerts and re-deliver any that arrived while the client was
disconnected.

---

## Concurrency

One active session is permitted per peer username. If a second client
authenticates with the same credentials, the existing session is terminated
and replaced by the new one.

---

## Client Implementation Examples

### Python (using `slixmpp`)

```python
import slixmpp
import asyncio
from xml.etree import ElementTree as ET

CAP_NS = "urn:oasis:names:tc:emergency:cap:1.2"

class CAPClient(slixmpp.ClientXMPP):
    def __init__(self, jid, password, server, port):
        super().__init__(jid, password)
        self._server = server
        self._port = port
        self.add_event_handler("session_start", self.on_start)
        self.add_event_handler("message", self.on_message)

    async def on_start(self, event):
        # No presence needed — receive-only client
        pass

    def on_message(self, msg):
        if msg["type"] != "headline":
            return
        alert = msg.xml.find(f"{{{CAP_NS}}}alert")
        if alert is None:
            return
        identifier  = alert.findtext(f"{{{CAP_NS}}}identifier")
        info        = alert.find(f"{{{CAP_NS}}}info")
        if info is None:
            return
        event_type  = info.findtext(f"{{{CAP_NS}}}event")
        severity    = info.findtext(f"{{{CAP_NS}}}severity")
        urgency     = info.findtext(f"{{{CAP_NS}}}urgency")
        instruction = info.findtext(f"{{{CAP_NS}}}instruction")
        area        = info.find(f"{{{CAP_NS}}}area")
        area_desc   = area.findtext(f"{{{CAP_NS}}}areaDesc") if area is not None else None

        # SAME / UGC geocodes
        geocodes = {}
        if area is not None:
            for gc in area.findall(f"{{{CAP_NS}}}geocode"):
                name = gc.findtext(f"{{{CAP_NS}}}valueName")
                val  = gc.findtext(f"{{{CAP_NS}}}value")
                geocodes.setdefault(name, []).append(val)

        print(f"ALERT [{severity}/{urgency}] {event_type} — {identifier}")
        print(f"  Area: {area_desc}")
        if instruction:
            print(f"  Action: {instruction}")
        if "SAME" in geocodes:
            print(f"  SAME: {', '.join(geocodes['SAME'])}")

async def main():
    client = CAPClient(
        jid="ops-client@eag.example.com",
        password="s3cr3t",
        server="eag.example.com",
        port=5222,
    )
    client.connect((client._server, client._port), use_ssl=False)
    await client.disconnected

asyncio.run(main())
```

### Go

```go
package main

import (
    "encoding/base64"
    "encoding/xml"
    "fmt"
    "net"
    "strings"
    "time"
)

const (
    server   = "eag.example.com:5222"
    username = "ops-client"
    password = "s3cr3t"
    domain   = "eag.example.com"
    resource = "go-client"

    nsStream = "http://etherx.jabber.org/streams"
    nsClient = "jabber:client"
    nsSASL   = "urn:ietf:params:xml:ns:xmpp-sasl"
    nsBind   = "urn:ietf:params:xml:ns:xmpp-bind"
    nsCAP    = "urn:oasis:names:tc:emergency:cap:1.2"
)

type CAPAlert struct {
    XMLName    xml.Name  `xml:"alert"`
    Identifier string    `xml:"identifier"`
    Sender     string    `xml:"sender"`
    Sent       string    `xml:"sent"`
    Status     string    `xml:"status"`
    MsgType    string    `xml:"msgType"`
    Scope      string    `xml:"scope"`
    References string    `xml:"references"`
    Info       []CAPInfo `xml:"info"`
}

type CAPParam struct {
    ValueName string `xml:"valueName"`
    Value     string `xml:"value"`
}

type CAPInfo struct {
    Language     string     `xml:"language"`
    Category     string     `xml:"category"`
    Event        string     `xml:"event"`
    ResponseType string     `xml:"responseType"`
    Urgency      string     `xml:"urgency"`
    Severity     string     `xml:"severity"`
    Certainty    string     `xml:"certainty"`
    Effective    string     `xml:"effective"`
    Onset        string     `xml:"onset"`
    Expires      string     `xml:"expires"`
    SenderName   string     `xml:"senderName"`
    Headline     string     `xml:"headline"`
    Description  string     `xml:"description"`
    Instruction  string     `xml:"instruction"`
    Parameters   []CAPParam `xml:"parameter"`
    Areas        []struct {
        AreaDesc string     `xml:"areaDesc"`
        Geocodes []CAPParam `xml:"geocode"`
    } `xml:"area"`
}

func saslPlain(username, password string) string {
    raw := "\x00" + username + "\x00" + password
    return base64.StdEncoding.EncodeToString([]byte(raw))
}

func main() {
    conn, _ := net.Dial("tcp", server)
    defer conn.Close()

    send := func(s string) { fmt.Fprint(conn, s) }

    streamOpen := func() {
        fmt.Fprintf(conn,
            "<?xml version='1.0'?><stream:stream to=%q "+
            "xmlns=%q xmlns:stream=%q version='1.0'>",
            domain, nsClient, nsStream)
    }

    dec := xml.NewDecoder(conn)

    skipToStart := func() xml.StartElement {
        for {
            tok, _ := dec.Token()
            if s, ok := tok.(xml.StartElement); ok {
                return s
            }
        }
    }

    // ── Stream open + SASL ──────────────────────────────────────────────────
    streamOpen()
    dec = xml.NewDecoder(conn) // fresh decoder each stream restart

    // Skip stream:stream + features
    skipToStart() // stream:stream
    skipToStart() // stream:features (read and discard)
    dec.Skip()    //nolint:errcheck

    // Authenticate
    send(fmt.Sprintf(`<auth xmlns=%q mechanism="PLAIN">%s</auth>`,
        nsSASL, saslPlain(username, password)))

    skipToStart() // <success/> or <failure/>

    // ── Stream restart + bind ───────────────────────────────────────────────
    dec = xml.NewDecoder(conn)
    streamOpen()

    skipToStart() // stream:stream
    skipToStart() // stream:features
    dec.Skip()    //nolint:errcheck

    send(fmt.Sprintf(
        `<iq type='set' id='bind_1'><bind xmlns=%q><resource>%s</resource></bind></iq>`,
        nsBind, resource))

    skipToStart() // bind result
    dec.Skip()    //nolint:errcheck

    fmt.Println("connected, waiting for alerts...")

    // ── Receive alerts ──────────────────────────────────────────────────────
    for {
        tok, err := dec.Token()
        if err != nil {
            fmt.Println("disconnected:", err)
            return
        }
        start, ok := tok.(xml.StartElement)
        if !ok {
            continue
        }
        if start.Name.Local != "message" {
            dec.Skip() //nolint:errcheck
            continue
        }
        // Find CAP alert child
        type msg struct {
            Alert *CAPAlert `xml:"urn:oasis:names:tc:emergency:cap:1.2 alert"`
        }
        var m msg
        dec.DecodeElement(&m, &start) //nolint:errcheck
        if m.Alert == nil {
            continue
        }
        a := m.Alert
        fmt.Printf("[%s] %s — %s/%s\nArea: %s\nExpires: %s\n\n",
            a.Info.Severity, a.Info.Event, a.Info.Urgency, a.Info.Certainty,
            a.Info.Area.AreaDesc, a.Info.Expires)
    }
}
```

### JavaScript (Node.js using `node-xmpp-client`)

```javascript
const { Client, Stanza } = require('@xmpp/client')
const { xml } = require('@xmpp/xml')

const CAP_NS = 'urn:oasis:names:tc:emergency:cap:1.2'

const client = new Client({
  service: 'xmpp://eag.example.com:5222',
  domain:  'eag.example.com',
  username: 'ops-client',
  password: 's3cr3t',
  resource: 'js-client',
})

client.on('online', () => {
  console.log('connected, waiting for alerts...')
})

client.on('stanza', stanza => {
  if (!stanza.is('message') || stanza.attrs.type !== 'headline') return

  const alertEl = stanza.getChild('alert', CAP_NS)
  if (!alertEl) return

  const identifier  = alertEl.getChildText('identifier')
  const info        = alertEl.getChild('info')
  if (!info) return

  const event       = info.getChildText('event')
  const severity    = info.getChildText('severity')
  const urgency     = info.getChildText('urgency')
  const expires     = info.getChildText('expires')
  const instruction = info.getChildText('instruction')
  const area        = info.getChild('area')
  const areaDesc    = area?.getChildText('areaDesc')

  // SAME / UGC geocodes
  const geocodes = {}
  area?.getChildren('geocode').forEach(gc => {
    const name = gc.getChildText('valueName')
    const val  = gc.getChildText('value')
    if (name) (geocodes[name] = geocodes[name] || []).push(val)
  })

  // Source-specific parameters
  const params = {}
  info.getChildren('parameter').forEach(p => {
    const name = p.getChildText('valueName')
    const val  = p.getChildText('value')
    if (name) (params[name] = params[name] || []).push(val)
  })

  console.log(`[${severity}/${urgency}] ${event}`)
  console.log(`  Area:    ${areaDesc}`)
  if (instruction) console.log(`  Action:  ${instruction}`)
  if (geocodes.SAME) console.log(`  SAME:    ${geocodes.SAME.join(', ')}`)
  console.log(`  Expires: ${expires}`)
  console.log(`  ID:      ${identifier}`)
})

client.on('error', err => console.error(err))
client.start().catch(console.error)
```

---

## Namespace Reference

| Namespace URI | Purpose |
|---|---|
| `jabber:client` | XMPP client stream default namespace |
| `http://etherx.jabber.org/streams` | XMPP stream element namespace (`stream:`) |
| `urn:ietf:params:xml:ns:xmpp-tls` | STARTTLS feature and negotiation |
| `urn:ietf:params:xml:ns:xmpp-sasl` | SASL feature and authentication |
| `urn:ietf:params:xml:ns:xmpp-bind` | Resource binding |
| `urn:ietf:params:xml:ns:xmpp-session` | Session establishment (legacy) |
| `urn:oasis:names:tc:emergency:cap:1.2` | CAP 1.2 alert payload |

---

## Notes and Limitations

- **PLAIN only.** No SCRAM, DIGEST-MD5, or GSSAPI. Use TLS to protect credentials in transit.
- **No offline storage.** Alerts published while a client is disconnected are not queued at the XMPP layer. EAG will re-deliver unforwarded alerts on reconnect for short outages; extended outages may result in missed alerts.
- **Receive-only.** Clients send no alert stanzas. Incoming stanzas from clients (other than stream negotiation) are discarded.
- **One session per username.** A new login with the same credentials terminates the existing session.
- **No roster, presence, or MUC.** EAG implements only the subset of RFC 6120 needed for authenticated, receive-only CAP delivery.
- **No polygon in XMPP payload.** GeoJSON geometry is stored internally but not included in the XMPP stanza. Clients requiring polygon data should query `GET /api/v1/alerts/{id}` on the EAG REST API.
