// app.js — provides VM-style behaviors for the dashboard

const CSRF_COOKIE_NAME = 'scheduler_csrf';
const authContent = document.getElementById('auth-content');
const serviceStatus = document.getElementById('service-status');

function getCookie(name) {
  const matches = document.cookie.match(new RegExp('(^|; )' + name.replace(/([.$?*|{}()\[\]\\/+^])/g, '\\$1') + '=([^;]*)'));
  return matches ? decodeURIComponent(matches[2]) : undefined;
}

function withCSRF(headers = {}) {
  const token = getCookie(CSRF_COOKIE_NAME);
  if (token) headers['X-CSRF-Token'] = token;
  return headers;
}

function createLoginForm() {
  const form = document.createElement('form');
  form.id = 'login-form';
  form.className = 'form-row';

  const uWrap = document.createElement('div');
  const lUser = document.createElement('label'); lUser.htmlFor = 'username'; lUser.textContent = 'Username';
  const inpUser = document.createElement('input'); inpUser.id = 'username'; inpUser.name = 'username'; inpUser.autocomplete = 'username'; inpUser.required = true;
  uWrap.appendChild(lUser); uWrap.appendChild(inpUser);

  const pWrap = document.createElement('div');
  const lPass = document.createElement('label'); lPass.htmlFor = 'password'; lPass.textContent = 'Password';
  const inpPass = document.createElement('input'); inpPass.id = 'password'; inpPass.name = 'password'; inpPass.type = 'password'; inpPass.autocomplete = 'current-password'; inpPass.required = true;
  pWrap.appendChild(lPass); pWrap.appendChild(inpPass);

  const btn = document.createElement('button'); btn.type = 'submit'; btn.textContent = 'Sign in';

  form.appendChild(uWrap); form.appendChild(pWrap); form.appendChild(btn);

  form.addEventListener('submit', async (event) => {
    event.preventDefault();
    const username = inpUser.value.trim();
    const password = inpPass.value;
    authContent.classList.add('loading');

    try {
      const response = await fetch('/api/v1/auth/login', {
        method: 'POST',
        credentials: 'same-origin',
        headers: Object.assign({ 'Content-Type': 'application/json' }, withCSRF()),
        body: JSON.stringify({ username, password }),
      });

      if (!response.ok) throw new Error('Login failed');

      await refreshAuth();
    } catch (err) {
      authContent.innerHTML = '<p class="notice">Unable to sign in. Confirm backend credentials and retry.</p>';
      setTimeout(refreshAuth, 2000);
    } finally {
      authContent.classList.remove('loading');
    }
  });

  return form;
}

function renderLoginForm() {
  authContent.innerHTML = '';
  authContent.appendChild(createLoginForm());
  const hint = document.createElement('p');
  hint.style.marginTop = '12px';
  hint.style.color = 'var(--muted)';
  hint.style.fontSize = '0.95rem';
  hint.textContent = 'Use the API login endpoint to sign in and unlock admin/workstation data.';
  authContent.appendChild(hint);

  // Hide admin panels when logged out
  showAdminPanels(null);
}

function renderUserInfo(user) {
  authContent.innerHTML = '';
  const wrap = document.createElement('div');
  wrap.className = 'notice';
  wrap.innerHTML = `
    <p><strong>Signed in as:</strong> ${user.username}</p>
    <p><strong>Role:</strong> ${user.role}</p>
    <p><strong>Assigned workstations:</strong> ${user.assigned_workstation_ids.length ? user.assigned_workstation_ids.join(', ') : 'All'}</p>
  `;
  authContent.appendChild(wrap);

  const btn = document.createElement('button'); btn.id = 'logout-button'; btn.textContent = 'Sign out';
  btn.addEventListener('click', async () => {
    await fetch('/api/v1/auth/logout', { method: 'POST', credentials: 'same-origin', headers: withCSRF() });
    await refreshAuth();
  });
  authContent.appendChild(btn);

  // Show/hide admin panels based on role
  showAdminPanels(user.role);
}

async function refreshAuth() {
  authContent.classList.add('loading');
  try {
    const response = await fetch('/api/v1/auth/me', { credentials: 'same-origin' });
    if (response.ok) {
      const user = await response.json();
      renderUserInfo(user);
    } else {
      renderLoginForm();
    }
  } catch (err) {
    authContent.innerHTML = '<p class="notice">Unable to reach auth API. Backend may not be running.</p>';
  } finally {
    authContent.classList.remove('loading');
  }
}

async function fetchHealth() {
  try {
    const response = await fetch('/healthz', { cache: 'no-store' });
    if (!response.ok) throw new Error('unhealthy');
    const payload = await response.json();
    serviceStatus.textContent = `Online • ${payload.status} • ${new Date(payload.time).toLocaleTimeString()}`;
    serviceStatus.classList.remove('failure');
  } catch (err) {
    serviceStatus.textContent = 'Offline • health check failed';
    serviceStatus.classList.add('failure');
  }
}

// --- API helpers ---
async function apiGet(path) {
  const res = await fetch(path, { credentials: 'same-origin', headers: { Accept: 'application/json' } });
  if (!res.ok) {
    const text = await res.text().catch(() => '');
    const err = new Error(`GET ${path} failed: ${res.status}`);
    err.status = res.status;
    err.body = text;
    throw err;
  }
  return res.json();
}

// --- Admin views ---
function renderWorkstations(list) {
  const el = document.getElementById('workstations-list');
  if (!list || list.length === 0) {
    el.innerHTML = '<p class="notice">No workstations returned.</p>';
    return;
  }
  const ul = document.createElement('ul');
  ul.className = 'endpoint-list';
  list.forEach(ws => {
    const li = document.createElement('li');
    const name = document.createElement('span'); name.textContent = ws.name || ws.id || 'unnamed';
    const meta = document.createElement('span'); meta.textContent = ws.id || '';
    li.appendChild(name); li.appendChild(meta);
    ul.appendChild(li);
  });
  el.innerHTML = '';
  el.appendChild(ul);
}

function renderUsers(list) {
  const el = document.getElementById('users-list');
  if (!list || list.length === 0) {
    el.innerHTML = '<p class="notice">No users returned.</p>';
    return;
  }
  const ul = document.createElement('ul');
  ul.className = 'endpoint-list';
  list.forEach(u => {
    const li = document.createElement('li');
    const name = document.createElement('span'); name.textContent = u.username || u.id || 'unknown';
    const meta = document.createElement('span'); meta.textContent = u.role || '';
    li.appendChild(name); li.appendChild(meta);
    ul.appendChild(li);
  });
  el.innerHTML = '';
  el.appendChild(ul);
}

async function loadWorkstations() {
  const el = document.getElementById('workstations-list');
  el.innerHTML = '<p class="notice">Loading workstations...</p>';
  try {
    const data = await apiGet('/api/v1/admin/workstations');
    renderWorkstations(data || []);
  } catch (err) {
    if (err.status === 401 || err.status === 403) {
      el.innerHTML = '<p class="notice">Unauthorized. Sign in as an admin to view workstations.</p>';
    } else {
      el.innerHTML = `<p class="notice">Failed to load workstations: ${err.message}</p>`;
    }
  }
}

async function loadUsers() {
  const el = document.getElementById('users-list');
  el.innerHTML = '<p class="notice">Loading users...</p>';
  try {
    const data = await apiGet('/api/v1/admin/users');
    renderUsers(data || []);
  } catch (err) {
    if (err.status === 401 || err.status === 403) {
      el.innerHTML = '<p class="notice">Unauthorized. Sign in as an admin to view users.</p>';
    } else {
      el.innerHTML = `<p class="notice">Failed to load users: ${err.message}</p>`;
    }
  }
}

function initAdminPanel() {
  const btnW = document.getElementById('btn-load-workstations');
  const btnU = document.getElementById('btn-load-users');
  const adminPanel = document.getElementById('admin-panel');
  const usersPanel = document.getElementById('users-panel');

  // Hide admin panels until user logs in as admin
  if (adminPanel) adminPanel.style.display = 'none';
  if (usersPanel) usersPanel.style.display = 'none';

  if (btnW) btnW.addEventListener('click', loadWorkstations);
  if (btnU) btnU.addEventListener('click', loadUsers);
}

function showAdminPanels(role) {
  const adminPanel = document.getElementById('admin-panel');
  const usersPanel = document.getElementById('users-panel');
  if (role === 'admin') {
    if (adminPanel) adminPanel.style.display = 'block';
    if (usersPanel) usersPanel.style.display = 'block';
  } else {
    if (adminPanel) adminPanel.style.display = 'none';
    if (usersPanel) usersPanel.style.display = 'none';
  }
}

// Initialize
fetchHealth();
refreshAuth();
setInterval(fetchHealth, 30000);
initAdminPanel();
