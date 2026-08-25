import {useCallback, useEffect, useLayoutEffect, useRef, useState} from 'react'
import {createPortal} from 'react-dom'
import './App.css'
import {EventsOn} from '../wailsjs/runtime/runtime'
import {
  ClearLogs,
  ApplyUpdate,
  CheckUpdate,
  DeleteLocalAccount,
  GetLocalAccounts,
  GetLocalAccountPassword,
  GetLogs,
  GetStatus,
  GetVersion,
  DiscoverP99LoginProxyInstalls,
  ScanP99LoginProxyInstalls,
  ImportLocalAccountsFromPath,
  ExportLocalAccountsCSV,
  OpenFolderInFileManager,
  OpenReleaseURL,
  PickEQDirectory,
  PickLocalAccountsCSVFile,
  PickP99ProxyConfigFile,
  PickP99ProxyDataDirectory,
  SaveLocalAccount,
  SaveSource,
  PreviewSourceJSON,
  GetEqHostState,
  SaveEqHostContent,
  RestoreEqHostBackup,
  OpenEQDirectory,
  SetActiveSource,
  SetConnectionMode,
  SetEQDirectory,
  SetListenPort,
  DeleteSource,
  ShareLocalAccount,
  UnshareLocalAccount,
} from '../wailsjs/go/app/App'

const TABS = [
  {id: 'proxy', label: 'Connections'},
  {id: 'accounts', label: 'Accounts'},
  {id: 'eq', label: 'EverQuest'},
  {id: 'logs', label: 'Logs'},
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

/** Case-insensitive substring match across any of the given field values. */
function rowMatchesFilter(query, values) {
  const needle = String(query || '').trim().toLowerCase()
  if (!needle) return true
  return values.some((v) => {
    if (v == null || v === '') return false
    if (Array.isArray(v)) {
      return v.some((x) => String(x ?? '').toLowerCase().includes(needle))
    }
    return String(v).toLowerCase().includes(needle)
  })
}

export default function App() {
  const [tab, setTab] = useState('proxy')
  const [accountSub, setAccountSub] = useState('sso')
  const [ssoFilter, setSsoFilter] = useState('')
  const [localFilter, setLocalFilter] = useState('')
  const [status, setStatus] = useState(null)
  const [localAccounts, setLocalAccounts] = useState([])
  const [localForm, setLocalForm] = useState(blankLocalForm)
  const [accountModal, setAccountModal] = useState(false)
  const [importModal, setImportModal] = useState(false)
  const [p99Installs, setP99Installs] = useState([])
  const [p99InstallsLoading, setP99InstallsLoading] = useState(false)
  const [p99ScanDeep, setP99ScanDeep] = useState(false)
  const [importResult, setImportResult] = useState(null)
  const [showLocalPassword, setShowLocalPassword] = useState(false)
  const [shareModal, setShareModal] = useState(false)
  const [shareForm, setShareForm] = useState({name: '', userIds: [], roleIds: [], groupIds: [], shared: false})
  const [sourcesModal, setSourcesModal] = useState(false)
  const [sourceForm, setSourceForm] = useState(null) // null = list view; object = edit/add
  const [sourceJSON, setSourceJSON] = useState('')
  const [eqDir, setEqDir] = useState('')
  const [eqHost, setEqHost] = useState({current: '', backup: '', has_backup: false})
  const [eqHostEditing, setEqHostEditing] = useState(false)
  const [eqHostDraft, setEqHostDraft] = useState('')
  const [listenPort, setListenPort] = useState('6998')
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState('')
  const [update, setUpdate] = useState(null)
  const [updateStatus, setUpdateStatus] = useState(null)
  const [updateChecking, setUpdateChecking] = useState(false)
  const [updateInstalling, setUpdateInstalling] = useState(false)
  const [deleteLocalConfirm, setDeleteLocalConfirm] = useState(null)
  const [theme, setTheme] = useState(() => readStoredTheme())
  const [logs, setLogs] = useState([])
  const logEndRef = useRef(null)
  const [logAutoScroll, setLogAutoScroll] = useState(true)

  const applyUpdateInfo = useCallback((u) => {
    if (!u) return
    setUpdateStatus(u)
    if (u.update_available) setUpdate(u)
    else setUpdate(null)
  }, [])

  const checkForUpdates = useCallback(async () => {
    setUpdateChecking(true)
    try {
      const u = await CheckUpdate()
      applyUpdateInfo(u)
      setError('')
    } catch (e) {
      setError(String(e))
    } finally {
      setUpdateChecking(false)
    }
  }, [applyUpdateInfo])

  const installUpdate = useCallback(async () => {
    setUpdateInstalling(true)
    setError('')
    try {
      await ApplyUpdate()
      // App should quit and relaunch; keep spinner if it doesn't.
    } catch (e) {
      setError(String(e))
      setUpdateInstalling(false)
    }
  }, [])

  useEffect(() => {
    GetVersion()
      .then((v) => {
        setUpdateStatus((prev) => prev ?? {current: v, update_available: false})
      })
      .catch(() => {})
    return EventsOn('update-checked', applyUpdateInfo)
  }, [applyUpdateInfo])

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
    CheckUpdate().then(applyUpdateInfo).catch(() => {})
    return () => clearInterval(id)
  }, [refresh, refreshLocal, applyUpdateInfo])

  useEffect(() => {
    if (tab !== 'eq') return
    GetEqHostState()
      .then((st) => {
        const next = st || {current: '', backup: '', has_backup: false}
        setEqHost(next)
        setEqHostDraft(next.current || '')
        setEqHostEditing(false)
      })
      .catch((e) => setError(String(e)))
  }, [tab, status?.eq_directory])

  useEffect(() => {
    if (tab !== 'logs') return
    let cancelled = false
    const load = async () => {
      try {
        const lines = await GetLogs(1000)
        if (!cancelled) setLogs(lines || [])
      } catch (e) {
        if (!cancelled) setError(String(e))
      }
    }
    load()
    const id = setInterval(load, 1500)
    return () => {
      cancelled = true
      clearInterval(id)
    }
  }, [tab])

  useEffect(() => {
    if (tab !== 'logs' || !logAutoScroll) return
    logEndRef.current?.scrollIntoView({behavior: 'smooth'})
  }, [logs, tab, logAutoScroll])

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

  async function selectLocal(acc) {
    let password = ''
    try {
      password = await GetLocalAccountPassword(acc.name || '')
    } catch (_) {
      password = ''
    }
    setLocalForm({
      name: acc.name || '',
      password: password || '',
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

  async function openImportModal() {
    setImportResult(null)
    setImportModal(true)
    setP99ScanDeep(false)
    setP99InstallsLoading(true)
    try {
      const installs = await DiscoverP99LoginProxyInstalls()
      setP99Installs(installs || [])
    } catch {
      setP99Installs([])
    } finally {
      setP99InstallsLoading(false)
    }
  }

  async function scanP99Installs() {
    setP99ScanDeep(true)
    setP99InstallsLoading(true)
    setImportResult(null)
    try {
      const installs = await ScanP99LoginProxyInstalls()
      setP99Installs(installs || [])
    } catch (e) {
      setError(String(e))
      setP99Installs([])
    } finally {
      setP99InstallsLoading(false)
      setP99ScanDeep(false)
    }
  }

  function closeImportModal() {
    setImportModal(false)
    setImportResult(null)
  }

  async function importAccountsFromPath(path) {
    const res = await ImportLocalAccountsFromPath(path)
    setImportResult(res)
    await refreshLocal()
    return res
  }

  async function withNativeDialog(fn) {
    setBusy(true)
    try {
      await fn()
    } catch (e) {
      setError(String(e))
    } finally {
      setBusy(false)
    }
  }

  async function pickCSVFile() {
    await withNativeDialog(async () => {
      const path = await PickLocalAccountsCSVFile()
      if (!path) return
      await importAccountsFromPath(path)
    })
  }

  async function pickP99ConfigFile(startDir = '') {
    await withNativeDialog(async () => {
      const path = await PickP99ProxyConfigFile(startDir)
      if (!path) return
      await importAccountsFromPath(path)
    })
  }

  async function pickP99InstallFolder(startDir = '') {
    await withNativeDialog(async () => {
      const dir = await PickP99ProxyDataDirectory(startDir)
      if (!dir) return
      await importAccountsFromPath(dir)
    })
  }

  function openShareModal(acc) {
    setShareForm({
      name: acc.name,
      userIds: [...(acc.shared_user_ids || [])],
      roleIds: [...(acc.shared_role_ids || [])],
      groupIds: [...(acc.shared_group_ids || [])],
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

  function toggleShareRole(roleId) {
    setShareForm((f) => {
      const has = f.roleIds.includes(roleId)
      return {
        ...f,
        roleIds: has ? f.roleIds.filter((id) => id !== roleId) : [...f.roleIds, roleId],
      }
    })
  }

  function toggleShareGroup(groupId) {
    setShareForm((f) => {
      const has = f.groupIds.includes(groupId)
      return {
        ...f,
        groupIds: has ? f.groupIds.filter((id) => id !== groupId) : [...f.groupIds, groupId],
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
  const shareRoles = (status?.sso_roles?.length ? status.sso_roles : adminRoles) || []
  const shareGroups = status?.sso_groups || []
  const canConfigureShares = shareTargets.length > 0 || shareRoles.length > 0 || shareGroups.length > 0

  function formatShareGrantSummary(acc) {
    if (!acc.shared) return '—'
    const parts = []
    const users = (acc.shared_user_ids || []).length
    const roles = (acc.shared_role_ids || []).length
    const groups = (acc.shared_group_ids || []).length
    if (users) parts.push(`${users} user${users === 1 ? '' : 's'}`)
    if (roles) parts.push(`${roles} role${roles === 1 ? '' : 's'}`)
    if (groups) parts.push(`${groups} group${groups === 1 ? '' : 's'}`)
    return parts.length ? parts.join(', ') : 'owner only'
  }

  function ssoAccessLabel(a) {
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
      parts.push(u.display_name || u.discord_id || `#${uid}`)
    }
    for (const gid of a.group_ids || []) {
      const g = (status?.sso_groups || []).find((x) => x.id === gid)
      parts.push(g?.name || `group #${gid}`)
    }
    return parts.length ? parts.join(', ') : 'all'
  }

  function ssoLoggedInLabel(a) {
    const fromDaemon = (status?.sso_online || []).find((o) => o.account_id === a.id)
    if (fromDaemon?.character_name) return fromDaemon.character_name
    const localOnline = status?.online || []
    for (const ch of a.characters || []) {
      if (localOnline.some((n) => n.toLowerCase() === ch.toLowerCase())) return ch
    }
    return ''
  }

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

  const filteredSSOAccounts = (status?.sso_accounts || []).filter((a) => {
    const access = ssoAccessLabel(a)
    const loggedIn = ssoLoggedInLabel(a)
    return rowMatchesFilter(ssoFilter, [
      a.username,
      a.aliases,
      a.tags,
      a.characters,
      access,
      loggedIn,
      a.required_role_id,
      a.required_role_ids,
    ])
  })
  const sortedSSOAccounts = ssoSort.sorted(filteredSSOAccounts, {
    username: (a) => a.username || '',
    aliases: (a) => (a.aliases || []).join(', '),
    tags: (a) => (a.tags || []).join(', '),
    characters: (a) => (a.characters || []).join(', '),
    access: (a) => ssoAccessLabel(a),
    logged: (a) => ssoLoggedInLabel(a),
  })
  const filteredLocalAccounts = localAccounts.filter((acc) => {
    const shareUserNames = (acc.shared_user_ids || []).map((uid) => {
      const u = discordUserFromID(uid)
      return [u.display_name, u.discord_id]
    }).flat()
    const shareRoleNames = (acc.shared_role_ids || []).map((rid) => roleNameById(shareRoles, rid))
    const shareGroupNames = (acc.shared_group_ids || []).map((gid) => {
      const g = shareGroups.find((x) => x.id === gid)
      return g?.name || `group #${gid}`
    })
    return rowMatchesFilter(localFilter, [
      acc.name,
      acc.aliases,
      formatShareGrantSummary(acc),
      shareUserNames,
      shareRoleNames,
      shareGroupNames,
      acc.in_use_by,
      acc.last_login_by,
      acc.shared ? 'shared' : '',
    ])
  })
  const sortedLocalAccounts = localSort.sorted(filteredLocalAccounts, {
    name: (a) => a.name || '',
    aliases: (a) => (a.aliases || []).join(', '),
    shared: (a) => (a.shared
      ? (a.shared_user_ids || []).length + (a.shared_role_ids || []).length + (a.shared_group_ids || []).length + 1
      : 0),
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
              {update.can_apply && (
                <button
                  className="secondary"
                  type="button"
                  disabled={busy || updateInstalling}
                  onClick={() => installUpdate()}
                >
                  {updateInstalling ? 'Installing…' : 'Install & restart'}
                </button>
              )}{' '}
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
                      placeholder={'{\n  "name": "Guild",\n  "host": "identity.example.com:443",\n  "token": "..."\n}'}
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
                <div className="table-wrap">
                  <table className="data-table">
                    <thead>
                      <tr>
                        <SortTh sortKey="name" sort={sourcesSort.sort} onSort={sourcesSort.onSort}>Name</SortTh>
                        <SortTh sortKey="host" sort={sourcesSort.sort} onSort={sourcesSort.onSort} className="col-host">Host</SortTh>
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
                            <td className="mono col-ellipsis col-host" title={src.host || ''}>{src.host || '—'}</td>
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
                        <label className="list-filter">
                          <span className="sr-only">Filter accounts</span>
                          <input
                            type="search"
                            value={ssoFilter}
                            onChange={(e) => setSsoFilter(e.target.value)}
                            placeholder="Filter accounts…"
                            autoComplete="off"
                            spellCheck={false}
                          />
                        </label>
                      </div>
                      {(status.sso_accounts?.length || 0) === 0 ? (
                        <p className="empty">No accounts available yet. Admins manage them in the web admin.</p>
                      ) : sortedSSOAccounts.length === 0 ? (
                        <p className="empty">No accounts match “{ssoFilter.trim()}”.</p>
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
                                const loggedIn = ssoLoggedInLabel(a)
                                const access = ssoAccessLabel(a)
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
                    <div className="status-head-actions">
                      <label className="list-filter">
                        <span className="sr-only">Filter accounts</span>
                        <input
                          type="search"
                          value={localFilter}
                          onChange={(e) => setLocalFilter(e.target.value)}
                          placeholder="Filter accounts…"
                          autoComplete="off"
                          spellCheck={false}
                        />
                      </label>
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
                              onClick: () => openImportModal(),
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
                        {localAccounts.length === 0 ? (
                          <tr>
                            <td colSpan={4} className="empty-cell">No local accounts yet.</td>
                          </tr>
                        ) : sortedLocalAccounts.length === 0 ? (
                          <tr>
                            <td colSpan={4} className="empty-cell">No accounts match “{localFilter.trim()}”.</td>
                          </tr>
                        ) : (
                          sortedLocalAccounts.map((acc) => {
                            const canOpenShare = !!status?.sso_connected && (canConfigureShares || !!acc.shared)
                            const shareTitle = !status?.sso_connected
                              ? 'Connect with Login w/ SSO to share'
                              : (!canConfigureShares && !acc.shared
                                ? 'No SSO users, roles, or groups available to share with yet'
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
                                        {formatShareGrantSummary(acc)}
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
                                    onClick={() => setDeleteLocalConfirm(acc.name)}
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
              <div className="row eqhost-actions">
                <button
                  type="button"
                  disabled={busy}
                  onClick={() => run(() => SetEQDirectory(eqDir))}
                >
                  Save path
                </button>
                <button
                  type="button"
                  className="secondary"
                  disabled={busy || !status?.eq_directory}
                  onClick={() => run(() => OpenEQDirectory())}
                >
                  Open folder
                </button>
              </div>

              {status?.eq_directory ? (
                <>
                  <div className="eqhost-block">
                    <div className="row status-head">
                      <h2 className="sub flush">eqhost.txt</h2>
                      {!eqHostEditing ? (
                        <button
                          type="button"
                          className="secondary"
                          disabled={busy}
                          onClick={() => {
                            setEqHostDraft(eqHost.current || '')
                            setEqHostEditing(true)
                          }}
                        >
                          Edit
                        </button>
                      ) : (
                        <div className="row eqhost-edit-actions">
                          <button
                            type="button"
                            className="secondary"
                            disabled={busy}
                            onClick={() => {
                              setEqHostDraft(eqHost.current || '')
                              setEqHostEditing(false)
                            }}
                          >
                            Cancel
                          </button>
                          <button
                            type="button"
                            disabled={busy}
                            onClick={() => run(async () => {
                              await SaveEqHostContent(eqHostDraft)
                              setEqHostEditing(false)
                              const st = await GetEqHostState()
                              setEqHost(st || {current: '', backup: '', has_backup: false})
                              setEqHostDraft(st?.current || '')
                            })}
                          >
                            Save
                          </button>
                        </div>
                      )}
                    </div>
                    <textarea
                      className="mono eqhost-text"
                      value={eqHostEditing ? eqHostDraft : (eqHost.current || '')}
                      onChange={(e) => setEqHostDraft(e.target.value)}
                      readOnly={!eqHostEditing}
                      placeholder="eqhost.txt not found in the install directory"
                      rows={6}
                    />
                    {!eqHost.current && !eqHostEditing ? (
                      <p className="hint">No eqhost.txt yet. Click Edit to create one.</p>
                    ) : null}
                  </div>

                  <div className="eqhost-block">
                    <div className="row status-head">
                      <h2 className="sub flush">Backup (eqhost.txt.bak)</h2>
                      <button
                        type="button"
                        className="secondary"
                        disabled={busy || !eqHost.has_backup}
                        onClick={() => run(async () => {
                          await RestoreEqHostBackup()
                          const st = await GetEqHostState()
                          setEqHost(st || {current: '', backup: '', has_backup: false})
                          setEqHostDraft(st?.current || '')
                          setEqHostEditing(false)
                        })}
                      >
                        Restore backup
                      </button>
                    </div>
                    <textarea
                      className="mono eqhost-text readonly"
                      value={eqHost.backup || ''}
                      readOnly
                      placeholder={eqHost.has_backup ? '' : 'No backup yet — saving eqhost.txt creates eqhost.txt.bak once'}
                      rows={6}
                    />
                  </div>
                </>
              ) : (
                <p className="hint">Save an EverQuest install path to view and edit eqhost.txt.</p>
              )}

              {status?.online?.length > 0 && (
                <>
                  <h2 className="sub">Online (from logs)</h2>
                  <p className="meta">{status.online.join(', ')}</p>
                </>
              )}
            </div>
          </section>
        )}

        {tab === 'logs' && (
          <section className="panel logs-panel">
            <div className="panel-scroll logs-scroll">
              <div className="row status-head">
                <h2 className="flush">Application log</h2>
                <div className="row">
                  <label className="checkbox-inline">
                    <input
                      type="checkbox"
                      checked={logAutoScroll}
                      onChange={(e) => setLogAutoScroll(e.target.checked)}
                    />
                    Auto-scroll
                  </label>
                  <button
                    type="button"
                    className="secondary"
                    disabled={busy}
                    onClick={() => run(async () => {
                      await ClearLogs()
                      setLogs([])
                    })}
                  >
                    Clear
                  </button>
                </div>
              </div>
              <p className="hint">
                Proxy, SSO, eqhost, and other backend activity. Refreshes while this tab is open.
              </p>
              <div className="log-view" role="log" aria-live="polite" aria-relevant="additions">
                {logs.length === 0 ? (
                  <p className="empty">No log entries yet.</p>
                ) : (
                  logs.map((line, i) => (
                    <div key={`${line.time}-${i}`} className={`log-line level-${(line.level || 'info').toLowerCase()}`}>
                      <span className="log-time">{line.time}</span>
                      <span className="log-level">{line.level}</span>
                      <span className="log-msg">{line.message}</span>
                      {line.attrs ? <span className="log-attrs">{line.attrs}</span> : null}
                    </div>
                  ))
                )}
                <div ref={logEndRef} />
              </div>
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

              <h2 className="sub">Updates</h2>
              <p className="hint">
                Current version:{' '}
                <strong>{status?.version || updateStatus?.current || '—'}</strong>
              </p>
              <div className="row">
                <button
                  type="button"
                  className="secondary"
                  disabled={busy || updateChecking || updateInstalling}
                  onClick={() => run(checkForUpdates)}
                >
                  {updateChecking ? 'Checking…' : 'Check for updates'}
                </button>
                {updateStatus?.update_available && updateStatus?.can_apply && (
                  <button
                    type="button"
                    disabled={busy || updateInstalling}
                    onClick={() => installUpdate()}
                  >
                    {updateInstalling ? 'Installing…' : 'Install & restart'}
                  </button>
                )}
              </div>
              {updateStatus && !updateChecking && (
                <p className={`hint update-status${updateStatus.error ? ' err' : ''}`}>
                  {updateStatus.error ? (
                    <>Could not check for updates: {updateStatus.error}</>
                  ) : updateStatus.update_available ? (
                    <>
                      Update available: <strong>{updateStatus.latest}</strong> (you have{' '}
                      {updateStatus.current}).{' '}
                      <button
                        type="button"
                        className="secondary"
                        onClick={() => OpenReleaseURL(updateStatus.release_url)}
                      >
                        Open release
                      </button>
                    </>
                  ) : (
                    <>You&apos;re on the latest version ({updateStatus.current}).</>
                  )}
                </p>
              )}
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

      {importModal && (
        <div className="modal-backdrop" onClick={closeImportModal} role="presentation">
          <div
            className="modal modal-wide"
            role="dialog"
            aria-modal="true"
            aria-labelledby="import-modal-title"
            onClick={(e) => e.stopPropagation()}
          >
            <h2 id="import-modal-title">Import local accounts</h2>
            <p className="hint">
              Import name/password rows from a CSV file or from an existing P99 Login Proxy install.
            </p>

            <div className="import-section">
              <h3 className="sub flush">CSV file</h3>
              <p className="hint">Choose any <code>local_accounts.csv</code> or compatible export.</p>
              <button
                type="button"
                className="secondary"
                disabled={busy}
                onClick={pickCSVFile}
              >
                Choose CSV file…
              </button>
            </div>

            <div className="import-section">
              <div className="row status-head">
                <h3 className="sub flush">P99 Login Proxy</h3>
                <button
                  type="button"
                  className="secondary"
                  disabled={busy || p99InstallsLoading}
                  onClick={scanP99Installs}
                >
                  {p99InstallsLoading && p99ScanDeep ? 'Scanning…' : 'Scan all folders'}
                </button>
              </div>
              <p className="hint">
                If you already use P99 Login Proxy, open its data folder and select{' '}
                <code>proxyconfig.ini</code> or <code>local_accounts.csv</code>.
                Use <strong>Scan all folders</strong> to search your home directory recursively.
              </p>
              {p99InstallsLoading ? (
                <p className="hint">{p99ScanDeep ? 'Scanning all folders…' : 'Searching for installs…'}</p>
              ) : p99Installs.length === 0 ? (
                <p className="hint">No P99 Login Proxy installs found. Try Scan all folders.</p>
              ) : (
                <ul className="p99-install-list">
                  {p99Installs.map((inst) => (
                    <li key={inst.config_path} className="p99-install-item">
                      <div className="mono p99-install-path" title={inst.config_path}>
                        {inst.config_dir}
                      </div>
                      {inst.eq_directory ? (
                        <div className="hint p99-install-eq">EQ: {inst.eq_directory}</div>
                      ) : null}
                      <div className="row p99-install-actions">
                        <button
                          type="button"
                          className="secondary"
                          disabled={busy}
                          onClick={() => withNativeDialog(() => OpenFolderInFileManager(inst.config_dir))}
                        >
                          Open folder
                        </button>
                        <button
                          type="button"
                          className="secondary"
                          disabled={busy || !inst.has_accounts}
                          title={inst.has_accounts ? inst.accounts_csv : 'Accounts CSV not found'}
                          onClick={() => withNativeDialog(() => importAccountsFromPath(inst.config_path))}
                        >
                          Import from config
                        </button>
                        <button
                          type="button"
                          className="secondary"
                          disabled={busy}
                          onClick={() => pickP99ConfigFile(inst.config_dir)}
                        >
                          Choose file…
                        </button>
                      </div>
                    </li>
                  ))}
                </ul>
              )}
              <div className="row p99-browse-actions">
                <button
                  type="button"
                  className="secondary"
                  disabled={busy}
                  onClick={() => pickP99InstallFolder('')}
                >
                  Browse for install folder…
                </button>
                <button
                  type="button"
                  className="secondary"
                  disabled={busy}
                  onClick={() => pickP99ConfigFile('')}
                >
                  Choose config or CSV…
                </button>
              </div>
            </div>

            {importResult?.message ? (
              <p className="hint import-result">{importResult.message}</p>
            ) : null}

            <div className="modal-actions">
              <button type="button" className="secondary" disabled={busy} onClick={closeImportModal}>
                Done
              </button>
            </div>
          </div>
        </div>
      )}

      {deleteLocalConfirm && (
        <div className="modal-backdrop" onClick={() => setDeleteLocalConfirm(null)} role="presentation">
          <div
            className="modal"
            role="dialog"
            aria-modal="true"
            aria-labelledby="delete-local-title"
            onClick={(e) => e.stopPropagation()}
          >
            <h2 id="delete-local-title">Delete local account</h2>
            <p className="hint">
              Delete local account <strong>{deleteLocalConfirm}</strong>? This cannot be undone.
            </p>
            <div className="modal-actions">
              <button type="button" className="secondary" disabled={busy} onClick={() => setDeleteLocalConfirm(null)}>
                Cancel
              </button>
              <button
                type="button"
                className="secondary danger"
                disabled={busy}
                onClick={() => run(async () => {
                  const name = deleteLocalConfirm
                  await DeleteLocalAccount(name)
                  setDeleteLocalConfirm(null)
                  if (localForm.name === name) closeLocalModal()
                })}
              >
                Delete
              </button>
            </div>
          </div>
        </div>
      )}

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
        const roles = (status?.sso_roles?.length ? status.sso_roles : (status?.sso_admin_roles || []))
        const groups = status?.sso_groups || []
        const canSaveShares = !!status?.sso_connected
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
                Publishes this local account to SSO as a private share. Selected Discord users, roles, and/or
                access groups can log in with it over SSO; others cannot see it. Your local copy stays on this machine.
              </p>
              {!status?.sso_connected ? (
                <p className="empty">Connect with Login w/ SSO to share.</p>
              ) : (
                <>
                  {directory.length > 0 ? (
                    <>
                      <h3 className="share-section-title">Users</h3>
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
                    </>
                  ) : null}
                  {roles.length > 0 ? (
                    <>
                      <h3 className="share-section-title">Discord roles</h3>
                      <div className="role-checklist">
                        {roles.map((r) => (
                          <label key={r.id} className="role-check">
                            <input
                              type="checkbox"
                              checked={shareForm.roleIds.includes(r.id)}
                              onChange={() => toggleShareRole(r.id)}
                            />
                            <span className="role-check-label">
                              <span className="mode-label">{r.name || r.id}</span>
                              <span className="mode-hint mono">{r.id}</span>
                            </span>
                          </label>
                        ))}
                      </div>
                    </>
                  ) : null}
                  {groups.length > 0 ? (
                    <>
                      <h3 className="share-section-title">Access groups</h3>
                      <div className="role-checklist">
                        {groups.map((g) => (
                          <label key={g.id} className="role-check">
                            <input
                              type="checkbox"
                              checked={shareForm.groupIds.includes(g.id)}
                              onChange={() => toggleShareGroup(g.id)}
                            />
                            <span className="role-check-label">
                              <span className="mode-label">{g.name || `Group #${g.id}`}</span>
                              {g.description ? (
                                <span className="mode-hint">{g.description}</span>
                              ) : null}
                            </span>
                          </label>
                        ))}
                      </div>
                    </>
                  ) : null}
                  {!directory.length && !roles.length && !groups.length ? (
                    <p className="empty">No SSO users, Discord roles, or access groups are available yet.</p>
                  ) : null}
                </>
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
                  onClick={() => run(async () => {
                    await ShareLocalAccount(shareForm.name, shareForm.userIds, shareForm.roleIds, shareForm.groupIds)
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
            className="modal modal-wide modal-sources"
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
                          <SortTh sortKey="host" sort={sourcesSort.sort} onSort={sourcesSort.onSort} className="col-host">Host</SortTh>
                          <SortTh sortKey="token" sort={sourcesSort.sort} onSort={sourcesSort.onSort} className="col-token">Token</SortTh>
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
                            <td className="mono col-ellipsis col-host" title={src.host || ''}>{src.host || '—'}</td>
                            <td className="col-token">{src.has_token ? 'saved' : 'missing'}</td>
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
                      placeholder="identity.example.com:443"
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
                  Host is host:port only (e.g. <code>identity.example.com:443</code>). Changing host clears the
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
