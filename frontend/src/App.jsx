import {useCallback, useEffect, useLayoutEffect, useRef, useState} from 'react'
import {createPortal} from 'react-dom'
import './App.css'
import {
  CheckUpdate,
  DeleteLocalAccount,
  GetLocalAccounts,
  GetStatus,
  ImportLocalAccountsCSV,
  ExportLocalAccountsCSV,
  OpenReleaseURL,
  PickEQDirectory,
  SaveLocalAccount,
  SaveSource,
  PreviewSourceJSON,
  SetActiveSource,
  SetConnectionMode,
  SetEQDirectory,
  SetListenPort,
  DeleteSource,
  ShareLocalAccount,
  UnshareLocalAccount,
} from '../wailsjs/go/main/App'

const TABS = [
  {id: 'proxy', label: 'Connections'},
  {id: 'accounts', label: 'Accounts'},
  {id: 'eq', label: 'EverQuest'},
  {id: 'settings', label: 'Settings'},
]

const ACCOUNT_SUBTABS_BASE = [
  {id: 'sso', label: 'SSO'},
  {id: 'local', label: 'Local'},
]

function formatWhen(iso) {
  if (!iso) return '—'
  const t = Date.parse(iso)
  if (Number.isNaN(t)) return iso
  const diff = Date.now() - t
  const sec = Math.round(diff / 1000)
  if (sec < 60) return 'just now'
  const min = Math.round(sec / 60)
  if (min < 60) return `${min}m ago`
  const hr = Math.round(min / 60)
  if (hr < 48) return `${hr}h ago`
  const d = Math.round(hr / 24)
  if (d < 14) return `${d}d ago`
  try {
    return new Date(t).toLocaleString()
  } catch {
    return iso
  }
}

const THEME_KEY = 'alfred-identity-theme'

function readStoredTheme() {
  try {
    const t = localStorage.getItem(THEME_KEY)
    return t === 'dark' ? 'dark' : 'light'
  } catch {
    return 'light'
  }
}

function applyTheme(theme) {
  const t = theme === 'dark' ? 'dark' : 'light'
  document.documentElement.setAttribute('data-theme', t)
  try {
    localStorage.setItem(THEME_KEY, t)
  } catch {
    /* ignore */
  }
  return t
}

const CONNECTION_MODES = [
  {id: 'login_sso', label: 'Login w/ SSO', hint: 'UDP proxy + guild SSO accounts'},
  {id: 'login_only', label: 'Login Only', hint: 'UDP proxy with local accounts only (no SSO)'},
  {id: 'disabled', label: 'Disabled', hint: 'Proxy and SSO stopped'},
]

function ActionMenu({disabled, label = 'More actions', items}) {
  const [open, setOpen] = useState(false)
  const rootRef = useRef(null)
  const menuRef = useRef(null)
  const [menuPos, setMenuPos] = useState(null)

  const updateMenuPos = useCallback(() => {
    const el = rootRef.current
    if (!el) return
    const r = el.getBoundingClientRect()
    const width = 180
    let left = r.right - width
    if (left < 8) left = 8
    let top = r.bottom + 4
    const estHeight = 8 + items.length * 36
    if (top + estHeight > window.innerHeight - 8) {
      top = Math.max(8, r.top - estHeight - 4)
    }
    setMenuPos({top, left, width})
  }, [items.length])

  useLayoutEffect(() => {
    if (!open) {
      setMenuPos(null)
      return undefined
    }
    updateMenuPos()
    window.addEventListener('resize', updateMenuPos)
    window.addEventListener('scroll', updateMenuPos, true)
    return () => {
      window.removeEventListener('resize', updateMenuPos)
      window.removeEventListener('scroll', updateMenuPos, true)
    }
  }, [open, updateMenuPos])

  useEffect(() => {
    if (!open) return undefined
    function onDocDown(e) {
      if (rootRef.current?.contains(e.target)) return
      if (menuRef.current?.contains(e.target)) return
      setOpen(false)
    }
    function onKey(e) {
      if (e.key === 'Escape') setOpen(false)
    }
    document.addEventListener('mousedown', onDocDown)
    document.addEventListener('keydown', onKey)
    return () => {
      document.removeEventListener('mousedown', onDocDown)
      document.removeEventListener('keydown', onKey)
    }
  }, [open])

  return (
    <div className={`action-menu ${open ? 'open' : ''}`} ref={rootRef}>
      <button
        type="button"
        className="secondary action-menu-trigger"
        disabled={disabled}
        aria-haspopup="menu"
        aria-expanded={open}
        aria-label={label}
        title={label}
        onClick={() => setOpen((o) => !o)}
      >
        <span className="action-menu-caret" aria-hidden="true"/>
      </button>
      {open && menuPos && createPortal(
        <ul
          ref={menuRef}
          className="action-menu-list"
          role="menu"
          aria-label={label}
          style={{
            position: 'fixed',
            top: menuPos.top,
            left: menuPos.left,
            width: menuPos.width,
          }}
        >
          {items.map((item) => (
            <li key={item.id} role="presentation">
              <button
                type="button"
                role="menuitem"
                className="action-menu-item"
                disabled={disabled || item.disabled}
                onClick={() => {
                  setOpen(false)
                  item.onClick?.()
                }}
              >
                {item.label}
              </button>
            </li>
          ))}
        </ul>,
        document.body,
      )}
    </div>
  )
}

function ModeDropdown({value, disabled, onChange, compact = false, id}) {
  const [open, setOpen] = useState(false)
  const rootRef = useRef(null)
  const menuRef = useRef(null)
  const [menuPos, setMenuPos] = useState(null)
  const active = CONNECTION_MODES.find((m) => m.id === value) || CONNECTION_MODES[2]

  const updateMenuPos = useCallback(() => {
    const el = rootRef.current
    if (!el) return
    const r = el.getBoundingClientRect()
    const width = compact ? Math.max(r.width, 280) : Math.max(r.width, 320)
    let left = r.left
    if (left + width > window.innerWidth - 8) {
      left = Math.max(8, window.innerWidth - width - 8)
    }
    let top = r.bottom + 6
    const estHeight = 220
    if (top + estHeight > window.innerHeight - 8) {
      top = Math.max(8, r.top - estHeight - 6)
    }
    setMenuPos({top, left, width})
  }, [compact])

  useLayoutEffect(() => {
    if (!open) {
      setMenuPos(null)
      return undefined
    }
    updateMenuPos()
    window.addEventListener('resize', updateMenuPos)
    window.addEventListener('scroll', updateMenuPos, true)
    return () => {
      window.removeEventListener('resize', updateMenuPos)
      window.removeEventListener('scroll', updateMenuPos, true)
    }
  }, [open, updateMenuPos])

  useEffect(() => {
    if (!open) return undefined
    function onDocDown(e) {
      if (rootRef.current?.contains(e.target)) return
      if (menuRef.current?.contains(e.target)) return
      setOpen(false)
    }
    function onKey(e) {
      if (e.key === 'Escape') setOpen(false)
    }
    document.addEventListener('mousedown', onDocDown)
    document.addEventListener('keydown', onKey)
    return () => {
      document.removeEventListener('mousedown', onDocDown)
      document.removeEventListener('keydown', onKey)
    }
  }, [open])

  return (
    <div
      className={`mode-dropdown ${open ? 'open' : ''} ${compact ? 'compact' : ''}`}
      ref={rootRef}
    >
      <button
        type="button"
        id={id}
        className="mode-trigger"
        disabled={disabled}
        aria-haspopup="listbox"
        aria-expanded={open}
        aria-label="Connection mode"
        onClick={() => setOpen((o) => !o)}
      >
        <span className="mode-trigger-text">
          <span className="mode-label">{active.label}</span>
          {!compact && <span className="mode-hint">{active.hint}</span>}
        </span>
        <span className="mode-caret" aria-hidden="true"/>
      </button>
      {open && menuPos && createPortal(
        <ul
          ref={menuRef}
          className="mode-menu"
          role="listbox"
          aria-label="Connection mode"
          style={{
            position: 'fixed',
            top: menuPos.top,
            left: menuPos.left,
            width: menuPos.width,
          }}
        >
          {CONNECTION_MODES.map((m) => {
            const selected = m.id === active.id
            return (
              <li key={m.id} role="presentation">
                <button
                  type="button"
                  role="option"
                  aria-selected={selected}
                  className={selected ? 'mode-option active' : 'mode-option'}
                  disabled={disabled}
                  onClick={() => {
                    setOpen(false)
                    if (m.id !== active.id) onChange(m.id)
                  }}
                >
                  <span className="mode-option-top">
                    <span className="mode-label">{m.label}</span>
                    {selected && <span className="mode-check" aria-hidden="true">✓</span>}
                  </span>
                  <span className="mode-hint">{m.hint}</span>
                </button>
              </li>
            )
          })}
        </ul>,
        document.body,
      )}
    </div>
  )
}

function portFromListen(listen) {
  if (!listen || !listen.includes(':')) return '6998'
  const parts = listen.split(':')
  return parts[parts.length - 1] || '6998'
}

function blankLocalForm() {
  return {name: '', password: '', aliases: '', editing: false}
}

function roleNameById(roles, id) {
  if (!id) return ''
  const hit = (roles || []).find((r) => r.id === id)
  return hit?.name || id
}

function SortTh({sortKey, sort, onSort, children, className = ''}) {
  const active = sort?.key === sortKey
  const mark = active ? (sort.dir > 0 ? ' ▲' : ' ▼') : ''
  return (
    <th
      className={`sortable ${className}`.trim()}
      onClick={() => onSort(sortKey)}
      aria-sort={active ? (sort.dir > 0 ? 'ascending' : 'descending') : 'none'}
    >
      {children}{mark}
    </th>
  )
}

function useTableSort(defaultKey, defaultDir = 1) {
  const [sort, setSort] = useState({key: defaultKey, dir: defaultDir})
  const onSort = useCallback((key) => {
    setSort((s) => (s.key === key ? {key, dir: -s.dir} : {key, dir: 1}))
  }, [])
  const sorted = useCallback((rows, getters) => {
    const get = getters[sort.key]
    if (!get || !rows?.length) return rows || []
    const dir = sort.dir || 1
    return [...rows].sort((a, b) => {
      const va = get(a)
      const vb = get(b)
      if (typeof va === 'number' && typeof vb === 'number') return (va - vb) * dir
      return String(va ?? '').localeCompare(String(vb ?? ''), undefined, {sensitivity: 'base', numeric: true}) * dir
    })
  }, [sort])
  return {sort, onSort, sorted}
}

export default function App() {
  const [tab, setTab] = useState('proxy')
  const [accountSub, setAccountSub] = useState('sso')
  const [status, setStatus] = useState(null)
  const [localAccounts, setLocalAccounts] = useState([])
  const [localForm, setLocalForm] = useState(blankLocalForm)
  const [accountModal, setAccountModal] = useState(false)
  const [showLocalPassword, setShowLocalPassword] = useState(false)
  const [shareModal, setShareModal] = useState(false)
  const [shareForm, setShareForm] = useState({name: '', userIds: [], shared: false})
  const [sourcesModal, setSourcesModal] = useState(false)
  const [sourceForm, setSourceForm] = useState(null) // null = list view; object = edit/add
  const [sourceJSON, setSourceJSON] = useState('')
  const [eqDir, setEqDir] = useState('')
  const [listenPort, setListenPort] = useState('6998')
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState('')
  const [update, setUpdate] = useState(null)
  const [theme, setTheme] = useState(() => readStoredTheme())

  useEffect(() => {
    applyTheme(theme)
  }, [theme])

  const refreshLocal = useCallback(async () => {
    try {
      const accs = await GetLocalAccounts()
      setLocalAccounts(accs || [])
    } catch (e) {
      setError(String(e))
    }
  }, [])

  const refresh = useCallback(async () => {
    try {
      const s = await GetStatus()
      setStatus(s)
      if (s?.eq_directory) setEqDir(s.eq_directory)
      if (s?.listen) setListenPort(portFromListen(s.listen))
      setError('')
    } catch (e) {
      setError(String(e))
    }
  }, [])

  useEffect(() => {
    refresh()
    refreshLocal()
    const id = setInterval(() => {
      refresh()
      // Keep share "in use" / last-login indicators fresh while SSO is up.
      refreshLocal()
    }, 4000)
    CheckUpdate().then((u) => {
      if (u?.update_available) setUpdate(u)
    }).catch(() => {})
    return () => clearInterval(id)
  }, [refresh, refreshLocal])

  async function run(fn) {
    setBusy(true)
    try {
      await fn()
      await refresh()
      await refreshLocal()
    } catch (e) {
      setError(String(e))
    } finally {
      setBusy(false)
    }
  }

  function selectLocal(acc) {
    setLocalForm({
      name: acc.name || '',
      password: acc.password || '',
      aliases: (acc.aliases || []).join(', '),
      editing: true,
    })
    setShowLocalPassword(false)
    setAccountModal(true)
  }

  function startNewLocal() {
    setLocalForm(blankLocalForm())
    setShowLocalPassword(false)
    setAccountModal(true)
  }

  function closeLocalModal() {
    setAccountModal(false)
    setShowLocalPassword(false)
    setLocalForm(blankLocalForm())
  }

  function openShareModal(acc) {
    setShareForm({
      name: acc.name,
      userIds: [...(acc.shared_user_ids || [])],
      shared: !!acc.shared,
    })
    setShareModal(true)
  }

  function toggleShareUser(userId) {
    setShareForm((f) => {
      const has = f.userIds.includes(userId)
      return {
        ...f,
        userIds: has ? f.userIds.filter((id) => id !== userId) : [...f.userIds, userId],
      }
    })
  }

  function openSourcesModal() {
    setSourceForm(null)
    setSourcesModal(true)
  }

  function startNewSource() {
    setSourceForm({id: '', name: '', host: '', token: '', notes: '', editing: false, has_token: false})
  }

  function startEditSource(src) {
    setSourceForm({
      id: src.id || '',
      name: src.name || '',
      host: src.host || '',
      token: '',
      notes: src.notes || '',
      editing: true,
      has_token: !!src.has_token,
    })
  }

  function closeSourcesModal() {
    setSourcesModal(false)
    setSourceForm(null)
  }

  async function importSourceFromJSON(raw) {
    const json = String(raw || '').trim()
    if (!json) throw new Error('Paste source JSON from Discord /alfred-identity-sso get')
    const list = await PreviewSourceJSON(json)
    const items = Array.isArray(list) ? list : []
    if (items.length === 0) {
      throw new Error('No source found in JSON')
    }
    const first = items[0] || {}
    setSourceJSON('')
    setSourcesModal(true)
    setSourceForm({
      id: '',
      name: first.name || '',
      host: first.host || '',
      token: first.token || '',
      notes: first.notes || '',
      editing: false,
      has_token: false,
      fromImport: true,
      importExtra: Math.max(0, items.length - 1),
      importQueue: items.slice(1),
    })
  }

  const sources = status?.sources || []
  const ssoConnected = !!status?.sso_connected
  const accountSubtabs = [
    ...ACCOUNT_SUBTABS_BASE,
    ...(ssoConnected ? [{id: 'shared', label: 'Shared'}] : []),
  ]
  const adminUsers = status?.sso_admin_users || []
  const adminRoles = status?.sso_admin_roles || []
  const directory = status?.sso_directory || []
  const myUserId = status?.sso_user_id || 0
  const shareTargets = directory.filter((u) => u.id !== myUserId)

  function discordUserFromID(id) {
    const fromDir = directory.find((u) => u.id === id)
    if (fromDir) return fromDir
    const fromAdmin = adminUsers.find((u) => u.id === id)
    if (fromAdmin) {
      return {id: fromAdmin.id, discord_id: fromAdmin.discord_id, display_name: fromAdmin.display_name}
    }
    return {id, discord_id: '', display_name: `User #${id}`}
  }

  // Discord users you've shared local accounts with → accounts list
  const outgoingSharesByUser = (() => {
    const map = new Map()
    for (const acc of localAccounts) {
      if (!acc.shared) continue
      for (const uid of acc.shared_user_ids || []) {
        if (!map.has(uid)) map.set(uid, [])
        map.get(uid).push(acc.name)
      }
    }
    return [...map.entries()].map(([uid, names]) => ({
      user: discordUserFromID(uid),
      accounts: names.sort((a, b) => a.localeCompare(b)),
    })).sort((a, b) =>
      String(a.user.display_name || a.user.discord_id).localeCompare(
        String(b.user.display_name || b.user.discord_id),
      ))
  })()

  // Restricted SSO accounts shared with you (owned by someone else)
  const incomingSharedAccounts = (status?.sso_accounts || [])
    .filter((a) => a.restricted && a.owner_user_id && a.owner_user_id !== myUserId)
    .map((a) => ({
      account: a,
      owner: discordUserFromID(a.owner_user_id),
    }))
    .sort((a, b) => String(a.account.username || '').localeCompare(String(b.account.username || '')))

  const ssoSort = useTableSort('username')
  const localSort = useTableSort('name')
  const outgoingSort = useTableSort('name')
  const incomingSort = useTableSort('account')
  const sourcesSort = useTableSort('name')

  const sortedSSOAccounts = ssoSort.sorted(status?.sso_accounts || [], {
    username: (a) => a.username || '',
    aliases: (a) => (a.aliases || []).join(', '),
    tags: (a) => (a.tags || []).join(', '),
    characters: (a) => (a.characters || []).join(', '),
    access: (a) => {
      if (a.disabled) return 'disabled'
      if (a.restricted) return 'private share'
      const parts = []
      const roleIDs = (a.required_role_ids && a.required_role_ids.length)
        ? a.required_role_ids
        : (a.required_role_id ? [a.required_role_id] : [])
      for (const rid of roleIDs) parts.push(roleNameById(adminRoles, rid))
      const userIDs = (a.required_user_ids && a.required_user_ids.length)
        ? a.required_user_ids
        : (a.required_user_id ? [a.required_user_id] : [])
      for (const uid of userIDs) {
        const u = discordUserFromID(uid)
        parts.push(u.display_name || u.discord_id || '')
      }
      for (const gid of a.group_ids || []) {
        const g = (status?.sso_groups || []).find((x) => x.id === gid)
        parts.push(g?.name || `group #${gid}`)
      }
      return parts.length ? parts.join(', ') : 'all'
    },
    logged: (a) => {
      const fromDaemon = (status?.sso_online || []).find((o) => o.account_id === a.id)
      return fromDaemon?.character_name || ''
    },
  })
  const sortedLocalAccounts = localSort.sorted(localAccounts, {
    name: (a) => a.name || '',
    aliases: (a) => (a.aliases || []).join(', '),
    shared: (a) => (a.shared ? (a.shared_user_ids || []).length + 1 : 0),
  })
  const sortedOutgoing = outgoingSort.sorted(outgoingSharesByUser, {
    name: (r) => r.user.display_name || r.user.discord_id || '',
    discord: (r) => r.user.discord_id || '',
    accounts: (r) => (r.accounts || []).join(', '),
  })
  const sortedIncoming = incomingSort.sorted(incomingSharedAccounts, {
    account: (r) => r.account.username || '',
    owner: (r) => r.owner.display_name || r.owner.discord_id || '',
    discord: (r) => r.owner.discord_id || '',
  })
  const sortedSources = sourcesSort.sorted(sources, {
    name: (s) => s.name || s.id || '',
    host: (s) => s.host || '',
    token: (s) => (s.has_token ? 'saved' : 'missing'),
    status: (s) => {
      const active = s.id === status?.active_source
      const connected = active && status?.sso_connected
      return connected ? 'connected' : active ? 'active' : ''
    },
  })

  const activeMode =
    CONNECTION_MODES.find((m) => m.id === status?.connection_mode) || CONNECTION_MODES[2]
  const activeSourceName =
    sources.find((x) => x.id === status?.active_source)?.name
    || status?.active_source
    || 'no source'
  const onlineCount = status?.online?.length || 0

  useEffect(() => {
    if (!ssoConnected && accountSub === 'shared') {
      setAccountSub('sso')
    }
  }, [accountSub, ssoConnected])

  return (
    <div className="shell">
      <div className="top">
        <nav className="tabs" role="tablist">
          {TABS.map((t) => (
            <button
              key={t.id}
              type="button"
              role="tab"
              aria-selected={tab === t.id}
              className={tab === t.id ? 'tab active' : 'tab'}
              onClick={() => setTab(t.id)}
            >
              {t.label}
            </button>
          ))}
        </nav>
      </div>

      {(update?.update_available || error) && (
        <div className="banners">
          {update?.update_available && (
            <div className="banner">
              Update available: <b>{update.latest}</b>{' '}
              <button className="secondary" type="button" onClick={() => OpenReleaseURL(update.release_url)}>
                Open release
              </button>
            </div>
          )}
          {error && <div className="banner err">{error}</div>}
        </div>
      )}

      <main>
        {tab === 'proxy' && (
          <section className="panel">
            <div className="panel-scroll">
              <h2>Connection mode</h2>
              <p className="hint">
                Controls the UDP login proxy and whether the active SSO source is used.
              </p>
              <label htmlFor="connection-mode">Mode</label>
              <ModeDropdown
                id="connection-mode"
                value={activeMode.id}
                disabled={busy}
                onChange={(next) => run(async () => {
                  await SetConnectionMode(next)
                })}
              />

              <div className="row status-head">
                <h2 className="sub flush">SSO sources</h2>
                <button type="button" className="secondary" disabled={busy} onClick={openSourcesModal}>
                  Manage sources…
                </button>
              </div>
              <p className="hint">
                Choose which guild daemon to use. Account lists on the SSO tab follow the active
                source. Local accounts are unchanged.
              </p>
              {sortedSources.length === 0 ? (
                <div className="source-dropzone empty-cta">
                  <h3>Add your first SSO source</h3>
                  <p>
                    Run <code>/alfred-identity-sso get</code> in Discord, copy the Alfred Identity
                    source JSON, paste it below, then set Connection mode to Login w/ SSO.
                  </p>
                  <div className="source-json-row">
                    <textarea
                      className="mono"
                      value={sourceJSON}
                      onChange={(e) => setSourceJSON(e.target.value)}
                      placeholder={'{\n  "name": "Guild",\n  "host": "127.0.0.1:8181",\n  "token": "..."\n}'}
                      disabled={busy}
                      rows={6}
                    />
                    <button
                      type="button"
                      disabled={busy || !sourceJSON.trim()}
                      onClick={() => run(() => importSourceFromJSON(sourceJSON))}
                    >
                      Add from JSON
                    </button>
                  </div>
                  <div className="source-dropzone-actions">
                    <button type="button" className="secondary" disabled={busy} onClick={() => {
                      openSourcesModal()
                      startNewSource()
                    }}>
                      Enter details manually
                    </button>
                  </div>
                </div>
              ) : (
                <>
                  <div className="source-json-row compact">
                    <textarea
                      className="mono"
                      value={sourceJSON}
                      onChange={(e) => setSourceJSON(e.target.value)}
                      placeholder="Paste another source JSON from Discord…"
                      disabled={busy}
                      rows={4}
                    />
                    <button
                      type="button"
                      className="secondary"
                      disabled={busy || !sourceJSON.trim()}
                      onClick={() => run(() => importSourceFromJSON(sourceJSON))}
                    >
                      Add from JSON
                    </button>
                  </div>
                  <div className="table-wrap">
                    <table className="data-table">
                      <thead>
                        <tr>
                          <SortTh sortKey="name" sort={sourcesSort.sort} onSort={sourcesSort.onSort}>Name</SortTh>
                          <SortTh sortKey="host" sort={sourcesSort.sort} onSort={sourcesSort.onSort}>Host</SortTh>
                          <SortTh sortKey="status" sort={sourcesSort.sort} onSort={sourcesSort.onSort}>Status</SortTh>
                          <th className="col-actions">Active</th>
                        </tr>
                      </thead>
                      <tbody>
                        {sortedSources.map((src) => {
                          const active = src.id === status?.active_source
                          const connected = active && status?.sso_connected
                          return (
                            <tr key={src.id} className={active ? 'row-selected' : ''}>
                              <td>{src.name || src.id}</td>
                              <td className="mono">{src.host || '—'}</td>
                              <td>
                                {connected ? 'connected' : active ? 'active' : '—'}
                              </td>
                              <td className="col-actions">
                                {active ? (
                                  <span className="badge live">active</span>
                                ) : (
                                  <button
                                    type="button"
                                    className="secondary"
                                    disabled={busy}
                                    onClick={() => run(() => SetActiveSource(src.id))}
                                  >
                                    Use this source
                                  </button>
                                )}
                              </td>
                            </tr>
                          )
                        })}
                      </tbody>
                    </table>
                  </div>
                </>
              )}
            </div>
          </section>
        )}

        {tab === 'accounts' && (
          <section className="panel">
            <nav className="subtabs" role="tablist" aria-label="Account type">
              {accountSubtabs.map((t) => (
                <button
                  key={t.id}
                  type="button"
                  role="tab"
                  aria-selected={accountSub === t.id}
                  className={accountSub === t.id ? 'subtab active' : 'subtab'}
                  onClick={() => setAccountSub(t.id)}
                >
                  {t.label}
                </button>
              ))}
            </nav>

            <div className="panel-scroll">
              {accountSub === 'sso' && (
                <>
                  <p className="hint">
                    SSO accounts from the active source (view only). Manage accounts, aliases, tags, and
                    characters in the web admin. Login cycling uses shared <b>tags</b>; aliases are unique
                    to one account; character names log into that account when typed at the EQ login screen.
                  </p>
                  {!status?.sso_connected ? (
                    <p className="empty">Connect with Login w/ SSO on the Connections tab to see guild accounts.</p>
                  ) : (
                    <>
                      <div className="row status-head">
                        <h2>Accounts</h2>
                      </div>
                      {(status.sso_accounts?.length || 0) === 0 ? (
                        <p className="empty">No accounts available yet. Admins manage them in the web admin.</p>
                      ) : (
                        <div className="table-wrap">
                          <table className="data-table">
                            <thead>
                              <tr>
                                <SortTh sortKey="username" sort={ssoSort.sort} onSort={ssoSort.onSort}>Account</SortTh>
                                <SortTh sortKey="aliases" sort={ssoSort.sort} onSort={ssoSort.onSort}>Aliases</SortTh>
                                <SortTh sortKey="tags" sort={ssoSort.sort} onSort={ssoSort.onSort}>Tags</SortTh>
                                <SortTh sortKey="characters" sort={ssoSort.sort} onSort={ssoSort.onSort}>Characters</SortTh>
                                <SortTh sortKey="access" sort={ssoSort.sort} onSort={ssoSort.onSort}>Access</SortTh>
                                <SortTh sortKey="logged" sort={ssoSort.sort} onSort={ssoSort.onSort}>Logged in</SortTh>
                              </tr>
                            </thead>
                            <tbody>
                              {sortedSSOAccounts.map((a) => {
                                const fromDaemon = (status.sso_online || []).find((o) => o.account_id === a.id)
                                let loggedIn = fromDaemon?.character_name || ''
                                if (!loggedIn) {
                                  const localOnline = status.online || []
                                  for (const ch of a.characters || []) {
                                    if (localOnline.some((n) => n.toLowerCase() === ch.toLowerCase())) {
                                      loggedIn = ch
                                      break
                                    }
                                  }
                                }
                                const accessParts = []
                                if (a.disabled) {
                                  accessParts.push('disabled')
                                } else if (a.restricted) {
                                  accessParts.push('private share')
                                } else {
                                  const roleIDs = (a.required_role_ids && a.required_role_ids.length)
                                    ? a.required_role_ids
                                    : (a.required_role_id ? [a.required_role_id] : [])
                                  for (const rid of roleIDs) {
                                    accessParts.push(roleNameById(adminRoles, rid))
                                  }
                                  const userIDs = (a.required_user_ids && a.required_user_ids.length)
                                    ? a.required_user_ids
                                    : (a.required_user_id ? [a.required_user_id] : [])
                                  for (const uid of userIDs) {
                                    const u = discordUserFromID(uid)
                                    accessParts.push(u.display_name || u.discord_id || `#${uid}`)
                                  }
                                  for (const gid of a.group_ids || []) {
                                    const g = (status?.sso_groups || []).find((x) => x.id === gid)
                                    accessParts.push(g?.name || `group #${gid}`)
                                  }
                                  if (!accessParts.length) accessParts.push('all')
                                }
                                const access = accessParts.join(', ')
                                const aliases = (a.aliases || []).filter(
                                  (al) => !a.username || al.toLowerCase() !== a.username.toLowerCase(),
                                )
                                const tags = a.tags || []
                                const chars = a.characters || []
                                return (
                                  <tr key={a.id} className={loggedIn ? 'row-selected' : ''}>
                                    <td className="mono">{a.username || '—'}</td>
                                    <td>{aliases.length ? aliases.join(', ') : '—'}</td>
                                    <td>{tags.length ? tags.join(', ') : '—'}</td>
                                    <td>{chars.length ? chars.join(', ') : '—'}</td>
                                    <td title={(a.required_role_ids || []).join(', ') || a.required_role_id || ''}>{access}</td>
                                    <td>{loggedIn ? <span className="chip online">{loggedIn}</span> : '—'}</td>
                                  </tr>
                                )
                              })}
                            </tbody>
                          </table>
                        </div>
                      )}
                    </>
                  )}
                </>
              )}

              {accountSub === 'local' && (
                <>
                  <p className="hint">
                    Personal accounts on this machine (CSV). Checked before SSO when the typed name matches.
                    With SSO connected, you can share an account to selected Discord users (copies credentials
                    to the daemon as a private share).
                  </p>

                  <div className="row status-head">
                    <h2>Accounts</h2>
                    <div className="btn-split">
                      <button type="button" className="secondary" disabled={busy} onClick={startNewLocal}>
                        Add account
                      </button>
                      <ActionMenu
                        disabled={busy}
                        label="More account actions"
                        items={[
                          {
                            id: 'import-csv',
                            label: 'Import CSV…',
                            onClick: () => run(() => ImportLocalAccountsCSV()),
                          },
                          {
                            id: 'export-csv',
                            label: 'Export CSV…',
                            onClick: () => run(() => ExportLocalAccountsCSV()),
                          },
                        ]}
                      />
                    </div>
                  </div>

                  <div className="table-wrap">
                    <table className="data-table">
                      <thead>
                        <tr>
                          <SortTh sortKey="name" sort={localSort.sort} onSort={localSort.onSort}>Account</SortTh>
                          <SortTh sortKey="aliases" sort={localSort.sort} onSort={localSort.onSort}>Aliases</SortTh>
                          <SortTh sortKey="shared" sort={localSort.sort} onSort={localSort.onSort}>Shared with</SortTh>
                          <th className="col-actions">Actions</th>
                        </tr>
                      </thead>
                      <tbody>
                        {sortedLocalAccounts.length === 0 ? (
                          <tr>
                            <td colSpan={4} className="empty-cell">No local accounts yet.</td>
                          </tr>
                        ) : (
                          sortedLocalAccounts.map((acc) => {
                            const shareCount = (acc.shared_user_ids || []).length
                            const canOpenShare = !!status?.sso_connected && (shareTargets.length > 0 || !!acc.shared)
                            const shareTitle = !status?.sso_connected
                              ? 'Connect with Login w/ SSO to share'
                              : (shareTargets.length === 0 && !acc.shared
                                ? 'No other SSO users to share with yet'
                                : undefined)
                            // Prefer live fields from GetLocalAccounts; fall back to status.share_activity.
                            let inUseOther = !!acc.in_use_other
                            let inUseBy = acc.in_use_by || ''
                            let lastOther = !!acc.last_login_other
                            let lastBy = acc.last_login_by || ''
                            let lastAt = acc.last_login_at || ''
                            const sid = acc.shared_sso_account_id
                            if (sid && status?.share_activity) {
                              const on = (status.share_activity.online || []).find(
                                (o) => o.account_id === sid && !o.actor_is_owner,
                              )
                              if (on) {
                                inUseOther = true
                                inUseBy = on.user_display_name || on.user_discord_id || 'someone'
                              }
                              const login = (status.share_activity.logins || []).find(
                                (e) => e.account_id === sid && !e.actor_is_owner,
                              )
                              if (login) {
                                lastOther = true
                                lastBy = login.actor_name || login.actor_discord_id || 'someone'
                                lastAt = login.created_at || lastAt
                              }
                            }
                            return (
                              <tr key={acc.name} className={inUseOther ? 'row-in-use-other' : undefined}>
                                <td className="mono">
                                  <div className="local-acct-cell">
                                    <span>{acc.name}</span>
                                    {inUseOther ? (
                                      <span className="pill pill-in-use" title={`In use by ${inUseBy}`}>
                                        In use · {inUseBy}
                                      </span>
                                    ) : null}
                                  </div>
                                </td>
                                <td>{(acc.aliases || []).length ? acc.aliases.join(', ') : '—'}</td>
                                <td>
                                  {acc.shared ? (
                                    <div className="share-meta">
                                      <div>
                                        {shareCount
                                          ? `${shareCount} user${shareCount === 1 ? '' : 's'}`
                                          : 'owner only'}
                                      </div>
                                      {inUseOther ? (
                                        <div className="share-activity in-use">
                                          In use now · {inUseBy}
                                        </div>
                                      ) : acc.in_use ? (
                                        <div className="share-activity">
                                          In use{inUseBy ? ` · ${inUseBy}` : ''}
                                        </div>
                                      ) : null}
                                      {lastOther || lastBy ? (
                                        <div className={`share-activity${lastOther ? ' other' : ''}`}>
                                          Last login · {lastBy || '—'}
                                          {lastAt ? ` · ${formatWhen(lastAt)}` : ''}
                                        </div>
                                      ) : null}
                                    </div>
                                  ) : '—'}
                                </td>
                                <td className="col-actions">
                                  <button type="button" className="secondary" disabled={busy} onClick={() => selectLocal(acc)}>
                                    Edit
                                  </button>
                                  <button
                                    type="button"
                                    className="secondary"
                                    disabled={busy || !canOpenShare}
                                    title={shareTitle}
                                    onClick={() => openShareModal(acc)}
                                  >
                                    Share
                                  </button>
                                  <button
                                    type="button"
                                    className="secondary"
                                    disabled={busy}
                                    onClick={() => run(async () => {
                                      await DeleteLocalAccount(acc.name)
                                      if (localForm.name === acc.name) closeLocalModal()
                                    })}
                                  >
                                    Remove
                                  </button>
                                </td>
                              </tr>
                            )
                          })
                        )}
                      </tbody>
                    </table>
                  </div>
                </>
              )}

              {accountSub === 'shared' && (
                <>
                  <p className="hint">
                    Discord users involved in Local → Share (private SSO copies), plus accounts others
                    shared with you. When someone uses an account you shared, it shows under activity below
                    and on the Local accounts list.
                  </p>
                  {!status?.sso_connected ? (
                    <p className="empty">Connect with Login w/ SSO to see shared users.</p>
                  ) : (
                    <>
                      <div className="row status-head">
                        <h2>Activity on your shares</h2>
                      </div>
                      {(status?.share_activity?.online || []).length > 0 ? (
                        <div className="table-wrap" style={{marginBottom: '1rem'}}>
                          <table className="data-table">
                            <thead>
                              <tr>
                                <th>EQ account</th>
                                <th>Who</th>
                                <th>Character</th>
                                <th>Status</th>
                              </tr>
                            </thead>
                            <tbody>
                              {(status.share_activity.online || []).map((o) => (
                                <tr key={`on-${o.account_id}-${o.user_id}`}>
                                  <td className="mono">{o.account_username || `#${o.account_id}`}</td>
                                  <td>
                                    <div className="discord-user-name">
                                      {o.user_display_name || o.user_discord_id || '—'}
                                      {o.actor_is_owner ? ' (you)' : ''}
                                    </div>
                                    {o.user_discord_id ? (
                                      <div className="muted mono">{o.user_discord_id}</div>
                                    ) : null}
                                  </td>
                                  <td className="mono">{o.character_name || '—'}</td>
                                  <td className="share-activity in-use">In use now</td>
                                </tr>
                              ))}
                            </tbody>
                          </table>
                        </div>
                      ) : (
                        <p className="hint" style={{marginBottom: '1rem'}}>No shared accounts in use right now.</p>
                      )}
                      <div className="table-wrap" style={{marginBottom: '1.25rem'}}>
                        <table className="data-table">
                          <thead>
                            <tr>
                              <th>When</th>
                              <th>Who</th>
                              <th>EQ account</th>
                              <th>Typed name</th>
                            </tr>
                          </thead>
                          <tbody>
                            {(status?.share_activity?.logins || []).length === 0 ? (
                              <tr>
                                <td colSpan={4} className="empty-cell">
                                  No SSO logins on your shared accounts yet.
                                </td>
                              </tr>
                            ) : (
                              (status.share_activity.logins || []).map((e) => (
                                <tr key={e.id} className={e.actor_is_owner ? '' : 'share-login-other'}>
                                  <td title={e.created_at || ''}>{formatWhen(e.created_at)}</td>
                                  <td>
                                    <div className="discord-user-name">
                                      {e.actor_name || e.actor_discord_id || '—'}
                                      {e.actor_is_owner ? ' (you)' : ''}
                                    </div>
                                    {e.actor_discord_id ? (
                                      <div className="muted mono">{e.actor_discord_id}</div>
                                    ) : null}
                                  </td>
                                  <td className="mono">{e.account_username || `#${e.account_id}`}</td>
                                  <td className="mono">{e.detail || '—'}</td>
                                </tr>
                              ))
                            )}
                          </tbody>
                        </table>
                      </div>

                      <div className="row status-head">
                        <h2>Users you shared with</h2>
                      </div>
                      <div className="table-wrap">
                        <table className="data-table">
                          <thead>
                            <tr>
                              <SortTh sortKey="name" sort={outgoingSort.sort} onSort={outgoingSort.onSort}>Discord user</SortTh>
                              <SortTh sortKey="discord" sort={outgoingSort.sort} onSort={outgoingSort.onSort}>Discord ID</SortTh>
                              <SortTh sortKey="accounts" sort={outgoingSort.sort} onSort={outgoingSort.onSort}>Local accounts</SortTh>
                            </tr>
                          </thead>
                          <tbody>
                            {sortedOutgoing.length === 0 ? (
                              <tr>
                                <td colSpan={3} className="empty-cell">
                                  No Discord users yet. Use Share on a Local account to grant access.
                                </td>
                              </tr>
                            ) : (
                              sortedOutgoing.map(({user, accounts}) => (
                                <tr key={user.id}>
                                  <td>
                                    <div className="discord-user-name">{user.display_name || '—'}</div>
                                  </td>
                                  <td className="mono">{user.discord_id || '—'}</td>
                                  <td className="mono">{accounts.join(', ')}</td>
                                </tr>
                              ))
                            )}
                          </tbody>
                        </table>
                      </div>

                      <div className="row status-head" style={{marginTop: '1.25rem'}}>
                        <h2>Shared with you</h2>
                      </div>
                      <div className="table-wrap">
                        <table className="data-table">
                          <thead>
                            <tr>
                              <SortTh sortKey="account" sort={incomingSort.sort} onSort={incomingSort.onSort}>EQ account</SortTh>
                              <SortTh sortKey="owner" sort={incomingSort.sort} onSort={incomingSort.onSort}>Owner (Discord)</SortTh>
                              <SortTh sortKey="discord" sort={incomingSort.sort} onSort={incomingSort.onSort}>Discord ID</SortTh>
                            </tr>
                          </thead>
                          <tbody>
                            {sortedIncoming.length === 0 ? (
                              <tr>
                                <td colSpan={3} className="empty-cell">
                                  No private shares from other Discord users.
                                </td>
                              </tr>
                            ) : (
                              sortedIncoming.map(({account, owner}) => (
                                <tr key={account.id}>
                                  <td className="mono">{account.username || `#${account.id}`}</td>
                                  <td>{owner.display_name || '—'}</td>
                                  <td className="mono">{owner.discord_id || '—'}</td>
                                </tr>
                              ))
                            )}
                          </tbody>
                        </table>
                      </div>
                    </>
                  )}
                </>
              )}
            </div>
          </section>
        )}

        {tab === 'eq' && (
          <section className="panel">
            <h2>EverQuest install</h2>
            <div className="panel-scroll">
              <p className="hint">Used for log watching (online presence) and rewriting <code>eqhost.txt</code>.</p>
              <label>Install directory</label>
              <div className="row path-row">
                <input value={eqDir} onChange={(e) => setEqDir(e.target.value)} placeholder="/path/to/EverQuest"/>
                <button
                  type="button"
                  className="secondary"
                  disabled={busy}
                  onClick={() => run(async () => {
                    const picked = await PickEQDirectory()
                    if (picked) setEqDir(picked)
                  })}
                >
                  Browse…
                </button>
              </div>
              <button
                type="button"
                disabled={busy}
                onClick={() => run(() => SetEQDirectory(eqDir))}
              >
                Save path
              </button>
              {status?.online?.length > 0 && (
                <>
                  <h2 className="sub">Online (from logs)</h2>
                  <p className="meta">{status.online.join(', ')}</p>
                </>
              )}
            </div>
          </section>
        )}

        {tab === 'settings' && (
          <section className="panel">
            <div className="panel-scroll">
              <h2>Appearance</h2>
              <p className="hint">Choose light or dark. Preference is saved on this machine.</p>
              <div className="theme-toggle" role="group" aria-label="Theme">
                <button
                  type="button"
                  className={theme === 'light' ? 'active' : ''}
                  aria-pressed={theme === 'light'}
                  onClick={() => setTheme('light')}
                >
                  Light
                </button>
                <button
                  type="button"
                  className={theme === 'dark' ? 'active' : ''}
                  aria-pressed={theme === 'dark'}
                  onClick={() => setTheme('dark')}
                >
                  Dark
                </button>
              </div>

              <h2 className="sub">UDP proxy</h2>
              <p className="hint">
                Local login proxy listen address. Bound to 127.0.0.1 only.
              </p>
              <label>Listen port</label>
              <div className="row">
                <input
                  className="port"
                  value={listenPort}
                  onChange={(e) => setListenPort(e.target.value.replace(/[^\d]/g, ''))}
                  inputMode="numeric"
                  placeholder="6998"
                />
                <button
                  type="button"
                  className="secondary"
                  disabled={busy}
                  onClick={() => run(async () => {
                    const p = parseInt(listenPort, 10)
                    if (!p) throw new Error('Enter a port number')
                    await SetListenPort(p)
                  })}
                >
                  Save port
                </button>
              </div>
              <p className="hint">Default 6998. Changing the port while the proxy is on will restart it.</p>
            </div>
          </section>
        )}
      </main>

      <footer className="statusbar" aria-label="Status">
        <div className={`status-item mode-item ${status?.proxy_enabled ? 'on' : 'off'}`}>
          <span className="pulse" aria-hidden="true"/>
          <div className="text">
            <div className="label">Mode</div>
            <ModeDropdown
              compact
              value={activeMode.id}
              disabled={busy}
              onChange={(next) => run(async () => {
                await SetConnectionMode(next)
              })}
            />
          </div>
        </div>
        <div className={`status-item ${status?.proxy_enabled ? 'on' : 'off'}`}>
          <span className="pulse" aria-hidden="true"/>
          <div className="text">
            <div className="label">Proxy</div>
            <div className="value" title={status?.listen || ''}>
              {status?.proxy_enabled ? (status.listen || 'Listening') : 'Stopped'}
            </div>
          </div>
        </div>
        <div className={`status-item ${status?.sso_connected ? 'on' : 'off'}`}>
          <span className="pulse" aria-hidden="true"/>
          <div className="text">
            <div className="label">SSO</div>
            <div className="value" title={activeSourceName}>
              {status?.sso_connected ? activeSourceName : 'Disconnected'}
            </div>
          </div>
        </div>
        <div className={`status-item ${status?.eq_directory ? 'on' : 'warn'}`}>
          <span className="pulse" aria-hidden="true"/>
          <div className="text">
            <div className="label">EQ path</div>
            <div className="value" title={status?.eq_directory || ''}>
              {status?.eq_directory ? 'Configured' : 'Not set'}
            </div>
          </div>
        </div>
        <div className={`status-item ${onlineCount > 0 ? 'on' : 'idle'}`}>
          <span className="pulse" aria-hidden="true"/>
          <div className="text">
            <div className="label">Online</div>
            <div className="value">{onlineCount}</div>
          </div>
        </div>
      </footer>

      {accountModal && (
        <div className="modal-backdrop" onClick={closeLocalModal} role="presentation">
          <div
            className="modal"
            role="dialog"
            aria-modal="true"
            aria-labelledby="account-modal-title"
            onClick={(e) => e.stopPropagation()}
          >
            <h2 id="account-modal-title">{localForm.editing ? 'Edit account' : 'Add account'}</h2>
            <div className="form-grid">
              <div>
                <label>Account name</label>
                <input
                  value={localForm.name}
                  onChange={(e) => setLocalForm((f) => ({...f, name: e.target.value}))}
                  placeholder="equsername"
                  disabled={localForm.editing}
                  autoFocus={!localForm.editing}
                />
              </div>
              <div>
                <label>Password</label>
                <div className="password-field">
                  <input
                    type={showLocalPassword ? 'text' : 'password'}
                    value={localForm.password}
                    onChange={(e) => setLocalForm((f) => ({...f, password: e.target.value}))}
                    placeholder="Password"
                    autoComplete="off"
                    autoFocus={localForm.editing}
                  />
                  <button
                    type="button"
                    className="password-toggle"
                    disabled={busy}
                    aria-label={showLocalPassword ? 'Hide password' : 'Show password'}
                    aria-pressed={showLocalPassword}
                    onClick={() => setShowLocalPassword((v) => !v)}
                  >
                    {showLocalPassword ? (
                      <svg viewBox="0 0 24 24" width="18" height="18" aria-hidden="true">
                        <path fill="currentColor" d="M12 7c2.76 0 5 2.24 5 5 0 .65-.13 1.26-.36 1.83l2.92 2.92c1.51-1.26 2.7-2.89 3.43-4.75-1.73-4.39-6-7.5-11-7.5-1.4 0-2.74.25-3.98.7l2.16 2.16C10.74 7.13 11.35 7 12 7zM2 4.27l2.28 2.28.46.46C3.08 8.3 1.78 10.02 1 12c1.73 4.39 6 7.5 11 7.5 1.55 0 3.03-.3 4.38-.84l.42.42L19.73 22 21 20.73 3.27 3 2 4.27zM7.53 9.8l1.55 1.55c-.05.21-.08.43-.08.65 0 1.66 1.34 3 3 3 .22 0 .44-.03.65-.08l1.55 1.55c-.67.33-1.41.53-2.2.53-2.76 0-5-2.24-5-5 0-.79.2-1.53.53-2.2zm4.31-.78 3.15 3.15.02-.16c0-1.66-1.34-3-3-3l-.17.01z"/>
                      </svg>
                    ) : (
                      <svg viewBox="0 0 24 24" width="18" height="18" aria-hidden="true">
                        <path fill="currentColor" d="M12 4.5C7 4.5 2.73 7.61 1 12c1.73 4.39 6 7.5 11 7.5s9.27-3.11 11-7.5c-1.73-4.39-6-7.5-11-7.5zM12 17c-2.76 0-5-2.24-5-5s2.24-5 5-5 5 2.24 5 5-2.24 5-5 5zm0-8c-1.66 0-3 1.34-3 3s1.34 3 3 3 3-1.34 3-3-1.34-3-3-3z"/>
                      </svg>
                    )}
                  </button>
                </div>
              </div>
              <div className="form-span">
                <label>Aliases (comma-separated)</label>
                <input
                  value={localForm.aliases}
                  onChange={(e) => setLocalForm((f) => ({...f, aliases: e.target.value}))}
                  placeholder="tank, box1"
                />
              </div>
            </div>
            <div className="modal-actions">
              <button type="button" className="secondary" disabled={busy} onClick={closeLocalModal}>
                Cancel
              </button>
              <button
                type="button"
                disabled={busy}
                onClick={() => run(async () => {
                  if (!localForm.name.trim()) throw new Error('Enter an account name')
                  const aliases = localForm.aliases
                    .split(',')
                    .map((s) => s.trim())
                    .filter(Boolean)
                  await SaveLocalAccount(localForm.name.trim(), localForm.password, aliases)
                  closeLocalModal()
                })}
              >
                {localForm.editing ? 'Save changes' : 'Add account'}
              </button>
            </div>
          </div>
        </div>
      )}

      {shareModal && (() => {
        const myId = status?.sso_user_id || 0
        const directory = (status?.sso_directory || []).filter((u) => u.id !== myId)
        const canSaveShares = status?.sso_connected && directory.length > 0
        return (
          <div className="modal-backdrop" onClick={() => setShareModal(false)} role="presentation">
            <div
              className="modal modal-wide"
              role="dialog"
              aria-modal="true"
              aria-labelledby="share-modal-title"
              onClick={(e) => e.stopPropagation()}
            >
              <h2 id="share-modal-title">Share — {shareForm.name}</h2>
              <p className="hint">
                Publishes this local account to SSO as a private share. Selected users can log in with it
                over SSO; others cannot see it. Your local copy stays on this machine.
              </p>
              {!status?.sso_connected ? (
                <p className="empty">Connect with Login w/ SSO to share.</p>
              ) : directory.length === 0 ? (
                <p className="empty">No other SSO users yet. Someone else must create a token first.</p>
              ) : (
                <div className="role-checklist">
                  {directory.map((u) => (
                    <label key={u.id} className="role-check">
                      <input
                        type="checkbox"
                        checked={shareForm.userIds.includes(u.id)}
                        onChange={() => toggleShareUser(u.id)}
                      />
                      <span className="role-check-label">
                        <span className="mode-label">{u.display_name || u.discord_id}</span>
                        <span className="mode-hint mono">{u.discord_id}</span>
                      </span>
                    </label>
                  ))}
                </div>
              )}
              <div className="modal-actions">
                {shareForm.shared ? (
                  <button
                    type="button"
                    className="secondary danger"
                    disabled={busy || !status?.sso_connected}
                    onClick={() => run(async () => {
                      await UnshareLocalAccount(shareForm.name)
                      setShareModal(false)
                    })}
                  >
                    Stop sharing
                  </button>
                ) : null}
                <button type="button" className="secondary" disabled={busy} onClick={() => setShareModal(false)}>
                  Cancel
                </button>
                <button
                  type="button"
                  disabled={busy || !canSaveShares}
                  title={!canSaveShares && status?.sso_connected ? 'No other SSO users to share with' : undefined}
                  onClick={() => run(async () => {
                    await ShareLocalAccount(shareForm.name, shareForm.userIds)
                    setShareModal(false)
                  })}
                >
                  Save shares
                </button>
              </div>
            </div>
          </div>
        )
      })()}

      {sourcesModal && (
        <div className="modal-backdrop" onClick={closeSourcesModal} role="presentation">
          <div
            className="modal modal-wide"
            role="dialog"
            aria-modal="true"
            aria-labelledby="sources-modal-title"
            onClick={(e) => e.stopPropagation()}
          >
            <h2 id="sources-modal-title">
              {sourceForm ? (sourceForm.editing ? 'Edit SSO source' : 'Add SSO source') : 'Manage SSO sources'}
            </h2>

            {!sourceForm ? (
              <>
                <p className="hint">
                  Paste source JSON from Discord <code>/alfred-identity-sso get</code>, or enter
                  details manually. On the Connections tab, pick which source is active.
                </p>
                <div className="source-json-row compact">
                  <textarea
                    className="mono"
                    value={sourceJSON}
                    onChange={(e) => setSourceJSON(e.target.value)}
                    placeholder="Paste Alfred Identity source JSON from Discord…"
                    disabled={busy}
                    rows={5}
                  />
                  <button
                    type="button"
                    className="secondary"
                    disabled={busy || !sourceJSON.trim()}
                    onClick={() => run(() => importSourceFromJSON(sourceJSON))}
                  >
                    Add from JSON
                  </button>
                </div>
                <div className="row status-head">
                  <span/>
                  <button type="button" className="secondary" disabled={busy} onClick={startNewSource}>
                    Add manually
                  </button>
                </div>
                {sortedSources.length === 0 ? (
                  <div className="source-empty-modal">
                    <p>
                      <strong>No SSO sources yet.</strong> Run{' '}
                      <code>/alfred-identity-sso get</code> in Discord and paste the Alfred Identity
                      source JSON above.
                    </p>
                  </div>
                ) : (
                  <div className="table-wrap">
                    <table className="data-table">
                      <thead>
                        <tr>
                          <SortTh sortKey="name" sort={sourcesSort.sort} onSort={sourcesSort.onSort}>Name</SortTh>
                          <SortTh sortKey="host" sort={sourcesSort.sort} onSort={sourcesSort.onSort}>Host</SortTh>
                          <SortTh sortKey="token" sort={sourcesSort.sort} onSort={sourcesSort.onSort}>Token</SortTh>
                          <th className="col-actions">Actions</th>
                        </tr>
                      </thead>
                      <tbody>
                        {sortedSources.map((src) => (
                          <tr key={src.id} className={src.id === status?.active_source ? 'row-selected' : ''}>
                            <td>
                              {src.name || src.id}
                              {src.id === status?.active_source ? (
                                <span className="badge live" style={{marginLeft: '0.4rem'}}>active</span>
                              ) : null}
                            </td>
                            <td className="mono">{src.host || '—'}</td>
                            <td>{src.has_token ? 'saved' : 'missing'}</td>
                            <td className="col-actions">
                              <button type="button" className="secondary" disabled={busy} onClick={() => startEditSource(src)}>
                                Edit
                              </button>
                              <button
                                type="button"
                                className="secondary danger"
                                disabled={busy}
                                onClick={() => run(() => DeleteSource(src.id))}
                              >
                                Remove
                              </button>
                            </td>
                          </tr>
                        ))}
                      </tbody>
                    </table>
                  </div>
                )}
                <div className="modal-actions">
                  <button type="button" className="secondary" disabled={busy} onClick={closeSourcesModal}>
                    Done
                  </button>
                </div>
              </>
            ) : (
              <>
                <div className="form-grid">
                  <div className="form-span">
                    <label>Name</label>
                    <input
                      value={sourceForm.name}
                      onChange={(e) => setSourceForm((f) => ({...f, name: e.target.value}))}
                      placeholder="Guild daemon"
                      autoFocus={!sourceForm.fromImport}
                    />
                  </div>
                  <div className="form-span">
                    <label>Host</label>
                    <input
                      value={sourceForm.host}
                      onChange={(e) => setSourceForm((f) => ({...f, host: e.target.value}))}
                      placeholder="127.0.0.1:8181"
                      className="mono"
                    />
                  </div>
                  <div className="form-span">
                    <label>SSO token</label>
                    <input
                      type="password"
                      value={sourceForm.token}
                      onChange={(e) => setSourceForm((f) => ({...f, token: e.target.value}))}
                      placeholder={sourceForm.editing && sourceForm.has_token ? 'Leave blank to keep current' : 'Paste token'}
                      autoComplete="off"
                      autoFocus={!!sourceForm.fromImport}
                    />
                  </div>
                  <div className="form-span">
                    <label>Notes (optional)</label>
                    <input
                      value={sourceForm.notes}
                      onChange={(e) => setSourceForm((f) => ({...f, notes: e.target.value}))}
                      placeholder="Optional"
                    />
                  </div>
                </div>
                <p className="hint">
                  Host is host:port only (e.g. <code>127.0.0.1:8181</code>). Changing host clears the
                  saved token until you paste a new one.
                </p>
                {sourceForm.fromImport ? (
                  <p className="hint">
                    Loaded from JSON. Review the fields, then save
                    {sourceForm.importExtra
                      ? ` (${sourceForm.importExtra} more in the paste will open next)`
                      : ''}.
                  </p>
                ) : null}
                <div className="modal-actions">
                  <button
                    type="button"
                    className="secondary"
                    disabled={busy}
                    onClick={() => setSourceForm(null)}
                  >
                    Back
                  </button>
                  <button
                    type="button"
                    disabled={busy}
                    onClick={() => run(async () => {
                      if (!sourceForm.name.trim()) throw new Error('Enter a name')
                      if (!sourceForm.host.trim()) throw new Error('Enter a host')
                      if (!sourceForm.editing && !sourceForm.token.trim()) {
                        throw new Error('Enter an SSO token')
                      }
                      await SaveSource({
                        id: sourceForm.id,
                        name: sourceForm.name.trim(),
                        host: sourceForm.host.trim(),
                        token: sourceForm.token,
                        notes: sourceForm.notes.trim(),
                      })
                      const queue = sourceForm.importQueue || []
                      if (queue.length > 0) {
                        const next = queue[0]
                        setSourceForm({
                          id: '',
                          name: next.name || '',
                          host: next.host || '',
                          token: next.token || '',
                          notes: next.notes || '',
                          editing: false,
                          has_token: false,
                          fromImport: true,
                          importExtra: queue.length - 1,
                          importQueue: queue.slice(1),
                        })
                      } else {
                        setSourceForm(null)
                      }
                    })}
                  >
                    {sourceForm.editing ? 'Save changes' : 'Add source'}
                  </button>
                </div>
              </>
            )}
          </div>
        </div>
      )}
    </div>
  )
}
