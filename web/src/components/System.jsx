import React, { useState, useCallback } from 'react'
import { api } from '../api.js'
import { fmtTime, useAutoRefresh, usePeerEvents } from './shared.jsx'

export default function System() {
  const [status, setStatus]   = useState(null)
  const [loading, setLoading] = useState(false)
  const [error, setError]     = useState(null)
  const [lastUpdate, setLastUpdate] = useState(null)
  const [sweeping, setSweeping] = useState(false)
  const [sweepMsg, setSweepMsg] = useState(null)

  const refresh = useCallback(async () => {
    setLoading(true)
    try {
      const s = await api.getStatus()
      setStatus(s)
      setLastUpdate(new Date())
      setError(null)
    } catch (e) { setError(e.message) }
    finally { setLoading(false) }
  }, [])

  useAutoRefresh(refresh, 30000, false)
  usePeerEvents(refresh)

  const runExpiry = async () => {
    setSweeping(true); setSweepMsg(null)
    try {
      await api.runExpiry()
      setSweepMsg('Expiry sweep triggered.')
      setTimeout(() => setSweepMsg(null), 4000)
    } catch (e) { setSweepMsg('Error: ' + e.message) }
    finally { setSweeping(false) }
  }

  const fmtUptime = (secs) => {
    if (!secs) return '—'
    const d = Math.floor(secs / 86400)
    const h = Math.floor((secs % 86400) / 3600)
    const m = Math.floor((secs % 3600) / 60)
    const s = secs % 60
    return [d&&`${d}d`, h&&`${h}h`, m&&`${m}m`, `${s}s`].filter(Boolean).join(' ')
  }

  return (
    <div>
      <div className="page-header">
        <div className="page-title">System</div>
        <div className="page-actions">
          {lastUpdate && <span className="last-updated">Updated {fmtTime(lastUpdate)}</span>}
          <button onClick={refresh}>Refresh</button>
        </div>
      </div>

      {error && <div className="error-msg">{error}</div>}
      {loading && !status && <div className="loading">Loading…</div>}

      {status && (
        <>
          <div className="section-title">Runtime</div>
          <div className="grid-3" style={{ marginBottom:20 }}>
            <div className="stat-card">
              <div className="stat-label">Version</div>
              <div style={{ fontFamily:'var(--font-mono)', fontSize:20 }}>{status.version}</div>
            </div>
            <div className="stat-card">
              <div className="stat-label">Uptime</div>
              <div style={{ fontFamily:'var(--font-mono)', fontSize:18 }}>{fmtUptime(status.uptime_seconds)}</div>
            </div>
            <div className="stat-card">
              <div className="stat-label">DB Driver</div>
              <div style={{ fontFamily:'var(--font-mono)', fontSize:18, color:'var(--accent)' }}>
                {status.database?.driver}
              </div>
            </div>
          </div>

          <div className="section-title">Database</div>
          <div className="grid-3" style={{ marginBottom:20 }}>
            <div className="stat-card">
              <div className="stat-label">Total Active Alerts</div>
              <div className="stat-value">{status.database?.alert_count?.toLocaleString()}</div>
            </div>
            <div className="stat-card">
              <div className="stat-label">Soft-Deleted (pending hard-delete)</div>
              <div className="stat-value" style={{ color: status.database?.expired_pending_hard_delete > 0 ? 'var(--warning)' : 'var(--text)' }}>
                {status.database?.expired_pending_hard_delete?.toLocaleString()}
              </div>
            </div>
          </div>

          <div className="section-title">Expiry Sweep</div>
          <div className="card" style={{ marginBottom:20, display:'flex', alignItems:'center', gap:16 }}>
            <button className="primary" onClick={runExpiry} disabled={sweeping}>
              {sweeping ? 'Running…' : 'Run Expiry Sweep Now'}
            </button>
            {sweepMsg && <span style={{ fontFamily:'var(--font-mono)', fontSize:13, color: sweepMsg.startsWith('Error') ? 'var(--critical)' : 'var(--ok)' }}>{sweepMsg}</span>}
            <span style={{ fontSize:12, color:'var(--muted)' }}>
              Soft-deletes expired alerts and hard-deletes stale soft-deleted records.
            </span>
          </div>

          <div className="section-title">Feed Sources</div>
          <div className="table-wrap card" style={{ marginBottom:20 }}>
            <table>
              <thead><tr>
                <th>ID</th><th>Name</th><th>Enabled</th>
                <th>Last Polled</th><th>Status</th><th>Alert Count</th>
              </tr></thead>
              <tbody>
                {status.feeds?.map(f => (
                  <tr key={f.id}>
                    <td style={{ fontFamily:'var(--font-mono)', fontSize:12, color:'var(--muted)' }}>{f.id}</td>
                    <td style={{ fontFamily:'var(--font-ui)', fontWeight:600 }}>{f.name}</td>
                    <td>{f.enabled ? <span style={{color:'var(--ok)'}}>Yes</span> : <span style={{color:'var(--muted)'}}>No</span>}</td>
                    <td style={{ fontFamily:'var(--font-mono)', fontSize:12 }}>
                      {f.last_polled ? fmtTime(f.last_polled) : '—'}
                    </td>
                    <td>
                      <span style={{
                        fontFamily:'var(--font-mono)', fontSize:12,
                        color: f.last_status === 'ok' ? 'var(--ok)' : f.last_status?.startsWith('error') ? 'var(--critical)' : 'var(--muted)'
                      }}>{f.last_status || '—'}</span>
                    </td>
                    <td style={{ fontFamily:'var(--font-mono)' }}>{f.alert_count}</td>
                  </tr>
                ))}
                {!status.feeds?.length && <tr><td colSpan={6} style={{color:'var(--muted)'}}>No feeds.</td></tr>}
              </tbody>
            </table>
          </div>

          <div className="section-title">XMPP Server</div>
          <div style={{ display:'flex', gap:12, flexWrap:'wrap', marginBottom:12 }}>
            {[
              { label:'C2S', cfg: status.xmpp_server?.c2s, extra: cfg => cfg?.starttls ? 'STARTTLS' : 'Plain' },
              { label:'C2S TLS', cfg: status.xmpp_server?.c2s_tls, extra: () => 'Direct TLS' },
            ].map(({ label, cfg, extra }) => (
              <div key={label} className="stat-card" style={{ minWidth:160 }}>
                <div className="stat-label">{label}{cfg?.enabled ? ` — Port ${cfg.port}` : ''}</div>
                <div style={{ marginTop:6, fontSize:13, fontFamily:'var(--font-ui)' }}>
                  {cfg?.enabled
                    ? <><span style={{color:'var(--ok)'}}>● Listening</span> <span style={{color:'var(--muted)',fontSize:11}}>{extra(cfg)}</span></>
                    : <span style={{color:'var(--border)'}}>○ Disabled</span>}
                </div>
              </div>
            ))}
            <div className="stat-card" style={{ minWidth:140 }}>
              <div className="stat-label">Connected Peers</div>
              <div className="stat-value" style={{ color: status.xmpp_server?.peer_count > 0 ? 'var(--ok)' : 'var(--muted)' }}>
                {status.xmpp_server?.peer_count ?? 0}
              </div>
            </div>
          </div>

          <div className="table-wrap card">
            <table>
              <thead><tr>
                <th>Status</th><th>Peer Name</th><th>Username</th>
                <th>Remote Address</th><th>Connected At</th>
              </tr></thead>
              <tbody>
                {status.xmpp_server?.connected?.map(c => (
                  <tr key={c.id}>
                    <td><span style={{ display:'inline-block', width:8, height:8, borderRadius:'50%', background:'var(--ok)', boxShadow:'0 0 5px var(--ok)' }} /></td>
                    <td style={{ fontFamily:'var(--font-ui)', fontWeight:600 }}>{c.name}</td>
                    <td style={{ fontFamily:'var(--font-mono)', fontSize:12 }}>{c.username}</td>
                    <td style={{ fontFamily:'var(--font-mono)', fontSize:12 }}>{c.remote_addr}</td>
                    <td style={{ fontFamily:'var(--font-mono)', fontSize:12 }}>{c.connected_at ? new Date(c.connected_at).toLocaleTimeString() : '—'}</td>
                  </tr>
                ))}
                {!status.xmpp_server?.connected?.length && (
                  <tr><td colSpan={5} style={{color:'var(--muted)',textAlign:'center',padding:'16px 0'}}>No peers connected.</td></tr>
                )}
              </tbody>
            </table>
          </div>
        </>
      )}
    </div>
  )
}
