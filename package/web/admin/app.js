(function () {
  const API = '/api/v1';
  const TOKEN_KEY = 'admin_token';

  function getToken() { return localStorage.getItem(TOKEN_KEY); }
  function clearToken() { localStorage.removeItem(TOKEN_KEY); }

  function redirectToLogin() {
    clearToken();
    window.location.href = '/admin/login.html';
  }

  async function api(url, options = {}) {
    const token = getToken();
    const headers = { 'Content-Type': 'application/json', ...(options.headers || {}) };
    if (token) headers['Authorization'] = 'Bearer ' + token;
    const res = await fetch(API + url, { ...options, headers });
    const data = await res.json().catch(() => ({}));
    if (res.status === 401 || res.status === 403) {
      redirectToLogin();
      throw new Error(data.message || '未授权');
    }
    if (data.code !== 0 && data.code !== 200) {
      throw new Error(data.message || '请求失败');
    }
    return data;
  }

  async function apiUpload(relPath, file) {
    const token = getToken();
    const fd = new FormData();
    fd.append('file', file);
    const res = await fetch(API + '/oss/' + relPath, {
      method: 'POST',
      headers: token ? { 'Authorization': 'Bearer ' + token } : {},
      body: fd
    });
    const data = await res.json().catch(() => ({}));
    if (res.status === 401 || res.status === 403) {
      redirectToLogin();
      throw new Error(data.message || '未授权');
    }
    if (data.code !== 0 && data.code !== 200) {
      throw new Error(data.message || '上传失败');
    }
    return data.data?.url || (API + '/oss/' + relPath);
  }

  function getExt(filename) {
    const i = filename.lastIndexOf('.');
    return i >= 0 ? filename.slice(i + 1) : 'jpg';
  }

  function showModal(title, innerHTML, onConfirm, onCancel) {
    const overlay = document.createElement('div');
    overlay.className = 'modal-overlay';
    overlay.innerHTML = `<div class="modal modal-wide"><h4>${title}</h4>${innerHTML}<div class="modal-actions"><button class="btn" id="modal-cancel">取消</button><button class="btn btn-primary" id="modal-confirm">确定</button></div></div>`;
    document.body.appendChild(overlay);
    const close = () => overlay.remove();
    overlay.querySelector('#modal-cancel').onclick = () => { if (onCancel) onCancel(); close(); };
    overlay.querySelector('#modal-confirm').onclick = async () => {
      try {
        await onConfirm(overlay);
        close();
      } catch (e) { alert(e.message); }
    };
  }

  const logoutBtn = document.getElementById('logout-btn');
  const moduleContent = document.getElementById('module-content');
  logoutBtn.addEventListener('click', redirectToLogin);

  const routes = ['users', 'posts', 'questions', 'answers', 'schools'];
  function getRoute() {
    const hash = (location.hash || '#/users').slice(2) || 'users';
    return routes.includes(hash) ? hash : 'users';
  }

  function route() {
    const r = getRoute();
    document.querySelectorAll('.nav-item').forEach(el => el.classList.toggle('active', el.dataset.route === r));
    if (r === 'users') renderUsers();
    else if (r === 'posts') renderPosts();
    else if (r === 'questions') renderQuestions();
    else if (r === 'answers') renderAnswers();
    else if (r === 'schools') renderSchools();
  }
  window.addEventListener('hashchange', route);

  function renderTable(caption, columns, rows, page, total, pageSize, onPageChange, extraHeader) {
    const totalPages = Math.ceil(total / pageSize) || 1;
    let html = `<div class="module-header"><h3>${caption}</h3>${extraHeader || ''}</div><div class="table-wrap"><table><thead><tr>`;
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
    html += '</tbody></table></div><div class="pagination"><button ' + (page <= 1 ? 'disabled' : '') + ' data-page="' + (page - 1) + '">上一页</button><span>第 ' + page + '/' + totalPages + ' 页，共 ' + total + ' 条</span><button ' + (page >= totalPages ? 'disabled' : '') + ' data-page="' + (page + 1) + '">下一页</button></div>';
    moduleContent.innerHTML = html;
    moduleContent.querySelectorAll('.pagination button[data-page]').forEach(btn => {
      btn.addEventListener('click', () => onPageChange(parseInt(btn.dataset.page, 10)));
    });
  }

  const ROLE_MAP = { 1: '普通用户', 2: '管理员', 3: '超级管理员', 4: '匿名用户' };
  const STATUS_MAP = { 1: '正常', 2: '禁用' };

  let userPage = 1;
  async function renderUsers() {
    moduleContent.innerHTML = '<p>加载中...</p>';
    try {
      const data = await api(`/admin/users?page=${userPage}&pageSize=15`);
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
        u._actions = (u.status === 1 ? `<button class="btn btn-danger btn-sm" data-disable="${u.id}">禁用</button>` : `<button class="btn btn-success btn-sm" data-restore="${u.id}">恢复</button>`) +
          ` <button class="btn btn-sm" data-edit-user="${u.id}">编辑</button>` +
          ` <select class="role-select btn-sm" data-id="${u.id}" style="padding:2px 4px;background:var(--bg-dark);color:var(--text);border:1px solid #3d424d;border-radius:4px;">${[1,2,3,4].map(v => `<option value="${v}" ${u.role === v ? 'selected' : ''}>${ROLE_MAP[v]}</option>`).join('')}</select>`;
      });
      const extra = '<button class="btn btn-primary" id="add-user-btn">新建用户</button>';
      renderTable('用户管理', columns, list, userPage, total, 15, (p) => { userPage = p; renderUsers(); }, extra);

      document.getElementById('add-user-btn')?.addEventListener('click', showCreateUserModal);
      moduleContent.querySelectorAll('[data-disable]').forEach(btn => btn.addEventListener('click', () => disableUser(parseInt(btn.dataset.disable, 10))));
      moduleContent.querySelectorAll('[data-restore]').forEach(btn => btn.addEventListener('click', () => restoreUser(parseInt(btn.dataset.restore, 10))));
      moduleContent.querySelectorAll('[data-edit-user]').forEach(btn => btn.addEventListener('click', () => showEditUserModal(parseInt(btn.dataset.editUser, 10), list.find(u => u.id === parseInt(btn.dataset.editUser, 10)))));
      moduleContent.querySelectorAll('.role-select').forEach(sel => sel.addEventListener('change', () => updateUserRole(parseInt(sel.dataset.id, 10), parseInt(sel.value, 10))));
    } catch (e) { moduleContent.innerHTML = '<p class="error">' + e.message + '</p>'; }
  }

  async function disableUser(id) { try { await api(`/admin/users/${id}`, { method: 'DELETE' }); renderUsers(); } catch (e) { alert(e.message); } }
  async function restoreUser(id) { try { await api(`/admin/users/${id}/restore`, { method: 'POST' }); renderUsers(); } catch (e) { alert(e.message); } }
  async function updateUserRole(id, role) { try { await api(`/admin/users/${id}/role`, { method: 'PUT', body: JSON.stringify({ role }) }); renderUsers(); } catch (e) { alert(e.message); } }

  function showCreateUserModal() {
    showModal('新建用户', `
      <label>用户名 <input type="text" id="u-username" placeholder="用户名" required></label>
      <label>密码 <input type="password" id="u-password" placeholder="密码" required></label>
      <label>学校ID <input type="number" id="u-school_id" placeholder="0"></label>
      <label>角色 <select id="u-role"><option value="1">普通用户</option><option value="2">管理员</option><option value="3">超管</option><option value="4">匿名</option></select></label>
      <label>状态 <select id="u-status"><option value="1">正常</option><option value="2">禁用</option></select></label>
    `, async (ov) => {
      const d = await api('/admin/users', { method: 'POST', body: JSON.stringify({
        username: ov.querySelector('#u-username').value.trim(),
        password: ov.querySelector('#u-password').value,
        school_id: parseInt(ov.querySelector('#u-school_id').value || '0', 10) || 0,
        role: parseInt(ov.querySelector('#u-role').value, 10),
        status: parseInt(ov.querySelector('#u-status').value, 10)
      })});
      renderUsers();
    });
  }

  function showEditUserModal(id, user) {
    if (!user) return;
    showModal('编辑用户 #' + id, `
      <label>学校ID <input type="number" id="ue-school_id" value="${user.school_id || 0}"></label>
      <label>头像 <input type="file" id="ue-avatar" accept="image/*"></label>
      <label>背景图 <input type="file" id="ue-background" accept="image/*"></label>
    `, async (ov) => {
      const updates = { school_id: parseInt(ov.querySelector('#ue-school_id').value || '0', 10) || 0 };
      const avatarFile = ov.querySelector('#ue-avatar').files[0];
      const bgFile = ov.querySelector('#ue-background').files[0];
      if (avatarFile) {
        const ext = getExt(avatarFile.name);
        updates.avatar = await apiUpload('user/' + id + '/avatar.' + ext, avatarFile);
      }
      if (bgFile) {
        const ext = getExt(bgFile.name);
        updates.background = await apiUpload('user/' + id + '/background.' + ext, bgFile);
      }
      await api('/admin/users/' + id, { method: 'PUT', body: JSON.stringify(updates) });
      renderUsers();
    });
  }

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
        a._actions = (a.status === 1 ? `<button class="btn btn-danger btn-sm" data-disable="${a.id}">禁用</button>` : `<button class="btn btn-success btn-sm" data-restore="${a.id}">恢复</button>`) +
          ` <button class="btn btn-sm" data-edit="${a.id}">编辑</button>`;
      });
      const extra = '<button class="btn btn-primary" id="add-post-btn">新建帖子</button>';
      renderTable('帖子管理', columns, list, postPage, total, 15, (p) => { postPage = p; renderPosts(); }, extra);

      document.getElementById('add-post-btn')?.addEventListener('click', () => showArticleModal('posts', '新建帖子', null));
      moduleContent.querySelectorAll('[data-disable]').forEach(btn => btn.addEventListener('click', () => articleAction('posts', parseInt(btn.dataset.disable, 10), 'disable')));
      moduleContent.querySelectorAll('[data-restore]').forEach(btn => btn.addEventListener('click', () => articleAction('posts', parseInt(btn.dataset.restore, 10), 'restore')));
      moduleContent.querySelectorAll('[data-edit]').forEach(btn => btn.addEventListener('click', () => showArticleModal('posts', '编辑帖子', list.find(x => x.id === parseInt(btn.dataset.edit, 10)))));
    } catch (e) { moduleContent.innerHTML = '<p class="error">' + e.message + '</p>'; }
  }

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
        a._actions = (a.status === 1 ? `<button class="btn btn-danger btn-sm" data-disable="${a.id}">禁用</button>` : `<button class="btn btn-success btn-sm" data-restore="${a.id}">恢复</button>`) +
          ` <button class="btn btn-sm" data-edit="${a.id}">编辑</button>`;
      });
      const extra = '<button class="btn btn-primary" id="add-question-btn">新建提问</button>';
      renderTable('提问管理', columns, list, questionPage, total, 15, (p) => { questionPage = p; renderQuestions(); }, extra);

      document.getElementById('add-question-btn')?.addEventListener('click', () => showArticleModal('questions', '新建提问', null));
      moduleContent.querySelectorAll('[data-disable]').forEach(btn => btn.addEventListener('click', () => articleAction('questions', parseInt(btn.dataset.disable, 10), 'disable')));
      moduleContent.querySelectorAll('[data-restore]').forEach(btn => btn.addEventListener('click', () => articleAction('questions', parseInt(btn.dataset.restore, 10), 'restore')));
      moduleContent.querySelectorAll('[data-edit]').forEach(btn => btn.addEventListener('click', () => showArticleModal('questions', '编辑提问', list.find(x => x.id === parseInt(btn.dataset.edit, 10)))));
    } catch (e) { moduleContent.innerHTML = '<p class="error">' + e.message + '</p>'; }
  }

  let answerPage = 1;
  let questionListForAnswer = [];
  async function renderAnswers() {
    moduleContent.innerHTML = '<p>加载中...</p>';
    try {
      const [dataAnswer, dataQ] = await Promise.all([
        api(`/admin/answers?page=${answerPage}&pageSize=15&include_invalid=1`),
        api(`/admin/questions?page=1&pageSize=500&include_invalid=1`)
      ]);
      questionListForAnswer = dataQ.data?.list || [];
      const list = dataAnswer.data?.list || [];
      const total = dataAnswer.data?.total || 0;
      const columns = [
        { key: 'id', label: 'ID' },
        { key: 'parent_id', label: '提问ID' },
        { key: 'title', label: '标题', render: r => (r.title || r.content || '').slice(0, 40) + (r.content?.length > 40 ? '...' : '') },
        { key: 'status', label: '状态', render: r => `<span class="status-badge status-${r.status === 1 ? 'valid' : 'invalid'}">${r.status === 1 ? '正常' : '已禁用'}</span>` },
        { key: 'created_at', label: '创建时间', render: r => (r.created_at || '').slice(0, 19) },
      ];
      list.forEach(a => {
        a._actions = (a.status === 1 ? `<button class="btn btn-danger btn-sm" data-disable="${a.id}">禁用</button>` : `<button class="btn btn-success btn-sm" data-restore="${a.id}">恢复</button>`) +
          ` <button class="btn btn-sm" data-edit="${a.id}">编辑</button>`;
      });
      const extra = '<button class="btn btn-primary" id="add-answer-btn">新建回答</button>';
      renderTable('回答管理', columns, list, answerPage, total, 15, (p) => { answerPage = p; renderAnswers(); }, extra);

      document.getElementById('add-answer-btn')?.addEventListener('click', () => showArticleModal('answers', '新建回答', null));
      moduleContent.querySelectorAll('[data-disable]').forEach(btn => btn.addEventListener('click', () => articleAction('answers', parseInt(btn.dataset.disable, 10), 'disable')));
      moduleContent.querySelectorAll('[data-restore]').forEach(btn => btn.addEventListener('click', () => articleAction('answers', parseInt(btn.dataset.restore, 10), 'restore')));
      moduleContent.querySelectorAll('[data-edit]').forEach(btn => btn.addEventListener('click', () => showArticleModal('answers', '编辑回答', list.find(x => x.id === parseInt(btn.dataset.edit, 10)))));
    } catch (e) { moduleContent.innerHTML = '<p class="error">' + e.message + '</p>'; }
  }

  const TYPE_MAP = { posts: 'posts', questions: 'questions', answers: 'answers' };

  function showArticleModal(type, title, row) {
    const isAnswer = type === 'answers';
    const parentOpts = isAnswer ? questionListForAnswer.map(q => `<option value="${q.id}" ${row?.parent_id === q.id ? 'selected' : ''}>#${q.id} ${(q.title||'').slice(0,30)}</option>`).join('') : '';
    const parentDefault = isAnswer ? '<option value="">请选择提问</option>' : '';
    const parentHtml = isAnswer ? `<label>父提问ID <select id="art-parent_id" required>${parentDefault}${parentOpts}</select></label>` : '';
    showModal(title, `
      ${parentHtml}
      <label>标题 <input type="text" id="art-title" value="${row ? (row.title || '') : ''}" placeholder="标题" required></label>
      <label>内容 <textarea id="art-content" rows="4" placeholder="正文">${row ? (row.content || '') : ''}</textarea></label>
      <label>用户ID <input type="number" id="art-user_id" value="${row?.user_id || ''}" placeholder="0"></label>
      <label>学校ID <input type="number" id="art-school_id" value="${row?.school_id || ''}" placeholder="0"></label>
      <label>公开 <select id="art-publish_status"><option value="1" ${row?.publish_status === 1 ? 'selected' : ''}>私密</option><option value="2" ${!row || row.publish_status === 2 ? 'selected' : ''}>公开</option></select></label>
      <label>图片 <input type="file" id="art-images" accept="image/*" multiple></label>
    `, async (ov) => {
      const payload = {
        title: ov.querySelector('#art-title').value.trim(),
        content: ov.querySelector('#art-content').value.trim(),
        user_id: parseInt(ov.querySelector('#art-user_id').value || '0', 10) || undefined,
        school_id: parseInt(ov.querySelector('#art-school_id').value || '0', 10) || undefined,
        publish_status: parseInt(ov.querySelector('#art-publish_status').value, 10)
      };
      if (isAnswer) {
        const pid = parseInt(ov.querySelector('#art-parent_id').value, 10);
        if (!pid) throw new Error('请选择父提问');
        payload.parent_id = pid;
      }
      const files = ov.querySelector('#art-images').files;

      if (row) {
        let images = row.images && Array.isArray(row.images) ? [...row.images] : [];
        if (files && files.length) {
          for (let i = 0; i < files.length; i++) {
            const ext = getExt(files[i].name);
            const url = await apiUpload(`article/${row.id}/image_${images.length + i + 1}.${ext}`, files[i]);
            images.push(url);
          }
        }
        await api(`/admin/${type}/${row.id}`, { method: 'PUT', body: JSON.stringify({
          title: payload.title, content: payload.content,
          publish_status: payload.publish_status,
          ...(images.length ? { images } : {})
        })});
      } else {
        const createPayload = { title: payload.title, content: payload.content };
        if (payload.user_id) createPayload.user_id = payload.user_id;
        if (payload.school_id) createPayload.school_id = payload.school_id;
        createPayload.publish_status = payload.publish_status;
        if (isAnswer) createPayload.parent_id = payload.parent_id;
        const d = await api(`/admin/${type}`, { method: 'POST', body: JSON.stringify(createPayload) });
        const newId = d.data?.id;
        if (newId && files && files.length) {
          const images = [];
          for (let i = 0; i < files.length; i++) {
            const ext = getExt(files[i].name);
            const url = await apiUpload(`article/${newId}/image_${i + 1}.${ext}`, files[i]);
            images.push(url);
          }
          await api(`/admin/${type}/${newId}`, { method: 'PUT', body: JSON.stringify({ images }) });
        }
      }
      if (type === 'posts') renderPosts();
      else if (type === 'questions') renderQuestions();
      else renderAnswers();
    });
  }

  async function articleAction(type, id, action) {
    try {
      if (action === 'disable') await api(`/admin/${type}/${id}`, { method: 'DELETE' });
      else await api(`/admin/${type}/${id}/restore`, { method: 'POST' });
      if (type === 'posts') renderPosts();
      else if (type === 'questions') renderQuestions();
      else renderAnswers();
    } catch (e) { alert(e.message); }
  }

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
        s._actions = (s.status === 1 ? `<button class="btn btn-danger btn-sm" data-disable="${s.id}">下架</button>` : `<button class="btn btn-success btn-sm" data-restore="${s.id}">恢复</button>`) +
          ` <button class="btn btn-sm" data-edit-school="${s.id}">编辑</button>`;
      });
      const extra = '<button class="btn btn-primary" id="add-school-btn">新建学校</button>';
      renderTable('学校管理', columns, list, schoolPage, total, 15, (p) => { schoolPage = p; renderSchools(); }, extra);

      document.getElementById('add-school-btn')?.addEventListener('click', showCreateSchoolModal);
      moduleContent.querySelectorAll('[data-disable]').forEach(btn => btn.addEventListener('click', () => schoolAction(parseInt(btn.dataset.disable, 10), 'disable')));
      moduleContent.querySelectorAll('[data-restore]').forEach(btn => btn.addEventListener('click', () => schoolAction(parseInt(btn.dataset.restore, 10), 'restore')));
      moduleContent.querySelectorAll('[data-edit-school]').forEach(btn => btn.addEventListener('click', () => showEditSchoolModal(list.find(s => s.id === parseInt(btn.dataset.editSchool, 10)))));
    } catch (e) { moduleContent.innerHTML = '<p class="error">' + e.message + '</p>'; }
  }

  async function schoolAction(id, action) {
    try {
      if (action === 'disable') await api(`/admin/schools/${id}`, { method: 'DELETE' });
      else await api(`/admin/schools/${id}/restore`, { method: 'POST' });
      renderSchools();
    } catch (e) { alert(e.message); }
  }

  function showCreateSchoolModal() {
    showModal('新建学校', `
      <label>学校名称 <input type="text" id="s-name" placeholder="学校名称"></label>
      <label>登录地址 <input type="text" id="s-login_url" placeholder="https://"></label>
    `, async (ov) => {
      await api('/admin/schools', { method: 'POST', body: JSON.stringify({
        name: ov.querySelector('#s-name').value.trim(),
        login_url: ov.querySelector('#s-login_url').value.trim()
      })});
      renderSchools();
    });
  }

  function showEditSchoolModal(school) {
    if (!school) return;
    showModal('编辑学校 #' + school.id, `
      <label>学校名称 <input type="text" id="se-name" value="${school.name || ''}" placeholder="学校名称"></label>
      <label>登录地址 <input type="text" id="se-login_url" value="${school.login_url || ''}" placeholder="https://"></label>
    `, async (ov) => {
      await api('/admin/schools/' + school.id, { method: 'PUT', body: JSON.stringify({
        name: ov.querySelector('#se-name').value.trim(),
        login_url: ov.querySelector('#se-login_url').value.trim()
      })});
      renderSchools();
    });
  }

  if (!getToken()) { redirectToLogin(); return; }
  api('/admin/users?page=1&pageSize=1').then(() => route()).catch(() => redirectToLogin());
})();
