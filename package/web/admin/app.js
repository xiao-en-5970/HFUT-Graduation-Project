(function () {
  const API = '/api/v1';
  const TOKEN_KEY = 'admin_token';

  function getToken() {
    return localStorage.getItem(TOKEN_KEY);
  }

  function setToken(token) {
    localStorage.setItem(TOKEN_KEY, token);
  }

  function clearToken() {
    localStorage.removeItem(TOKEN_KEY);
  }

  async function api(url, options = {}) {
    const token = getToken();
    const headers = {
      'Content-Type': 'application/json',
      ...(options.headers || {}),
    };
    if (token) headers['Authorization'] = 'Bearer ' + token;
    const res = await fetch(API + url, { ...options, headers });
    const data = await res.json().catch(() => ({}));
    if (res.status === 401 || res.status === 403) {
      clearToken();
      showLogin();
      throw new Error(data.message || '未授权');
    }
    if (data.code !== 0 && data.code !== 200) {
      throw new Error(data.message || '请求失败');
    }
    return data;
  }

  const loginScreen = document.getElementById('login-screen');
  const dashboardScreen = document.getElementById('dashboard-screen');
  const loginForm = document.getElementById('login-form');
  const loginError = document.getElementById('login-error');
  const logoutBtn = document.getElementById('logout-btn');
  const moduleContent = document.getElementById('module-content');

  function showLogin() {
    loginScreen.classList.remove('hidden');
    dashboardScreen.classList.add('hidden');
    loginError.textContent = '';
  }

  function showDashboard() {
    loginScreen.classList.add('hidden');
    dashboardScreen.classList.remove('hidden');
  }

  loginForm.addEventListener('submit', async (e) => {
    e.preventDefault();
    loginError.textContent = '';
    const username = loginForm.username.value.trim();
    const password = loginForm.password.value;
    try {
      const data = await fetch(API + '/admin/login', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ username, password }),
      }).then(r => r.json());
      if (data.code !== 0 && data.code !== 200) {
        loginError.textContent = data.message || '登录失败';
        return;
      }
      const token = data.data?.token || data.data;
      if (!token) {
        loginError.textContent = '登录响应异常';
        return;
      }
      setToken(token);
      showDashboard();
      route();
    } catch (err) {
      loginError.textContent = err.message || '网络错误';
    }
  });

  logoutBtn.addEventListener('click', () => {
    clearToken();
    showLogin();
  });

  // 路由
  const routes = ['users', 'posts', 'questions', 'answers', 'schools'];
  const routeNames = { users: '用户', posts: '帖子', questions: '提问', answers: '回答', schools: '学校' };

  function getRoute() {
    const hash = (location.hash || '#/users').slice(2) || 'users';
    return routes.includes(hash) ? hash : 'users';
  }

  function route() {
    const r = getRoute();
    document.querySelectorAll('.nav-item').forEach(el => {
      el.classList.toggle('active', el.dataset.route === r);
    });
    if (r === 'users') renderUsers();
    else if (r === 'posts') renderPosts();
    else if (r === 'questions') renderQuestions();
    else if (r === 'answers') renderAnswers();
    else if (r === 'schools') renderSchools();
  }

  window.addEventListener('hashchange', route);

  // 通用分页表格
  function renderTable(caption, columns, rows, page, total, pageSize, onPageChange, extraHeader) {
    const totalPages = Math.ceil(total / pageSize) || 1;
    let html = `<div class="module-header"><h3>${caption}</h3>`;
    if (extraHeader) html += extraHeader;
    html += '</div>';
    html += '<div class="table-wrap"><table><thead><tr>';
    columns.forEach(c => { html += `<th>${c.label}</th>`; });
    html += '<th>操作</th></tr></thead><tbody>';
    if (!rows.length) {
      html += `<tr><td colspan="${columns.length + 1}" style="text-align:center;color:var(--text-muted)">暂无数据</td></tr>`;
    } else {
      rows.forEach(row => {
        html += '<tr>';
        columns.forEach(c => { html += `<td>${c.render ? c.render(row) : (row[c.key] ?? '-')}</td>`; });
        html += `<td>${row._actions || ''}</td></tr>`;
      });
    }
    html += '</tbody></table></div>';
    html += `<div class="pagination">`;
    html += `<button ${page <= 1 ? 'disabled' : ''} data-page="${page - 1}">上一页</button>`;
    html += `<span>第 ${page}/${totalPages} 页，共 ${total} 条</span>`;
    html += `<button ${page >= totalPages ? 'disabled' : ''} data-page="${page + 1}">下一页</button>`;
    html += `</div>`;
    moduleContent.innerHTML = html;
    moduleContent.querySelectorAll('.pagination button[data-page]').forEach(btn => {
      btn.addEventListener('click', () => onPageChange(parseInt(btn.dataset.page, 10)));
    });
  }

  // 用户管理
  let userPage = 1;
  const userPageSize = 15;
  const ROLE_MAP = { 1: '普通用户', 2: '管理员', 3: '超级管理员', 4: '匿名用户' };
  const STATUS_MAP = { 1: '正常', 2: '禁用' };

  async function renderUsers() {
    moduleContent.innerHTML = '<p>加载中...</p>';
    try {
      const data = await api(`/admin/users?page=${userPage}&pageSize=${userPageSize}`);
      const list = data.data?.list || [];
      const total = data.data?.total || 0;

      const columns = [
        { key: 'id', label: 'ID' },
        { key: 'username', label: '用户名' },
        { key: 'status', label: '状态', render: r => `<span class="status-badge status-${r.status === 1 ? 'valid' : 'invalid'}">${STATUS_MAP[r.status] || r.status}</span>` },
        { key: 'role', label: '角色', render: r => `<span class="role-${r.role}">${ROLE_MAP[r.role] || r.role}</span>` },
        { key: 'created_at', label: '注册时间', render: r => (r.created_at || '').slice(0, 19) },
      ];

      list.forEach(u => {
        u._actions = '';
        if (u.status === 1) {
          u._actions += `<button class="btn btn-danger btn-sm" data-disable="${u.id}">禁用</button>`;
        } else {
          u._actions += `<button class="btn btn-success btn-sm" data-restore="${u.id}">恢复</button>`;
        }
        u._actions += ` <select class="role-select btn-sm" data-id="${u.id}" style="padding:2px 4px;background:var(--bg-dark);color:var(--text);border:1px solid #3d424d;border-radius:4px;">
          ${[1,2,3,4].map(v => `<option value="${v}" ${u.role === v ? 'selected' : ''}>${ROLE_MAP[v]}</option>`).join('')}
        </select>`;
      });

      renderTable('用户管理', columns, list, userPage, total, userPageSize, (p) => { userPage = p; renderUsers(); });

      moduleContent.querySelectorAll('[data-disable]').forEach(btn => {
        btn.addEventListener('click', () => disableUser(parseInt(btn.dataset.disable, 10)));
      });
      moduleContent.querySelectorAll('[data-restore]').forEach(btn => {
        btn.addEventListener('click', () => restoreUser(parseInt(btn.dataset.restore, 10)));
      });
      moduleContent.querySelectorAll('.role-select').forEach(sel => {
        sel.addEventListener('change', () => updateUserRole(parseInt(sel.dataset.id, 10), parseInt(sel.value, 10)));
      });
    } catch (e) {
      moduleContent.innerHTML = '<p class="error">' + e.message + '</p>';
    }
  }

  async function disableUser(id) {
    try {
      await api(`/admin/users/${id}`, { method: 'DELETE' });
      renderUsers();
    } catch (e) { alert(e.message); }
  }

  async function restoreUser(id) {
    try {
      await api(`/admin/users/${id}/restore`, { method: 'POST' });
      renderUsers();
    } catch (e) { alert(e.message); }
  }

  async function updateUserRole(id, role) {
    try {
      await api(`/admin/users/${id}/role`, { method: 'PUT', body: JSON.stringify({ role }) });
      renderUsers();
    } catch (e) { alert(e.message); }
  }

  // 帖子
  let postPage = 1;
  async function renderPosts() {
    moduleContent.innerHTML = '<p>加载中...</p>';
    try {
      const data = await api(`/admin/posts?page=${postPage}&pageSize=15&include_invalid=1`);
      const list = data.data?.list || [];
      const total = data.data?.total || 0;

      const columns = [
        { key: 'id', label: 'ID' },
        { key: 'title', label: '标题', render: r => (r.title || '').slice(0, 40) + (r.title?.length > 40 ? '...' : '') },
        { key: 'status', label: '状态', render: r => `<span class="status-badge status-${r.status === 1 ? 'valid' : 'invalid'}">${r.status === 1 ? '正常' : '已禁用'}</span>` },
        { key: 'created_at', label: '创建时间', render: r => (r.created_at || '').slice(0, 19) },
      ];
      list.forEach(a => {
        a._actions = a.status === 1
          ? `<button class="btn btn-danger btn-sm" data-disable="${a.id}">禁用</button>`
          : `<button class="btn btn-success btn-sm" data-restore="${a.id}">恢复</button>`;
      });

      renderTable('帖子管理', columns, list, postPage, total, 15, (p) => { postPage = p; renderPosts(); });

      moduleContent.querySelectorAll('[data-disable]').forEach(btn => {
        btn.addEventListener('click', () => articleAction('posts', parseInt(btn.dataset.disable, 10), 'disable'));
      });
      moduleContent.querySelectorAll('[data-restore]').forEach(btn => {
        btn.addEventListener('click', () => articleAction('posts', parseInt(btn.dataset.restore, 10), 'restore'));
      });
    } catch (e) {
      moduleContent.innerHTML = '<p class="error">' + e.message + '</p>';
    }
  }

  async function articleAction(type, id, action) {
    try {
      if (action === 'disable') {
        await api(`/admin/${type}/${id}`, { method: 'DELETE' });
      } else {
        await api(`/admin/${type}/${id}/restore`, { method: 'POST' });
      }
      if (type === 'posts') renderPosts();
      else if (type === 'questions') renderQuestions();
      else if (type === 'answers') renderAnswers();
    } catch (e) { alert(e.message); }
  }

  // 提问
  let questionPage = 1;
  async function renderQuestions() {
    moduleContent.innerHTML = '<p>加载中...</p>';
    try {
      const data = await api(`/admin/questions?page=${questionPage}&pageSize=15&include_invalid=1`);
      const list = data.data?.list || [];
      const total = data.data?.total || 0;

      const columns = [
        { key: 'id', label: 'ID' },
        { key: 'title', label: '标题', render: r => (r.title || '').slice(0, 40) + (r.title?.length > 40 ? '...' : '') },
        { key: 'status', label: '状态', render: r => `<span class="status-badge status-${r.status === 1 ? 'valid' : 'invalid'}">${r.status === 1 ? '正常' : '已禁用'}</span>` },
        { key: 'created_at', label: '创建时间', render: r => (r.created_at || '').slice(0, 19) },
      ];
      list.forEach(a => {
        a._actions = a.status === 1
          ? `<button class="btn btn-danger btn-sm" data-disable="${a.id}">禁用</button>`
          : `<button class="btn btn-success btn-sm" data-restore="${a.id}">恢复</button>`;
      });

      renderTable('提问管理', columns, list, questionPage, total, 15, (p) => { questionPage = p; renderQuestions(); });

      moduleContent.querySelectorAll('[data-disable]').forEach(btn => {
        btn.addEventListener('click', () => articleAction('questions', parseInt(btn.dataset.disable, 10), 'disable'));
      });
      moduleContent.querySelectorAll('[data-restore]').forEach(btn => {
        btn.addEventListener('click', () => articleAction('questions', parseInt(btn.dataset.restore, 10), 'restore'));
      });
    } catch (e) {
      moduleContent.innerHTML = '<p class="error">' + e.message + '</p>';
    }
  }

  // 回答
  let answerPage = 1;
  async function renderAnswers() {
    moduleContent.innerHTML = '<p>加载中...</p>';
    try {
      const data = await api(`/admin/answers?page=${answerPage}&pageSize=15&include_invalid=1`);
      const list = data.data?.list || [];
      const total = data.data?.total || 0;

      const columns = [
        { key: 'id', label: 'ID' },
        { key: 'parent_id', label: '提问ID' },
        { key: 'title', label: '标题', render: r => (r.title || r.content || '').slice(0, 40) + (r.content?.length > 40 ? '...' : '') },
        { key: 'status', label: '状态', render: r => `<span class="status-badge status-${r.status === 1 ? 'valid' : 'invalid'}">${r.status === 1 ? '正常' : '已禁用'}</span>` },
        { key: 'created_at', label: '创建时间', render: r => (r.created_at || '').slice(0, 19) },
      ];
      list.forEach(a => {
        a._actions = a.status === 1
          ? `<button class="btn btn-danger btn-sm" data-disable="${a.id}">禁用</button>`
          : `<button class="btn btn-success btn-sm" data-restore="${a.id}">恢复</button>`;
      });

      renderTable('回答管理', columns, list, answerPage, total, 15, (p) => { answerPage = p; renderAnswers(); });

      moduleContent.querySelectorAll('[data-disable]').forEach(btn => {
        btn.addEventListener('click', () => articleAction('answers', parseInt(btn.dataset.disable, 10), 'disable'));
      });
      moduleContent.querySelectorAll('[data-restore]').forEach(btn => {
        btn.addEventListener('click', () => articleAction('answers', parseInt(btn.dataset.restore, 10), 'restore'));
      });
    } catch (e) {
      moduleContent.innerHTML = '<p class="error">' + e.message + '</p>';
    }
  }

  // 学校
  let schoolPage = 1;
  async function renderSchools() {
    moduleContent.innerHTML = '<p>加载中...</p>';
    try {
      const data = await api(`/admin/schools?page=${schoolPage}&pageSize=15&include_invalid=1`);
      const list = data.data?.list || [];
      const total = data.data?.total || 0;

      const columns = [
        { key: 'id', label: 'ID' },
        { key: 'name', label: '学校名称', render: r => r.name || '-' },
        { key: 'login_url', label: '登录地址', render: r => (r.login_url || '-').slice(0, 40) },
        { key: 'status', label: '状态', render: r => `<span class="status-badge status-${r.status === 1 ? 'valid' : 'invalid'}">${r.status === 1 ? '正常' : '已下架'}</span>` },
        { key: 'created_at', label: '创建时间', render: r => (r.created_at || '').slice(0, 19) },
      ];
      list.forEach(s => {
        s._actions = s.status === 1
          ? `<button class="btn btn-danger btn-sm" data-disable="${s.id}">下架</button>`
          : `<button class="btn btn-success btn-sm" data-restore="${s.id}">恢复</button>`;
      });

      const extraHeader = '<button class="btn btn-primary" id="add-school-btn">新增学校</button>';
      renderTable('学校管理', columns, list, schoolPage, total, 15, (p) => { schoolPage = p; renderSchools(); }, extraHeader);

      document.getElementById('add-school-btn')?.addEventListener('click', showAddSchoolModal);
      moduleContent.querySelectorAll('[data-disable]').forEach(btn => {
        btn.addEventListener('click', () => schoolAction(parseInt(btn.dataset.disable, 10), 'disable'));
      });
      moduleContent.querySelectorAll('[data-restore]').forEach(btn => {
        btn.addEventListener('click', () => schoolAction(parseInt(btn.dataset.restore, 10), 'restore'));
      });
    } catch (e) {
      moduleContent.innerHTML = '<p class="error">' + e.message + '</p>';
    }
  }

  async function schoolAction(id, action) {
    try {
      if (action === 'disable') {
        await api(`/admin/schools/${id}`, { method: 'DELETE' });
      } else {
        await api(`/admin/schools/${id}/restore`, { method: 'POST' });
      }
      renderSchools();
    } catch (e) { alert(e.message); }
  }

  function showAddSchoolModal() {
    const overlay = document.createElement('div');
    overlay.className = 'modal-overlay';
    overlay.innerHTML = `
      <div class="modal">
        <h4>新增学校</h4>
        <input type="text" id="school-name" placeholder="学校名称">
        <input type="text" id="school-login-url" placeholder="登录地址">
        <div class="actions">
          <button class="btn" id="modal-cancel">取消</button>
          <button class="btn btn-primary" id="modal-confirm">确定</button>
        </div>
      </div>`;
    document.body.appendChild(overlay);

    const close = () => overlay.remove();

    overlay.querySelector('#modal-cancel').onclick = close;
    overlay.querySelector('#modal-confirm').onclick = async () => {
      const name = overlay.querySelector('#school-name').value.trim();
      const login_url = overlay.querySelector('#school-login-url').value.trim();
      try {
        await api('/admin/schools', { method: 'POST', body: JSON.stringify({ name, login_url }) });
        close();
        renderSchools();
      } catch (e) { alert(e.message); }
    };
  }

  // 初始化
  if (getToken()) {
    api('/admin/users?page=1&pageSize=1').then(() => {
      showDashboard();
      route();
    }).catch(() => {
      showLogin();
    });
  } else {
    showLogin();
  }
})();
