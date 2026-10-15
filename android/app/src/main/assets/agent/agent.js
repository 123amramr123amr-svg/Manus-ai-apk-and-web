/* Standalone agent mode: an OpenManus-style assistant that runs entirely on
   the device. It talks to any OpenAI-compatible LLM endpoint using the user's
   own key and uses free public services for search and page reading. */
'use strict';

const AGENT_I18N = window.MANUS_AGENT_I18N || { ar: {}, en: {} };
const PROVIDERS = window.MANUS_AGENT_PROVIDERS || [];
const TOOL_SCHEMAS = window.MANUS_AGENT_TOOLS || [];
const TOOL_IMPL = window.MANUS_AGENT_TOOLS_IMPL || {};

const KEYS = {
  lang: 'manus.agent.lang',
  settings: 'manus.agent.settings',
  sessions: 'manus.agent.sessions',
  current: 'manus.agent.current'
};

const DEFAULTS = {
  providerId: 'deepseek',
  baseUrl: 'https://api.deepseek.com/v1',
  apiKey: '',
  model: 'deepseek-chat',
  temperature: 0.3,
  maxSteps: 6,
  tools: true,
  systemPrompt: ''
};

const state = {
  lang: localStorage.getItem(KEYS.lang) || 'ar',
  settings: Object.assign({}, DEFAULTS, readJSON(KEYS.settings, {})),
  sessions: readJSON(KEYS.sessions, []),
  currentId: localStorage.getItem(KEYS.current) || '',
  messages: [],
  running: false,
  abort: null,
  step: 0
};

const $ = (selector) => document.querySelector(selector);

function readJSON(key, fallback) {
  try {
    const raw = localStorage.getItem(key);
    return raw ? JSON.parse(raw) : fallback;
  } catch (error) {
    return fallback;
  }
}

function saveSettings() {
  localStorage.setItem(KEYS.settings, JSON.stringify(state.settings));
}

function saveSessions() {
  localStorage.setItem(KEYS.sessions, JSON.stringify(state.sessions.slice(0, 40)));
}

function t(key, fallback) {
  const dict = AGENT_I18N[state.lang] || {};
  if (dict[key] !== undefined) return dict[key];
  const ar = AGENT_I18N.ar || {};
  if (ar[key] !== undefined) return ar[key];
  return fallback !== undefined ? fallback : key;
}

function applyI18n() {
  document.documentElement.lang = state.lang;
  document.documentElement.dir = state.lang === 'ar' ? 'rtl' : 'ltr';
  document.querySelectorAll('[data-i18n]').forEach((node) => {
    node.textContent = t(node.dataset.i18n, node.textContent);
  });
  document.querySelectorAll('[data-i18n-placeholder]').forEach((node) => {
    node.placeholder = t(node.dataset.i18nPlaceholder, node.placeholder);
  });
  document.querySelectorAll('[data-i18n-title]').forEach((node) => {
    node.title = t(node.dataset.i18nTitle, node.title);
  });
  document.title = 'Manus AI — ' + t('agent.title');
}

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

function escapeHtml(text) {
  return String(text === undefined || text === null ? '' : text)
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;');
}

/* Inline markdown used inside rendered answers. */
function inlineMarkdown(text) {
  return escapeHtml(text)
    .replace(/`([^`]+)`/g, '<code>$1</code>')
    .replace(/\*\*([^*]+)\*\*/g, '<strong>$1</strong>')
    .replace(/\[([^\]]+)\]\((https?:\/\/[^)\s]+)\)/g, '<a href="$2" target="_blank" rel="noopener">$1</a>')
    .replace(/(^|[\s(])(https?:\/\/[^\s)<]+)/g, '$1<a href="$2" target="_blank" rel="noopener">$2</a>');
}

/* --------------------------------------------------------- markdown render */
function renderMarkdown(source) {
  const lines = String(source || '').split('\n');
  const out = [];
  let inCode = false;
  let listType = '';
  const codeLines = [];

  const closeList = () => {
    if (listType) {
      out.push(listType === 'ol' ? '</ol>' : '</ul>');
      listType = '';
    }
  };

  lines.forEach((line) => {
    if (/^\s*```/.test(line)) {
      if (inCode) {
        out.push('<pre><code>' + escapeHtml(codeLines.join('\n')) + '</code></pre>');
        codeLines.length = 0;
        inCode = false;
      } else {
        closeList();
        inCode = true;
      }
      return;
    }
    if (inCode) {
      codeLines.push(line);
      return;
    }
    if (!line.trim()) {
      closeList();
      return;
    }

    const heading = /^(#{1,4})\s+(.*)$/.exec(line);
    if (heading) {
      closeList();
      const level = Math.min(heading[1].length + 2, 5);
      out.push('<h' + level + '>' + inlineMarkdown(heading[2]) + '</h' + level + '>');
      return;
    }

    const ordered = /^\s*\d+[.)]\s+(.*)$/.exec(line);
    if (ordered) {
      if (listType !== 'ol') {
        closeList();
        out.push('<ol>');
        listType = 'ol';
      }
      out.push('<li>' + inlineMarkdown(ordered[1]) + '</li>');
      return;
    }

    const bullet = /^\s*[-*•]\s+(.*)$/.exec(line);
    if (bullet) {
      if (listType !== 'ul') {
        closeList();
        out.push('<ul>');
        listType = 'ul';
      }
      out.push('<li>' + inlineMarkdown(bullet[1]) + '</li>');
      return;
    }

    if (/^\s*(---+|\*\*\*+)\s*$/.test(line)) {
      closeList();
      out.push('<hr>');
      return;
    }

    closeList();
    out.push('<p>' + inlineMarkdown(line) + '</p>');
  });

  if (inCode && codeLines.length) {
    out.push('<pre><code>' + escapeHtml(codeLines.join('\n')) + '</code></pre>');
  }
  closeList();
  return out.join('');
}

/* ------------------------------------------------------------- LLM client */
function providerById(id) {
  return PROVIDERS.find((provider) => provider.id === id) || PROVIDERS[PROVIDERS.length - 1];
}

function endpoint(path) {
  const base = String(state.settings.baseUrl || '').replace(/\/+$/, '');
  if (!base) throw new Error(t('agent.error.baseUrl'));
  return base + path;
}

function requestHeaders() {
  const provider = providerById(state.settings.providerId);
  const headers = { 'Content-Type': 'application/json' };
  if (state.settings.apiKey || !provider.noKey) {
    headers.Authorization = 'Bearer ' + state.settings.apiKey;
  }
  Object.assign(headers, provider.extraHeaders || {});
  return headers;
}

function accumulateToolCalls(target, deltas) {
  (deltas || []).forEach((delta) => {
    const index = typeof delta.index === 'number' ? delta.index : target.length;
    if (!target[index]) {
      target[index] = { id: '', type: 'function', function: { name: '', arguments: '' } };
    }
    if (delta.id) target[index].id = delta.id;
    if (delta.function && delta.function.name) target[index].function.name += delta.function.name;
    if (delta.function && delta.function.arguments) target[index].function.arguments += delta.function.arguments;
  });
}

function textFromDelta(delta) {
  if (!delta) return '';
  if (typeof delta.content === 'string') return delta.content;
  if (Array.isArray(delta.content)) {
    return delta.content.map((part) => (typeof part === 'string' ? part : part.text || '')).join('');
  }
  return '';
}

/** Streams one assistant turn: text deltas plus any requested tool calls. */
async function streamAssistant(apiMessages, useTools, onDelta, signal) {
  const body = {
    model: state.settings.model,
    messages: apiMessages,
    temperature: Number(state.settings.temperature) || 0,
    stream: true
  };
  if (useTools && TOOL_SCHEMAS.length) {
    body.tools = TOOL_SCHEMAS;
    body.tool_choice = 'auto';
  }

  const response = await fetch(endpoint('/chat/completions'), {
    method: 'POST',
    headers: requestHeaders(),
    body: JSON.stringify(body),
    signal
  });

  if (!response.ok) {
    const detail = await response.text();
    const error = new Error('HTTP ' + response.status + ' — ' + detail.slice(0, 300));
    error.status = response.status;
    throw error;
  }

  const contentType = response.headers.get('Content-Type') || '';
  if (!response.body || contentType.includes('application/json')) {
    const payload = await response.json();
    const message = ((payload.choices || [])[0] || {}).message || {};
    const text = typeof message.content === 'string' ? message.content : '';
    if (text) onDelta(text);
    return { text, toolCalls: message.tool_calls || [] };
  }

  const reader = response.body.getReader();
  const decoder = new TextDecoder();
  let buffer = '';
  let text = '';
  const toolCalls = [];

  for (;;) {
    const chunk = await reader.read();
    if (chunk.done) break;
    buffer += decoder.decode(chunk.value, { stream: true });
    const lines = buffer.split('\n');
    buffer = lines.pop();

    for (const rawLine of lines) {
      const line = rawLine.trim();
      if (!line || line.startsWith(':') || line.startsWith('event:')) continue;
      if (!line.startsWith('data:')) continue;
      const data = line.slice(5).trim();
      if (!data || data === '[DONE]') continue;

      let parsed = null;
      try {
        parsed = JSON.parse(data);
      } catch (error) {
        continue;
      }
      if (parsed.error) {
        throw new Error(parsed.error.message || JSON.stringify(parsed.error).slice(0, 300));
      }
      const delta = (((parsed.choices || [])[0] || {}).delta) || {};
      const piece = textFromDelta(delta);
      if (piece) {
        text += piece;
        onDelta(piece);
      }
      if (delta.tool_calls) accumulateToolCalls(toolCalls, delta.tool_calls);
    }
  }

  const cleaned = toolCalls.filter(Boolean).map((call, index) => ({
    id: call.id || 'call_' + index,
    type: 'function',
    function: { name: call.function.name, arguments: call.function.arguments || '{}' }
  }));
  return { text, toolCalls: cleaned };
}

/* ------------------------------------------------------------ agent loop */
function defaultSystemPrompt() {
  if (state.settings.systemPrompt) return state.settings.systemPrompt;
  return t('agent.systemDefault');
}

/** Converts the UI transcript into OpenAI chat messages. */
function toApiMessages() {
  const messages = [{ role: 'system', content: defaultSystemPrompt() }];
  state.messages.forEach((message) => {
    if (message.role === 'user') {
      messages.push({ role: 'user', content: message.text });
      return;
    }
    if (message.role !== 'assistant') return;

    const finished = (message.toolCalls || []).filter((call) => call.status === 'done' || call.status === 'error');
    const entry = { role: 'assistant', content: message.text || '' };
    if (finished.length) {
      entry.tool_calls = finished.map((call) => ({
        id: call.id,
        type: 'function',
        function: { name: call.name, arguments: call.argsText || '{}' }
      }));
    }
    messages.push(entry);
    finished.forEach((call) => {
      messages.push({
        role: 'tool',
        tool_call_id: call.id,
        name: call.name,
        content: JSON.stringify(call.status === 'error' ? { error: call.error } : call.result)
      });
    });
  });
  return messages;
}

function parseArgs(text) {
  try {
    return JSON.parse(text || '{}');
  } catch (error) {
    return {};
  }
}

async function executeTool(call) {
  const implementation = TOOL_IMPL[call.name];
  if (!implementation) throw new Error('unknown tool: ' + call.name);
  return implementation(parseArgs(call.argsText));
}

function toolsUnsupportedMessage(error) {
  return Boolean(error) && error.status === 400 && /tool|function|tool_choice/i.test(error.message || '');
}

/** Runs the plan → tool → answer loop until the model answers without tools. */
async function send(text) {
  if (state.running) return;
  const provider = providerById(state.settings.providerId);
  if (!state.settings.apiKey && !provider.noKey) {
    openSettings();
    toast(t('agent.needKey'), 'error');
    return;
  }
  if (!state.settings.model) {
    openSettings();
    toast(t('agent.needModel'), 'error');
    return;
  }

  state.messages.push({ role: 'user', text });
  ensureSession();
  renderAll();

  state.running = true;
  state.step = 0;
  state.abort = new AbortController();
  setRunning(true);

  let toolsActive = Boolean(state.settings.tools);
  const maxSteps = Math.max(1, Math.min(Number(state.settings.maxSteps) || 6, 12));

  try {
    for (let step = 0; step < maxSteps; step += 1) {
      state.step = step + 1;
      updateStepLabel();

      const assistant = { role: 'assistant', text: '', toolCalls: [], step: state.step };
      state.messages.push(assistant);
      const node = appendMessageNode(assistant);

      let turn = null;
      try {
        turn = await streamAssistant(toApiMessages().slice(0, -1), toolsActive, (piece) => {
          assistant.text += piece;
          updateAssistantNode(node, assistant);
        }, state.abort.signal);
      } catch (error) {
        if (toolsUnsupportedMessage(error) && toolsActive) {
          toolsActive = false;
          state.messages.pop();
          removeNode(node);
          toast(t('agent.toolsUnsupported'));
          step -= 1;
          continue;
        }
        throw error;
      }

      assistant.text = turn.text || assistant.text;
      assistant.toolCalls = turn.toolCalls.map((call) => ({
        id: call.id,
        name: call.function.name,
        argsText: call.function.arguments,
        status: 'pending'
      }));
      updateAssistantNode(node, assistant);
      persistCurrent();

      if (!assistant.toolCalls.length) break;

      for (const call of assistant.toolCalls) {
        call.status = 'running';
        updateAssistantNode(node, assistant);
        try {
          call.result = await executeTool(call);
          call.status = 'done';
        } catch (error) {
          call.status = 'error';
          call.error = String(error && error.message ? error.message : error);
        }
        updateAssistantNode(node, assistant);
        persistCurrent();
      }
    }
  } catch (error) {
    if (error && error.name === 'AbortError') {
      // Drop the empty assistant bubble left behind by the cancelled turn.
      const last = state.messages[state.messages.length - 1];
      if (last && last.role === 'assistant' && !last.text && !(last.toolCalls || []).length) {
        state.messages.pop();
      }
      toast(t('agent.stopped'));
    } else {
      const last = state.messages[state.messages.length - 1];
      if (last && last.role === 'assistant' && !last.text && !last.toolCalls.length) {
        last.error = String(error && error.message ? error.message : error);
      } else {
        state.messages.push({ role: 'assistant', text: '', error: String(error && error.message ? error.message : error) });
      }
      toast(t('agent.error.failed'), 'error');
    }
  } finally {
    state.running = false;
    state.abort = null;
    setRunning(false);
    updateStepLabel();
    renderAll();
    persistCurrent();
    renderSessions();
  }
}

function stopRun() {
  if (state.abort) state.abort.abort();
}

/* -------------------------------------------------------------- sessions */
function ensureSession() {
  if (state.currentId) return;
  state.currentId = 'chat_' + Date.now().toString(36);
  localStorage.setItem(KEYS.current, state.currentId);
}

function sessionTitle() {
  const first = state.messages.find((message) => message.role === 'user');
  return first ? first.text.slice(0, 60) : t('agent.newChat');
}

function persistCurrent() {
  if (!state.currentId || !state.messages.length) return;
  const record = {
    id: state.currentId,
    title: sessionTitle(),
    updatedAt: Date.now(),
    messages: state.messages.slice(-60)
  };
  const existing = state.sessions.find((session) => session.id === state.currentId);
  if (existing) Object.assign(existing, record);
  else state.sessions.unshift(record);
  saveSessions();
}

function newChat() {
  persistCurrent();
  state.currentId = '';
  localStorage.removeItem(KEYS.current);
  state.messages = [];
  renderAll();
  renderSessions();
  closeDrawer();
  $('#composer-text').focus();
}

function openSession(id) {
  const session = state.sessions.find((item) => item.id === id);
  if (!session) return;
  persistCurrent();
  state.currentId = id;
  localStorage.setItem(KEYS.current, id);
  state.messages = session.messages || [];
  renderAll();
  renderSessions();
  closeDrawer();
}

function deleteSession(id) {
  state.sessions = state.sessions.filter((session) => session.id !== id);
  saveSessions();
  if (state.currentId === id) {
    state.currentId = '';
    state.messages = [];
    localStorage.removeItem(KEYS.current);
  }
  renderAll();
  renderSessions();
}

function renderSessions() {
  const list = $('#sessions-list');
  list.textContent = '';
  if (!state.sessions.length) {
    list.append(el('li', 'empty', t('agent.noSessions')));
    return;
  }
  state.sessions.forEach((session) => {
    const item = el('li', session.id === state.currentId ? 'is-current' : '');
    const open = el('button', 'session-open', session.title || t('agent.newChat'));
    open.type = 'button';
    open.addEventListener('click', () => openSession(session.id));
    item.append(open);
    item.append(el('span', 'item-sub', new Date(session.updatedAt || Date.now()).toLocaleString()));
    const remove = el('button', 'session-delete', '\u2715');
    remove.type = 'button';
    remove.addEventListener('click', () => deleteSession(session.id));
    item.append(remove);
    list.append(item);
  });
}

/* --------------------------------------------------------------- rendering */
function renderToolCard(call) {
  const card = el('details', 'tool ' + call.status);
  const summary = el('summary');
  summary.append(el('span', 'tool-name', call.name));
  summary.append(el('span', 'badge ' + call.status, t('agent.tool.' + call.status, call.status)));
  card.append(summary);
  if (call.argsText) card.append(el('div', 'tool-args', call.argsText));
  if (call.status === 'done' && call.result !== undefined) {
    const text = typeof call.result === 'string' ? call.result : JSON.stringify(call.result, null, 2);
    card.append(el('pre', 'tool-result', text.slice(0, 4000)));
  }
  if (call.status === 'error') card.append(el('div', 'tool-error', call.error || ''));
  return card;
}

function updateAssistantNode(node, message) {
  const answer = node.querySelector('.answer');
  if (!answer) return;
  if (message.text) answer.innerHTML = renderMarkdown(message.text);
  else if (!message.error) answer.textContent = message.toolCalls && message.toolCalls.length ? '' : '…';

  const previousTools = node.querySelector('.tools');
  if (previousTools) previousTools.remove();
  if (message.toolCalls && message.toolCalls.length) {
    const box = el('div', 'tools');
    message.toolCalls.forEach((call) => box.append(renderToolCard(call)));
    node.append(box);
  }

  const previousError = node.querySelector('.error-box');
  if (previousError) previousError.remove();
  if (message.error) {
    const box = el('div', 'error-box');
    box.append(el('strong', null, t('agent.errorLabel') + ': '));
    box.append(el('span', null, message.error));
    node.append(box);
  }

  const previousActions = node.querySelector('.msg-actions');
  if (previousActions) previousActions.remove();
  if (message.text) {
    const row = el('div', 'msg-actions');
    const copy = el('button', 'ghost-btn', t('agent.copy'));
    copy.type = 'button';
    copy.addEventListener('click', () => copyText(message.text));
    row.append(copy);
    node.append(row);
  }
  scrollToEnd();
}

function renderMessage(message) {
  if (message.role === 'user') {
    const node = el('div', 'msg user');
    node.append(el('div', 'body', message.text));
    return node;
  }
  const node = el('div', 'msg assistant');
  const head = el('div', 'msg-head');
  head.append(el('span', null, t('agent.assistant')));
  if (message.step) head.append(el('span', 'badge', t('agent.step') + ' ' + message.step));
  node.append(head);
  node.append(el('div', 'answer'));
  updateAssistantNode(node, message);
  return node;
}

function renderAll() {
  const list = $('#messages');
  list.textContent = '';
  state.messages.forEach((message) => list.append(renderMessage(message)));
  $('#welcome').hidden = state.messages.length > 0;
  scrollToEnd();
}

function appendMessageNode(message) {
  const node = renderMessage(message);
  $('#messages').append(node);
  $('#welcome').hidden = true;
  scrollToEnd();
  return node;
}

function removeNode(node) {
  if (node && node.parentNode) node.parentNode.removeChild(node);
}

function scrollToEnd() {
  const list = $('#messages');
  window.requestAnimationFrame(() => { list.scrollTop = list.scrollHeight; });
}

function setRunning(running) {
  $('#btn-send').hidden = running;
  $('#btn-stop').hidden = !running;
  $('#composer-text').disabled = running;
}

function updateStepLabel() {
  const node = $('#step-label');
  if (!state.running) {
    node.hidden = true;
    node.textContent = '';
    return;
  }
  node.hidden = false;
  node.textContent = t('agent.working') + ' — ' + t('agent.step') + ' ' +
    Math.max(state.step, 1) + '/' + state.settings.maxSteps;
}

function copyText(text) {
  if (navigator.clipboard && navigator.clipboard.writeText) {
    navigator.clipboard.writeText(text)
      .then(() => toast(t('agent.copied'), 'ok'))
      .catch(() => toast(t('agent.copyFailed'), 'error'));
    return;
  }
  toast(t('agent.copyFailed'), 'error');
}

function conversationMarkdown() {
  const lines = ['# ' + sessionTitle(), ''];
  state.messages.forEach((message) => {
    if (message.role === 'user') {
      lines.push('## ' + t('agent.you'), '', message.text, '');
      return;
    }
    lines.push('## ' + t('agent.assistant'), '');
    (message.toolCalls || []).forEach((call) => {
      lines.push('- \uD83D\uDD27 `' + call.name + '` — ' + (call.status === 'error' ? call.error : 'ok'));
    });
    if (message.toolCalls && message.toolCalls.length) lines.push('');
    if (message.text) lines.push(message.text, '');
    if (message.error) lines.push('> ' + t('agent.errorLabel') + ': ' + message.error, '');
  });
  return lines.join('\n');
}

function exportChat() {
  const markdown = conversationMarkdown();
  const name = 'manus-chat-' + new Date().toISOString().slice(0, 19).replace(/[:T]/g, '-') + '.md';
  if (window.Android && window.Android.saveText) {
    window.Android.saveText(name, markdown);
    toast(t('agent.exported') + ': ' + name, 'ok');
    return;
  }
  const blob = new Blob([markdown], { type: 'text/markdown;charset=utf-8' });
  const url = URL.createObjectURL(blob);
  const link = el('a');
  link.href = url;
  link.download = name;
  document.body.append(link);
  link.click();
  link.remove();
  URL.revokeObjectURL(url);
  toast(t('agent.exported'), 'ok');
}

/* --------------------------------------------------------------- settings */
function providerOptions() {
  const select = $('#set-provider');
  select.textContent = '';
  PROVIDERS.forEach((provider) => {
    const option = el('option', null, provider.name);
    option.value = provider.id;
    select.append(option);
  });
}

function applyProviderToForm(providerId, keepModel) {
  const provider = providerById(providerId);
  $('#set-base-url').value = provider.baseUrl || '';
  const list = $('#model-list');
  list.textContent = '';
  (provider.models || []).forEach((model) => {
    const option = el('option');
    option.value = model;
    list.append(option);
  });
  if (!keepModel) {
    $('#set-model').value = (provider.models || [])[0] || '';
  }
  const link = $('#key-link');
  link.href = provider.keyUrl || '#';
  link.hidden = !provider.keyUrl;
  $('#set-key').placeholder = provider.noKey ? (provider.noteKey || 'ollama') : 'sk-…';
}

function renderNotesPreview() {
  const notes = TOOL_IMPL.listNotesRaw ? TOOL_IMPL.listNotesRaw() : [];
  $('#notes-count').textContent = String(notes.length);
}

function fillSettingsForm() {
  providerOptions();
  $('#set-provider').value = state.settings.providerId;
  applyProviderToForm(state.settings.providerId, true);
  $('#set-base-url').value = state.settings.baseUrl || '';
  $('#set-model').value = state.settings.model || '';
  $('#set-key').value = state.settings.apiKey || '';
  $('#set-temperature').value = state.settings.temperature;
  $('#set-max-steps').value = state.settings.maxSteps;
  $('#set-tools').checked = Boolean(state.settings.tools);
  $('#set-system').value = state.settings.systemPrompt || '';
  $('#set-lang').value = state.lang;
  $('#btn-mode').hidden = !(window.Android && window.Android.switchMode);
  renderNotesPreview();
  updateProviderChip();
}

function saveSettingsFromForm() {
  state.settings.providerId = $('#set-provider').value;
  state.settings.baseUrl = $('#set-base-url').value.trim();
  state.settings.model = $('#set-model').value.trim();
  state.settings.apiKey = $('#set-key').value.trim();
  state.settings.temperature = Number($('#set-temperature').value) || 0;
  state.settings.maxSteps = Math.min(Math.max(Number($('#set-max-steps').value) || 6, 1), 12);
  state.settings.tools = $('#set-tools').checked;
  state.settings.systemPrompt = $('#set-system').value.trim();
  saveSettings();
  updateProviderChip();
  toast(t('agent.saved'), 'ok');
}

function updateProviderChip() {
  const provider = providerById(state.settings.providerId);
  $('#provider-chip').textContent = provider.name + ' · ' + (state.settings.model || '—');
}

async function testConnection() {
  saveSettingsFromForm();
  try {
    const response = await fetch(endpoint('/chat/completions'), {
      method: 'POST',
      headers: requestHeaders(),
      body: JSON.stringify({
        model: state.settings.model,
        messages: [{ role: 'user', content: 'ping' }],
        max_tokens: 5
      })
    });
    if (!response.ok) {
      const detail = await response.text();
      throw new Error('HTTP ' + response.status + ' ' + detail.slice(0, 160));
    }
    toast(t('agent.testOk'), 'ok');
  } catch (error) {
    toast(t('agent.testFail') + ': ' + (error && error.message ? error.message : error), 'error');
  }
}

function openSettings() {
  fillSettingsForm();
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

function openDrawer() {
  $('#drawer').hidden = false;
  $('#scrim').hidden = false;
}

function closeDrawer() {
  $('#drawer').hidden = true;
  $('#scrim').hidden = true;
}

/* ----------------------------------------------------------------- events */
function bindAgentEvents() {
  $('#btn-settings').addEventListener('click', openSettings);
  $('#btn-close-settings').addEventListener('click', closeSettings);
  $('#btn-save-settings').addEventListener('click', saveSettingsFromForm);
  $('#btn-test').addEventListener('click', testConnection);
  $('#btn-new-chat').addEventListener('click', newChat);
  $('#btn-sessions').addEventListener('click', openDrawer);
  $('#btn-close-drawer').addEventListener('click', closeDrawer);
  $('#scrim').addEventListener('click', closeDrawer);
  $('#btn-export').addEventListener('click', exportChat);
  $('#btn-mode').addEventListener('click', () => {
    if (window.Android && window.Android.switchMode) window.Android.switchMode();
  });

  $('#set-provider').addEventListener('change', (event) => applyProviderToForm(event.target.value, false));
  $('#set-lang').addEventListener('change', (event) => {
    state.lang = event.target.value;
    localStorage.setItem(KEYS.lang, state.lang);
    applyI18n();
    renderAll();
    renderSessions();
    fillSettingsForm();
  });

  $('#composer').addEventListener('submit', (event) => {
    event.preventDefault();
    const input = $('#composer-text');
    const text = input.value.trim();
    if (!text || state.running) return;
    input.value = '';
    input.style.height = 'auto';
    send(text);
  });
  $('#btn-stop').addEventListener('click', stopRun);
  $('#composer-text').addEventListener('keydown', (event) => {
    if (event.key === 'Enter' && !event.shiftKey) {
      event.preventDefault();
      $('#composer').requestSubmit();
    }
  });
  $('#composer-text').addEventListener('input', (event) => {
    event.target.style.height = 'auto';
    event.target.style.height = Math.min(event.target.scrollHeight, 160) + 'px';
  });

  document.querySelectorAll('[data-suggestion]').forEach((button) => {
    button.addEventListener('click', () => {
      $('#composer-text').value = t(button.dataset.suggestion);
      $('#composer').requestSubmit();
    });
  });

  window.addEventListener('online', () => toast(t('agent.online'), 'ok'));
  window.addEventListener('offline', () => toast(t('agent.offline'), 'error'));
}

function boot() {
  applyI18n();
  providerOptions();
  if (state.currentId) {
    const session = state.sessions.find((item) => item.id === state.currentId);
    if (session) state.messages = session.messages || [];
  }
  bindAgentEvents();
  fillSettingsForm();
  renderSessions();
  renderAll();
  setRunning(false);
  updateStepLabel();
  if (!state.settings.apiKey && !providerById(state.settings.providerId).noKey) {
    openSettings();
  }
}

document.addEventListener('DOMContentLoaded', boot);
