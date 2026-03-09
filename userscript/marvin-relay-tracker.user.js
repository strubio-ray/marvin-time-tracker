// ==UserScript==
// @name         Marvin Relay Tracker
// @namespace    com.strubio.marvin-relay-tracker
// @version      0.2.0
// @description  Overlay tracking controls on Amazing Marvin, synced with relay server via SSE
// @match        https://app.amazingmarvin.com/*
// @grant        GM.xmlHttpRequest
// @grant        GM_addStyle
// @grant        GM_getValue
// @grant        GM_setValue
// @connect      *
// @run-at       document-idle
// ==/UserScript==

(function () {
  'use strict';

  // === Config ===

  const Config = {
    async getRelayUrl() {
      return (await GM_getValue('relayUrl', 'http://localhost:8080')).replace(/\/+$/, '');
    },
    async setRelayUrl(url) {
      await GM_setValue('relayUrl', url.replace(/\/+$/, ''));
    },
    async getHideNative() {
      return await GM_getValue('hideNative', false);
    },
    async setHideNative(val) {
      await GM_setValue('hideNative', val);
    },
    async getCollapsed() {
      return await GM_getValue('collapsed', false);
    },
    async setCollapsed(val) {
      await GM_setValue('collapsed', val);
    },
    async getConnected() {
      return await GM_getValue('connected', true);
    },
    async setConnected(val) {
      await GM_setValue('connected', val);
    },
    async isFirstRun() {
      const url = await GM_getValue('relayUrl', null);
      return url === null;
    },
  };

  // === API ===

  const API = (() => {
    let pending = false;

    function gmRequest(method, url, data) {
      return new Promise((resolve, reject) => {
        GM.xmlHttpRequest({
          method,
          url,
          headers: data ? { 'Content-Type': 'application/json' } : undefined,
          data: data ? JSON.stringify(data) : undefined,
          timeout: 15000,
          onload(resp) {
            if (resp.status >= 200 && resp.status < 300) {
              try {
                resolve(JSON.parse(resp.responseText));
              } catch {
                resolve(resp.responseText);
              }
            } else {
              reject(new Error(`HTTP ${resp.status}: ${resp.responseText}`));
            }
          },
          onerror(err) {
            reject(new Error(err.error || 'Network error'));
          },
          ontimeout() {
            reject(new Error('Request timeout'));
          },
        });
      });
    }

    function isValidUrl(str) {
      try {
        const url = new URL(str);
        return url.protocol === 'http:' || url.protocol === 'https:';
      } catch {
        return false;
      }
    }

    return {
      isValidUrl,

      async startTracking(taskId, title) {
        if (pending) return;
        pending = true;
        try {
          const url = await Config.getRelayUrl();
          return await gmRequest('POST', `${url}/start`, { taskId, title });
        } finally {
          pending = false;
        }
      },

      async stopTracking(taskId) {
        if (pending) return;
        pending = true;
        try {
          const url = await Config.getRelayUrl();
          const body = taskId ? { taskId } : {};
          return await gmRequest('POST', `${url}/stop`, body);
        } finally {
          pending = false;
        }
      },

      async getStatus() {
        const url = await Config.getRelayUrl();
        return await gmRequest('GET', `${url}/status`);
      },

      async getHistory(limit = 10) {
        const url = await Config.getRelayUrl();
        return await gmRequest('GET', `${url}/history?limit=${limit}`);
      },
    };
  })();

  // === State ===

  const State = (() => {
    const listeners = [];
    const state = {
      tracking: false,
      taskId: null,
      taskTitle: null,
      startedAt: null,
      connectionState: 'disconnected', // 'disconnected' | 'connecting' | 'connected' | 'reconnecting'
    };

    return {
      get() {
        return { ...state };
      },

      isConnected() {
        return state.connectionState === 'connected';
      },

      update(data) {
        if (data.tracking !== undefined) state.tracking = data.tracking;
        if (data.tracking) {
          if (data.taskId !== undefined) state.taskId = data.taskId;
          if (data.taskTitle !== undefined) state.taskTitle = data.taskTitle;
          if (data.startedAt !== undefined) state.startedAt = data.startedAt;
        } else if (data.tracking === false) {
          state.taskId = null;
          state.taskTitle = null;
          state.startedAt = null;
        }
        listeners.forEach((fn) => fn(state));
      },

      setConnectionState(val) {
        state.connectionState = val;
        listeners.forEach((fn) => fn(state));
      },

      onChange(fn) {
        listeners.push(fn);
      },
    };
  })();

  // === Toast ===

  const Toast = (() => {
    let container = null;
    const MAX_TOASTS = 3;

    function init(shadowRoot) {
      container = document.createElement('div');
      container.className = 'toast-container';
      shadowRoot.appendChild(container);
    }

    function show(message, type = 'info', duration = 5000) {
      if (!container) return;

      // Enforce max visible toasts
      while (container.children.length >= MAX_TOASTS) {
        container.removeChild(container.firstChild);
      }

      const toast = document.createElement('div');
      toast.className = `toast toast-${type}`;
      toast.textContent = message;

      container.appendChild(toast);

      // Trigger slide-in animation
      requestAnimationFrame(() => {
        toast.classList.add('toast-visible');
      });

      setTimeout(() => {
        toast.classList.remove('toast-visible');
        toast.classList.add('toast-hiding');
        toast.addEventListener('transitionend', () => {
          if (toast.parentNode) toast.remove();
        });
        // Fallback removal if transitionend doesn't fire
        setTimeout(() => { if (toast.parentNode) toast.remove(); }, 300);
      }, duration);
    }

    return { init, show };
  })();

  // === SSE ===

  const SSE = (() => {
    let eventSource = null;
    let reconnectTimer = null;
    let reconnectAttempt = 0;
    let intentionalDisconnect = false;

    const BASE_DELAY = 1000;
    const MAX_DELAY = 30000;

    function getReconnectDelay() {
      return Math.min(BASE_DELAY * Math.pow(2, reconnectAttempt), MAX_DELAY);
    }

    function scheduleReconnect() {
      if (reconnectTimer || intentionalDisconnect) return;
      State.setConnectionState('reconnecting');
      const delay = getReconnectDelay();
      reconnectAttempt++;
      reconnectTimer = setTimeout(() => {
        reconnectTimer = null;
        connectInternal();
      }, delay);
    }

    async function connectInternal() {
      if (intentionalDisconnect) return;

      const currentState = State.get().connectionState;
      if (currentState !== 'reconnecting') {
        State.setConnectionState('connecting');
      }

      // Health check before opening SSE
      try {
        const status = await API.getStatus();
        // Apply initial state from health check
        State.update({
          tracking: status.tracking,
          taskId: status.taskId || null,
          taskTitle: status.taskTitle || null,
          startedAt: status.startedAt || null,
        });
      } catch (err) {
        if (reconnectAttempt === 0) {
          // First attempt — show error toast
          Toast.show(`Cannot reach server: ${err.message}`, 'error');
        }
        scheduleReconnect();
        return;
      }

      const url = await Config.getRelayUrl();

      try {
        eventSource = new EventSource(`${url}/events`);
      } catch {
        scheduleReconnect();
        return;
      }

      eventSource.addEventListener('state', (e) => {
        try {
          const data = JSON.parse(e.data);
          State.update({
            tracking: data.tracking,
            taskId: data.taskId || null,
            taskTitle: data.taskTitle || null,
            startedAt: data.startedAt || null,
          });
        } catch { /* ignore parse errors */ }
      });

      eventSource.addEventListener('tracking_started', (e) => {
        try {
          const data = JSON.parse(e.data);
          State.update({
            tracking: true,
            taskId: data.taskId || null,
            taskTitle: data.taskTitle || null,
            startedAt: data.startedAt || null,
          });
        } catch { /* ignore */ }
      });

      eventSource.addEventListener('tracking_stopped', () => {
        State.update({ tracking: false });
      });

      eventSource.onopen = () => {
        const wasReconnecting = reconnectAttempt > 0;
        State.setConnectionState('connected');
        reconnectAttempt = 0;
        if (wasReconnecting) {
          Toast.show('Reconnected to server', 'success', 3000);
        }
      };

      eventSource.onerror = () => {
        if (intentionalDisconnect) return;
        if (State.isConnected()) {
          Toast.show('Connection lost, reconnecting...', 'warning');
        }
        if (eventSource) {
          eventSource.close();
          eventSource = null;
        }
        scheduleReconnect();
      };
    }

    return {
      async connect() {
        intentionalDisconnect = false;
        reconnectAttempt = 0;
        await connectInternal();
      },

      disconnect() {
        intentionalDisconnect = true;
        if (reconnectTimer) {
          clearTimeout(reconnectTimer);
          reconnectTimer = null;
        }
        if (eventSource) {
          eventSource.close();
          eventSource = null;
        }
        reconnectAttempt = 0;
        State.setConnectionState('disconnected');
      },
    };
  })();

  // === UI ===

  const UI = (() => {
    let root = null;
    let shadow = null;
    let timerInterval = null;
    let elements = {};

    function formatElapsed(startedAtMs) {
      if (!startedAtMs) return '00:00:00';
      const diff = Math.max(0, Math.floor((Date.now() - startedAtMs) / 1000));
      const h = String(Math.floor(diff / 3600)).padStart(2, '0');
      const m = String(Math.floor((diff % 3600) / 60)).padStart(2, '0');
      const s = String(diff % 60).padStart(2, '0');
      return `${h}:${m}:${s}`;
    }

    function formatDuration(ms) {
      if (!ms || ms <= 0) return '0:00';
      const totalSec = Math.floor(ms / 1000);
      const h = Math.floor(totalSec / 3600);
      const m = String(Math.floor((totalSec % 3600) / 60)).padStart(2, '0');
      const s = String(totalSec % 60).padStart(2, '0');
      return h > 0 ? `${h}:${m}:${s}` : `${m}:${s}`;
    }

    function formatRelativeTime(timestampMs) {
      const diff = Date.now() - timestampMs;
      const mins = Math.floor(diff / 60000);
      if (mins < 1) return 'just now';
      if (mins < 60) return `${mins}m ago`;
      const hours = Math.floor(mins / 60);
      if (hours < 24) return `${hours}h ago`;
      const days = Math.floor(hours / 24);
      return `${days}d ago`;
    }

    function createPanel() {
      root = document.createElement('div');
      root.id = 'marvin-relay-root';
      shadow = root.attachShadow({ mode: 'closed' });

      const style = document.createElement('style');
      style.textContent = `
        :host {
          all: initial;
          font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
          font-size: 13px;
        }
        .panel {
          position: fixed;
          bottom: 16px;
          right: 16px;
          z-index: 999999;
          background: #1e1e2e;
          color: #cdd6f4;
          border-radius: 10px;
          box-shadow: 0 4px 20px rgba(0,0,0,0.4);
          min-width: 220px;
          overflow: hidden;
          transition: all 0.2s ease;
        }
        .header {
          display: flex;
          justify-content: space-between;
          align-items: center;
          padding: 8px 12px;
          background: #313244;
          cursor: pointer;
          user-select: none;
        }
        .header-title {
          font-weight: 600;
          font-size: 12px;
          text-transform: uppercase;
          letter-spacing: 0.5px;
        }
        .header-icons {
          display: flex;
          gap: 6px;
        }
        .icon-btn {
          background: none;
          border: none;
          color: #cdd6f4;
          cursor: pointer;
          font-size: 14px;
          padding: 2px;
          opacity: 0.7;
          transition: opacity 0.15s;
        }
        .icon-btn:hover { opacity: 1; }
        .body { padding: 12px; }
        .body.hidden { display: none; }
        .status {
          margin-bottom: 8px;
          padding: 8px;
          border-radius: 6px;
          text-align: center;
          position: relative;
        }
        .status.idle { background: #45475a; }
        .status.tracking { background: #1e4620; }
        .status.disconnected { background: #5c2020; }
        .status.connecting { background: #45475a; }
        .status.reconnecting { background: #4a3c20; }
        .status-row {
          display: flex;
          align-items: center;
          justify-content: center;
          gap: 4px;
        }
        .task-title {
          font-weight: 600;
          margin-bottom: 4px;
          word-break: break-word;
        }
        .elapsed {
          font-size: 20px;
          font-weight: 700;
          font-variant-numeric: tabular-nums;
          letter-spacing: 1px;
        }
        .controls { margin-top: 8px; }
        .btn {
          width: 100%;
          padding: 8px;
          border: none;
          border-radius: 6px;
          cursor: pointer;
          font-size: 13px;
          font-weight: 600;
          transition: background 0.15s;
        }
        .btn-stop { background: #f38ba8; color: #1e1e2e; }
        .btn-stop:hover:not(:disabled) { background: #eba0ac; }
        .btn:disabled { opacity: 0.5; cursor: not-allowed; }
        .connect-toggle {
          position: absolute;
          top: 6px;
          right: 6px;
          background: none;
          border: 1px solid rgba(205,214,244,0.2);
          border-radius: 4px;
          color: #cdd6f4;
          cursor: pointer;
          font-size: 12px;
          padding: 2px 5px;
          opacity: 0.6;
          transition: opacity 0.15s, color 0.15s, border-color 0.15s;
          line-height: 1;
        }
        .connect-toggle:hover { opacity: 1; }
        .connect-toggle.active {
          color: #a6e3a1;
          border-color: rgba(166,227,161,0.4);
          opacity: 0.8;
        }
        .settings { margin-top: 10px; border-top: 1px solid #45475a; padding-top: 10px; }
        .settings.hidden { display: none; }
        .settings label {
          display: block;
          font-size: 11px;
          margin-bottom: 4px;
          color: #a6adc8;
        }
        .settings input[type="text"] {
          width: 100%;
          box-sizing: border-box;
          padding: 6px 8px;
          background: #313244;
          border: 1px solid #45475a;
          border-radius: 4px;
          color: #cdd6f4;
          font-size: 12px;
          margin-bottom: 4px;
        }
        .settings input[type="text"]:focus {
          outline: none;
          border-color: #89b4fa;
        }
        .settings input[type="text"].input-error {
          border-color: #f38ba8;
        }
        .url-error {
          color: #f38ba8;
          font-size: 10px;
          margin-bottom: 6px;
          display: none;
        }
        .url-error.visible { display: block; }
        .checkbox-row {
          display: flex;
          align-items: center;
          gap: 6px;
          font-size: 12px;
        }
        .error {
          color: #f38ba8;
          font-size: 11px;
          margin-top: 6px;
          text-align: center;
        }
        .dot {
          display: inline-block;
          width: 6px;
          height: 6px;
          border-radius: 50%;
          margin-right: 6px;
          flex-shrink: 0;
        }
        .dot.green { background: #a6e3a1; }
        .dot.red { background: #f38ba8; }
        .dot.gray { background: #6c7086; }
        .dot.yellow { background: #f9e2af; }
        @keyframes pulse {
          0%, 100% { opacity: 1; }
          50% { opacity: 0.4; }
        }
        .dot.pulse { animation: pulse 1.5s ease-in-out infinite; }
        .history-section { margin-top: 10px; border-top: 1px solid #45475a; padding-top: 8px; }
        .history-section.hidden { display: none; }
        .history-toggle {
          background: none;
          border: none;
          color: #a6adc8;
          cursor: pointer;
          font-size: 11px;
          padding: 0;
          width: 100%;
          text-align: left;
          transition: color 0.15s;
        }
        .history-toggle:hover { color: #cdd6f4; }
        .history-list { margin-top: 6px; }
        .history-list.hidden { display: none; }
        .history-entry {
          display: flex;
          justify-content: space-between;
          align-items: center;
          padding: 4px 6px;
          border-radius: 4px;
          font-size: 11px;
          margin-bottom: 2px;
        }
        .history-entry:nth-child(odd) { background: #313244; }
        .history-entry:nth-child(even) { background: #2a2b3d; }
        .history-task {
          flex: 1;
          overflow: hidden;
          text-overflow: ellipsis;
          white-space: nowrap;
          margin-right: 8px;
        }
        .history-meta {
          display: flex;
          gap: 8px;
          flex-shrink: 0;
          color: #a6adc8;
          font-size: 10px;
        }
        .history-duration { color: #a6e3a1; font-weight: 600; }
        .history-empty {
          color: #6c7086;
          font-size: 11px;
          text-align: center;
          padding: 8px;
        }
        .toast-container {
          position: fixed;
          top: 16px;
          right: 16px;
          z-index: 9999999;
          display: flex;
          flex-direction: column;
          gap: 6px;
          pointer-events: none;
        }
        .toast {
          padding: 8px 14px;
          border-radius: 6px;
          font-size: 12px;
          font-weight: 500;
          box-shadow: 0 4px 12px rgba(0,0,0,0.3);
          transform: translateX(120%);
          transition: transform 0.25s ease, opacity 0.25s ease;
          opacity: 0;
          pointer-events: auto;
          max-width: 300px;
          word-break: break-word;
        }
        .toast-visible {
          transform: translateX(0);
          opacity: 1;
        }
        .toast-hiding {
          transform: translateX(120%);
          opacity: 0;
        }
        .toast-error { background: #45273a; color: #f38ba8; border: 1px solid rgba(243,139,168,0.3); }
        .toast-warning { background: #45392a; color: #fab387; border: 1px solid rgba(250,179,135,0.3); }
        .toast-info { background: #2a3545; color: #89b4fa; border: 1px solid rgba(137,180,250,0.3); }
        .toast-success { background: #2a4530; color: #a6e3a1; border: 1px solid rgba(166,227,161,0.3); }
      `;

      const panel = document.createElement('div');
      panel.className = 'panel';
      panel.innerHTML = `
        <div class="header">
          <span class="header-title">Relay Tracker</span>
          <div class="header-icons">
            <button class="icon-btn settings-toggle" title="Settings">&#9881;</button>
            <button class="icon-btn collapse-toggle" title="Collapse">&#9660;</button>
          </div>
        </div>
        <div class="body">
          <div class="status disconnected">
            <div class="status-row">
              <span class="dot red"></span>
              <span class="status-label">Disconnected</span>
            </div>
            <button class="connect-toggle" title="Connect to relay server">&#9211;</button>
          </div>
          <div class="controls hidden">
            <button class="btn btn-stop" disabled>Stop</button>
          </div>
          <div class="error hidden"></div>
          <div class="history-section hidden">
            <button class="history-toggle">Recent sessions &#9660;</button>
            <div class="history-list hidden"></div>
          </div>
          <div class="settings hidden">
            <label>Relay Server URL</label>
            <input type="text" class="relay-url" placeholder="http://localhost:8080">
            <div class="url-error">Invalid URL format</div>
            <div class="checkbox-row">
              <input type="checkbox" class="hide-native" id="mrt-hide-native">
              <label for="mrt-hide-native">Hide native tracking buttons</label>
            </div>
          </div>
        </div>
      `;

      shadow.appendChild(style);
      shadow.appendChild(panel);

      elements = {
        panel,
        body: panel.querySelector('.body'),
        status: panel.querySelector('.status'),
        statusLabel: panel.querySelector('.status-label'),
        controls: panel.querySelector('.controls'),
        stopBtn: panel.querySelector('.btn-stop'),
        connectToggle: panel.querySelector('.connect-toggle'),
        error: panel.querySelector('.error'),
        historySection: panel.querySelector('.history-section'),
        historyToggle: panel.querySelector('.history-toggle'),
        historyList: panel.querySelector('.history-list'),
        settings: panel.querySelector('.settings'),
        settingsToggle: panel.querySelector('.settings-toggle'),
        collapseToggle: panel.querySelector('.collapse-toggle'),
        relayUrl: panel.querySelector('.relay-url'),
        urlError: panel.querySelector('.url-error'),
        hideNative: panel.querySelector('.hide-native'),
      };

      // Event handlers
      elements.collapseToggle.addEventListener('click', async (e) => {
        e.stopPropagation();
        const collapsed = elements.body.classList.toggle('hidden');
        elements.collapseToggle.innerHTML = collapsed ? '&#9650;' : '&#9660;';
        await Config.setCollapsed(collapsed);
      });

      elements.settingsToggle.addEventListener('click', (e) => {
        e.stopPropagation();
        elements.settings.classList.toggle('hidden');
      });

      elements.connectToggle.addEventListener('click', async (e) => {
        e.stopPropagation();
        const current = State.get();
        if (current.connectionState === 'disconnected') {
          await Config.setConnected(true);
          SSE.connect();
        } else {
          await Config.setConnected(false);
          SSE.disconnect();
        }
      });

      elements.stopBtn.addEventListener('click', async () => {
        if (!State.isConnected()) {
          Toast.show('Not connected to relay server', 'warning');
          return;
        }
        const current = State.get();
        // Optimistic: show idle immediately
        State.update({ tracking: false });
        try {
          await API.stopTracking(current.taskId);
        } catch (err) {
          // Revert on error
          State.update({
            tracking: true,
            taskId: current.taskId,
            taskTitle: current.taskTitle,
            startedAt: current.startedAt,
          });
          Toast.show(`Failed to stop: ${err.message}`, 'error');
        }
      });

      let urlTimeout = null;
      elements.relayUrl.addEventListener('input', () => {
        clearTimeout(urlTimeout);
        urlTimeout = setTimeout(async () => {
          const url = elements.relayUrl.value.trim();
          if (!url) return;

          if (!API.isValidUrl(url)) {
            elements.relayUrl.classList.add('input-error');
            elements.urlError.classList.add('visible');
            return;
          }

          elements.relayUrl.classList.remove('input-error');
          elements.urlError.classList.remove('visible');
          await Config.setRelayUrl(url);

          // Reconnect if currently connected
          const current = State.get();
          if (current.connectionState !== 'disconnected') {
            SSE.disconnect();
            SSE.connect();
          }
        }, 800);
      });

      elements.hideNative.addEventListener('change', async () => {
        const hide = elements.hideNative.checked;
        await Config.setHideNative(hide);
        DOM.applyNativeHiding(hide);
      });

      let historyOpen = false;
      elements.historyToggle.addEventListener('click', async () => {
        historyOpen = !historyOpen;
        elements.historyList.classList.toggle('hidden', !historyOpen);
        elements.historyToggle.innerHTML = historyOpen ? 'Recent sessions &#9650;' : 'Recent sessions &#9660;';

        if (historyOpen) {
          await loadHistory();
        }
      });

      document.body.appendChild(root);
    }

    async function loadHistory() {
      try {
        const sessions = await API.getHistory(10);
        if (!Array.isArray(sessions) || sessions.length === 0) {
          elements.historyList.innerHTML = '<div class="history-empty">No sessions yet</div>';
          return;
        }

        elements.historyList.innerHTML = sessions.map((s) => `
          <div class="history-entry">
            <span class="history-task" title="${escapeHtml(s.title || 'Unknown')}">${escapeHtml(s.title || 'Unknown')}</span>
            <span class="history-meta">
              <span class="history-duration">${formatDuration(s.duration)}</span>
              <span>${formatRelativeTime(s.stoppedAt)}</span>
            </span>
          </div>
        `).join('');
      } catch {
        elements.historyList.innerHTML = '<div class="history-empty">Failed to load history</div>';
      }
    }

    function showInlineError(msg) {
      elements.error.textContent = msg;
      elements.error.classList.remove('hidden');
      setTimeout(() => elements.error.classList.add('hidden'), 5000);
    }

    function render(state) {
      // Update timer
      if (timerInterval) {
        clearInterval(timerInterval);
        timerInterval = null;
      }

      const cs = state.connectionState;
      const isConnected = cs === 'connected';
      const isActive = cs !== 'disconnected';

      // Update connect toggle
      elements.connectToggle.classList.toggle('active', isActive);
      elements.connectToggle.title = isActive ? 'Disconnect from relay server' : 'Connect to relay server';

      // Update stop button disabled state
      elements.stopBtn.disabled = !isConnected;

      // Show/hide history section (only when connected)
      elements.historySection.classList.toggle('hidden', !isConnected);

      if (cs === 'disconnected') {
        elements.status.className = 'status disconnected';
        elements.status.innerHTML = `
          <div class="status-row"><span class="dot red"></span><span class="status-label">Disconnected</span></div>
          <button class="connect-toggle" title="Connect to relay server">&#9211;</button>
        `;
        elements.controls.classList.add('hidden');
        rebindConnectToggle();
      } else if (cs === 'connecting') {
        elements.status.className = 'status connecting';
        elements.status.innerHTML = `
          <div class="status-row"><span class="dot yellow pulse"></span><span class="status-label">Connecting...</span></div>
          <button class="connect-toggle active" title="Disconnect from relay server">&#9211;</button>
        `;
        elements.controls.classList.add('hidden');
        rebindConnectToggle();
      } else if (cs === 'reconnecting') {
        elements.status.className = 'status reconnecting';
        elements.status.innerHTML = `
          <div class="status-row"><span class="dot yellow pulse"></span><span class="status-label">Reconnecting...</span></div>
          <button class="connect-toggle active" title="Disconnect from relay server">&#9211;</button>
        `;
        elements.controls.classList.add('hidden');
        rebindConnectToggle();
      } else if (state.tracking) {
        elements.status.className = 'status tracking';
        elements.status.innerHTML = `
          <span class="dot green"></span>
          <div class="task-title">${escapeHtml(state.taskTitle || 'Unknown task')}</div>
          <div class="elapsed">${formatElapsed(state.startedAt)}</div>
          <button class="connect-toggle active" title="Disconnect from relay server">&#9211;</button>
        `;
        elements.controls.classList.remove('hidden');
        timerInterval = setInterval(() => {
          const el = shadow.querySelector('.elapsed');
          if (el) el.textContent = formatElapsed(state.startedAt);
        }, 1000);
        rebindConnectToggle();
      } else {
        // Connected but idle
        elements.status.className = 'status idle';
        elements.status.innerHTML = `
          <div class="status-row"><span class="dot gray"></span><span class="status-label">No active task</span></div>
          <button class="connect-toggle active" title="Disconnect from relay server">&#9211;</button>
        `;
        elements.controls.classList.add('hidden');
        rebindConnectToggle();
      }
    }

    function rebindConnectToggle() {
      const btn = shadow.querySelector('.connect-toggle');
      if (!btn) return;
      elements.connectToggle = btn;
      btn.addEventListener('click', async (e) => {
        e.stopPropagation();
        const current = State.get();
        if (current.connectionState === 'disconnected') {
          await Config.setConnected(true);
          SSE.connect();
        } else {
          await Config.setConnected(false);
          SSE.disconnect();
        }
      });
    }

    function escapeHtml(str) {
      const div = document.createElement('div');
      div.textContent = str;
      return div.innerHTML;
    }

    return {
      async init() {
        createPanel();

        // Restore collapsed state
        const collapsed = await Config.getCollapsed();
        if (collapsed) {
          elements.body.classList.add('hidden');
          elements.collapseToggle.innerHTML = '&#9650;';
        }

        // Load settings
        elements.relayUrl.value = await Config.getRelayUrl();
        elements.hideNative.checked = await Config.getHideNative();

        // Show settings if first run
        if (await Config.isFirstRun()) {
          elements.settings.classList.remove('hidden');
        }

        return { render, shadow };
      },

      showInlineError,
    };
  })();

  // === DOM ===

  const DOM = (() => {
    let observer = null;
    let onStartClick = null;
    let nativeStyleEl = null;

    const TASK_SELECTOR = 'div[data-item-id][data-item-type="task"]';
    const TITLE_SELECTOR = '.TitlePart';
    const BUTTON_CLASS = 'mrt-start-btn';

    const NATIVE_HIDE_CSS = `
      .trackingIcon,
      .timerButton,
      .timer-button,
      [class*="TrackingControls"],
      .time-tracking-button {
        display: none !important;
      }
    `;

    function injectButton(taskEl) {
      if (taskEl.querySelector(`.${BUTTON_CLASS}`)) return;

      const btn = document.createElement('button');
      btn.className = BUTTON_CLASS;
      btn.textContent = '\u25B6';
      btn.title = 'Start tracking (Relay)';
      Object.assign(btn.style, {
        background: 'none',
        border: '1px solid rgba(166,227,161,0.5)',
        borderRadius: '4px',
        color: '#a6e3a1',
        cursor: 'pointer',
        fontSize: '10px',
        padding: '2px 5px',
        marginLeft: '4px',
        opacity: '0.6',
        transition: 'opacity 0.15s',
        verticalAlign: 'middle',
      });
      btn.addEventListener('mouseenter', () => {
        if (!btn.disabled) btn.style.opacity = '1';
      });
      btn.addEventListener('mouseleave', () => {
        btn.style.opacity = btn.disabled ? '0.3' : '0.6';
      });

      btn.addEventListener('click', async (e) => {
        e.stopPropagation();
        e.preventDefault();

        if (btn.disabled) return;

        const itemId = taskEl.getAttribute('data-item-id');
        const titleEl = taskEl.querySelector(TITLE_SELECTOR);
        const title = titleEl ? titleEl.textContent.trim() : 'Unknown';

        if (onStartClick) {
          await onStartClick(itemId, title);
        }
      });

      // Apply initial disabled state
      applyButtonState(btn);

      const titleEl = taskEl.querySelector(TITLE_SELECTOR);
      if (titleEl) {
        titleEl.parentElement.appendChild(btn);
      }
    }

    function applyButtonState(btn) {
      const disabled = !State.isConnected();
      btn.disabled = disabled;
      btn.style.opacity = disabled ? '0.3' : '0.6';
      btn.style.cursor = disabled ? 'not-allowed' : 'pointer';
      btn.title = disabled ? 'Connect to relay server first' : 'Start tracking (Relay)';
    }

    function updateButtonStates() {
      const buttons = document.querySelectorAll(`.${BUTTON_CLASS}`);
      buttons.forEach(applyButtonState);
    }

    function scanAndInject(container) {
      const tasks = container.querySelectorAll(TASK_SELECTOR);
      tasks.forEach(injectButton);
    }

    return {
      init(startClickHandler) {
        onStartClick = startClickHandler;

        // Initial scan
        scanAndInject(document.body);

        // Observe for new task elements
        const target = document.querySelector('.List2Wrapper') || document.body;
        observer = new MutationObserver((mutations) => {
          for (const mutation of mutations) {
            for (const node of mutation.addedNodes) {
              if (node.nodeType !== Node.ELEMENT_NODE) continue;
              if (node.matches && node.matches(TASK_SELECTOR)) {
                injectButton(node);
              }
              if (node.querySelectorAll) {
                scanAndInject(node);
              }
            }
          }
        });
        observer.observe(target, { childList: true, subtree: true });
      },

      updateButtonStates,

      applyNativeHiding(hide) {
        if (hide && !nativeStyleEl) {
          nativeStyleEl = GM_addStyle(NATIVE_HIDE_CSS);
        } else if (!hide && nativeStyleEl) {
          nativeStyleEl.remove();
          nativeStyleEl = null;
        }
      },
    };
  })();

  // === Init ===

  async function init() {
    // 1. Create UI
    const { render, shadow } = await UI.init();

    // 2. Init toast system
    Toast.init(shadow);

    // 3. Wire state changes to UI and DOM buttons
    State.onChange(render);
    State.onChange(() => DOM.updateButtonStates());

    // 4. Wire DOM start clicks with connection guard
    const handleStartClick = async (taskId, title) => {
      if (!State.isConnected()) {
        Toast.show('Not connected to relay server', 'warning');
        return;
      }

      // Optimistic update
      State.update({
        tracking: true,
        taskId,
        taskTitle: title,
        startedAt: Date.now(),
      });

      try {
        const resp = await API.startTracking(taskId, title);
        // Use server startedAt if available
        if (resp && resp.startedAt) {
          State.update({
            tracking: true,
            taskId,
            taskTitle: title,
            startedAt: resp.startedAt,
          });
        }
      } catch (err) {
        // Revert on error
        State.update({ tracking: false });
        Toast.show(`Failed to start: ${err.message}`, 'error');
      }
    };

    // 5. Connect SSE based on persisted preference
    const shouldConnect = await Config.getConnected();
    if (shouldConnect) {
      await SSE.connect();
    }

    // 6. Wait for Marvin DOM, then start DOM observer
    const waitForMarvin = () => {
      return new Promise((resolve) => {
        const check = () => {
          if (document.querySelector('.List2Wrapper') || document.querySelector('[data-item-id]')) {
            resolve();
          } else {
            setTimeout(check, 500);
          }
        };
        check();
        // Give up after 30s and init anyway
        setTimeout(resolve, 30000);
      });
    };

    await waitForMarvin();
    DOM.init(handleStartClick);

    // 7. Apply native hiding if configured
    const hideNative = await Config.getHideNative();
    DOM.applyNativeHiding(hideNative);
  }

  init();
})();
