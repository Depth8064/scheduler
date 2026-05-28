// app.js — provides VM-style behaviors for the dashboard

const CSRF_COOKIE_NAME = 'scheduler_csrf';
const authContent = document.getElementById('auth-content');
const serviceStatus = document.getElementById('service-status');
let currentUserRole = null;
let currentUserId = null;
let activeAssignmentWorkstationId = null;

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
  currentUserRole = null;
  currentUserId = null;
  authContent.innerHTML = '';
  authContent.appendChild(createLoginForm());
  const hint = document.createElement('p');
  hint.style.marginTop = '12px';
  hint.style.color = 'var(--muted)';
  hint.style.fontSize = '0.95rem';
  hint.textContent = 'Use the API login endpoint to sign in and unlock admin/workstation data.';
  authContent.appendChild(hint);

  showAdminPanels(null);
}

function renderUserInfo(user) {
  currentUserRole = user.role;
  currentUserId = user.user_id;
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
    await initPage();
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

async function apiRequest(method, path, payload) {
  const options = {
    method,
    credentials: 'same-origin',
    headers: {
      Accept: 'application/json',
      ...withCSRF(),
    },
  };

  if (payload !== undefined) {
    options.headers['Content-Type'] = 'application/json';
    options.body = JSON.stringify(payload);
  }

  const res = await fetch(path, options);
  if (!res.ok) {
    const text = await res.text().catch(() => '');
    const err = new Error(`${method} ${path} failed: ${res.status}`);
    err.status = res.status;
    err.body = text;
    throw err;
  }

  if (res.status === 204) {
    return null;
  }

  return res.json();
}

function apiGet(path) {
  return apiRequest('GET', path);
}

function apiPost(path, payload) {
  return apiRequest('POST', path, payload);
}

function apiPatch(path, payload) {
  return apiRequest('PATCH', path, payload);
}

function apiPut(path, payload) {
  return apiRequest('PUT', path, payload);
}

function apiDelete(path) {
  return apiRequest('DELETE', path);
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

function formatDateTime(isoString) {
  if (!isoString) return '';
  const date = new Date(isoString);
  return Number.isNaN(date.valueOf()) ? isoString : date.toLocaleString();
}

function createMetaItem(label, value) {
  const span = document.createElement('span');
  span.textContent = `${label}: ${value}`;
  return span;
}

function renderUserList(list) {
  const el = document.getElementById('users-list');
  if (!el) return;
  if (!list || list.length === 0) {
    el.innerHTML = '<p class="notice">No users returned.</p>';
    return;
  }

  const ul = document.createElement('ul');
  ul.className = 'endpoint-list';

  list.forEach(user => {
    const li = document.createElement('li');
    const title = document.createElement('div');
    title.innerHTML = `<strong>${user.username}</strong> <span>${user.role}</span>`;

    const meta = document.createElement('div');
    meta.className = 'meta-row';
    meta.appendChild(createMetaItem('Status', user.active ? 'Active' : 'Inactive'));
    meta.appendChild(createMetaItem('Created', formatDateTime(user.created_at)));
    meta.appendChild(createMetaItem('Updated', formatDateTime(user.updated_at)));

    const actions = document.createElement('div');
    actions.className = 'meta-row';
    const roleButton = document.createElement('button');
    roleButton.textContent = user.role === 'admin' ? 'Make workstation' : 'Make admin';
    roleButton.addEventListener('click', async () => {
      roleButton.disabled = true;
      try {
        await updateUser(user.id, { role: user.role === 'admin' ? 'workstation' : 'admin' });
      } finally {
        roleButton.disabled = false;
      }
    });

    const activeButton = document.createElement('button');
    activeButton.textContent = user.active ? 'Deactivate' : 'Activate';
    activeButton.addEventListener('click', async () => {
      activeButton.disabled = true;
      try {
        await updateUser(user.id, { active: !user.active });
      } finally {
        activeButton.disabled = false;
      }
    });

    actions.appendChild(roleButton);
    actions.appendChild(activeButton);

    if (user.id !== currentUserId) {
      const deleteButton = document.createElement('button');
      deleteButton.textContent = 'Delete';
      deleteButton.style.background = '#b91c1c';
      deleteButton.addEventListener('click', async () => {
        if (!confirm(`Delete user ${user.username}?`)) return;
        deleteButton.disabled = true;
        try {
          await deleteUser(user.id);
        } finally {
          deleteButton.disabled = false;
        }
      });
      actions.appendChild(deleteButton);
    }

    li.appendChild(title);
    li.appendChild(meta);
    li.appendChild(actions);
    ul.appendChild(li);
  });

  el.innerHTML = '';
  el.appendChild(ul);
}

function renderWorkstationList(list) {
  const el = document.getElementById('workstations-list');
  if (!el) return;
  if (!list || list.length === 0) {
    el.innerHTML = '<p class="notice">No workstations returned.</p>';
    return;
  }

  const ul = document.createElement('ul');
  ul.className = 'endpoint-list';

  list.forEach(workstation => {
    const li = document.createElement('li');
    const title = document.createElement('div');
    title.innerHTML = `<strong>${workstation.name}</strong> <span>${workstation.station_type}</span>`;

    const meta = document.createElement('div');
    meta.className = 'meta-row';
    meta.appendChild(createMetaItem('Status', workstation.active ? 'Active' : 'Inactive'));
    meta.appendChild(createMetaItem('Created', formatDateTime(workstation.created_at)));
    meta.appendChild(createMetaItem('Updated', formatDateTime(workstation.updated_at)));

    const actions = document.createElement('div');
    actions.className = 'meta-row';
    const activeButton = document.createElement('button');
    activeButton.textContent = workstation.active ? 'Disable' : 'Enable';
    activeButton.addEventListener('click', async () => {
      activeButton.disabled = true;
      try {
        await updateWorkstation(workstation.id, { active: !workstation.active });
      } finally {
        activeButton.disabled = false;
      }
    });

    const manageButton = document.createElement('button');
    manageButton.textContent = 'Manage access';
    manageButton.addEventListener('click', () => openAssignmentPanel(workstation));

    const deleteButton = document.createElement('button');
    deleteButton.textContent = 'Delete';
    deleteButton.style.background = '#b91c1c';
    deleteButton.addEventListener('click', async () => {
      if (!confirm(`Delete workstation ${workstation.name}?`)) return;
      deleteButton.disabled = true;
      try {
        await deleteWorkstation(workstation.id);
      } finally {
        deleteButton.disabled = false;
      }
    });

    actions.appendChild(activeButton);
    actions.appendChild(manageButton);
    actions.appendChild(deleteButton);

    li.appendChild(title);
    li.appendChild(meta);
    li.appendChild(actions);
    ul.appendChild(li);
  });

  el.innerHTML = '';
  el.appendChild(ul);
}

function showUnauthorizedMessage(container, message) {
  if (!container) return;
  container.innerHTML = `<p class="notice">${message}</p>`;
}

async function loadUsersPage() {
  const listElement = document.getElementById('users-list');
  if (!listElement) return;

  if (currentUserRole !== 'admin') {
    showUnauthorizedMessage(listElement, 'Sign in as an admin to manage users.');
    return;
  }

  listElement.innerHTML = '<p class="notice">Loading users...</p>';
  try {
    const users = await apiGet('/api/v1/admin/users');
    renderUserList(users);
  } catch (err) {
    showUnauthorizedMessage(listElement, err.status === 401 || err.status === 403 ? 'Unauthorized. Admin access is required.' : `Unable to load users: ${err.message}`);
  }
}

async function loadWorkstationsPage() {
  const listElement = document.getElementById('workstations-list');
  const assignmentsPanel = document.getElementById('assignments-panel');
  if (!listElement) return;

  if (assignmentsPanel) {
    assignmentsPanel.style.display = 'none';
  }

  if (currentUserRole !== 'admin') {
    showUnauthorizedMessage(listElement, 'Sign in as an admin to manage workstations.');
    return;
  }

  listElement.innerHTML = '<p class="notice">Loading workstations...</p>';
  try {
    const workstations = await apiGet('/api/v1/admin/workstations');
    renderWorkstationList(workstations);
  } catch (err) {
    showUnauthorizedMessage(listElement, err.status === 401 || err.status === 403 ? 'Unauthorized. Admin access is required.' : `Unable to load workstations: ${err.message}`);
  }
}

async function createUser(event) {
  event.preventDefault();
  const form = event.target;
  const username = form.querySelector('#user-username').value.trim();
  const password = form.querySelector('#user-password').value;
  const role = form.querySelector('#user-role').value;
  const active = form.querySelector('#user-active').checked;

  const button = form.querySelector('button[type="submit"]');
  button.disabled = true;
  try {
    await apiPost('/api/v1/admin/users', { username, password, role, active });
    form.reset();
    form.querySelector('#user-active').checked = true;
    await loadUsersPage();
  } catch (err) {
    alert(err.body || err.message);
  } finally {
    button.disabled = false;
  }
}

async function createWorkstation(event) {
  event.preventDefault();
  const form = event.target;
  const name = form.querySelector('#workstation-name').value.trim();
  const stationType = form.querySelector('#workstation-type').value.trim();
  const active = form.querySelector('#workstation-active').checked;

  const button = form.querySelector('button[type="submit"]');
  button.disabled = true;
  try {
    await apiPost('/api/v1/admin/workstations', { name, station_type: stationType, active });
    form.reset();
    form.querySelector('#workstation-active').checked = true;
    await loadWorkstationsPage();
  } catch (err) {
    alert(err.body || err.message);
  } finally {
    button.disabled = false;
  }
}

async function updateUser(id, payload) {
  await apiPatch(`/api/v1/admin/users/${id}`, payload);
  await loadUsersPage();
}

async function deleteUser(id) {
  await apiDelete(`/api/v1/admin/users/${id}`);
  await loadUsersPage();
}

async function updateWorkstation(id, payload) {
  await apiPatch(`/api/v1/admin/workstations/${id}`, payload);
  await loadWorkstationsPage();
}

async function deleteWorkstation(id) {
  await apiDelete(`/api/v1/admin/workstations/${id}`);
  await loadWorkstationsPage();
}

async function openAssignmentPanel(workstation) {
  const panel = document.getElementById('assignments-panel');
  const header = document.getElementById('assignment-header');
  const checkboxes = document.getElementById('workstation-user-checkboxes');
  const status = document.getElementById('assignment-status');

  if (!panel || !header || !checkboxes || !status) return;

  activeAssignmentWorkstationId = workstation.id;
  panel.style.display = 'block';
  status.textContent = '';
  header.textContent = `Manage access for ${workstation.name}`;
  checkboxes.innerHTML = '<p class="notice">Loading users and assignments...</p>';

  try {
    const [users, assigned] = await Promise.all([
      apiGet('/api/v1/admin/users'),
      apiGet(`/api/v1/admin/workstations/${workstation.id}/users`),
    ]);

    const assignedIds = new Set(assigned.map(user => user.id));
    if (!users || users.length === 0) {
      checkboxes.innerHTML = '<p class="notice">No users available to assign.</p>';
      return;
    }

    checkboxes.innerHTML = '';
    users.forEach(user => {
      const row = document.createElement('label');
      row.style.display = 'grid';
      row.style.gridTemplateColumns = '1fr auto';
      row.style.alignItems = 'center';
      row.style.gap = '12px';

      const text = document.createElement('span');
      text.textContent = `${user.username} (${user.role})`;
      const checkbox = document.createElement('input');
      checkbox.type = 'checkbox';
      checkbox.value = user.id;
      checkbox.checked = assignedIds.has(user.id);
      row.appendChild(text);
      row.appendChild(checkbox);
      checkboxes.appendChild(row);
    });
  } catch (err) {
    checkboxes.innerHTML = `<p class="notice">Unable to load assignment data: ${err.message}</p>`;
  }
}

async function saveWorkstationAssignments(event) {
  event.preventDefault();
  const form = event.target;
  const checkboxes = form.querySelectorAll('#workstation-user-checkboxes input[type="checkbox"]');
  const status = document.getElementById('assignment-status');
  if (!status) return;

  if (!activeAssignmentWorkstationId) {
    status.textContent = 'No workstation selected for assignment.';
    return;
  }

  const selectedUserIds = Array.from(checkboxes)
    .filter(input => input.checked)
    .map(input => input.value);

  status.textContent = 'Saving assignment…';
  try {
    await apiPut(`/api/v1/admin/workstations/${activeAssignmentWorkstationId}/users`, { user_ids: selectedUserIds });
    status.textContent = 'Workstation assignments updated.';
    await loadWorkstationsPage();
  } catch (err) {
    status.textContent = `Failed to save assignments: ${err.message}`;
  }
}

function bindUsersPage() {
  const form = document.getElementById('user-create-form');
  if (form) form.addEventListener('submit', createUser);
}

function bindWorkstationsPage() {
  const createForm = document.getElementById('workstation-create-form');
  const assignForm = document.getElementById('workstation-assign-form');
  if (createForm) createForm.addEventListener('submit', createWorkstation);
  if (assignForm) assignForm.addEventListener('submit', saveWorkstationAssignments);
}

async function initPage() {
  const page = document.body.dataset.page;
  if (page === 'users') {
    bindUsersPage();
    await loadUsersPage();
  }
  if (page === 'workstations') {
    bindWorkstationsPage();
    await loadWorkstationsPage();
  }
}

function initAdminPanel() {
  const btnW = document.getElementById('btn-load-workstations');
  const btnU = document.getElementById('btn-load-users');
  const adminPanel = document.getElementById('admin-panel');
  const usersPanel = document.getElementById('users-panel');

  if (adminPanel) adminPanel.style.display = 'none';
  if (usersPanel) usersPanel.style.display = 'none';

  if (btnW) btnW.addEventListener('click', loadWorkstations);
  if (btnU) btnU.addEventListener('click', loadUsers);
}

// Initialize
fetchHealth();
setInterval(fetchHealth, 30000);
initAdminPanel();
refreshAuth();
