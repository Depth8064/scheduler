const CSRF_COOKIE_NAME = 'scheduler_csrf';
const page = document.body.dataset.page;
const authContent = document.getElementById('auth-content');
const serviceStatus = document.getElementById('service-status');
const globalBanner = document.getElementById('global-banner');

let currentUser = null;
let activeAssignmentWorkstationId = null;
let sessionRedirectScheduled = false;

function parseErrorBody(text) {
  if (!text) return '';
  try {
    const parsed = JSON.parse(text);
    if (parsed && typeof parsed.error === 'string') {
      return parsed.error;
    }
  } catch (err) {
    // Keep original body when not JSON.
  }
  return text;
}

function setGlobalBanner(message, tone = 'info') {
  if (!globalBanner) return;
  if (!message) {
    globalBanner.hidden = true;
    globalBanner.textContent = '';
    globalBanner.classList.remove('error', 'success');
    return;
  }

  globalBanner.hidden = false;
  globalBanner.textContent = message;
  globalBanner.classList.remove('error', 'success');
  if (tone === 'error' || tone === 'success') {
    globalBanner.classList.add(tone);
  }
}

function getCookie(name) {
  const matches = document.cookie.match(new RegExp('(^|; )' + name.replace(/([.$?*|{}()\[\]\\/+^])/g, '\\$1') + '=([^;]*)'));
  return matches ? decodeURIComponent(matches[2]) : undefined;
}

function withCSRF(headers = {}) {
  const token = getCookie(CSRF_COOKIE_NAME);
  if (token) headers['X-CSRF-Token'] = token;
  return headers;
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

  let res;
  try {
    res = await fetch(path, options);
  } catch (err) {
    const networkErr = new Error('Network error. Check connection and try again.');
    networkErr.cause = err;
    throw networkErr;
  }

  if (res.status === 401 && page !== 'login' && !sessionRedirectScheduled) {
    sessionRedirectScheduled = true;
    setGlobalBanner('Your session expired. Redirecting to sign in...', 'error');
    window.setTimeout(() => {
      window.location.assign('/login');
    }, 700);
  }

  if (!res.ok) {
    const text = await res.text().catch(() => '');
    const err = new Error(`${method} ${path} failed: ${res.status}`);
    err.status = res.status;
    err.body = parseErrorBody(text);
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

function roleHomePath(role) {
  return role === 'admin' ? '/dashboard' : '/operator';
}

function setRoleNavigation(role) {
  const navDashboard = document.getElementById('nav-dashboard');
  const navUsers = document.getElementById('nav-users');
  const navWorkstations = document.getElementById('nav-workstations');
  const navAdmin = document.getElementById('nav-admin');
  const navOperator = document.getElementById('nav-operator');

  [navDashboard, navUsers, navWorkstations, navAdmin, navOperator].forEach(link => {
    if (link) link.style.display = 'none';
  });

  if (role === 'admin') {
    if (navDashboard) navDashboard.style.display = 'inline-flex';
    if (navUsers) navUsers.style.display = 'inline-flex';
    if (navWorkstations) navWorkstations.style.display = 'inline-flex';
    if (navAdmin) navAdmin.style.display = 'inline-flex';
  } else if (role === 'workstation') {
    if (navOperator) navOperator.style.display = 'inline-flex';
  }
}

function renderAuthPanel(user) {
  if (!authContent) return;
  authContent.innerHTML = '';

  const wrap = document.createElement('div');
  wrap.className = 'notice';
  const identityLine = document.createElement('p');
  identityLine.textContent = `Signed in as: ${user.username}`;
  const roleLine = document.createElement('p');
  roleLine.textContent = `Role: ${user.role}`;
  const stationLine = document.createElement('p');
  const assigned = user.assigned_workstation_ids || [];
  stationLine.textContent = `Assigned workstations: ${assigned.length ? assigned.join(', ') : 'All workstations'}`;
  wrap.appendChild(identityLine);
  wrap.appendChild(roleLine);
  wrap.appendChild(stationLine);
  authContent.appendChild(wrap);

  const button = document.createElement('button');
  button.textContent = 'Sign out';
  button.addEventListener('click', async () => {
    button.disabled = true;
    try {
      await fetch('/api/v1/auth/logout', { method: 'POST', credentials: 'same-origin', headers: withCSRF() });
      window.location.assign('/login');
    } catch (err) {
      setGlobalBanner('Unable to sign out right now. Please try again.', 'error');
      button.disabled = false;
    }
  });
  authContent.appendChild(button);
}

async function fetchCurrentUser() {
  let response;
  try {
    response = await fetch('/api/v1/auth/me', { credentials: 'same-origin', cache: 'no-store' });
  } catch (err) {
    return null;
  }
  if (!response.ok) {
    return null;
  }
  return response.json();
}

async function ensureProtectedUser() {
  const user = await fetchCurrentUser();
  if (!user) {
    window.location.assign('/login');
    return null;
  }

  currentUser = user;
  setRoleNavigation(user.role);
  renderAuthPanel(user);
  return user;
}

async function fetchHealth() {
  if (!serviceStatus) return;

  try {
    serviceStatus.textContent = 'Checking scheduler status...';
    const response = await fetch('/healthz', { cache: 'no-store' });
    if (!response.ok) throw new Error('unhealthy');
    const payload = await response.json();
    serviceStatus.textContent = `Online • ${payload.status} • ${new Date(payload.time).toLocaleTimeString()}`;
    serviceStatus.classList.remove('failure');
    setGlobalBanner('');
  } catch (err) {
    serviceStatus.textContent = 'Offline • health check failed';
    serviceStatus.classList.add('failure');
  }
}

function setLoginPending(loginForm, pending) {
  if (!loginForm) return;
  const button = loginForm.querySelector('#login-submit');
  const username = loginForm.querySelector('#username');
  const password = loginForm.querySelector('#password');
  if (button) {
    button.disabled = pending;
    button.textContent = pending ? 'Signing in...' : 'Sign in';
  }
  if (username) username.disabled = pending;
  if (password) password.disabled = pending;
}

function setLoginStatus(message, tone = 'info') {
  const loginStatus = document.getElementById('login-status');
  if (!loginStatus) return;
  if (!message) {
    loginStatus.style.display = 'none';
    loginStatus.textContent = '';
    loginStatus.classList.remove('notice-success', 'notice-error');
    return;
  }

  loginStatus.style.display = 'block';
  loginStatus.textContent = message;
  loginStatus.classList.remove('notice-success', 'notice-error');
  if (tone === 'success') loginStatus.classList.add('notice-success');
  if (tone === 'error') loginStatus.classList.add('notice-error');
}

function setLoginError(message) {
  const loginError = document.getElementById('login-error');
  if (!loginError) return;
  if (!message) {
    loginError.style.display = 'none';
    loginError.textContent = '';
    return;
  }

  loginError.style.display = 'block';
  loginError.textContent = message;
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

function showUnauthorizedMessage(container, message) {
  if (!container) return;
  container.innerHTML = `<p class="notice">${message}</p>`;
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

    if (currentUser && user.id !== currentUser.user_id) {
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

async function loadUsersPage() {
  const listElement = document.getElementById('users-list');
  if (!listElement) return;
  if (!currentUser || currentUser.role !== 'admin') {
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
  if (assignmentsPanel) assignmentsPanel.style.display = 'none';

  if (!currentUser || currentUser.role !== 'admin') {
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
  if (!currentUser || currentUser.role !== 'admin') {
    alert('Admin access is required to create users.');
    return;
  }

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
    alert(err.body || err.message || 'Failed to create user');
  } finally {
    button.disabled = false;
  }
}

async function createWorkstation(event) {
  event.preventDefault();
  if (!currentUser || currentUser.role !== 'admin') {
    alert('Admin access is required to create workstations.');
    return;
  }

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
    alert(err.body || err.message || 'Failed to create workstation');
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

  status.textContent = 'Saving assignment...';
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

function bindAdminPageButtons() {
  const loadUsersBtn = document.getElementById('btn-load-users');
  const loadWorkstationsBtn = document.getElementById('btn-load-workstations');

  if (loadUsersBtn) {
    loadUsersBtn.addEventListener('click', loadUsersPage);
  }
  if (loadWorkstationsBtn) {
    loadWorkstationsBtn.addEventListener('click', loadWorkstationsPage);
  }
}

function renderOperatorAssignments(user) {
  const target = document.getElementById('operator-assigned-workstations');
  if (!target) return;

  const ids = user.assigned_workstation_ids || [];
  if (ids.length === 0) {
    target.innerHTML = '<p class="notice">No workstation assignments found.</p>';
    return;
  }

  const ul = document.createElement('ul');
  ul.className = 'endpoint-list';
  ids.forEach(id => {
    const li = document.createElement('li');
    li.innerHTML = `<span>Assigned workstation</span><code>${id}</code>`;
    ul.appendChild(li);
  });
  target.innerHTML = '';
  target.appendChild(ul);
}

function bindOperatorForm() {
  const form = document.getElementById('operator-count-form');
  const status = document.getElementById('operator-count-status');
  if (!form || !status) return;

  form.addEventListener('submit', (event) => {
    event.preventDefault();
    status.style.display = 'block';
    status.textContent = 'Count API is not available in backend yet. UI is staged for the upcoming execution-progress endpoint.';
  });
}

async function initLoginPage() {
  const loginForm = document.getElementById('login-form');
  if (!loginForm) return;

  setLoginStatus('Checking for active session...');
  const existing = await fetchCurrentUser();
  if (existing) {
    setLoginStatus('Session found. Redirecting...', 'success');
    window.location.assign(roleHomePath(existing.role));
    return;
  }
  setLoginStatus('');

  loginForm.addEventListener('submit', async (event) => {
    event.preventDefault();
    const username = loginForm.querySelector('#username').value.trim();
    const password = loginForm.querySelector('#password').value;

    setLoginPending(loginForm, true);
    setLoginError('');
    setLoginStatus('Signing in...');

    try {
      await apiPost('/api/v1/auth/login', { username, password });
      const user = await fetchCurrentUser();
      if (!user) {
        throw new Error('Unable to load user session');
      }
      setLoginStatus('Sign-in successful. Redirecting...', 'success');
      window.location.assign(roleHomePath(user.role));
    } catch (err) {
      const detail = err.status === 401
        ? 'Sign in failed. Verify your username and password and try again.'
        : (err.body || err.message || 'Sign in failed. Please try again.');
      setLoginStatus('');
      setLoginError(detail);
    } finally {
      setLoginPending(loginForm, false);
    }
  });
}

async function initProtectedPage() {
  const user = await ensureProtectedUser();
  if (!user) return;

  const expectedRole = page === 'operator' ? 'workstation' : 'admin';
  if (user.role !== expectedRole) {
    window.location.assign(roleHomePath(user.role));
    return;
  }

  await fetchHealth();
  setInterval(fetchHealth, 30000);

  if (page === 'users') {
    bindUsersPage();
    await loadUsersPage();
    return;
  }

  if (page === 'workstations') {
    bindWorkstationsPage();
    await loadWorkstationsPage();
    return;
  }

  if (page === 'admin') {
    bindAdminPageButtons();
    return;
  }

  if (page === 'operator') {
    renderOperatorAssignments(user);
    bindOperatorForm();
  }
}

async function initApp() {
  if (page === 'login') {
    await initLoginPage();
    return;
  }
  await initProtectedPage();
}

initApp();
