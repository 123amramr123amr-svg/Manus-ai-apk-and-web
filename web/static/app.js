/* Manus AI web app — frontend logic (vanilla ES2020, no build step). */
'use strict';

const I18N = window.MANUS_I18N || { ar: {}, en: {} };
const STORAGE = { apiKey: 'manus.apiKey', lang: 'manus.lang', task: 'manus.lastTask' };
const POLL_MS = 4000;

const state = {
  lang: localStorage.getItem(STORAGE.lang) || 'ar',
  apiKey: localStorage.getItem(STORAGE.apiKey) || '',
  server: null,
  profiles: [],
  locales: [],
  taskId: localStorage.getItem(STORAGE.task) || '',
  task: null,
  messages: [],
  tasks: [],
  projects: [],
  pendingFiles: [],
  pollTimer: null,
  view: 'chat',
  installPrompt: null
};

const $ = (selector) => document.querySelector(selector);
const $$ = (selector) => Array.from(document.querySelectorAll(selector));

/* ------------------------------------------------------------------ i18n */
function t(key, fallback) {
  const dict = I18N[state.lang] || {};
  if (dict[key] !== undefined) return dict[key];
  const ar = I18N.ar || {};
  if (ar[key] !== undefined) return ar[key];
  return fallback !== undefined ? fallback : key;
}

function applyI18n() {
  document.documentElement.lang = state.lang;
  document.documentElement.dir = state.lang === 'ar' ? 'rtl' : 'ltr';
  $$('[data-i18n]').forEach((node) => { node.textContent = t(node.dataset.i18n, node.textContent); });
  $$('[data-i18n-placeholder]').forEach((node) => { node.placeholder = t(node.dataset.i18nPlaceholder, node.placeholder); });
  $$('[data-i18n-title]').forEach((node) => { node.title = t(node.dataset.i18nTitle, node.title); });
  $('#btn-lang').textContent = t('lang.switch', 'EN');
  document.title = state.lang === 'ar' ? 'Manus AI — مساعدك الذكي' : 'Manus AI — your agent console';
}

/* --------------------------------------------------------------- helpers */
function el(tag, className, text) {
  const node = document.createElement(tag);
  if (className) node.className = className;
  if (text !== undefined && text !== null) node.textContent = String(text);
  return node;
}

function toast(message, kind) {
  const node = $('#toast');
  node.textContent = message;
  node.className = 'toast' + (kind ? ' is-' + kind : '');
  node.hidden = false;
  clearTimeout(toast.timer);
  toast.timer = setTimeout(() => { node.hidden = true; }, 4200);
}

function fmtDate(value) {
  if (!value) return '—';
  const date = new Date(Number(value));
  if (Number.isNaN(date.getTime())) return '—';
  return date.toLocaleString(state.lang === 'ar' ? 'ar-EG' : 'en-US', {
    dateStyle: 'short', timeStyle: 'short'
  });
}

function fmtNumber(value) {
  const number = Number(value);
  if (!Number.isFinite(number)) return '0';
  return number.toLocaleString(state.lang === 'ar' ? 'ar-EG' : 'en-US', { maximumFractionDigits: 2 });
}

function fmtBytes(bytes) {
  const size = Number(bytes) || 0;
  if (size < 1024) return size + ' B';
  if (size < 1024 * 1024) return (size / 1024).toFixed(1) + ' KB';
  return (size / (1024 * 1024)).toFixed(1) + ' MB';
}

function emptyItem(messageKey) {
  return el('li', 'empty', t(messageKey));
}

/* ------------------------------------------------------------ API client */
class ApiError extends Error {
  constructor(message, status, code) {
    super(message);
    this.status = status;
    this.code = code;
  }
}

async function api(path, options = {}) {
  const headers = Object.assign({}, options.headers);
  if (state.apiKey) headers['X-Manus-Api-Key'] = state.apiKey;

  let body = options.body;
  if (body && !(body instanceof FormData)) {
    headers['Content-Type'] = 'application/json';
    body = JSON.stringify(body);
  }

  const response = await fetch(path, {
    method: options.method || 'GET',
    headers,
    body,
    cache: 'no-store'
  });

  const text = await response.text();
  let data = null;
  if (text) {
    try { data = JSON.parse(text); } catch { data = { message: text }; }
  }

  if (!response.ok) {
    const message = (data && data.message) || response.statusText || 'request failed';
    throw new ApiError(message, response.status, data && data.error);
  }
  return data;
}

/* --------------------------------------------------------------- startup */
async function boot() {
  applyI18n();
  bindEvents();
  registerServiceWorker();

  try {
    state.server = await api('/api/config');
    $('#app-version').textContent = state.server.version || 'dev';
    renderServerInfo();
  } catch (error) {
    setBrand(t('brand.error'), 'is-error');
  }

  const hasKey = state.apiKey || (state.server && state.server.server_key_configured);
  if (hasKey) {
    setBrand(t('brand.ready'), 'is-ok');
  } else {
    setBrand(t('brand.noKey'), '');
    openSettings();
  }

  await loadProfiles();
  await loadTasks();
  await loadProjects();
  if (state.taskId) {
    $('#task-select').value = state.taskId;
    await openTask(state.taskId, false);
  }
}

function setBrand(text, className) {
  const node = $('#brand-status');
  node.textContent = text;
  node.className = 'brand-status' + (className ? ' ' + className : '');
}

function renderServerInfo() {
  const server = state.server || {};
  const box = $('#server-info');
  box.textContent = '';
  const rows = [
    [t('settings.baseUrl'), server.base_url || 'https://api.manus.ai'],
    [t('settings.serverKey'), server.server_key_configured ? t('settings.serverKeyYes') : t('settings.serverKeyNo')],
    [t('settings.webhookToken'), server.webhook_token_required ? t('settings.required') : t('settings.notRequired')],
    [t('settings.maxUpload'), fmtBytes(server.max_upload_bytes)]
  ];
  rows.forEach(([label, value]) => {
    const row = el('div');
    row.append(el('span', null, label + ': '), el('code', null, value));
    box.append(row);
  });
}

function openSettings() {
  $('#api-key').value = state.apiKey;
  renderServerInfo();
  const dialog = $('#settings-dialog');
  if (typeof dialog.showModal === 'function') {
    if (!dialog.open) dialog.showModal();
  } else {
    dialog.setAttribute('open', '');
  }
}

function closeSettings() {
  const dialog = $('#settings-dialog');
  if (typeof dialog.close === 'function') dialog.close();
  else dialog.removeAttribute('open');
}

function refreshBrand() {
  const hasKey = Boolean(state.apiKey) || Boolean(state.server && state.server.server_key_configured);
  setBrand(hasKey ? t('brand.ready') : t('brand.noKey'), hasKey ? 'is-ok' : '');
}

async function saveKey() {
  state.apiKey = $('#api-key').value.trim();
  if (state.apiKey) localStorage.setItem(STORAGE.apiKey, state.apiKey);
  else localStorage.removeItem(STORAGE.apiKey);
  refreshBrand();
  toast(t('toast.keySaved'), 'ok');
  await loadProfiles();
  await loadTasks();
}

function clearKey() {
  state.apiKey = '';
  localStorage.removeItem(STORAGE.apiKey);
  $('#api-key').value = '';
  refreshBrand();
  toast(t('toast.keyCleared'));
}

async function testKey() {
  try {
    const credits = await api('/api/credits');
    const total = credits && credits.data ? credits.data.total_credits : 0;
    toast(t('toast.connected') + ' (' + fmtNumber(total) + ')', 'ok');
  } catch (error) {
    toast(error.message, 'error');
  }
}

/* --------------------------------------------------------------- events */
function bindEvents() {
  $('#btn-lang').addEventListener('click', () => {
    state.lang = state.lang === 'ar' ? 'en' : 'ar';
    localStorage.setItem(STORAGE.lang, state.lang);
    applyI18n();
    renderMessages();
    renderTasks();
    renderCredits();
    renderUsage();
    renderTools();
    renderEvents();
    renderServerInfo();
    refreshBrand();
  });

  $('#btn-settings').addEventListener('click', openSettings);
  $('#btn-close-settings').addEventListener('click', closeSettings);
  $('#btn-save-key').addEventListener('click', saveKey);
  $('#btn-clear-key').addEventListener('click', clearKey);
  $('#btn-test-key').addEventListener('click', testKey);
  $('#btn-install').addEventListener('click', triggerInstall);

  $$('.tab').forEach((tab) => tab.addEventListener('click', () => showView(tab.dataset.view)));

  $('#btn-load-tasks').addEventListener('click', loadTasks);
  $('#btn-load-usage').addEventListener('click', () => { loadCredits(); loadUsage(); });
  $('#btn-load-tools').addEventListener('click', loadTools);
  $('#btn-load-events').addEventListener('click', loadEvents);
  $('#btn-clear-events').addEventListener('click', clearEvents);
  $('#btn-refresh-task').addEventListener('click', () => refreshCurrentTask(true));
  $('#btn-stop-task').addEventListener('click', stopTask);

  $('#task-select').addEventListener('change', (event) => {
    const value = event.target.value;
    if (!value) {
      state.taskId = '';
      state.task = null;
      state.messages = [];
      localStorage.removeItem(STORAGE.task);
      renderMessages();
      updateChatChrome();
      return;
    }
    openTask(value, true);
  });

  $('#new-task-form').addEventListener('submit', createTask);
  $('#btn-upload-new').addEventListener('click', () => $('#file-input').click());
  $('#composer').addEventListener('submit', sendMessage);
  $('#btn-attach').addEventListener('click', () => $('#file-input').click());
  $('#file-input').addEventListener('change', handleFiles);
  $('#composer-text').addEventListener('keydown', (event) => {
    if (event.key === 'Enter' && !event.shiftKey) {
      event.preventDefault();
      $('#composer').requestSubmit();
    }
  });

  $('#project-form').addEventListener('submit', createProject);
  $('#webhook-form').addEventListener('submit', createWebhook);
  $('#btn-use-own-url').addEventListener('click', () => {
    $('#webhook-url').value = window.location.origin + '/api/webhook';
  });

  window.addEventListener('online', () => toast(t('toast.online'), 'ok'));
  window.addEventListener('offline', () => toast(t('toast.offline'), 'error'));
  window.addEventListener('beforeinstallprompt', (event) => {
    event.preventDefault();
    state.installPrompt = event;
  });
  document.addEventListener('visibilitychange', () => {
    if (!document.hidden) refreshCurrentTask(false);
  });
}

function showView(view) {
  state.view = view;
  $$('.view').forEach((node) => node.classList.toggle('is-active', node.id === 'view-' + view));
  $$('.tab').forEach((node) => node.classList.toggle('is-active', node.dataset.view === view));
  if (view === 'tasks') loadTasks();
  if (view === 'usage') { loadCredits(); loadUsage(); }
  if (view === 'tools') loadTools();
  if (view === 'activity') loadEvents();
}

function triggerInstall() {
  if (state.installPrompt) {
    state.installPrompt.prompt();
    state.installPrompt = null;
    return;
  }
  if (window.matchMedia('(display-mode: standalone)').matches) {
    toast(t('toast.installed'), 'ok');
    return;
  }
  toast(t('toast.installHint'));
}

/* ------------------------------------------------------- profiles & tasks */
async function loadProfiles() {
  try {
    const data = await api('/api/profiles');
    state.profiles = (data && data.profiles) || [];
    state.locales = (data && data.locales) || [];

    const profileSelect = $('#nt-profile');
    profileSelect.textContent = '';
    state.profiles.forEach((profile) => {
      const option = el('option', null, profile.id);
      option.value = profile.id;
      profileSelect.append(option);
    });
    if (!state.profiles.length) profileSelect.append(el('option', null, 'manus-1.6'));

    const localeSelect = $('#nt-locale');
    localeSelect.textContent = '';
    state.locales.forEach((locale) => {
      const option = el('option', null, locale);
      option.value = locale;
      localeSelect.append(option);
    });
    localeSelect.value = state.lang === 'ar' ? 'ar-EG' : 'en-US';
  } catch (error) {
    /* profiles are cosmetic: keep the defaults already in the DOM */
  }
}

async function loadTasks() {
  try {
    const limit = Number($('#tasks-limit').value) || 20;
    const data = await api('/api/tasks?limit=' + limit);
    state.tasks = (data && data.tasks) || [];
    renderTasks();
    renderTaskOptions();
  } catch (error) {
    if (error.code !== 'missing_api_key') toast(error.message, 'error');
  }
}

function renderTaskOptions() {
  const select = $('#task-select');
  const current = state.taskId;
  select.textContent = '';
  const none = el('option', null, t('chat.noTask'));
  none.value = '';
  select.append(none);

  state.tasks.forEach((task) => {
    const option = el('option', null, (task.title || task.id) + ' — ' + (task.status || ''));
    option.value = task.id;
    select.append(option);
  });

  if (current && !state.tasks.some((task) => task.id === current)) {
    const option = el('option', null, current);
    option.value = current;
    select.append(option);
  }
  select.value = current;
}

function renderTasks() {
  const list = $('#tasks-list');
  list.textContent = '';
  const tasks = state.tasks || [];
  $('#tasks-empty').hidden = tasks.length > 0;
  if (!tasks.length) return;

  tasks.forEach((task) => {
    const item = el('li');
    const head = el('div', 'item-title');
    head.append(el('strong', null, task.title || t('tasks.noTitle')));
    head.append(el('span', 'badge ' + (task.status || ''), task.status || t('common.unknown')));
    item.append(head);
    item.append(el('div', 'item-sub',
      t('tasks.created') + ': ' + fmtDate(task.created_at) + ' • ' +
      t('tasks.credits') + ': ' + fmtNumber(task.credit_usage)));
    item.append(el('div', 'item-sub', t('chat.taskId') + ': ' + task.id));

    const actions = el('div', 'item-actions');
    actions.append(button(t('tasks.open'), 'ghost-btn', () => { openInChat(task.id); }));
    actions.append(button(t('common.rename'), 'ghost-btn', () => renameTask(task)));
    actions.append(button(t('chat.stop'), 'ghost-btn', async () => {
      try {
        await api('/api/tasks/' + encodeURIComponent(task.id) + '/stop', { method: 'POST' });
        toast(t('toast.stopped'), 'ok');
        loadTasks();
      } catch (error) { toast(error.message, 'error'); }
    }));
    actions.append(button(t('common.delete'), 'danger-btn', async () => {
      if (!window.confirm(t('common.confirmDelete'))) return;
      try {
        await api('/api/tasks/' + encodeURIComponent(task.id), { method: 'DELETE' });
        if (state.taskId === task.id) {
          state.taskId = '';
          state.task = null;
          state.messages = [];
          localStorage.removeItem(STORAGE.task);
          renderMessages();
          updateChatChrome();
        }
        toast(t('toast.taskDeleted'), 'ok');
        loadTasks();
      } catch (error) { toast(error.message, 'error'); }
    }));
    item.append(actions);
    list.append(item);
  });
}

function button(label, className, onClick) {
  const node = el('button', className, label);
  node.type = 'button';
  node.addEventListener('click', onClick);
  return node;
}

async function renameTask(task) {
  const title = window.prompt(t('tasks.newTitle'), task.title || '');
  if (title === null) return;
  try {
    await api('/api/tasks/' + encodeURIComponent(task.id), { method: 'PATCH', body: { title } });
    loadTasks();
    if (state.taskId === task.id) refreshCurrentTask(true);
  } catch (error) { toast(error.message, 'error'); }
}

function openInChat(taskId) {
  state.taskId = taskId;
  localStorage.setItem(STORAGE.task, taskId);
  $('#task-select').value = taskId;
  showView('chat');
  openTask(taskId, true);
}

/* ------------------------------------------------------------------ chat */
async function openTask(taskId, showError) {
  state.taskId = taskId;
  localStorage.setItem(STORAGE.task, taskId);
  updateChatChrome();
  try {
    state.task = await api('/api/tasks/' + encodeURIComponent(taskId));
    await loadMessages();
  } catch (error) {
    if (showError) toast(error.message, 'error');
  }
  updateChatChrome();
  startPolling();
}

async function refreshCurrentTask(showError) {
  if (!state.taskId) return;
  try {
    state.task = await api('/api/tasks/' + encodeURIComponent(state.taskId));
    await loadMessages();
    updateChatChrome();
  } catch (error) {
    if (showError) toast(error.message, 'error');
  }
}

async function loadMessages() {
  if (!state.taskId) return;
  const data = await api('/api/tasks/' + encodeURIComponent(state.taskId) + '/messages?limit=100&verbose=true');
  state.messages = (data && data.messages) || [];
  renderMessages();
}

function startPolling() {
  stopPolling();
  if (!state.taskId) return;
  state.pollTimer = setInterval(async () => {
    if (document.hidden || state.view !== 'chat' || !state.taskId) return;
    try {
      state.task = await api('/api/tasks/' + encodeURIComponent(state.taskId));
      await loadMessages();
      updateChatChrome();
    } catch (error) { /* transient errors are ignored while polling */ }
  }, POLL_MS);
}

function stopPolling() {
  if (state.pollTimer) clearInterval(state.pollTimer);
  state.pollTimer = null;
}

function updateChatChrome() {
  const hasTask = Boolean(state.taskId);
  $('#composer').hidden = !hasTask;
  $('#chat-empty').hidden = hasTask || state.messages.length > 0;
  $('#btn-stop-task').disabled = !hasTask;
  $('#btn-refresh-task').disabled = !hasTask;

  const meta = $('#chat-meta');
  meta.textContent = '';
  if (!hasTask) {
    meta.hidden = true;
    return;
  }
  meta.hidden = false;
  const task = state.task || {};
  if (task.id) meta.append(el('span', null, t('chat.taskId') + ': ' + task.id));
  if (task.agent_status) meta.append(el('span', 'badge ' + task.agent_status, task.agent_status));
  if (task.credit_usage !== undefined) meta.append(el('span', null, t('chat.credits') + ': ' + fmtNumber(task.credit_usage)));
  if (task.updated_at) meta.append(el('span', null, t('chat.updated') + ': ' + fmtDate(task.updated_at)));
  if (task.task_url) {
    const link = el('a', 'ghost-btn', t('chat.openInManus'));
    link.href = task.task_url;
    link.target = '_blank';
    link.rel = 'noopener';
    meta.append(link);
  }
}

function extractParts(payload) {
  const texts = [];
  const files = [];
  if (!payload || typeof payload !== 'object') return { texts, files };

  const content = payload.content;
  if (Array.isArray(content)) {
    content.forEach((part) => {
      if (!part || typeof part !== 'object') return;
      const type = String(part.type || '');
      if (type === 'text' || type === 'thinking' || type === 'reasoning' || type === 'markdown') {
        const text = part.text || part.content || part.thinking;
        if (text) texts.push(String(text));
      } else if (part.file_name || part.filename || part.file_url || part.url) {
        files.push(String(part.file_name || part.filename || part.file_url || part.url));
      }
    });
  }
  if (!texts.length && typeof payload.text === 'string') texts.push(payload.text);
  if (!texts.length && typeof payload.message === 'string') texts.push(payload.message);
  return { texts, files };
}

function renderMessages() {
  const box = $('#chat-messages');
  box.textContent = '';
  const messages = (state.messages || []).slice()
    .sort((a, b) => Number(a.timestamp || 0) - Number(b.timestamp || 0));

  messages.forEach((message) => box.append(renderMessage(message)));
  $('#chat-empty').hidden = Boolean(state.taskId) || messages.length > 0;

  if (state.view === 'chat') {
    window.requestAnimationFrame(() => { box.scrollIntoView({ block: 'end' }); });
  }
}

function renderMessage(message) {
  const type = String(message.type || '');
  let payload = null;
  let kind = 'status';
  let label = t('chat.system');

  if (type === 'user_message') { payload = message.user_message; kind = 'user'; label = t('chat.you'); }
  else if (type === 'assistant_message') { payload = message.assistant_message; kind = 'assistant'; label = t('chat.assistant'); }
  else if (type === 'error_message') { payload = message.error_message; kind = 'error'; label = t('chat.errorLabel'); }
  else if (type === 'status_update') { payload = message.status_update; kind = 'status'; label = t('chat.statusLabel'); }

  const node = el('div', 'msg ' + kind);
  const head = el('div', 'msg-head');
  head.append(el('span', null, label));
  head.append(el('span', null, fmtDate(message.timestamp)));
  node.append(head);

  const { texts, files } = extractParts(payload);
  const body = el('div', 'body', texts.length ? texts.join('\n\n') : t('chat.noText'));
  node.append(body);

  if (files.length) {
    const attach = el('div', 'item-sub', t('chat.attachments') + ': ' + files.join(', '));
    node.append(attach);
  }

  if (payload) {
    const details = el('details');
    details.append(el('summary', null, t('chat.raw')));
    details.append(el('pre', null, JSON.stringify(payload, null, 2)));
    node.append(details);
  }
  return node;
}

/* --------------------------------------------------------- chat actions */
async function createTask(event) {
  event.preventDefault();
  const prompt = $('#nt-prompt').value.trim();
  if (!prompt) return;

  const body = {
    prompt,
    title: $('#nt-title').value.trim(),
    agent_profile: $('#nt-profile').value,
    locale: $('#nt-locale').value,
    project_id: $('#nt-project').value,
    interactive: $('#nt-interactive').checked,
    attachments: state.pendingFiles.map((file) => ({ file_id: file.file_id }))
  };

  const submit = event.target.querySelector('button[type="submit"]');
  submit.disabled = true;
  try {
    const created = await api('/api/tasks', { method: 'POST', body });
    toast(t('toast.taskCreated'), 'ok');
    $('#nt-prompt').value = '';
    $('#nt-title').value = '';
    state.pendingFiles = [];
    renderPending();
    await loadTasks();
    openInChat(created.task_id);
  } catch (error) {
    toast(error.message, 'error');
  } finally {
    submit.disabled = false;
  }
}

async function sendMessage(event) {
  event.preventDefault();
  if (!state.taskId) return;

  const input = $('#composer-text');
  const message = input.value.trim();
  if (!message && !state.pendingFiles.length) return;

  const body = {
    message: message || t('chat.attachments'),
    attachments: state.pendingFiles.map((file) => ({ file_id: file.file_id }))
  };

  input.value = '';
  try {
    await api('/api/tasks/' + encodeURIComponent(state.taskId) + '/messages', { method: 'POST', body });
    state.pendingFiles = [];
    renderPending();
    toast(t('toast.messageSent'), 'ok');
    await refreshCurrentTask(false);
  } catch (error) {
    toast(error.message, 'error');
    input.value = message;
  }
}

async function stopTask() {
  if (!state.taskId) return;
  try {
    await api('/api/tasks/' + encodeURIComponent(state.taskId) + '/stop', { method: 'POST' });
    toast(t('toast.stopped'), 'ok');
    await refreshCurrentTask(false);
    loadTasks();
  } catch (error) {
    toast(error.message, 'error');
  }
}

async function handleFiles(event) {
  const files = Array.from(event.target.files || []);
  event.target.value = '';
  for (const file of files) {
    try {
      const uploaded = await uploadFile(file);
      state.pendingFiles.push(uploaded);
      renderPending();
      toast(t('toast.fileUploaded') + ' — ' + file.name, 'ok');
    } catch (error) {
      toast(file.name + ': ' + error.message, 'error');
    }
  }
}

async function uploadFile(file) {
  const form = new FormData();
  form.append('file', file, file.name);
  const uploaded = await api('/api/files', { method: 'POST', body: form });
  return { file_id: uploaded.file_id, filename: uploaded.filename || file.name, size: uploaded.size_bytes };
}

function renderPending() {
  const box = $('#pending-attachments');
  box.textContent = '';
  box.hidden = state.pendingFiles.length === 0;
  state.pendingFiles.forEach((file, index) => {
    const chip = el('span', 'chip');
    chip.append(el('span', null, file.filename + ' (' + fmtBytes(file.size) + ')'));
    chip.append(button('\u2715', null, () => {
      state.pendingFiles.splice(index, 1);
      renderPending();
    }));
    box.append(chip);
  });
}

/* ---------------------------------------------------- credits & usage */
async function loadCredits() {
  try {
    const data = await api('/api/credits');
    state.credits = (data && data.data) || null;
  } catch (error) {
    state.credits = null;
  }
  renderCredits();
}

function renderCredits() {
  const box = $('#credits');
  box.textContent = '';
  const credits = state.credits;
  if (!credits) {
    box.append(el('p', 'hint', t('common.loading')));
    return;
  }
  const cells = [
    [t('usage.total'), credits.total_credits],
    [t('usage.free'), credits.free_credits],
    [t('usage.periodic'), credits.periodic_credits],
    [t('usage.addon'), credits.addon_credits],
    [t('usage.pro'), credits.pro_monthly_credits],
    [t('usage.refresh'), credits.refresh_credits]
  ];
  cells.forEach(([label, value]) => {
    const cell = el('div', 'credit');
    cell.append(el('b', null, fmtNumber(value)));
    cell.append(el('span', null, label));
    box.append(cell);
  });
  if (credits.next_refresh_time) {
    box.append(el('p', 'hint', t('usage.nextRefresh') + ': ' + fmtDate(credits.next_refresh_time)));
  }
}

async function loadUsage() {
  try {
    const data = await api('/api/usage?limit=25');
    state.usage = (data && data.records) || [];
  } catch (error) {
    state.usage = [];
    if (error.code !== 'missing_api_key') toast(error.message, 'error');
  }
  renderUsage();
}

function renderUsage() {
  const list = $('#usage-list');
  list.textContent = '';
  const records = state.usage || [];
  if (!records.length) {
    list.append(emptyItem('usage.empty'));
    return;
  }
  records.forEach((record) => {
    const item = el('li');
    const head = el('div', 'item-title');
    head.append(el('strong', null, record.title || record.task_id || t('common.unknown')));
    head.append(el('span', 'badge', fmtNumber(record.credits)));
    item.append(head);
    item.append(el('div', 'item-sub', fmtDate(record.created_at) + (record.type ? ' • ' + record.type : '')));
    list.append(item);
  });
}

/* --------------------------------------------------------------- tools */
async function loadTools() {
  await Promise.all([loadProjects(), loadAgents(), loadSkills(), loadConnectors(), loadBrowsers(), loadWebhooks()]);
}

async function loadProjects() {
  try {
    const data = await api('/api/projects');
    state.projects = (data && data.projects) || [];
  } catch (error) {
    state.projects = [];
  }
  renderProjects();
}

function renderProjects() {
  const list = $('#projects-list');
  list.textContent = '';
  const projects = state.projects || [];
  if (!projects.length) {
    list.append(emptyItem('tools.empty'));
  } else {
    projects.forEach((project) => {
      const item = el('li');
      item.append(el('strong', null, project.name || project.id));
      if (project.instruction) item.append(el('div', 'item-sub', project.instruction));
      list.append(item);
    });
  }

  const select = $('#nt-project');
  const current = select.value;
  select.textContent = '';
  const none = el('option', null, '—');
  none.value = '';
  select.append(none);
  projects.forEach((project) => {
    const option = el('option', null, project.name || project.id);
    option.value = project.id;
    select.append(option);
  });
  select.value = current;
}

async function createProject(event) {
  event.preventDefault();
  const name = $('#project-name').value.trim();
  if (!name) return;
  try {
    await api('/api/projects', {
      method: 'POST',
      body: { name, instruction: $('#project-instruction').value.trim() }
    });
    $('#project-name').value = '';
    $('#project-instruction').value = '';
    toast(t('toast.projectCreated'), 'ok');
    await loadProjects();
  } catch (error) {
    toast(error.message, 'error');
  }
}

async function loadAgents() {
  const list = $('#agents-list');
  list.textContent = '';
  try {
    const data = await api('/api/agents');
    const agents = (data && data.agents) || [];
    if (!agents.length) list.append(emptyItem('tools.empty'));
    agents.forEach((agent) => {
      const item = el('li');
      item.append(el('strong', null, agent.nickname || agent.id));
      item.append(el('div', 'item-sub', agent.about || agent.task_id || ''));
      list.append(item);
    });
  } catch (error) {
    list.append(emptyItem('tools.empty'));
  }
}

async function loadSkills() {
  const list = $('#skills-list');
  list.textContent = '';
  try {
    const data = await api('/api/skills');
    const skills = (data && data.skills) || [];
    if (!skills.length) list.append(emptyItem('tools.empty'));
    skills.forEach((skill) => {
      const item = el('li');
      item.append(el('strong', null, skill.name || skill.id));
      item.append(el('div', 'item-sub', skill.description || ''));
      list.append(item);
    });
  } catch (error) {
    list.append(emptyItem('tools.empty'));
  }
}

async function loadConnectors() {
  const list = $('#connectors-list');
  list.textContent = '';
  try {
    const data = await api('/api/connectors');
    const connectors = (data && data.connectors) || [];
    if (!connectors.length) list.append(emptyItem('tools.empty'));
    connectors.forEach((connector) => {
      const item = el('li');
      item.append(el('strong', null, connector.name || connector.id));
      item.append(el('div', 'item-sub', connector.description || connector.category || ''));
      list.append(item);
    });
  } catch (error) {
    list.append(emptyItem('tools.empty'));
  }
}

async function loadBrowsers() {
  const list = $('#browsers-list');
  list.textContent = '';
  try {
    const data = await api('/api/browser-clients');
    const clients = (data && data.clients) || [];
    if (!clients.length) list.append(emptyItem('tools.empty'));
    clients.forEach((client) => {
      const item = el('li');
      item.append(el('strong', null, client.client_name || client.client_id));
      item.append(el('div', 'item-sub', client.user_agent || ''));
      list.append(item);
    });
  } catch (error) {
    list.append(emptyItem('tools.empty'));
  }
}

/* ------------------------------------------------------------ webhooks */
async function loadWebhooks() {
  const list = $('#webhooks-list');
  list.textContent = '';
  $('#webhook-hint').textContent = t('tools.webhookHint') + ' ' + window.location.origin + '/api/webhook';
  try {
    const data = await api('/api/webhooks');
    const webhooks = (data && data.webhooks) || [];
    if (!webhooks.length) list.append(emptyItem('tools.empty'));
    webhooks.forEach((webhook) => {
      const item = el('li');
      const head = el('div', 'item-title');
      head.append(el('strong', null, webhook.url));
      head.append(el('span', 'badge ' + (webhook.status || ''), webhook.status || ''));
      item.append(head);
      item.append(el('div', 'item-sub', fmtDate(webhook.created_at)));
      const actions = el('div', 'item-actions');
      actions.append(button(t('common.delete'), 'danger-btn', async () => {
        try {
          await api('/api/webhooks/' + encodeURIComponent(webhook.id), { method: 'DELETE' });
          toast(t('toast.webhookDeleted'), 'ok');
          loadWebhooks();
        } catch (error) { toast(error.message, 'error'); }
      }));
      item.append(actions);
      list.append(item);
    });
  } catch (error) {
    list.append(emptyItem('tools.empty'));
  }
}

async function createWebhook(event) {
  event.preventDefault();
  const url = $('#webhook-url').value.trim();
  if (!url) return;
  try {
    await api('/api/webhooks', { method: 'POST', body: { url } });
    $('#webhook-url').value = '';
    toast(t('toast.webhookCreated'), 'ok');
    await loadWebhooks();
  } catch (error) {
    toast(error.message, 'error');
  }
}

/* ------------------------------------------------------------ activity */
async function loadEvents() {
  const list = $('#events-list');
  list.textContent = '';
  try {
    const data = await api('/api/webhook/events');
    state.events = (data && data.events) || [];
  } catch (error) {
    state.events = [];
  }
  renderEvents();
}

function renderEvents() {
  const list = $('#events-list');
  list.textContent = '';
  const events = state.events || [];
  $('#events-empty').hidden = events.length > 0;
  if (!events.length) return;

  events.forEach((event) => {
    const item = el('li');
    const head = el('div', 'item-title');
    head.append(el('strong', null, t('activity.' + event.kind, event.event_type)));
    head.append(el('span', 'badge ' + event.kind, event.event_type || ''));
    item.append(head);
    item.append(el('div', 'item-sub', t('activity.receivedAt') + ': ' + fmtDate(new Date(event.received_at).getTime())));

    const detail = event.task_detail || {};
    const taskId = detail.task_id || detail.id;
    if (taskId) item.append(el('div', 'item-sub', t('chat.taskId') + ': ' + taskId));
    if (detail.title) item.append(el('div', 'item-sub', detail.title));

    if (event.raw) {
      const details = el('details');
      details.append(el('summary', null, t('chat.raw')));
      details.append(el('pre', null, event.raw));
      item.append(details);
    }
    list.append(item);
  });
}

async function clearEvents() {
  try {
    await api('/api/webhook/events', { method: 'DELETE' });
    state.events = [];
    renderEvents();
    toast(t('toast.eventsCleared'), 'ok');
  } catch (error) {
    toast(error.message, 'error');
  }
}

/* ----------------------------------------------------------------- boot */
document.addEventListener('DOMContentLoaded', () => {
  boot().catch((error) => toast(error.message, 'error'));
});
