(function () {
  const API = '/api/v1';
  const TOKEN_KEY = 'admin_token';

  function getToken() { return localStorage.getItem(TOKEN_KEY); }
  function setToken(token) { localStorage.setItem(TOKEN_KEY, token); }

  // 若已登录且 token 有效，直接进入平台
  if (getToken()) {
    fetch(API + '/admin/users?page=1&pageSize=1', {
      headers: { 'Authorization': 'Bearer ' + getToken() },
    }).then(r => r.json()).then(data => {
      if (data.code === 200 || data.code === 0) {
        window.location.href = '/admin/';
        return;
      }
    }).catch(() => {});
  }

  const form = document.getElementById('login-form');
  const errorEl = document.getElementById('login-error');

  form.addEventListener('submit', async (e) => {
    e.preventDefault();
    errorEl.textContent = '';
    const username = form.username.value.trim();
    const password = form.password.value;
    try {
      const res = await fetch(API + '/admin/login', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ username, password }),
      });
      const data = await res.json();
      if (data.code !== 0 && data.code !== 200) {
        errorEl.textContent = data.message || '登录失败';
        return;
      }
      const token = data.data?.token || data.data;
      if (!token) {
        errorEl.textContent = '登录响应异常';
        return;
      }
      setToken(token);
      window.location.href = '/admin/';
    } catch (err) {
      errorEl.textContent = err.message || '网络错误';
    }
  });
})();
