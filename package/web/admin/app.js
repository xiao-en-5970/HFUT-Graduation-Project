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

  async function apiUpload(relPath, file, options = {}) {
    return new Promise((resolve, reject) => {
      const xhr = new XMLHttpRequest();
      const fd = new FormData();
      fd.append('file', file);
      const token = getToken();
      const uploadUrl = API + '/oss/' + String(relPath).replace(/^\//, '');

      xhr.upload.onprogress = (e) => {
        if (e.lengthComputable && options.onProgress) {
          options.onProgress(Math.round((e.loaded / e.total) * 100), e.loaded, e.total);
        }
      };

      xhr.onload = () => {
        if (xhr.status === 401 || xhr.status === 403) {
          redirectToLogin();
          reject(new Error('未授权'));
          return;
        }
        let data = {};
        try { data = JSON.parse(xhr.responseText); } catch (_) {}
        if (data.code !== 0 && data.code !== 200) {
          reject(new Error(data.message || '上传失败'));
          return;
        }
        resolve(data.data?.url || uploadUrl);
      };
      xhr.onerror = () => reject(new Error('上传失败，请检查网络或 API 地址'));

      xhr.open('POST', uploadUrl);
      if (token) xhr.setRequestHeader('Authorization', 'Bearer ' + token);
      xhr.send(fd);
    });
  }

    /** multipart/form-data（如管理端商品图上传） */
    async function apiForm(url, formData, options = {}) {
        const token = getToken();
        const headers = {...(options.headers || {})};
        if (token) headers['Authorization'] = 'Bearer ' + token;
        const res = await fetch(API + url, {method: options.method || 'POST', headers, body: formData});
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

  function ensureUploadProgress(overlay) {
    let wrap = overlay.querySelector('.upload-progress-wrap');
    if (!wrap) {
      wrap = document.createElement('div');
      wrap.className = 'upload-progress-wrap';
      wrap.innerHTML = '<div class="upload-progress-label">上传中...</div><div class="upload-progress-bar"><div class="upload-progress-fill"></div></div>';
      const modal = overlay.querySelector('.modal');
      const actions = overlay.querySelector('.modal-actions');
      modal.insertBefore(wrap, actions);
    }
    return {
      show(label) {
        wrap.style.display = 'block';
        wrap.querySelector('.upload-progress-label').textContent = label || '上传中...';
        wrap.querySelector('.upload-progress-fill').style.width = '0%';
      },
      update(pct, label) {
        wrap.querySelector('.upload-progress-fill').style.width = pct + '%';
        if (label != null) wrap.querySelector('.upload-progress-label').textContent = label;
      },
      hide() { wrap.style.display = 'none'; }
    };
  }

  function getExt(filename) {
    const i = filename.lastIndexOf('.');
    return i >= 0 ? filename.slice(i + 1) : 'jpg';
  }

  function showModal(title, innerHTML, onConfirm, onCancel, onMount) {
    const overlay = document.createElement('div');
    overlay.className = 'modal-overlay';
    overlay.innerHTML = `<div class="modal modal-wide"><h4>${title}</h4>${innerHTML}<div class="modal-actions"><button class="btn" id="modal-cancel">取消</button><button class="btn btn-primary" id="modal-confirm">确定</button></div></div>`;
    document.body.appendChild(overlay);
    if (onMount) onMount(overlay);
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

    const routes = ['users', 'posts', 'questions', 'answers', 'goods', 'orders', 'schools', 'bind-school'];
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
    else if (r === 'goods') renderGoods();
    else if (r === 'orders') renderOrders();
    else if (r === 'schools') renderSchools();
    else if (r === 'bind-school') renderBindSchool();
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
  const ARTICLE_STATUS_MAP = {1: '正常', 2: '已删除', 3: '草稿'};
    const GOOD_STATUS_MAP = {1: '在售', 2: '下架', 3: '已售出'};
    const GOODS_TYPE_MAP = {1: '送货上门', 2: '自提', 3: '在线商品'};

  /** 列表单图展示（头像/背景），与帖子图片同一套 display 逻辑 */
  function renderListImage(url, title) {
    if (!url) return '-';
    return `<div class="list-images-wrap"><a href="${url}" target="_blank" title="${title || '图片'}"><img src="${url}" alt="${title || '图片'}" class="list-thumb" onerror="this.parentElement.style.display='none'"/></a></div>`;
  }
  /** 列表多图展示（帖子/提问/回答），与单图同一套 display 逻辑 */
  function renderListImages(urls) {
    const imgs = urls && Array.isArray(urls) ? urls : [];
    if (!imgs.length) return '-';
    return `<div class="list-images-wrap">${imgs.map((u, i) => `<a href="${u || ''}" target="_blank" title="图${i + 1}"><img src="${u || ''}" alt="图${i + 1}" class="list-thumb" onerror="this.parentElement.style.display='none'"/></a>`).join('')}</div>`;
  }

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
        { key: 'school_id', label: '学校', render: r => r.school_id ? `#${r.school_id}` : '<span class="text-muted">未绑定</span>' },
        { key: 'avatar', label: '头像', render: r => renderListImage(r.avatar, '头像') },
        { key: 'background', label: '背景图', render: r => renderListImage(r.background, '背景图') },
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
      moduleContent.querySelectorAll('[data-edit-user]').forEach(btn => btn.addEventListener('click', () => {
        const user = list.find(u => u.id == btn.dataset.editUser || Number(u.id) === parseInt(btn.dataset.editUser, 10));
        if (!user) {
          alert('未找到对应用户');
          return;
        }
        showEditUserModal(Number(user.id) || parseInt(btn.dataset.editUser, 10), user);
      }));
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
    const existingAvatar = user.avatar ? `<div class="existing-images"><span class="label">已上传头像：</span><a href="${user.avatar}" target="_blank" title="头像"><img src="${user.avatar}" alt="头像" onerror="this.parentElement.style.display='none'"/></a></div>` : '';
    const existingBg = user.background ? `<div class="existing-images"><span class="label">已上传背景图：</span><a href="${user.background}" target="_blank" title="背景图"><img src="${user.background}" alt="背景图" onerror="this.parentElement.style.display='none'"/></a></div>` : '';
    showModal('编辑用户 #' + id, `
      <label>学校ID <input type="number" id="ue-school_id" value="${user.school_id || 0}"></label>
      ${existingAvatar}
      <label>头像（可更换）<input type="file" id="ue-avatar" accept="image/*"></label>
      <div id="ue-avatar-preview" class="file-preview"></div>
      ${existingBg}
      <label>背景图（可更换）<input type="file" id="ue-background" accept="image/*"></label>
      <div id="ue-background-preview" class="file-preview"></div>
    `, async (ov) => {
      const updates = { school_id: parseInt(ov.querySelector('#ue-school_id').value || '0', 10) || 0 };
      const avatarFile = ov.querySelector('#ue-avatar').files[0];
      const bgFile = ov.querySelector('#ue-background').files[0];
      const progress = ensureUploadProgress(ov);
      const confirmBtn = ov.querySelector('#modal-confirm');
      try {
        if (avatarFile || bgFile) {
          confirmBtn.disabled = true;
          if (avatarFile) {
            progress.show('上传头像...');
            const ext = getExt(avatarFile.name);
            updates.avatar = await apiUpload('user/' + id + '/avatar.' + ext, avatarFile, {
              onProgress: (p) => progress.update(p, `上传头像 ${p}%`)
            });
            progress.update(100);
          }
          if (bgFile) {
            progress.show('上传背景图...');
            const ext = getExt(bgFile.name);
            updates.background = await apiUpload('user/' + id + '/background.' + ext, bgFile, {
              onProgress: (p) => progress.update(p, `上传背景图 ${p}%`)
            });
            progress.update(100);
          }
        }
        await api('/admin/users/' + id, { method: 'PUT', body: JSON.stringify(updates) });
        renderUsers();
      } finally {
        progress.hide();
        confirmBtn.disabled = false;
      }
    }, null, (ov) => {
      ov.querySelector('#ue-avatar')?.addEventListener('change', function() {
        const preview = ov.querySelector('#ue-avatar-preview');
        if (!preview) return;
        preview.innerHTML = '';
        if (this.files?.[0]) {
          const img = document.createElement('img');
          img.src = URL.createObjectURL(this.files[0]);
          img.alt = '头像预览';
          img.className = 'preview-thumb';
          preview.appendChild(img);
        }
      });
      ov.querySelector('#ue-background')?.addEventListener('change', function() {
        const preview = ov.querySelector('#ue-background-preview');
        if (!preview) return;
        preview.innerHTML = '';
        if (this.files?.[0]) {
          const img = document.createElement('img');
          img.src = URL.createObjectURL(this.files[0]);
          img.alt = '背景图预览';
          img.className = 'preview-thumb';
          preview.appendChild(img);
        }
      });
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
        { key: 'images', label: '图片', render: r => renderListImages(r.images) },
        {
          key: 'status',
          label: '状态',
          render: r => `<span class="status-badge status-${r.status === 1 ? 'valid' : r.status === 3 ? 'draft' : 'invalid'}">${ARTICLE_STATUS_MAP[r.status] || r.status}</span>`
        },
        { key: 'created_at', label: '创建时间', render: r => (r.created_at || '').slice(0, 19) },
      ];
      list.forEach(a => {
        if (a.status === 1) {
          a._actions = `<button class="btn btn-danger btn-sm" data-disable="${a.id}">禁用</button> <button class="btn btn-sm" data-edit="${a.id}">编辑</button>`;
        } else if (a.status === 2) {
          a._actions = `<button class="btn btn-success btn-sm" data-restore="${a.id}">恢复</button> <button class="btn btn-sm" data-edit="${a.id}">编辑</button>`;
        } else {
          a._actions = `<button class="btn btn-primary btn-sm" data-publish="${a.id}">发布</button> <button class="btn btn-sm" data-edit="${a.id}">编辑</button>`;
        }
      });
      const extra = '<button class="btn btn-primary" id="add-post-btn">新建帖子</button>';
      renderTable('帖子管理', columns, list, postPage, total, 15, (p) => { postPage = p; renderPosts(); }, extra);

      document.getElementById('add-post-btn')?.addEventListener('click', () => createDraftThenShowModal('posts', '编辑帖子'));
      moduleContent.querySelectorAll('[data-disable]').forEach(btn => btn.addEventListener('click', () => articleAction('posts', parseInt(btn.dataset.disable, 10), 'disable')));
      moduleContent.querySelectorAll('[data-restore]').forEach(btn => btn.addEventListener('click', () => articleAction('posts', parseInt(btn.dataset.restore, 10), 'restore')));
      moduleContent.querySelectorAll('[data-publish]').forEach(btn => btn.addEventListener('click', () => articleAction('posts', parseInt(btn.dataset.publish, 10), 'publish')));
      moduleContent.querySelectorAll('[data-edit]').forEach(btn => btn.addEventListener('click', () => {
        const row = list.find(x => x.id == btn.dataset.edit || Number(x.id) === parseInt(btn.dataset.edit, 10));
        if (!row) {
          alert('未找到对应记录');
          return;
        }
        showArticleModal('posts', '编辑帖子', row);
      }));
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
        { key: 'images', label: '图片', render: r => renderListImages(r.images) },
        {
          key: 'status',
          label: '状态',
          render: r => `<span class="status-badge status-${r.status === 1 ? 'valid' : r.status === 3 ? 'draft' : 'invalid'}">${ARTICLE_STATUS_MAP[r.status] || r.status}</span>`
        },
        { key: 'created_at', label: '创建时间', render: r => (r.created_at || '').slice(0, 19) },
      ];
      list.forEach(a => {
        if (a.status === 1) {
          a._actions = `<button class="btn btn-danger btn-sm" data-disable="${a.id}">禁用</button> <button class="btn btn-sm" data-edit="${a.id}">编辑</button>`;
        } else if (a.status === 2) {
          a._actions = `<button class="btn btn-success btn-sm" data-restore="${a.id}">恢复</button> <button class="btn btn-sm" data-edit="${a.id}">编辑</button>`;
        } else {
          a._actions = `<button class="btn btn-primary btn-sm" data-publish="${a.id}">发布</button> <button class="btn btn-sm" data-edit="${a.id}">编辑</button>`;
        }
      });
      const extra = '<button class="btn btn-primary" id="add-question-btn">新建提问</button>';
      renderTable('提问管理', columns, list, questionPage, total, 15, (p) => { questionPage = p; renderQuestions(); }, extra);

      document.getElementById('add-question-btn')?.addEventListener('click', () => createDraftThenShowModal('questions', '编辑提问'));
      moduleContent.querySelectorAll('[data-disable]').forEach(btn => btn.addEventListener('click', () => articleAction('questions', parseInt(btn.dataset.disable, 10), 'disable')));
      moduleContent.querySelectorAll('[data-restore]').forEach(btn => btn.addEventListener('click', () => articleAction('questions', parseInt(btn.dataset.restore, 10), 'restore')));
      moduleContent.querySelectorAll('[data-publish]').forEach(btn => btn.addEventListener('click', () => articleAction('questions', parseInt(btn.dataset.publish, 10), 'publish')));
      moduleContent.querySelectorAll('[data-edit]').forEach(btn => btn.addEventListener('click', () => {
        const row = list.find(x => x.id == btn.dataset.edit || Number(x.id) === parseInt(btn.dataset.edit, 10));
        if (!row) {
          alert('未找到对应记录');
          return;
        }
        showArticleModal('questions', '编辑提问', row);
      }));
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
        { key: 'images', label: '图片', render: r => renderListImages(r.images) },
        {
          key: 'status',
          label: '状态',
          render: r => `<span class="status-badge status-${r.status === 1 ? 'valid' : r.status === 3 ? 'draft' : 'invalid'}">${ARTICLE_STATUS_MAP[r.status] || r.status}</span>`
        },
        { key: 'created_at', label: '创建时间', render: r => (r.created_at || '').slice(0, 19) },
      ];
      list.forEach(a => {
        if (a.status === 1) {
          a._actions = `<button class="btn btn-danger btn-sm" data-disable="${a.id}">禁用</button> <button class="btn btn-sm" data-edit="${a.id}">编辑</button>`;
        } else if (a.status === 2) {
          a._actions = `<button class="btn btn-success btn-sm" data-restore="${a.id}">恢复</button> <button class="btn btn-sm" data-edit="${a.id}">编辑</button>`;
        } else {
          a._actions = `<button class="btn btn-primary btn-sm" data-publish="${a.id}">发布</button> <button class="btn btn-sm" data-edit="${a.id}">编辑</button>`;
        }
      });
      const extra = '<button class="btn btn-primary" id="add-answer-btn">新建回答</button>';
      renderTable('回答管理', columns, list, answerPage, total, 15, (p) => { answerPage = p; renderAnswers(); }, extra);

      document.getElementById('add-answer-btn')?.addEventListener('click', () => showAnswerParentPicker());
      moduleContent.querySelectorAll('[data-disable]').forEach(btn => btn.addEventListener('click', () => articleAction('answers', parseInt(btn.dataset.disable, 10), 'disable')));
      moduleContent.querySelectorAll('[data-restore]').forEach(btn => btn.addEventListener('click', () => articleAction('answers', parseInt(btn.dataset.restore, 10), 'restore')));
      moduleContent.querySelectorAll('[data-publish]').forEach(btn => btn.addEventListener('click', () => articleAction('answers', parseInt(btn.dataset.publish, 10), 'publish')));
      moduleContent.querySelectorAll('[data-edit]').forEach(btn => btn.addEventListener('click', () => {
        const row = list.find(x => x.id == btn.dataset.edit || Number(x.id) === parseInt(btn.dataset.edit, 10));
        if (!row) {
          alert('未找到对应记录');
          return;
        }
        showArticleModal('answers', '编辑回答', row);
      }));
    } catch (e) { moduleContent.innerHTML = '<p class="error">' + e.message + '</p>'; }
  }

    let goodPage = 1;

    async function renderGoods() {
        moduleContent.innerHTML = '<p>加载中...</p>';
        try {
            const data = await api(`/admin/goods?page=${goodPage}&pageSize=15&include_invalid=1`);
            const list = data.data?.list || [];
            const total = data.data?.total || 0;
            const columns = [
                {key: 'id', label: 'ID'},
                {
                    key: 'title',
                    label: '标题',
                    render: r => (r.title || '').slice(0, 36) + ((r.title || '').length > 36 ? '...' : '')
                },
                {key: 'images', label: '图片', render: r => renderListImages(r.images)},
                {key: 'user_id', label: '用户ID', render: r => r.user_id ?? '-'},
                {key: 'school_id', label: '学校ID', render: r => r.school_id ?? '-'},
                {key: 'goods_type', label: '类别', render: r => GOODS_TYPE_MAP[r.goods_type] || r.goods_type || '-'},
                {key: 'pickup_addr', label: '自提地址', render: r => (r.pickup_addr || '-').slice(0, 24)},
                {key: 'price', label: '价格(分)', render: r => r.price ?? 0},
                {key: 'stock', label: '库存', render: r => r.stock ?? 0},
                {key: 'good_status', label: '销售状态', render: r => GOOD_STATUS_MAP[r.good_status] || r.good_status},
                {
                    key: 'status',
                    label: '记录',
                    render: r => `<span class="status-badge status-${r.status === 1 ? 'valid' : 'invalid'}">${r.status === 1 ? '正常' : '已禁用'}</span>`
                },
                {key: 'created_at', label: '创建时间', render: r => (r.created_at || '').slice(0, 19)},
            ];
            list.forEach(g => {
                const ok = g.status === 1;
                if (ok) {
                    g._actions = `<button class="btn btn-danger btn-sm" data-good-disable="${g.id}">禁用</button> ` +
                        `<button class="btn btn-sm" data-good-edit="${g.id}">编辑</button> ` +
                        (g.good_status === 1
                            ? `<button class="btn btn-sm" data-good-off="${g.id}">下架</button>`
                            : `<button class="btn btn-primary btn-sm" data-good-pub="${g.id}">上架</button>`);
                } else {
                    g._actions = `<button class="btn btn-success btn-sm" data-good-restore="${g.id}">恢复</button> <button class="btn btn-sm" data-good-edit="${g.id}">编辑</button>`;
                }
            });
            const extra = '<button class="btn btn-primary" id="add-good-btn">新建商品</button>';
            renderTable('商品管理', columns, list, goodPage, total, 15, (p) => {
                goodPage = p;
                renderGoods();
            }, extra);

            document.getElementById('add-good-btn')?.addEventListener('click', () => showGoodModal(null));
            moduleContent.querySelectorAll('[data-good-disable]').forEach(btn => btn.addEventListener('click', () => goodAction(parseInt(btn.dataset.goodDisable, 10), 'disable')));
            moduleContent.querySelectorAll('[data-good-restore]').forEach(btn => btn.addEventListener('click', () => goodAction(parseInt(btn.dataset.goodRestore, 10), 'restore')));
            moduleContent.querySelectorAll('[data-good-pub]').forEach(btn => btn.addEventListener('click', () => goodAction(parseInt(btn.dataset.goodPub, 10), 'publish')));
            moduleContent.querySelectorAll('[data-good-off]').forEach(btn => btn.addEventListener('click', () => goodAction(parseInt(btn.dataset.goodOff, 10), 'off')));
            moduleContent.querySelectorAll('[data-good-edit]').forEach(btn => btn.addEventListener('click', () => {
                const row = list.find(x => String(x.id) === String(btn.dataset.goodEdit));
                if (!row) {
                    alert('未找到对应记录');
                    return;
                }
                showGoodModal(row);
            }));
        } catch (e) {
            moduleContent.innerHTML = '<p class="error">' + e.message + '</p>';
        }
    }

    async function goodAction(id, action) {
        try {
            if (action === 'disable') await api(`/admin/goods/${id}`, {method: 'DELETE'});
            else if (action === 'restore') await api(`/admin/goods/${id}/restore`, {method: 'POST'});
            else if (action === 'publish') await api(`/admin/goods/${id}/publish`, {method: 'POST'});
            else if (action === 'off') await api(`/admin/goods/${id}/off-shelf`, {method: 'POST'});
            renderGoods();
        } catch (e) {
            alert(e.message);
        }
    }

    function showGoodModal(row) {
        const isEdit = row && row.id != null;
        const uid = row?.user_id ?? '';
        const sid = row?.school_id ?? '';
        const initialImgs = row?.images && Array.isArray(row.images) ? [...row.images] : [];
        showModal(isEdit ? '编辑商品 #' + row.id : '新建商品', `
      <label>用户ID <input type="number" id="g-user_id" value="${uid}" required></label>
      <label>学校ID <input type="number" id="g-school_id" value="${sid}" required></label>
      <label>标题 <input type="text" id="g-title" value=""></label>
      <label>内容 <textarea id="g-content" rows="4" required></textarea></label>
      <label>价格（分）<input type="number" id="g-price" value="${row?.price ?? 0}" min="0"></label>
      <label>标价（分）<input type="number" id="g-marked_price" value="${row?.marked_price ?? 0}" min="0"></label>
      <label>库存 <input type="number" id="g-stock" value="${row?.stock ?? 0}" min="0"></label>
      <label>商品类别 <select id="g-goods_type">
        <option value="1">送货上门</option>
        <option value="2">自提</option>
        <option value="3">在线商品</option>
      </select></label>
      <label>自提地址（自提类填写）<input type="text" id="g-pickup_addr" placeholder="约定提货地点"></label>
      <label>销售状态 <select id="g-good_status">
        <option value="1">在售</option>
        <option value="2">下架</option>
        <option value="3">已售出</option>
      </select></label>
      <div class="art-images-editor">
        <span class="label">图片（可删除、排序、新增）</span>
        <div id="g-images-list" class="art-images-list"></div>
        <label class="art-add-label">添加图片 <input type="file" id="g-images-add" accept="image/*" multiple></label>
      </div>
    `, async (ov) => {
            const payload = {
                user_id: parseInt(ov.querySelector('#g-user_id').value || '0', 10),
                school_id: parseInt(ov.querySelector('#g-school_id').value || '0', 10),
                title: ov.querySelector('#g-title').value.trim(),
                content: ov.querySelector('#g-content').value.trim(),
                price: parseInt(ov.querySelector('#g-price').value || '0', 10),
                marked_price: parseInt(ov.querySelector('#g-marked_price').value || '0', 10),
                stock: parseInt(ov.querySelector('#g-stock').value || '0', 10),
                goods_type: parseInt(ov.querySelector('#g-goods_type').value, 10),
                pickup_addr: ov.querySelector('#g-pickup_addr').value.trim(),
                good_status: parseInt(ov.querySelector('#g-good_status').value, 10)
            };
            if (!payload.user_id || !payload.school_id) throw new Error('请填写用户ID与学校ID');
            if (!payload.title || !payload.content) throw new Error('请填写标题与内容');

            let images = Array.from(ov.querySelectorAll('#g-images-list .art-img-item:not(.art-img-pending)')).map(el => el.dataset.url).filter(Boolean);
            const pending = Array.from(ov.querySelectorAll('#g-images-list .art-img-pending')).map(el => el._file).filter(Boolean);
            const progress = ensureUploadProgress(ov);
            const confirmBtn = ov.querySelector('#modal-confirm');

            try {
                confirmBtn.disabled = true;
                if (isEdit) {
                    if (pending.length) {
                        const fd = new FormData();
                        pending.forEach(f => fd.append('files', f));
                        progress.show('上传图片...');
                        const up = await apiForm(`/admin/goods/${row.id}/images`, fd);
                        (up.data?.urls || []).forEach(u => images.push(u));
                    }
                    await api(`/admin/goods/${row.id}`, {
                        method: 'PUT',
                        body: JSON.stringify({
                            user_id: payload.user_id,
                            school_id: payload.school_id,
                            title: payload.title,
                            content: payload.content,
                            price: payload.price,
                            marked_price: payload.marked_price,
                            stock: payload.stock,
                            goods_type: payload.goods_type,
                            pickup_addr: payload.pickup_addr,
                            good_status: payload.good_status,
                            images
                        })
                    });
                } else {
                    const createRes = await api('/admin/goods', {
                        method: 'POST',
                        body: JSON.stringify({
                            user_id: payload.user_id,
                            school_id: payload.school_id,
                            title: payload.title,
                            content: payload.content,
                            price: payload.price,
                            marked_price: payload.marked_price,
                            stock: payload.stock,
                            goods_type: payload.goods_type,
                            pickup_addr: payload.pickup_addr,
                            good_status: payload.good_status,
                            images: []
                        })
                    });
                    const newId = createRes.data?.id;
                    if (!newId) throw new Error('创建失败');
                    if (pending.length) {
                        const fd = new FormData();
                        pending.forEach(f => fd.append('files', f));
                        progress.show('上传图片...');
                        const up = await apiForm(`/admin/goods/${newId}/images`, fd);
                        const urls = up.data?.urls || [];
                        if (urls.length) {
                            await api(`/admin/goods/${newId}`, {method: 'PUT', body: JSON.stringify({images: urls})});
                        }
                    }
                }
            } finally {
                progress.hide();
                confirmBtn.disabled = false;
            }
            renderGoods();
        }, null, (ov) => {
            ov.querySelector('#g-title').value = row?.title || '';
            ov.querySelector('#g-content').value = row?.content || '';
            const gs = row?.good_status ?? 2;
            ov.querySelector('#g-good_status').value = String(gs);
            ov.querySelector('#g-goods_type').value = String(row?.goods_type ?? 1);
            ov.querySelector('#g-pickup_addr').value = row?.pickup_addr || '';
            const listEl = ov.querySelector('#g-images-list');
            const gid = isEdit ? row.id : null;

            function renderImages(urls) {
                listEl.innerHTML = urls.map((url, i) => `
          <span class="art-img-item" data-url="${(url || '').replace(/"/g, '&quot;')}" data-idx="${i}">
            <img src="${url || ''}" alt="图${i + 1}" onerror="this.style.display='none'"/>
            <button type="button" class="art-img-del" title="删除">×</button>
            <span class="art-img-move" title="上移">↑</span>
            <span class="art-img-move" title="下移">↓</span>
          </span>
        `).join('');
                listEl.querySelectorAll('.art-img-del').forEach(btn => {
                    btn.onclick = () => btn.closest('.art-img-item').remove();
                });
                listEl.querySelectorAll('.art-img-move').forEach((span) => {
                    const item = span.closest('.art-img-item');
                    const isUp = span.title === '上移';
                    span.onclick = () => {
                        if (isUp) {
                            const prev = item.previousElementSibling;
                            if (prev) listEl.insertBefore(item, prev);
                        } else {
                            const next = item.nextElementSibling;
                            if (next) listEl.insertBefore(next, item);
                        }
                    };
                });
            }

            function addImages(urls) {
                urls.forEach(url => {
                    const span = document.createElement('span');
                    span.className = 'art-img-item';
                    span.dataset.url = url || '';
                    span.innerHTML = `<img src="${url || ''}" alt="" onerror="this.style.display='none'"/><button type="button" class="art-img-del" title="删除">×</button><span class="art-img-move" title="上移">↑</span><span class="art-img-move" title="下移">↓</span>`;
                    span.querySelector('.art-img-del').onclick = () => span.remove();
                    span.querySelectorAll('.art-img-move').forEach((s) => {
                        s.onclick = () => {
                            if (s.title === '上移') {
                                const prev = span.previousElementSibling;
                                if (prev) listEl.insertBefore(span, prev);
                            } else {
                                const next = span.nextElementSibling;
                                if (next) listEl.insertBefore(next, span);
                            }
                        };
                    });
                    listEl.appendChild(span);
                });
            }

            function addPendingFiles(files) {
                Array.from(files).forEach(file => {
                    const url = URL.createObjectURL(file);
                    const span = document.createElement('span');
                    span.className = 'art-img-item art-img-pending';
                    span.dataset.url = '';
                    span._file = file;
                    span.innerHTML = `<img src="${url}" alt="" onerror="this.style.display='none'"/><button type="button" class="art-img-del" title="删除">×</button><span class="art-img-move" title="上移">↑</span><span class="art-img-move" title="下移">↓</span>`;
                    span.querySelector('.art-img-del').onclick = () => {
                        URL.revokeObjectURL(url);
                        span.remove();
                    };
                    span.querySelectorAll('.art-img-move').forEach((s) => {
                        s.onclick = () => {
                            if (s.title === '上移') {
                                const prev = span.previousElementSibling;
                                if (prev) listEl.insertBefore(span, prev);
                            } else {
                                const next = span.nextElementSibling;
                                if (next) listEl.insertBefore(next, span);
                            }
                        };
                    });
                    listEl.appendChild(span);
                });
            }

            renderImages(initialImgs);

            ov.addEventListener('change', async function (e) {
                if (e.target.id !== 'g-images-add' || !e.target.files) return;
                const files = Array.from(e.target.files || []);
                e.target.value = '';
                if (!files.length) return;
                if (!gid) {
                    addPendingFiles(files);
                    return;
                }
                const progress = ensureUploadProgress(ov);
                try {
                    const fd = new FormData();
                    files.forEach(f => fd.append('files', f));
                    progress.show('上传图片...');
                    const up = await apiForm(`/admin/goods/${gid}/images`, fd);
                    addImages(up.data?.urls || []);
                } catch (err) {
                    alert(err.message);
                } finally {
                    progress.hide();
                }
            });
        });
    }

  const TYPE_MAP = { posts: 'posts', questions: 'questions', answers: 'answers' };

  /** 帖子/提问：点击新建 → 立即创建空草稿 → 打开编辑弹窗 */
  async function createDraftThenShowModal(type, title) {
    try {
      const d = await api(`/admin/${type}`, {method: 'POST', body: JSON.stringify({publish_status: 2})});
      const newId = d.data?.id ?? d.data?.Id;
      if (newId == null || newId === '') throw new Error('创建草稿失败：未返回 id');
      const row = {id: newId, status: 3, title: '', content: '', images: [], publish_status: 2};
      showArticleModal(type, title, row);
    } catch (e) {
      alert(e.message);
    }
  }

  /** 回答：先选提问 → 创建草稿 → 打开编辑弹窗 */
  function showAnswerParentPicker() {
    if (!questionListForAnswer.length) {
      alert('暂无提问，请先创建提问');
      return;
    }
    const opts = questionListForAnswer.map(q => `<option value="${q.id}">#${q.id} ${(q.title || '').slice(0, 40)}</option>`).join('');
    showModal('选择要回复的提问', `<label>提问 <select id="answer-parent">${opts}</select></label>`, async (ov) => {
      const parentId = parseInt(ov.querySelector('#answer-parent').value, 10);
      if (!parentId) {
        throw new Error('请选择提问');
      }
      const d = await api('/admin/answers', {
        method: 'POST',
        body: JSON.stringify({parent_id: parentId, publish_status: 2})
      });
      const newId = d.data?.id;
      if (!newId) throw new Error('创建草稿失败');
      const row = {id: newId, parent_id: parentId, status: 3, title: '', content: '', images: [], publish_status: 2};
      showArticleModal('answers', '编辑回答', row);
    });
  }

  function showArticleModal(type, title, row) {
    const isAnswer = type === 'answers';
    const parentOpts = isAnswer ? questionListForAnswer.map(q => `<option value="${q.id}" ${row?.parent_id === q.id ? 'selected' : ''}>#${q.id} ${(q.title||'').slice(0,30)}</option>`).join('') : '';
    const parentDefault = isAnswer ? '<option value="">请选择提问</option>' : '';
    const parentHtml = isAnswer ? `<label>父提问ID <select id="art-parent_id" required>${parentDefault}${parentOpts}</select></label>` : '';
    const initialImgs = row?.images && Array.isArray(row.images) ? [...row.images] : [];
    showModal(title, `
      ${parentHtml}
      <label>标题 <input type="text" id="art-title" value="${row ? (row.title || '') : ''}" placeholder="标题" required></label>
      <label>内容 <textarea id="art-content" rows="4" placeholder="正文">${row ? (row.content || '') : ''}</textarea></label>
      <label>用户ID <input type="number" id="art-user_id" value="${row?.user_id || ''}" placeholder="0"></label>
      <label>学校ID <input type="number" id="art-school_id" value="${row?.school_id || ''}" placeholder="0"></label>
      <label>公开 <select id="art-publish_status"><option value="1" ${row?.publish_status === 1 ? 'selected' : ''}>私密</option><option value="2" ${!row || row.publish_status === 2 ? 'selected' : ''}>公开</option></select></label>
      <div class="art-images-editor">
        <span class="label">图片（可删除、调整顺序、新增）</span>
        <div id="art-images-list" class="art-images-list"></div>
        <label class="art-add-label">添加图片 <input type="file" id="art-images-add" accept="image/*" multiple></label>
      </div>
    `, async (ov) => {
      const payload = {
        title: ov.querySelector('#art-title').value.trim(),
        content: ov.querySelector('#art-content').value.trim(),
        user_id: parseInt(ov.querySelector('#art-user_id').value || '0', 10),
        school_id: parseInt(ov.querySelector('#art-school_id').value || '0', 10),
        publish_status: parseInt(ov.querySelector('#art-publish_status').value, 10)
      };
      if (isAnswer) {
        const pid = parseInt(ov.querySelector('#art-parent_id').value, 10);
        if (!pid) throw new Error('请选择父提问');
        payload.parent_id = pid;
      }
      let images = Array.from(ov.querySelectorAll('#art-images-list .art-img-item')).map(el => el.dataset.url).filter(Boolean);

      const progress = ensureUploadProgress(ov);
      const confirmBtn = ov.querySelector('#modal-confirm');
      try {
        if (row) {
          const rowId = row.id ?? row.ID;
          if (!rowId) {
            throw new Error('文章 ID 缺失');
          }
          const putBody = { title: payload.title, content: payload.content, publish_status: payload.publish_status, user_id: payload.user_id, school_id: payload.school_id };
          putBody.images = images;
          if (row.status === 3) putBody.status = 1;
          await api(`/admin/${type}/${rowId}`, {method: 'PUT', body: JSON.stringify(putBody)});
        } else {
          // 先创建草稿获取 id，再上传图片，最后 PUT 发布
          const createPayload = { title: payload.title, content: payload.content };
          if (payload.user_id) createPayload.user_id = payload.user_id;
          if (payload.school_id) createPayload.school_id = payload.school_id;
          createPayload.publish_status = payload.publish_status;
          if (isAnswer) createPayload.parent_id = payload.parent_id;
          const d = await api(`/admin/${type}`, { method: 'POST', body: JSON.stringify(createPayload) });
          const newId = d.data?.id;
          if (!newId) throw new Error('创建失败');
            confirmBtn.disabled = true;
          const toUpload = Array.from(ov.querySelectorAll('#art-images-list .art-img-pending')).map(el => el._file).filter(Boolean);
          let urls = [];
          for (let i = 0; i < toUpload.length; i++) {
            progress.show(`上传图片 ${i + 1}/${toUpload.length}`);
            const ext = getExt(toUpload[i].name);
            const path = `article/${newId}/img_${Date.now()}_${Math.random().toString(36).slice(2)}.${ext}`;
            urls.push(await apiUpload(path, toUpload[i], {onProgress: (p) => progress.update(p)}));
            }
          const putBody = {
            title: payload.title,
            content: payload.content,
            publish_status: payload.publish_status,
            user_id: payload.user_id,
            school_id: payload.school_id,
            images: urls,
            status: 1
          };
          await api(`/admin/${type}/${newId}`, {method: 'PUT', body: JSON.stringify(putBody)});
        }
      } finally {
        progress.hide();
        confirmBtn.disabled = false;
      }
      if (type === 'posts') renderPosts();
      else if (type === 'questions') renderQuestions();
      else renderAnswers();
    }, null, (ov) => {
      const listEl = ov.querySelector('#art-images-list');
      const rawId = row != null ? (row.id ?? row.ID) : undefined;
      const articleId = (rawId !== undefined && rawId !== null && rawId !== '') ? (Number(rawId) || rawId) : null;

      function renderImages(urls) {
        listEl.innerHTML = urls.map((url, i) => `
          <span class="art-img-item" data-url="${url || ''}" data-idx="${i}">
            <img src="${url || ''}" alt="图${i + 1}" onerror="this.style.display='none'"/>
            <button type="button" class="art-img-del" title="删除">×</button>
            <span class="art-img-move" title="上移">↑</span>
            <span class="art-img-move" title="下移">↓</span>
          </span>
        `).join('');
        listEl.querySelectorAll('.art-img-del').forEach(btn => {
          btn.onclick = () => btn.closest('.art-img-item').remove();
        });
        listEl.querySelectorAll('.art-img-move').forEach((span) => {
          const item = span.closest('.art-img-item');
          const isUp = span.title === '上移';
          span.onclick = () => {
            if (isUp) {
              const prev = item.previousElementSibling;
              if (prev) listEl.insertBefore(item, prev);
            } else {
              const next = item.nextElementSibling;
              if (next) listEl.insertBefore(next, item);
            }
          };
        });
      }

      function addImages(urls) {
        urls.forEach(url => {
          const span = document.createElement('span');
          span.className = 'art-img-item';
          span.dataset.url = url || '';
          span.innerHTML = `<img src="${url || ''}" alt="" onerror="this.style.display='none'"/><button type="button" class="art-img-del" title="删除">×</button><span class="art-img-move" title="上移">↑</span><span class="art-img-move" title="下移">↓</span>`;
          span.querySelector('.art-img-del').onclick = () => span.remove();
          span.querySelectorAll('.art-img-move').forEach((s, i) => {
            s.onclick = () => {
              if (s.title === '上移') {
                const prev = span.previousElementSibling;
                if (prev) listEl.insertBefore(span, prev);
              } else {
                const next = span.nextElementSibling;
                if (next) listEl.insertBefore(next, span);
              }
            };
          });
          listEl.appendChild(span);
        });
      }

      function addPendingFiles(files) {
        Array.from(files).forEach(file => {
          const url = URL.createObjectURL(file);
          const span = document.createElement('span');
          span.className = 'art-img-item art-img-pending';
          span.dataset.url = '';
          span._file = file;
          span.innerHTML = `<img src="${url}" alt="" onerror="this.style.display='none'"/><button type="button" class="art-img-del" title="删除">×</button><span class="art-img-move" title="上移">↑</span><span class="art-img-move" title="下移">↓</span>`;
          span.querySelector('.art-img-del').onclick = () => {
            URL.revokeObjectURL(url);
            span.remove();
          };
          span.querySelectorAll('.art-img-move').forEach((s) => {
            s.onclick = () => {
              if (s.title === '上移') {
                const prev = span.previousElementSibling;
                if (prev) listEl.insertBefore(span, prev);
              } else {
                const next = span.nextElementSibling;
                if (next) listEl.insertBefore(next, span);
              }
            };
          });
          listEl.appendChild(span);
        });
      }

      renderImages(initialImgs);

      ov.addEventListener('change', async function (e) {
        if (e.target.id !== 'art-images-add' || !e.target.files) return;
        const fileInput = e.target;
        const files = Array.from(fileInput.files || []);
        fileInput.value = '';
        if (!files.length) return;
        if (articleId != null) {
          const progress = ensureUploadProgress(ov);
          try {
            for (let i = 0; i < files.length; i++) {
              progress.show(`上传图片 ${i + 1}/${files.length}`);
              const ext = getExt(files[i].name);
              const path = `article/${articleId}/img_${Date.now()}_${Math.random().toString(36).slice(2)}.${ext}`;
              const url = await apiUpload(path, files[i], {onProgress: (p) => progress.update(p)});
              addImages([url]);
            }
          } catch (e) {
            alert(e.message);
          } finally {
            progress.hide();
          }
        } else {
          if (row && (row.id ?? row.Id) != null) {
            alert('上传失败：文章 ID 未正确传递，请关闭弹窗后重试');
            return;
          }
          addPendingFiles(files);
        }
      });
    });
  }

  async function articleAction(type, id, action) {
    try {
      if (action === 'disable') await api(`/admin/${type}/${id}`, { method: 'DELETE' });
      else if (action === 'restore') await api(`/admin/${type}/${id}/restore`, {method: 'POST'});
      else if (action === 'publish') await api(`/admin/${type}/${id}`, {
        method: 'PUT',
        body: JSON.stringify({status: 1})
      });
      if (type === 'posts') renderPosts();
      else if (type === 'questions') renderQuestions();
      else renderAnswers();
    } catch (e) { alert(e.message); }
  }

    let orderPage = 1;

    async function renderOrders() {
        moduleContent.innerHTML = '<p>加载中...</p>';
        try {
            const data = await api(`/admin/orders?page=${orderPage}&pageSize=15&include_invalid=1`);
            const list = data.data?.list || [];
            const total = data.data?.total || 0;
            const columns = [
                {key: 'id', label: '订单ID'},
                {key: 'user_id', label: '买家ID', render: r => r.user_id ?? '-'},
                {key: 'goods_id', label: '商品ID', render: r => r.goods_id ?? '-'},
                {key: 'good', label: '商品', render: r => (r.good && r.good.title) ? r.good.title.slice(0, 24) : '-'},
                {key: 'order_status_label', label: '状态', render: r => r.order_status_label || r.order_status},
                {key: 'receiver_addr', label: '收货', render: r => (r.receiver_addr || '-').slice(0, 20)},
                {key: 'created_at', label: '创建时间', render: r => (r.created_at || '').slice(0, 19)},
            ];
            list.forEach(row => {
                row._actions = `<button class="btn btn-sm btn-primary" data-order-detail="${row.id}">详情/聊天</button>`;
            });
            const extra = '<span class="text-muted" style="margin-left:8px">平台不经手资金，详见 doc/ORDER_AND_CHAT.md</span>';
            renderTable('订单演示（全站）', columns, list, orderPage, total, 15, (p) => {
                orderPage = p;
                renderOrders();
            }, extra);
            moduleContent.querySelectorAll('[data-order-detail]').forEach(btn => {
                btn.addEventListener('click', () => showOrderDetailModal(parseInt(btn.dataset.orderDetail, 10)));
            });
        } catch (e) {
            moduleContent.innerHTML = '<p class="error">' + e.message + '</p>';
        }
    }

    async function showOrderDetailModal(orderId) {
        try {
            const [dOrder, dMsg] = await Promise.all([
                api(`/admin/orders/${orderId}`),
                api(`/admin/orders/${orderId}/messages?page=1&pageSize=200`)
            ]);
            const o = dOrder.data || {};
            const msgs = dMsg.data?.list || [];
            const msgHtml = msgs.length
                ? msgs.map(m => `<div class="order-msg"><span class="order-msg-meta">#${m.id} 用户${m.sender_id} · ${(m.created_at || '').slice(0, 19)}</span>${m.msg_type === 2 ? `<div><img src="${m.image_url || ''}" class="list-thumb" alt=""/></div>` : `<div>${escapeHtml(m.content || '')}</div>`}</div>`).join('')
                : '<p class="text-muted">暂无消息</p>';
            const overlay = document.createElement('div');
            overlay.className = 'modal-overlay';
            overlay.innerHTML = `<div class="modal modal-wide"><h4>订单 #${orderId}</h4>
        <div class="order-detail-meta">
          <p><strong>状态</strong> ${o.order_status_label || o.order_status} · 买家 user_id: ${o.user_id} · 商品: ${o.good ? o.good.title : o.goods_id}</p>
          <p><strong>收货</strong> ${escapeHtml(o.receiver_addr || '')}</p>
          <p><strong>发货</strong> ${escapeHtml(o.sender_addr || '')}</p>
          <p><strong>双方同意时间</strong> 买方 ${(o.buyer_agreed_at || '').slice(0, 19) || '-'} / 卖方 ${(o.seller_agreed_at || '').slice(0, 19) || '-'}</p>
          <p><strong>完成时间</strong> ${(o.completed_at || '').slice(0, 19) || '-'}</p>
        </div>
        <h5>聊天记录</h5>
        <div class="order-chat-log">${msgHtml}</div>
        <div class="modal-actions"><button class="btn btn-primary" id="order-detail-close">关闭</button></div>
      </div>`;
            document.body.appendChild(overlay);
            overlay.querySelector('#order-detail-close').onclick = () => overlay.remove();
        } catch (e) {
            alert(e.message);
        }
    }

    function escapeHtml(s) {
        const d = document.createElement('div');
        d.textContent = s;
        return d.innerHTML;
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
      moduleContent.querySelectorAll('[data-edit-school]').forEach(btn => btn.addEventListener('click', () => {
        const school = list.find(s => s.id == btn.dataset.editSchool || Number(s.id) === parseInt(btn.dataset.editSchool, 10));
        if (!school) {
          alert('未找到对应学校');
          return;
        }
        showEditSchoolModal(school);
      }));
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
      <label>学校代码 <input type="text" id="s-code" placeholder="hfut"></label>
      <label>表单字段(JSON) <textarea id="s-form_fields" rows="4" placeholder='[{"key":"username","label_zh":"学号","label_en":"Student ID"},{"key":"password","label_zh":"密码","label_en":"Password"}]'></textarea></label>
      <label>验证码URL <input type="text" id="s-captcha_url" placeholder="空则用后端"></label>
    `, async (ov) => {
      let formFields = [];
      try {
        const raw = ov.querySelector('#s-form_fields').value.trim();
        formFields = raw ? JSON.parse(raw) : [];
      } catch (_) {}
      await api('/admin/schools', { method: 'POST', body: JSON.stringify({
        name: ov.querySelector('#s-name').value.trim(),
        login_url: ov.querySelector('#s-login_url').value.trim(),
        code: ov.querySelector('#s-code').value.trim(),
        form_fields: formFields.length ? formFields : undefined,
        captcha_url: ov.querySelector('#s-captcha_url').value.trim() || undefined
      })});
      renderSchools();
    });
  }

  function showEditSchoolModal(school) {
    if (!school) return;
    const formFieldsJson = school.form_fields && Array.isArray(school.form_fields) ? JSON.stringify(school.form_fields) : '[]';
    showModal('编辑学校 #' + school.id, `
      <label>学校名称 <input type="text" id="se-name" value="${school.name || ''}" placeholder="学校名称"></label>
      <label>登录地址 <input type="text" id="se-login_url" value="${school.login_url || ''}" placeholder="https://"></label>
      <label>学校代码 <input type="text" id="se-code" value="${school.code || ''}" placeholder="hfut"></label>
      <label>表单字段(JSON) <textarea id="se-form_fields" rows="4" placeholder='[{"key":"username","label_zh":"学号","label_en":"Student ID"},...]'>${formFieldsJson}</textarea></label>
      <label>验证码URL <input type="text" id="se-captcha_url" value="${school.captcha_url || ''}" placeholder="空则用后端"></label>
    `, async (ov) => {
      let formFields = [];
      try {
        formFields = JSON.parse(ov.querySelector('#se-form_fields').value.trim() || '[]');
      } catch (_) {}
      await api('/admin/schools/' + school.id, { method: 'PUT', body: JSON.stringify({
        name: ov.querySelector('#se-name').value.trim(),
        login_url: ov.querySelector('#se-login_url').value.trim(),
        code: ov.querySelector('#se-code').value.trim() || undefined,
        form_fields: formFields.length ? formFields : undefined,
        captcha_url: ov.querySelector('#se-captcha_url').value.trim() || undefined
      })});
      renderSchools();
    });
  }

  async function renderBindSchool() {
    moduleContent.innerHTML = '<p>加载中...</p>';
    try {
      const userData = await api('/user/info');
      const schoolId = userData.data?.school_id ?? 0;
      const schoolName = userData.data?.school_name || '';

      if (schoolId !== 0 && schoolId !== '0') {
        moduleContent.innerHTML = `
          <div class="module-header"><h3>绑定学校</h3></div>
          <div class="bind-school-already">
            <p>您已绑定学校：<strong>${schoolName || '学校#' + schoolId}</strong></p>
            <p class="text-muted">已绑定学校的用户不允许重复绑定。</p>
          </div>
        `;
        return;
      }

      const schoolsData = await api('/schools');
      const schoolList = schoolsData.data?.list || [];

      if (!schoolList.length) {
        moduleContent.innerHTML = `
          <div class="module-header"><h3>绑定学校</h3></div>
          <p class="text-muted">暂无可绑定的学校，请先在学校管理中新增并配置 code。</p>
        `;
        return;
      }

      const schoolOpts = schoolList.map(s => `<option value="${s.id}">${s.name || s.code || '#' + s.id}</option>`).join('');

      moduleContent.innerHTML = `
        <div class="module-header"><h3>绑定学校</h3></div>
        <div class="bind-school-form">
          <p class="text-muted">选择学校并填写认证信息，验证通过后完成绑定。</p>
          <label>选择学校 <select id="bind-school-select">${schoolOpts}</select></label>
          <div id="bind-form-fields"><p class="text-muted">请先选择学校</p></div>
          <div class="bind-actions">
            <button class="btn btn-primary" id="bind-submit">提交绑定</button>
          </div>
        </div>
      `;

      const schoolSelect = moduleContent.querySelector('#bind-school-select');
      const formFieldsContainer = moduleContent.querySelector('#bind-form-fields');

      let currentSchoolDetail = null;
      let captchaToken = '';

      async function loadSchoolDetail(sid) {
        try {
          const d = await api('/schools/' + sid);
          currentSchoolDetail = d.data;
          return currentSchoolDetail;
        } catch (e) {
          formFieldsContainer.innerHTML = '<p class="error">' + (e.message || '加载失败') + '</p>';
          return null;
        }
      }

      function renderFormFields(school) {
        if (!school) return;
        const fields = school.form_fields || [];
        const needCap = fields.some(f => f.key === 'captcha');
        formFieldsContainer.innerHTML = fields.map(f => {
          if (f.key === 'captcha') {
            return `<label>${f.label_zh || f.key} <input type="text" id="bind-${f.key}" placeholder="验证码" maxlength="8"><button type="button" id="bind-captcha-refresh" class="btn btn-sm">获取验证码</button><div id="bind-captcha-img" class="captcha-img-wrap"></div></label>`;
          }
          const type = f.key === 'password' ? 'password' : 'text';
          return `<label>${f.label_zh || f.key} <input type="${type}" id="bind-${f.key}" placeholder="${f.label_zh || f.key}"></label>`;
        }).join('');
        if (needCap) {
          formFieldsContainer.querySelector('#bind-captcha-refresh')?.addEventListener('click', () => fetchCaptcha(school.id));
        }
      }

      async function fetchCaptcha(sid) {
        try {
          const d = await api('/schools/' + sid + '/captcha');
          captchaToken = d.data?.token || '';
          const img = d.data?.image;
          const wrap = formFieldsContainer.querySelector('#bind-captcha-img');
          if (wrap && img) {
            wrap.innerHTML = `<img src="data:image/png;base64,${img}" alt="验证码" class="captcha-img"/>`;
          }
        } catch (e) {
          alert(e.message);
        }
      }

      async function onSchoolChange() {
        const sid = parseInt(schoolSelect.value, 10);
        captchaToken = '';
        formFieldsContainer.innerHTML = '<p class="text-muted">加载中...</p>';
        const school = await loadSchoolDetail(sid);
        if (school) {
          renderFormFields(school);
          if (school.form_fields?.some(f => f.key === 'captcha')) {
            fetchCaptcha(sid);
          }
        }
      }

      schoolSelect.addEventListener('change', onSchoolChange);

      // 初始加载默认学校详情
      onSchoolChange();

      moduleContent.querySelector('#bind-submit').addEventListener('click', async () => {
        const sid = parseInt(schoolSelect.value, 10);
        let school = currentSchoolDetail;
        if (!school) {
          school = await loadSchoolDetail(sid);
          if (!school) {
            alert('请先选择学校');
            return;
          }
        }
        const fields = school.form_fields || [];
        const body = { school_id: sid };
        fields.forEach(f => {
          const input = formFieldsContainer.querySelector('#bind-' + f.key);
          if (input) {
            if (f.key === 'captcha') {
              body.captcha = input.value.trim();
              body.captcha_token = captchaToken;
            } else {
              body[f.key] = input.value.trim();
            }
          }
        });
        if (!body.username || !body.password) {
          alert('请填写账号和密码');
          return;
        }
        if (fields.some(f => f.key === 'captcha') && (!body.captcha || !body.captcha_token)) {
          alert('请先获取验证码并填写');
          return;
        }
        try {
          await api('/user/bind/school', { method: 'POST', body: JSON.stringify(body) });
          alert('绑定成功');
          renderBindSchool();
        } catch (e) {
          alert(e.message);
        }
      });
    } catch (e) {
      moduleContent.innerHTML = '<p class="error">' + e.message + '</p>';
    }
  }

  if (!getToken()) { redirectToLogin(); return; }
  api('/admin/users?page=1&pageSize=1').then(() => route()).catch(() => redirectToLogin());
})();
