import { createApp } from './app.js';
import { fetchJSON, debounce } from './utils.js';

const API_BASE = '/api/v1';

class DashboardController {
  constructor(container) {
    this.container = container;
    this.state = {
      users: [],
      sessions: [],
      loading: true,
      error: null,
      filters: {
        status: 'all',
        search: '',
        sortBy: 'name',
        sortDir: 'asc',
        page: 1,
        perPage: 25,
      },
    };
    this.listeners = new Map();
    this.init();
  }

  async init() {
    try {
      await this.loadData();
      this.render();
      this.bindEvents();
    } catch (err) {
      this.state.error = err.message;
      this.render();
    }
  }

  async loadData() {
    this.state.loading = true;
    this.render();

    const [users, sessions] = await Promise.all([
      fetchJSON(`${API_BASE}/users`),
      fetchJSON(`${API_BASE}/sessions`),
    ]);

    this.state.users = users.data || [];
    this.state.sessions = sessions.data || [];
    this.state.loading = false;
  }

  getFilteredUsers() {
    let filtered = [...this.state.users];

    if (this.state.filters.status !== 'all') {
      filtered = filtered.filter(u => u.status === this.state.filters.status);
    }

    if (this.state.filters.search) {
      const query = this.state.filters.search.toLowerCase();
      filtered = filtered.filter(u =>
        u.name.toLowerCase().includes(query) ||
        u.email.toLowerCase().includes(query)
      );
    }

    const dir = this.state.filters.sortDir === 'asc' ? 1 : -1;
    const key = this.state.filters.sortBy;
    filtered.sort((a, b) => {
      if (a[key] < b[key]) return -dir;
      if (a[key] > b[key]) return dir;
      return 0;
    });

    const start = (this.state.filters.page - 1) * this.state.filters.perPage;
    const end = start + this.state.filters.perPage;
    return {
      items: filtered.slice(start, end),
      total: filtered.length,
      pages: Math.ceil(filtered.length / this.state.filters.perPage),
    };
  }

  bindEvents() {
    const searchInput = this.container.querySelector('#search-input');
    if (searchInput) {
      const handleSearch = debounce((e) => {
        this.state.filters.search = e.target.value;
        this.state.filters.page = 1;
        this.render();
      }, 300);
      searchInput.addEventListener('input', handleSearch);
      this.listeners.set(searchInput, { type: 'input', handler: handleSearch });
    }

    const statusFilter = this.container.querySelector('#status-filter');
    if (statusFilter) {
      const handleStatus = (e) => {
        this.state.filters.status = e.target.value;
        this.state.filters.page = 1;
        this.render();
      };
      statusFilter.addEventListener('change', handleStatus);
      this.listeners.set(statusFilter, { type: 'change', handler: handleStatus });
    }

    this.container.addEventListener('click', (e) => {
      const sortBtn = e.target.closest('[data-sort]');
      if (sortBtn) {
        const field = sortBtn.dataset.sort;
        if (this.state.filters.sortBy === field) {
          this.state.filters.sortDir =
            this.state.filters.sortDir === 'asc' ? 'desc' : 'asc';
        } else {
          this.state.filters.sortBy = field;
          this.state.filters.sortDir = 'asc';
        }
        this.render();
      }

      const pageBtn = e.target.closest('[data-page]');
      if (pageBtn) {
        this.state.filters.page = parseInt(pageBtn.dataset.page, 10);
        this.render();
      }

      const deleteBtn = e.target.closest('[data-delete]');
      if (deleteBtn) {
        const userId = deleteBtn.dataset.delete;
        this.deleteUser(userId);
      }
    });
  }

  async deleteUser(id) {
    if (!confirm('Are you sure you want to delete this user?')) return;

    try {
      await fetchJSON(`${API_BASE}/users/${id}`, { method: 'DELETE' });
      this.state.users = this.state.users.filter(u => u.id !== id);
      this.render();
    } catch (err) {
      this.state.error = err.message;
      this.render();
    }
  }

  render() {
    if (this.state.loading) {
      this.container.innerHTML = '<div class="loading">Loading...</div>';
      return;
    }

    if (this.state.error) {
      this.container.innerHTML = `<div class="error">${this.state.error}</div>`;
      return;
    }

    const { items, total, pages } = this.getFilteredUsers();
    const activeSessions = this.state.sessions.filter(s => s.active).length;

    this.container.innerHTML = `
      <div class="dashboard-header">
        <h1>Dashboard</h1>
        <div class="stats">
          <span class="stat">${total} users</span>
          <span class="stat">${activeSessions} active sessions</span>
        </div>
      </div>
      <div class="filters">
        <input id="search-input" type="text" placeholder="Search users..."
               value="${this.state.filters.search}">
        <select id="status-filter">
          <option value="all">All Status</option>
          <option value="active">Active</option>
          <option value="inactive">Inactive</option>
          <option value="suspended">Suspended</option>
        </select>
      </div>
      <table class="user-table">
        <thead>
          <tr>
            <th data-sort="name">Name</th>
            <th data-sort="email">Email</th>
            <th data-sort="status">Status</th>
            <th data-sort="created">Created</th>
            <th>Actions</th>
          </tr>
        </thead>
        <tbody>
          ${items.map(u => `
            <tr>
              <td>${u.name}</td>
              <td>${u.email}</td>
              <td><span class="badge badge-${u.status}">${u.status}</span></td>
              <td>${new Date(u.created).toLocaleDateString()}</td>
              <td>
                <button class="btn btn-sm" data-delete="${u.id}">Delete</button>
              </td>
            </tr>
          `).join('')}
        </tbody>
      </table>
      <div class="pagination">
        ${Array.from({ length: pages }, (_, i) => `
          <button data-page="${i + 1}"
                  class="${i + 1 === this.state.filters.page ? 'active' : ''}">
            ${i + 1}
          </button>
        `).join('')}
      </div>
    `;

    this.bindEvents();
  }

  destroy() {
    for (const [el, { type, handler }] of this.listeners) {
      el.removeEventListener(type, handler);
    }
    this.listeners.clear();
  }
}

const app = createApp('#app');
const dashboard = new DashboardController(app.root);

export { DashboardController };
