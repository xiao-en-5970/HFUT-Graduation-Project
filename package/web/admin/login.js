(function () {
    // login.js 仅负责一次性登录 + 写 access token；refresh token 由后端通过
    // HttpOnly + Secure + SameSite=Strict cookie 下发，浏览器 JS **读不到也写不到**——
    // 这是抵御 XSS 的关键，所以本文件**故意不引入 REFRESH_KEY 任何逻辑**。
  const API = '/api/v1';
    const ACCESS_KEY = 'admin_token'; // 历史 key——继续用作 access token

    function getToken() {
        return localStorage.getItem(ACCESS_KEY);
    }

    function setAccess(access) {
        localStorage.setItem(ACCESS_KEY, access);
    }

  if (getToken()) {
    fetch(API + '/admin/users?page=1&pageSize=1', {
      headers: { 'Authorization': 'Bearer ' + getToken() },
    }).then(r => r.json()).then(data => {
      if (data.code === 200 || data.code === 0) {
        window.location.href = '/admin/';
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
          // credentials: 'same-origin' 是浏览器默认值，足以让响应里的 Set-Cookie
          // 落到本站 cookie store；后续 /api/v1/user/refresh 自动带回。
          credentials: 'same-origin',
      });
      const data = await res.json();
      if (data.code !== 0 && data.code !== 200) {
        errorEl.textContent = data.message || '登录失败';
        return;
      }
        // 仅取 access_token；refresh 已经由 Set-Cookie 落到 HttpOnly cookie 里。
        const access = data.data?.access_token || data.data?.token || (typeof data.data === 'string' ? data.data : '');
        if (!access) {
        errorEl.textContent = '登录响应异常';
        return;
      }
        setAccess(access);
      window.location.href = '/admin/';
    } catch (err) {
      errorEl.textContent = err.message || '网络错误';
    }
  });
})();
