/* Tool implementations for the standalone agent.
   Everything runs inside the app/browser: no backend and no extra API keys.
   Search and page reading use the public r.jina.ai reader (free, no key). */
'use strict';

const READER = 'https://r.jina.ai/';
const NOTES_KEY = 'manus.agent.notes';

function trimText(value, limit) {
  const text = String(value === undefined || value === null ? '' : value).trim();
  if (text.length <= limit) return text;
  return text.slice(0, limit) + '\n…[truncated]';
}

async function fetchText(url, timeoutMs = 30000) {
  const controller = new AbortController();
  const timer = setTimeout(() => controller.abort(), timeoutMs);
  try {
    const response = await fetch(url, {
      signal: controller.signal,
      headers: { Accept: 'text/plain, text/markdown, */*' }
    });
    if (!response.ok) {
      throw new Error('HTTP ' + response.status + ' ' + response.statusText);
    }
    return await response.text();
  } finally {
    clearTimeout(timer);
  }
}

/** DuckDuckGo redirect links look like /l/?uddg=<encoded target>. */
function decodeResultUrl(href) {
  try {
    const match = /[?&]uddg=([^&]+)/.exec(href);
    if (match) return decodeURIComponent(match[1]);
    if (href.startsWith('//')) return 'https:' + href;
    return href;
  } catch (error) {
    return href;
  }
}

/** Turns the markdown produced by the reader for DuckDuckGo Lite into results. */
function parseSearchMarkdown(markdown, maxResults) {
  const results = [];
  const lines = markdown.split('\n');
  let current = null;

  lines.forEach((line) => {
    const link = /^\s*\d+\.\[([^\]]+)\]\(([^)]+)\)/.exec(line);
    if (link) {
      if (current) results.push(current);
      current = { title: link[1].trim(), url: decodeResultUrl(link[2].trim()), snippet: '' };
      return;
    }
    if (!current) return;
    const text = line.trim();
    if (!text) return;
    if (text.startsWith('Title:') || text.startsWith('URL Source:') || text.startsWith('Markdown Content:')) return;
    if (!current.snippet) current.snippet = text.slice(0, 300);
  });
  if (current) results.push(current);
  return results.slice(0, maxResults);
}

const TOOLS = {
  async web_search(args) {
    const query = String(args.query || '').trim();
    if (!query) throw new Error('query is required');
    const max = Math.min(Math.max(Number(args.max_results) || 5, 1), 10);
    const target = 'https://lite.duckduckgo.com/lite/?q=' + encodeURIComponent(query);
    const markdown = await fetchText(READER + target, 40000);
    const results = parseSearchMarkdown(markdown, max);
    if (!results.length) {
      return { query, results: [], note: 'no results parsed', raw: trimText(markdown, 1500) };
    }
    return { query, results };
  },

  async read_url(args) {
    const url = String(args.url || '').trim();
    if (!/^https?:\/\//i.test(url)) throw new Error('url must start with http:// or https://');
    const maxChars = Math.min(Math.max(Number(args.max_chars) || 6000, 500), 20000);
    const text = await fetchText(READER + url, 45000);
    return { url, content: trimText(text, maxChars) };
  },

  async wikipedia(args) {
    const query = String(args.query || '').trim();
    if (!query) throw new Error('query is required');
    const lang = String(args.lang || 'ar').trim().slice(0, 5) || 'ar';
    const searchUrl =
      'https://' + lang + '.wikipedia.org/w/api.php?action=query&list=search&format=json&origin=*&srlimit=3&srsearch=' +
      encodeURIComponent(query);
    const searchResponse = await fetch(searchUrl);
    if (!searchResponse.ok) throw new Error('HTTP ' + searchResponse.status);
    const search = await searchResponse.json();
    const hits = (search.query && search.query.search) || [];
    if (!hits.length) return { query, found: false };

    const title = hits[0].title;
    const summaryResponse = await fetch(
      'https://' + lang + '.wikipedia.org/api/rest_v1/page/summary/' + encodeURIComponent(title)
    );
    const summary = summaryResponse.ok ? await summaryResponse.json() : {};
    return {
      query,
      title,
      extract: trimText(summary.extract || hits[0].snippet || '', 2500),
      url: (summary.content_urls && summary.content_urls.desktop && summary.content_urls.desktop.page) || '',
      alternatives: hits.slice(1).map((hit) => hit.title)
    };
  }
};

/* ------------------------------------------------- local (offline) tools */
Object.assign(TOOLS, {
  async calculate(args) {
    const expression = String(args.expression || '').trim();
    if (!expression) throw new Error('expression is required');
    if (!/^[\d\s+\-*/%().,eE]+$/.test(expression)) {
      throw new Error('only numbers and + - * / % ( ) are allowed');
    }
    const normalised = expression.replace(/,/g, '');
    /* eslint-disable-next-line no-new-func */
    const value = Function('"use strict";return (' + normalised + ')')();
    if (typeof value !== 'number' || !Number.isFinite(value)) throw new Error('not a finite number');
    return { expression, result: value };
  },

  async current_time(args) {
    const now = new Date();
    const timezone = String(args.timezone || '').trim();
    let local = now.toString();
    if (timezone) {
      try {
        local = now.toLocaleString('en-GB', { timeZone: timezone, dateStyle: 'full', timeStyle: 'long' });
      } catch (error) {
        throw new Error('unknown timezone: ' + timezone);
      }
    }
    return { iso: now.toISOString(), local, timezone: timezone || Intl.DateTimeFormat().resolvedOptions().timeZone };
  },

  async save_note(args) {
    const title = String(args.title || '').trim();
    const content = String(args.content || '');
    if (!title) throw new Error('title is required');
    const notes = TOOLS.listNotesRaw();
    const index = notes.findIndex((note) => note.title === title);
    const entry = { title, content, updatedAt: new Date().toISOString() };
    if (index >= 0) notes[index] = entry;
    else notes.push(entry);
    localStorage.setItem(NOTES_KEY, JSON.stringify(notes));
    return { saved: true, title, notes: notes.length };
  },

  async list_notes() {
    const notes = TOOLS.listNotesRaw();
    return { count: notes.length, titles: notes.map((note) => note.title) };
  }
});

/** Notes are shared with the settings screen (read-only preview). */
TOOLS.listNotesRaw = function listNotesRaw() {
  try {
    const parsed = JSON.parse(localStorage.getItem(NOTES_KEY) || '[]');
    return Array.isArray(parsed) ? parsed : [];
  } catch (error) {
    return [];
  }
};

TOOLS.readNote = function readNote(title) {
  const note = TOOLS.listNotesRaw().find((item) => item.title === title);
  return note ? note.content : null;
};

window.MANUS_AGENT_TOOLS_IMPL = TOOLS;
