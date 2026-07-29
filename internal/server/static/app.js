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

  async function applyFilters() {
    const data = new FormData(form);
    const params = new URLSearchParams();
    params.set('org', org);

    const since = data.get('since');
    const until = data.get('until');
    if (since) params.set('since', since);
    if (until) params.set('until', until);
    for (const repo of data.getAll('repo')) params.append('repo', repo);
    for (const user of data.getAll('user')) params.append('user', user);

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
    applyFilters();
  });

  const clearBtn = document.getElementById('clear-filters');
  if (clearBtn) {
    clearBtn.addEventListener('click', () => setTimeout(applyFilters, 0));
  }
})();
