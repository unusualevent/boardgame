<template>
  <div class="user-management">
    <header class="page-header">
      <h1>User Management</h1>
      <button class="btn btn-primary" @click="showCreateModal = true">
        Add User
      </button>
    </header>

    <div class="filters-bar">
      <div class="search-wrapper">
        <input
          v-model="searchQuery"
          type="text"
          placeholder="Search by name or email..."
          class="search-input"
          @input="debouncedSearch"
        />
      </div>
      <div class="filter-group">
        <select v-model="statusFilter" class="filter-select">
          <option value="">All Statuses</option>
          <option value="active">Active</option>
          <option value="inactive">Inactive</option>
          <option value="suspended">Suspended</option>
        </select>
        <select v-model="roleFilter" class="filter-select">
          <option value="">All Roles</option>
          <option value="admin">Admin</option>
          <option value="editor">Editor</option>
          <option value="viewer">Viewer</option>
        </select>
      </div>
    </div>

    <div v-if="loading" class="loading-state">
      <div class="spinner"></div>
      <p>Loading users...</p>
    </div>

    <div v-else-if="error" class="error-state">
      <p class="error-message">{{ error }}</p>
      <button class="btn btn-secondary" @click="fetchUsers">Retry</button>
    </div>

    <table v-else class="data-table">
      <thead>
        <tr>
          <th @click="toggleSort('name')" class="sortable">
            Name
            <span v-if="sortField === 'name'" class="sort-icon">
              {{ sortDirection === 'asc' ? '&#9650;' : '&#9660;' }}
            </span>
          </th>
          <th @click="toggleSort('email')" class="sortable">
            Email
            <span v-if="sortField === 'email'" class="sort-icon">
              {{ sortDirection === 'asc' ? '&#9650;' : '&#9660;' }}
            </span>
          </th>
          <th>Role</th>
          <th @click="toggleSort('created_at')" class="sortable">
            Created
            <span v-if="sortField === 'created_at'" class="sort-icon">
              {{ sortDirection === 'asc' ? '&#9650;' : '&#9660;' }}
            </span>
          </th>
          <th>Status</th>
          <th>Actions</th>
        </tr>
      </thead>
      <tbody>
        <tr v-for="user in paginatedUsers" :key="user.id" class="user-row">
          <td class="user-name">
            <div class="user-avatar">{{ user.name.charAt(0) }}</div>
            <span>{{ user.name }}</span>
          </td>
          <td>{{ user.email }}</td>
          <td>
            <span :class="['badge', `badge-${user.role}`]">
              {{ user.role }}
            </span>
          </td>
          <td>{{ formatDate(user.created_at) }}</td>
          <td>
            <span :class="['status-dot', `status-${user.status}`]"></span>
            {{ user.status }}
          </td>
          <td class="actions-cell">
            <button class="btn btn-sm btn-secondary" @click="editUser(user)">
              Edit
            </button>
            <button class="btn btn-sm btn-danger" @click="confirmDelete(user)">
              Delete
            </button>
          </td>
        </tr>
      </tbody>
    </table>

    <div v-if="!loading && !error" class="pagination">
      <button
        class="btn btn-sm"
        :disabled="currentPage === 1"
        @click="currentPage--"
      >
        Previous
      </button>
      <span class="page-info">
        Page {{ currentPage }} of {{ totalPages }} ({{ filteredUsers.length }} users)
      </span>
      <button
        class="btn btn-sm"
        :disabled="currentPage === totalPages"
        @click="currentPage++"
      >
        Next
      </button>
    </div>

    <div v-if="showCreateModal" class="modal-overlay" @click.self="closeModal">
      <div class="modal-content">
        <h2>{{ editingUser ? 'Edit User' : 'Create User' }}</h2>
        <form @submit.prevent="saveUser">
          <div class="form-group">
            <label for="user-name">Name</label>
            <input
              id="user-name"
              v-model="formData.name"
              type="text"
              required
              class="form-input"
            />
          </div>
          <div class="form-group">
            <label for="user-email">Email</label>
            <input
              id="user-email"
              v-model="formData.email"
              type="email"
              required
              class="form-input"
            />
          </div>
          <div class="form-group">
            <label for="user-role">Role</label>
            <select id="user-role" v-model="formData.role" class="form-input">
              <option value="viewer">Viewer</option>
              <option value="editor">Editor</option>
              <option value="admin">Admin</option>
            </select>
          </div>
          <div class="form-actions">
            <button type="button" class="btn btn-secondary" @click="closeModal">
              Cancel
            </button>
            <button type="submit" class="btn btn-primary" :disabled="saving">
              {{ saving ? 'Saving...' : 'Save' }}
            </button>
          </div>
        </form>
      </div>
    </div>
  </div>
</template>

<script>
import { ref, computed, onMounted, watch } from 'vue';
import { useApi } from '../composables/useApi';
import { debounce } from '../utils/helpers';

export default {
  name: 'UserManagement',
  setup() {
    const { get, post, put, del } = useApi();

    const users = ref([]);
    const loading = ref(true);
    const error = ref(null);
    const saving = ref(false);

    const searchQuery = ref('');
    const statusFilter = ref('');
    const roleFilter = ref('');
    const sortField = ref('name');
    const sortDirection = ref('asc');
    const currentPage = ref(1);
    const perPage = 25;

    const showCreateModal = ref(false);
    const editingUser = ref(null);
    const formData = ref({ name: '', email: '', role: 'viewer' });

    const filteredUsers = computed(() => {
      let result = [...users.value];

      if (searchQuery.value) {
        const q = searchQuery.value.toLowerCase();
        result = result.filter(u =>
          u.name.toLowerCase().includes(q) ||
          u.email.toLowerCase().includes(q)
        );
      }

      if (statusFilter.value) {
        result = result.filter(u => u.status === statusFilter.value);
      }

      if (roleFilter.value) {
        result = result.filter(u => u.role === roleFilter.value);
      }

      result.sort((a, b) => {
        const dir = sortDirection.value === 'asc' ? 1 : -1;
        if (a[sortField.value] < b[sortField.value]) return -dir;
        if (a[sortField.value] > b[sortField.value]) return dir;
        return 0;
      });

      return result;
    });

    const totalPages = computed(() =>
      Math.ceil(filteredUsers.value.length / perPage)
    );

    const paginatedUsers = computed(() => {
      const start = (currentPage.value - 1) * perPage;
      return filteredUsers.value.slice(start, start + perPage);
    });

    async function fetchUsers() {
      loading.value = true;
      error.value = null;
      try {
        const response = await get('/api/v1/users');
        users.value = response.data;
      } catch (err) {
        error.value = err.message;
      } finally {
        loading.value = false;
      }
    }

    function toggleSort(field) {
      if (sortField.value === field) {
        sortDirection.value = sortDirection.value === 'asc' ? 'desc' : 'asc';
      } else {
        sortField.value = field;
        sortDirection.value = 'asc';
      }
    }

    function formatDate(dateStr) {
      return new Date(dateStr).toLocaleDateString('en-US', {
        year: 'numeric',
        month: 'short',
        day: 'numeric',
      });
    }

    function editUser(user) {
      editingUser.value = user;
      formData.value = { name: user.name, email: user.email, role: user.role };
      showCreateModal.value = true;
    }

    async function saveUser() {
      saving.value = true;
      try {
        if (editingUser.value) {
          await put(`/api/v1/users/${editingUser.value.id}`, formData.value);
        } else {
          await post('/api/v1/users', formData.value);
        }
        await fetchUsers();
        closeModal();
      } catch (err) {
        error.value = err.message;
      } finally {
        saving.value = false;
      }
    }

    async function confirmDelete(user) {
      if (!confirm(`Delete user "${user.name}"?`)) return;
      try {
        await del(`/api/v1/users/${user.id}`);
        await fetchUsers();
      } catch (err) {
        error.value = err.message;
      }
    }

    function closeModal() {
      showCreateModal.value = false;
      editingUser.value = null;
      formData.value = { name: '', email: '', role: 'viewer' };
    }

    const debouncedSearch = debounce(() => {
      currentPage.value = 1;
    }, 300);

    watch([statusFilter, roleFilter], () => {
      currentPage.value = 1;
    });

    onMounted(fetchUsers);

    return {
      users, loading, error, saving,
      searchQuery, statusFilter, roleFilter,
      sortField, sortDirection,
      currentPage, totalPages,
      filteredUsers, paginatedUsers,
      showCreateModal, editingUser, formData,
      fetchUsers, toggleSort, formatDate,
      editUser, saveUser, confirmDelete,
      closeModal, debouncedSearch,
    };
  },
};
</script>

<style scoped>
.user-management {
  padding: 24px;
  max-width: 1200px;
  margin: 0 auto;
}
.page-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 24px;
}
.filters-bar {
  display: flex;
  gap: 16px;
  margin-bottom: 16px;
}
.search-input {
  width: 300px;
  padding: 8px 12px;
  border: 1px solid #ddd;
  border-radius: 4px;
}
.filter-select {
  padding: 8px 12px;
  border: 1px solid #ddd;
  border-radius: 4px;
}
.data-table {
  width: 100%;
  border-collapse: collapse;
}
.data-table th,
.data-table td {
  padding: 12px;
  text-align: left;
  border-bottom: 1px solid #eee;
}
.data-table th.sortable {
  cursor: pointer;
  user-select: none;
}
.data-table th.sortable:hover {
  background: #f5f5f5;
}
.user-row:hover {
  background: #fafafa;
}
.user-name {
  display: flex;
  align-items: center;
  gap: 8px;
}
.user-avatar {
  width: 32px;
  height: 32px;
  border-radius: 50%;
  background: #4a90d9;
  color: white;
  display: flex;
  align-items: center;
  justify-content: center;
  font-weight: bold;
}
.modal-overlay {
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, 0.5);
  display: flex;
  align-items: center;
  justify-content: center;
}
.modal-content {
  background: white;
  padding: 24px;
  border-radius: 8px;
  width: 480px;
}
.form-group {
  margin-bottom: 16px;
}
.form-group label {
  display: block;
  margin-bottom: 4px;
  font-weight: 600;
}
.form-input {
  width: 100%;
  padding: 8px 12px;
  border: 1px solid #ddd;
  border-radius: 4px;
}
.form-actions {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
  margin-top: 24px;
}
</style>
