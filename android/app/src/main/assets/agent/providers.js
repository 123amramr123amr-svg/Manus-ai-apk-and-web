/* Provider presets and tool schemas for the standalone agent mode.
   Everything talks to OpenAI-compatible endpoints with the user's own key,
   so no Manus (or any other) server is required. */
window.MANUS_AGENT_PROVIDERS = [
  {
    id: 'openai',
    name: 'OpenAI',
    baseUrl: 'https://api.openai.com/v1',
    models: ['gpt-4o-mini', 'gpt-4o', 'gpt-4.1-mini', 'gpt-4.1', 'o4-mini'],
    keyUrl: 'https://platform.openai.com/api-keys',
    tools: true
  },
  {
    id: 'deepseek',
    name: 'DeepSeek',
    baseUrl: 'https://api.deepseek.com/v1',
    models: ['deepseek-chat', 'deepseek-reasoner'],
    keyUrl: 'https://platform.deepseek.com/api_keys',
    tools: true
  },
  {
    id: 'openrouter',
    name: 'OpenRouter',
    baseUrl: 'https://openrouter.ai/api/v1',
    models: [
      'openai/gpt-4o-mini',
      'anthropic/claude-3.7-sonnet',
      'google/gemini-2.5-flash',
      'deepseek/deepseek-chat',
      'meta-llama/llama-3.3-70b-instruct'
    ],
    keyUrl: 'https://openrouter.ai/keys',
    tools: true,
    extraHeaders: { 'HTTP-Referer': 'https://github.com/manus-ai-agent', 'X-Title': 'Manus AI Agent' }
  },
  {
    id: 'groq',
    name: 'Groq',
    baseUrl: 'https://api.groq.com/openai/v1',
    models: ['llama-3.3-70b-versatile', 'llama-3.1-8b-instant', 'qwen/qwen3-32b'],
    keyUrl: 'https://console.groq.com/keys',
    tools: true
  },
  {
    id: 'gemini',
    name: 'Google Gemini',
    baseUrl: 'https://generativelanguage.googleapis.com/v1beta/openai',
    models: ['gemini-2.5-flash', 'gemini-2.0-flash', 'gemini-2.5-pro'],
    keyUrl: 'https://aistudio.google.com/app/apikey',
    tools: true
  },
  {
    id: 'mistral',
    name: 'Mistral AI',
    baseUrl: 'https://api.mistral.ai/v1',
    models: ['mistral-large-latest', 'mistral-small-latest', 'open-mistral-nemo'],
    keyUrl: 'https://console.mistral.ai/api-keys/',
    tools: true
  },
  {
    id: 'xai',
    name: 'xAI (Grok)',
    baseUrl: 'https://api.x.ai/v1',
    models: ['grok-3-mini', 'grok-3', 'grok-2-1212'],
    keyUrl: 'https://console.x.ai/',
    tools: true
  },
  {
    id: 'ollama',
    name: 'Ollama (محلي / Local)',
    baseUrl: 'http://localhost:11434/v1',
    models: ['llama3.1', 'qwen2.5', 'mistral', 'gemma3'],
    keyUrl: 'https://ollama.com/download',
    tools: true,
    noKey: true,
    noteKey: 'ollama'
  },
  {
    id: 'custom',
    name: 'مخصص — OpenAI-compatible',
    baseUrl: '',
    models: [],
    keyUrl: '',
    tools: true
  }
];

/* Function-calling schemas (OpenAI tools format). */
window.MANUS_AGENT_TOOLS = [
  {
    type: 'function',
    function: {
      name: 'web_search',
      description:
        'Search the public web and return the top results with titles, URLs and snippets. ' +
        'Use this for any current or factual question before answering.',
      parameters: {
        type: 'object',
        properties: {
          query: { type: 'string', description: 'Search query' },
          max_results: { type: 'integer', description: 'How many results to return (1-10, default 5)' }
        },
        required: ['query']
      }
    }
  },
  {
    type: 'function',
    function: {
      name: 'read_url',
      description:
        'Fetch a web page and return its readable text (markdown). Use it after web_search to read a source in depth.',
      parameters: {
        type: 'object',
        properties: {
          url: { type: 'string', description: 'Absolute http(s) URL' },
          max_chars: { type: 'integer', description: 'Maximum characters to return (default 6000)' }
        },
        required: ['url']
      }
    }
  },
  {
    type: 'function',
    function: {
      name: 'wikipedia',
      description: 'Look up an encyclopaedia article summary on Wikipedia.',
      parameters: {
        type: 'object',
        properties: {
          query: { type: 'string', description: 'Topic to look up' },
          lang: { type: 'string', description: 'Language code such as ar or en (default ar)' }
        },
        required: ['query']
      }
    }
  },
  {
    type: 'function',
    function: {
      name: 'calculate',
      description: 'Evaluate a mathematical expression (numbers, parentheses, + - * / % ** and Math functions).',
      parameters: {
        type: 'object',
        properties: {
          expression: { type: 'string', description: 'Expression such as (1200*1.15)/3' }
        },
        required: ['expression']
      }
    }
  },
  {
    type: 'function',
    function: {
      name: 'current_time',
      description: 'Return the current date and time, optionally in a specific IANA timezone.',
      parameters: {
        type: 'object',
        properties: {
          timezone: { type: 'string', description: 'IANA timezone such as Africa/Cairo (optional)' }
        }
      }
    }
  },
  {
    type: 'function',
    function: {
      name: 'save_note',
      description: 'Save a note or draft on this device so it can be reused later.',
      parameters: {
        type: 'object',
        properties: {
          title: { type: 'string', description: 'Note title' },
          content: { type: 'string', description: 'Note body (markdown allowed)' }
        },
        required: ['title', 'content']
      }
    }
  },
  {
    type: 'function',
    function: {
      name: 'list_notes',
      description: 'List the titles of the notes saved on this device.',
      parameters: { type: 'object', properties: {} }
    }
  }
];
