import { writable, get } from 'svelte/store';
import { goto } from '$app/navigation';

function createAuth() {
  const { subscribe, set } = writable(
    typeof localStorage !== 'undefined' ? localStorage.getItem('ds_token') : null
  );
  return {
    subscribe,
    set(token) {
      if (typeof localStorage !== 'undefined') {
        token ? localStorage.setItem('ds_token', token) : localStorage.removeItem('ds_token');
      }
      set(token);
    },
    clear() { this.set(null); }
  };
}
export const authToken = createAuth();

const BASE = '/api';

async function req(method, path, body, admin = false) {
  const headers = { 'Content-Type': 'application/json' };
  const token = get(authToken);
  if (token) headers['Authorization'] = `Bearer ${token}`;
  const res = await fetch(`${BASE}${admin ? '/admin' : ''}${path}`, {
    method, headers, body: body ? JSON.stringify(body) : undefined
  });
  if (res.status === 401) { authToken.clear(); goto('/login'); throw new Error('Sesi habis'); }
  const data = await res.json().catch(() => ({}));
  if (!res.ok) throw new Error(data.error || `HTTP ${res.status}`);
  return data;
}

export const api = {
  // Auth
  login: (username, password) => req('POST', '/auth/login', { username, password }),

  // Public store
  getProducts: (qs = '') => req('GET', `/products${qs}`),
  getProduct: (id) => req('GET', `/products/${id}`),
  placeOrder: (body) => req('POST', '/orders', body),
  getPaymentConfig: () => req('GET', '/payment/config'),
  getInvoice: (no) => req('GET', `/invoice/${no}`),
  getPaymentMethods: () => req('GET', '/payment/methods'),

  // Admin - dashboard
  dashboard: () => req('GET', '/dashboard', null, true),

  // Admin - products
  adminProducts: (qs = '') => req('GET', `/products?admin=1${qs}`, null, true),
  createProduct: (body) => req('POST', '/products', body, true),
  updateProduct: (id, body) => req('PUT', `/products/${id}`, body, true),
  deleteProduct: (id) => req('DELETE', `/products/${id}`, null, true),
  toggleProduct: (id) => req('PATCH', `/products/${id}/toggle`, null, true),

  // Admin - stock items
  getStock: (productId) => req('GET', `/products/${productId}/stock`, null, true),
  addStock: (productId, items) => req('POST', `/products/${productId}/stock`, { items }, true),
  updateStockItem: (stockId, data) => req('PUT', `/stock/${stockId}`, { data }, true),
  deleteStockItem: (stockId) => req('DELETE', `/stock/${stockId}`, null, true),
  resetStockItem: (stockId) => req('PATCH', `/stock/${stockId}/reset`, null, true),

  // Admin - orders
  adminOrders: (status = '') => req('GET', `/orders${status ? '?status=' + status : ''}`, null, true),
  getOrder: (id) => req('GET', `/orders/${id}`, null, true),
  updateOrderStatus: (id, status) => req('PATCH', `/orders/${id}/status`, { status }, true),
  manualDeliver: (id, body) => req('POST', `/orders/${id}/deliver`, body, true),

  // Admin - payment
  getAdminPayment: () => req('GET', '/payment/config', null, true),
  updatePaymentConfig: (body) => req('PUT', '/payment/config', body, true),

  // Admin - scripts
  scriptLogs: () => req('GET', '/scripts/logs', null, true),

  // Admin - product image (multipart — tidak pakai req() biasa)
  uploadProductImage: async (productId, file) => {
    const tk = typeof localStorage !== 'undefined' ? localStorage.getItem('ds_token') : '';
    const form = new FormData();
    form.append('image', file);
    const res = await fetch(`/api/admin/products/${productId}/image`, {
      method: 'POST',
      headers: tk ? { Authorization: `Bearer ${tk}` } : {},
      body: form,
    });
    const data = await res.json().catch(() => ({}));
    if (!res.ok) throw new Error(data.error || `HTTP ${res.status}`);
    return data;
  },

  deleteProductImage: (productId) => req('DELETE', `/products/${productId}/image`, null, true),
};
