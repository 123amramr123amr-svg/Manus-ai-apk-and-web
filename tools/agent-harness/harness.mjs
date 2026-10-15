/* Headless harness: exercises the agent loop, SSE parsing and tools in Node
   with stubbed DOM + fetch. Run: node /tmp/agent_harness.mjs */
import fs from 'node:fs';

import { fileURLToPath } from 'node:url';
import path from 'node:path';

const AGENT_DIR = path.join(path.dirname(fileURLToPath(import.meta.url)), '../../web/agent');
const results = [];
const check = (name, condition, extra = '') => {
  results.push({ name, ok: Boolean(condition), extra });
  console.log((condition ? 'PASS ' : 'FAIL ') + name + (condition ? '' : ' -> ' + extra));
};

/* ------------------------------------------------------------ DOM stubs */
class Node {
  constructor(tag = 'div') {
    this.tagName = tag;
    this.children = [];
    this.style = {};
    this.dataset = {};
    this.classList = { toggle() {}, add() {}, remove() {} };
    this.hidden = false;
    this.textContent = '';
    this.innerHTML = '';
    this.value = '';
    this.scrollTop = 0;
    this.scrollHeight = 0;
    this.placeholder = '';
    this.title = '';
    this.href = '';
  }
  append(...nodes) { this.children.push(...nodes); }
  appendChild(node) { this.children.push(node); return node; }
  remove() {}
  addEventListener() {}
  setAttribute() {}
  querySelector() { return new Node(); }
  querySelectorAll() { return []; }
  focus() {}
  requestSubmit() {}
}

const nodes = new Map();
const getNode = (selector) => {
  if (!nodes.has(selector)) nodes.set(selector, new Node());
  return nodes.get(selector);
};

globalThis.localStorage = {
  _data: {},
  getItem(key) { return key in this._data ? this._data[key] : null; },
  setItem(key, value) { this._data[key] = String(value); },
  removeItem(key) { delete this._data[key]; }
};
globalThis.window = globalThis;
globalThis.document = {
  documentElement: { lang: '', dir: '' },
  querySelector: (selector) => getNode(selector),
  querySelectorAll: () => [],
  createElement: (tag) => new Node(tag),
  addEventListener: () => {},
  body: new Node('body')
};
Object.defineProperty(globalThis, 'navigator', { value: {}, configurable: true });
globalThis.requestAnimationFrame = (callback) => callback();

/* --------------------------------------------------------- load scripts */
const exportLists = {
  'tools.js': ['parseSearchMarkdown', 'decodeResultUrl', 'TOOLS'],
  'agent.js': ['renderMarkdown', 'inlineMarkdown', 'streamAssistant', 'toApiMessages', 'send',
    'stopRun', 'state', 'saveSettings', 'updateAssistantNode', 'conversationMarkdown']
};
for (const file of ['providers.js', 'tools.js', 'i18n.js', 'agent.js']) {
  const code = fs.readFileSync(`${AGENT_DIR}/${file}`, 'utf8');
  const names = exportLists[file] || [];
  const exported = names.length
    ? ';globalThis.__x = Object.assign(globalThis.__x || {}, { ' + names.join(', ') + ' });'
    : '';
  new Function(code + exported)();
}
const X = globalThis.__x;
check('scripts load and expose helpers',
  Boolean(X.parseSearchMarkdown && X.streamAssistant && X.send && X.TOOLS));

/* --------------------------------------------------------- tool helpers */
const sampleSearch = `Title: openmanus github at DuckDuckGo

URL Source: https://lite.duckduckgo.com/lite/?q=openmanus+github

Markdown Content:
1.[OpenManus - GitHub](https://duckduckgo.com/l/?uddg=https%3A%2F%2Fgithub.com%2FFoundationAgents%2FOpenManus&rut=abc)
Enjoy your own agent with **OpenManus**!
github.com/FoundationAgents/OpenManus

2.[OpenManus docs](https://duckduckgo.com/l/?uddg=https%3A%2F%2Fopenmanus.github.io%2F&rut=def)
OpenManus documentation page.
openmanus.github.io
`;

const parsed = X.parseSearchMarkdown(sampleSearch, 5);
check('search parses results', parsed.length === 2, JSON.stringify(parsed));
check('search decodes redirect urls',
  parsed[0].url === 'https://github.com/FoundationAgents/OpenManus' &&
  parsed[1].url === 'https://openmanus.github.io/',
  JSON.stringify(parsed.map((r) => r.url)));
check('search keeps snippets', /Enjoy your own agent/.test(parsed[0].snippet), parsed[0].snippet);

const mathOk = await X.TOOLS.calculate({ expression: '(1200*1.15)/3' });
check('calculator evaluates', mathOk.result === 460, JSON.stringify(mathOk));
let rejected = false;
try {
  await X.TOOLS.calculate({ expression: 'process.exit(1)' });
} catch (error) {
  rejected = true;
}
check('calculator rejects code injection', rejected);

const time = await X.TOOLS.current_time({ timezone: 'Africa/Cairo' });
check('current_time returns iso + local', Boolean(time.iso && time.local), JSON.stringify(time));

await X.TOOLS.save_note({ title: 'خطة', content: 'نص' });
const notes = await X.TOOLS.list_notes();
check('notes round-trip', notes.count === 1 && notes.titles[0] === 'خطة', JSON.stringify(notes));

/* -------------------------------------------------------- markdown render */
const rendered = X.renderMarkdown('# العنوان\n\n- أول\n- ثاني\n\n```js\nconst a = 1;\n```\n\n**عريض** و [رابط](https://example.com)');
check('markdown headings/lists/code',
  rendered.includes('<h3>') && rendered.includes('<ul>') && rendered.includes('<pre><code>') &&
  rendered.includes('<strong>عريض</strong>') && rendered.includes('href="https://example.com"'),
  rendered);
check('markdown escapes html', X.renderMarkdown('<script>alert(1)</script>').includes('&lt;script&gt;'));

/* --------------------------------------------------- SSE + tool-call loop */
const sseResponse = (chunks) => {
  const encoder = new TextEncoder();
  const stream = new ReadableStream({
    start(controller) {
      chunks.forEach((chunk) => controller.enqueue(encoder.encode(chunk)));
      controller.close();
    }
  });
  return new Response(stream, { status: 200, headers: { 'Content-Type': 'text/event-stream' } });
};

const chatChunksWithTool = [
  'data: {"choices":[{"delta":{"content":"سأبحث "}}]}\n\n',
  'data: {"choices":[{"delta":{"content":"الآن.","tool_calls":[{"index":0,"id":"call_1","type":"function","function":{"name":"web_","arguments":"{\\"que"}}]}}]}\n\n',
  'data: {"choices":[{"delta":{"tool_calls":[{"index":0,"function":{"name":"search","arguments":"ry\\":\\"openmanus\\"}"}}]}}]}\n\n',
  'data: [DONE]\n\n'
];
const finalChunks = [
  'data: {"choices":[{"delta":{"content":"## النتيجة\\n\\n- OpenManus مشروع مفتوح المصدر"}}]}\n\n',
  'data: [DONE]\n\n'
];

let chatCalls = 0;
globalThis.fetch = async (url) => {
  const target = String(url);
  if (target.includes('r.jina.ai')) {
    return new Response(sampleSearch, { status: 200, headers: { 'Content-Type': 'text/plain' } });
  }
  if (target.includes('/chat/completions')) {
    chatCalls += 1;
    return sseResponse(chatCalls === 1 ? chatChunksWithTool : finalChunks);
  }
  throw new Error('unexpected fetch: ' + target);
};

X.state.settings.apiKey = 'test-key';
X.state.settings.model = 'test-model';
X.state.settings.baseUrl = 'https://api.example.com/v1';
X.state.settings.tools = true;

const turn = await X.streamAssistant(
  [{ role: 'user', content: 'ابحث' }], true, () => {}, new AbortController().signal);
check('streaming accumulates text', turn.text === 'سأبحث الآن.', JSON.stringify(turn.text));
check('streaming accumulates tool call fragments',
  turn.toolCalls.length === 1 && turn.toolCalls[0].function.name === 'web_search' &&
  turn.toolCalls[0].function.arguments === '{"query":"openmanus"}',
  JSON.stringify(turn.toolCalls));

chatCalls = 0;
await X.send('ابحث عن OpenManus');
const transcript = X.state.messages;
const assistantWithTool = transcript.find((message) => message.toolCalls && message.toolCalls.length);
const finalAnswer = transcript[transcript.length - 1];
check('agent loop ran the tool',
  Boolean(assistantWithTool) && assistantWithTool.toolCalls[0].status === 'done',
  JSON.stringify(transcript));
check('tool returned search results',
  Boolean(assistantWithTool) && assistantWithTool.toolCalls[0].result.results.length === 2,
  JSON.stringify(assistantWithTool && assistantWithTool.toolCalls[0].result));
check('agent produced a final answer',
  finalAnswer.role === 'assistant' && /OpenManus/.test(finalAnswer.text), JSON.stringify(finalAnswer));

const apiMessages = X.toApiMessages();
check('api messages include the tool round trip',
  apiMessages[0].role === 'system' &&
  apiMessages.some((message) => message.role === 'assistant' && message.tool_calls) &&
  apiMessages.some((message) => message.role === 'tool' && message.tool_call_id === 'call_1'),
  JSON.stringify(apiMessages.map((message) => message.role)));

check('conversation exports to markdown',
  X.conversationMarkdown().includes('OpenManus'), 'export');

/* ------------------------------------------------- unsupported-tools retry */
chatCalls = 0;
globalThis.fetch = async (url) => {
  const target = String(url);
  if (target.includes('/chat/completions')) {
    chatCalls += 1;
    if (chatCalls === 1) {
      return new Response(JSON.stringify({ error: { message: 'tools are not supported' } }), {
        status: 400, headers: { 'Content-Type': 'application/json' }
      });
    }
    return sseResponse(finalChunks);
  }
  throw new Error('unexpected fetch: ' + target);
};
X.state.messages = [];
await X.send('اختبار بدون أدوات');
check('retries without tools on HTTP 400',
  chatCalls === 2 && X.state.messages[X.state.messages.length - 1].text.includes('OpenManus'),
  JSON.stringify(X.state.messages));

/* ------------------------------------------------------------- abort path */
globalThis.fetch = async () => {
  const error = new Error('aborted');
  error.name = 'AbortError';
  throw error;
};
X.state.messages = [];
await X.send('إلغاء');
check('abort is handled gracefully', X.state.messages.length === 1, JSON.stringify(X.state.messages));

const failed = results.filter((result) => !result.ok);
console.log('\n' + (results.length - failed.length) + '/' + results.length + ' checks passed');
process.exit(failed.length ? 1 : 0);
