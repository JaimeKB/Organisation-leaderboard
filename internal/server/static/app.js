(function () {
  const app = document.getElementById('app');
  if (!app) return;

  const org = app.dataset.org;
  const form = document.getElementById('filters');
  const tbody = document.getElementById('leaderboard-body');

  function escapeHtml(s) {
    return s.replace(/[&<>"']/g, (c) => ({
      '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;',
    }[c]));
  }

  function render(rows) {
    if (!rows.length) {
      tbody.innerHTML = '<tr><td colspan="4" class="muted">No commits in this range.</td></tr>';
      return;
    }
    tbody.innerHTML = rows.map((c, i) => `
      <tr>
        <td>${i + 1}</td>
        <td>${c.avatarUrl ? `<img src="${c.avatarUrl}" class="avatar-sm" alt="">` : ''}</td>
        <td>${escapeHtml(c.name || c.login)}</td>
        <td>${c.commitCount}</td>
      </tr>
    `).join('');
  }

  const EMPTY_MESSAGE = 'Select repositories and members, then click "Load data".';

  function showEmpty(message) {
    tbody.innerHTML = `<tr><td colspan="4" class="muted">${escapeHtml(message)}</td></tr>`;
  }

  async function loadData() {
    const data = new FormData(form);
    const repos = data.getAll('repo');
    const users = data.getAll('user');

    if (!repos.length || !users.length) {
      showEmpty('Select at least one repository and one member, then click "Load data".');
      return;
    }

    const params = new URLSearchParams();
    params.set('org', org);

    const since = data.get('since');
    const until = data.get('until');
    if (since) params.set('since', since);
    if (until) params.set('until', until);
    for (const repo of repos) params.append('repo', repo);
    for (const user of users) params.append('user', user);

    tbody.innerHTML = '<tr><td colspan="4" class="muted">Loading…</td></tr>';
    try {
      const res = await fetch('/api/leaderboard?' + params.toString());
      const payload = await res.json();
      if (!res.ok) throw new Error(payload.error || 'request failed');
      render(payload);
    } catch (err) {
      tbody.innerHTML = `<tr><td colspan="4" class="error">${escapeHtml(err.message)}</td></tr>`;
    }
  }

  form.addEventListener('submit', (e) => {
    e.preventDefault();
    loadData();
  });

  const clearBtn = document.getElementById('clear-filters');
  if (clearBtn) {
    clearBtn.addEventListener('click', () => {
      form.reset();
      showEmpty(EMPTY_MESSAGE);
    });
  }
})();
