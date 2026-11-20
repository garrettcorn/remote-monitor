// Storage module for persisting paired servers and preferences in localStorage.

const STORAGE_KEY = 'pc_monitor_servers_v1';
const ACTIVE_SERVER_KEY = 'pc_monitor_active_server_id';

export const ServerStore = {
  getServers() {
    try {
      const data = localStorage.getItem(STORAGE_KEY);
      return data ? JSON.parse(data) : [];
    } catch (e) {
      console.error('Failed to load servers from localStorage:', e);
      return [];
    }
  },

  getServer(id) {
    const servers = this.getServers();
    return servers.find(s => s.id === id) || null;
  },

  saveServer(server) {
    const servers = this.getServers();
    const existingIndex = servers.findIndex(s => s.id === server.id);

    if (existingIndex >= 0) {
      servers[existingIndex] = { ...servers[existingIndex], ...server, updatedAt: Date.now() };
    } else {
      server.id = server.id || 'srv_' + Math.random().toString(36).substring(2, 10);
      server.createdAt = Date.now();
      server.updatedAt = Date.now();
      servers.push(server);
    }

    try {
      localStorage.setItem(STORAGE_KEY, JSON.stringify(servers));
    } catch (e) {
      console.error('Failed to save server:', e);
    }
    return server;
  },

  // Update a server that is still paired; returns null (and saves nothing)
  // if it has been unpaired meanwhile. Background updates (snapshots,
  // health checks, pings) must use this, never saveServer, or an
  // unpaired server would be added straight back.
  updateServer(server) {
    if (!server?.id || !this.getServer(server.id)) return null;
    return this.saveServer(server);
  },

  deleteServer(id) {
    let servers = this.getServers();
    servers = servers.filter(s => s.id !== id);
    localStorage.setItem(STORAGE_KEY, JSON.stringify(servers));

    if (this.getActiveServerId() === id) {
      const nextId = servers.length > 0 ? servers[0].id : null;
      this.setActiveServerId(nextId);
    }
  },

  getActiveServerId() {
    return localStorage.getItem(ACTIVE_SERVER_KEY);
  },

  setActiveServerId(id) {
    if (id) {
      localStorage.setItem(ACTIVE_SERVER_KEY, id);
    } else {
      localStorage.removeItem(ACTIVE_SERVER_KEY);
    }
  },

  parsePairingParams(input) {
    if (!input) return null;
    let searchStr = input;

    // Handle full URL or hash fragment
    if (input.includes('#')) {
      searchStr = input.split('#')[1];
    } else if (input.includes('?')) {
      searchStr = input.split('?')[1];
    }

    const params = new URLSearchParams(searchStr);
    const name = params.get('name') || params.get('server') || 'Server';
    const tc = params.get('tc') || '';
    const token = params.get('token') || '';
    const tcport = parseInt(params.get('tcport') || '8080', 10);
    const port = parseInt(params.get('port') || '9120', 10);
    const direct = params.get('direct') || '';

    if (!tc && !direct && !token) {
      return null;
    }

    return {
      name,
      tailcatAddr: tc,
      tailcatPort: tcport,
      directURL: direct,
      directPort: port,
      token,
    };
  },

  exportConfig() {
    return JSON.stringify(this.getServers(), null, 2);
  },

  importConfig(jsonString) {
    try {
      const data = JSON.parse(jsonString);
      if (Array.isArray(data)) {
        localStorage.setItem(STORAGE_KEY, JSON.stringify(data));
        return true;
      }
    } catch (e) {
      console.error('Invalid configuration JSON:', e);
    }
    return false;
  }
};
