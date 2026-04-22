import React, { useState, useCallback } from 'react'
import { useNavigate } from 'react-router-dom'
import { api } from '../api.js'
import { StatusBadge, fmtTime, fmtRel, useAutoRefresh, usePeerEvents } from './shared.jsx'

export default function Dashboard() {
  const [stats, setStats]   = useState(null)
  const [status, setStatus] = useState(null)
  const [lastUpdate, setLastUpdate] = useState(null)
  const [error, setError]   = useState(null)
  const navigate = useNavigate()

  const refresh = useCallback(async () => {
    try {
      const [s, st] = await Promise.all([
        api.getAlertStats(),
        api.getStatus(),
      ])
      setStats(s)
      setStatus(st)
      setLastUpdate(new Date())
      setError(null)
    } catch (e) {
      setError(e.message)
    }
  }, [])

  useAutoRefresh(refresh, 30000, false)
  usePeerEvents(refresh)

  const sevOrder = ['Extreme','Severe','Moderate','Minor','Unknown']
  const enabledFeeds = status?.feeds?.filter(f => f.enabled) || []

  return (
    <div>
      <div className="page-header">
        <div className="page-title">Dashboard</div>
        <div className="page-actions">
          {lastUpdate && <span className="last-updated">Updated {fmtTime(lastUpdate)}</span>}
          <button onClick={refresh}>Refresh</button>
        </div>
      </div>

      {error && <div className="error-msg">{error}</div>}

      {stats && (
        <>
          <div className="section-title">Alert Severity</div>
          <div style={{ display:'flex', gap:12, flexWrap:'wrap', marginBottom:20 }}>
            {sevOrder.map(sev => {
              const count = stats.by_severity?.[sev] ?? 0
              const cls = sev==='Extreme'?'sev-extreme':sev==='Severe'?'sev-severe':sev==='Moderate'?'sev-moderate':sev==='Minor'?'sev-minor':'sev-unknown'
              return (
                <div key={sev} className={`stat-card ${cls}`} style={{ cursor:'pointer', minWidth:120 }}
                  onClick={() => navigate(`/alerts?severity=${sev}`)}>
                  <div className="stat-label">{sev}</div>
                  <div className="stat-value" style={{ fontSize:32 }}>{count}</div>
                </div>
              )
            })}
            <div className="stat-card" style={{ minWidth:120 }}>
              <div className="stat-label">Total Active</div>
              <div className="stat-value">{stats.total_active ?? 0}</div>
            </div>
            <div className="stat-card" style={{ minWidth:120 }}>
              <div className="stat-label">Forwarded</div>
              <div className="stat-value" style={{ color:'var(--ok)' }}>{stats.forwarded ?? 0}</div>
            </div>
            <div className="stat-card" style={{ minWidth:140 }}>
              <div className="stat-label">Pending Fwd</div>
              {stats.destinations_configured
                ? <div className="stat-value" style={{ color:'var(--warning)' }}>{stats.pending_forward ?? 0}</div>
                : <div className="stat-value" style={{ color:'var(--muted)', fontSize:22 }}>—
                    <div style={{ fontSize:10, color:'var(--muted)', fontWeight:400, textTransform:'none', fontFamily:'var(--font-ui)', marginTop:2 }}>no destinations</div>
                  </div>
              }
            </div>
          </div>
        </>
      )}

      {status && (
        <>
          <div className="section-title">Feed Sources</div>
          <div className="table-wrap card" style={{ marginBottom:20 }}>
            <table>
              <thead><tr>
                <th>Name</th><th>Type</th><th>Enabled</th>
                <th>Last Polled</th><th>Status</th><th>Alerts</th><th></th>
              </tr></thead>
              <tbody>
                {enabledFeeds.map(f => (
                  <tr key={f.id}>
                    <td style={{ fontFamily:'var(--font-ui)', fontWeight:600 }}>{f.name}</td>
                    <td><span className="badge" style={{ background:'var(--surface2)',color:'var(--muted)',border:'1px solid var(--border)' }}>{f.type || 'nws'}</span></td>
                    <td>{f.enabled ? <span style={{color:'var(--ok)'}}>●</span> : <span style={{color:'var(--muted)'}}>○</span>}</td>
                    <td style={{ fontFamily:'var(--font-mono)', fontSize:12 }}>{f.last_polled ? fmtRel(f.last_polled) : '—'}</td>
                    <td><StatusBadge status={f.last_status || '—'} /></td>
                    <td style={{ fontFamily:'var(--font-mono)' }}>{f.alert_count}</td>
                    <td><button onClick={() => api.pollFeed(f.id).catch(()=>{})}>Poll Now</button></td>
                  </tr>
                ))}
                {!enabledFeeds.length && (
                  <tr>
                    <td colSpan={7} style={{ color:'var(--muted)', textAlign:'center', padding:'16px 0' }}>
                      No enabled feed sources.
                    </td>
                  </tr>
                )}
              </tbody>
            </table>
          </div>

          <div className="section-title">XMPP Server</div>
          <div style={{ display:'flex', gap:12, flexWrap:'wrap', marginBottom:20 }}>
            {[
              { label:'C2S', cfg: status.xmpp_server?.c2s, extra: cfg => cfg?.starttls ? 'STARTTLS' : 'Plain' },
              { label:'C2S TLS', cfg: status.xmpp_server?.c2s_tls, extra: () => 'Direct TLS' },
            ].map(({ label, cfg, extra }) => (
              <div key={label} className="stat-card" style={{ minWidth:140 }}>
                <div className="stat-label">{label} {cfg?.enabled ? `— Port ${cfg.port}` : ''}</div>
                <div className="stat-value" style={{ fontSize:16, marginTop:4 }}>
                  {cfg?.enabled
                    ? <><span style={{color:'var(--ok)'}}>●</span> <span style={{fontSize:12,color:'var(--muted)',fontFamily:'var(--font-ui)'}}>{extra(cfg)}</span></>
                    : <span style={{color:'var(--border)'}}>○ Disabled</span>
                  }
                </div>
              </div>
            ))}
            <div className="stat-card" style={{ minWidth:120 }}>
              <div className="stat-label">Connected Peers</div>
              <div className="stat-value" style={{ color: status.xmpp_server?.peer_count > 0 ? 'var(--ok)' : 'var(--muted)' }}>
                {status.xmpp_server?.peer_count ?? 0}
              </div>
            </div>
          </div>
        </>
      )}

    </div>
  )
}
