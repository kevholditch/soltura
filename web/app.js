// State
let sessionId = null;
let history = [];
let currentAgentBubble = null;
let currentAgentText = '';

// View switching
function showView(viewId) {
  ['view-start', 'view-conversation', 'view-summary', 'view-vocab'].forEach(id => {
    document.getElementById(id).classList.add('hidden');
    document.getElementById(id).classList.remove('flex');
  });
  const target = document.getElementById(viewId);
  target.classList.remove('hidden');
  // Restore flex layout for views that use it
  if (['view-start', 'view-conversation', 'view-summary', 'view-vocab'].includes(viewId)) {
    target.classList.add('flex');
  }
}

// Start session
async function startSession(topic) {
  const startBtn = document.getElementById('start-btn');
  const startError = document.getElementById('start-error');
  startError.classList.add('hidden');
  startError.textContent = '';

  // Switch to conversation view immediately so the user gets feedback
  document.getElementById('message-thread').innerHTML = '';
  document.getElementById('session-topic').textContent = topic;
  showView('view-conversation');

  // Show a loading bubble while we wait for the opening question
  const thread = document.getElementById('message-thread');
  const loadingDiv = document.createElement('div');
  loadingDiv.className = 'agent-message';
  const loadingBubble = document.createElement('div');
  loadingBubble.className = 'agent-bubble loading-bubble';
  loadingBubble.innerHTML = '<span class="loading-dots"><span>.</span><span>.</span><span>.</span></span>';
  loadingDiv.appendChild(loadingBubble);
  thread.appendChild(loadingDiv);

  startBtn.disabled = true;

  try {
    const res = await fetch('/api/sessions', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ topic })
    });

    if (!res.ok) {
      const body = await res.json().catch(() => ({}));
      throw new Error(body.error || `Server error: ${res.status}`);
    }

    const data = await res.json();
    sessionId = data.session_id;
    history = [];

    // Replace loading bubble with the actual opening question
    loadingDiv.remove();
    addMessage('assistant', data.seed_content);
    history.push({ role: 'assistant', content: data.seed_content });

    document.getElementById('user-input').focus();
  } catch (err) {
    console.error('Failed to start session:', err);
    loadingDiv.remove();
    showView('view-start');
    startError.textContent = err.message;
    startError.classList.remove('hidden');
  } finally {
    startBtn.disabled = false;
    startBtn.textContent = 'Start Session';
  }
}

// Add message to UI
function addMessage(role, text) {
  const thread = document.getElementById('message-thread');
  const div = document.createElement('div');
  div.className = role === 'user' ? 'user-message' : 'agent-message';

  const bubble = document.createElement('div');
  bubble.className = role === 'user' ? 'user-bubble' : 'agent-bubble';
  if (role === 'assistant') {
    bubble.innerHTML = marked.parse(text);
  } else {
    bubble.textContent = text;
  }
  div.appendChild(bubble);
  thread.appendChild(div);
  thread.scrollTop = thread.scrollHeight;
  return bubble;
}

// Show an inline error message in the thread
function showError(message) {
  const thread = document.getElementById('message-thread');
  const div = document.createElement('div');
  div.className = 'error-message';
  div.textContent = message;
  thread.appendChild(div);
  thread.scrollTop = thread.scrollHeight;
}

// Submit turn
async function submitTurn(userText) {
  if (!userText.trim() || !sessionId) return;

  // Add user message immediately
  addMessage('user', userText);
  history.push({ role: 'user', content: userText });

  // Clear input, disable submit
  document.getElementById('user-input').value = '';
  document.getElementById('submit-btn').disabled = true;

  // Create streaming agent bubble
  const thread = document.getElementById('message-thread');
  const agentDiv = document.createElement('div');
  agentDiv.className = 'agent-message';
  const agentBubble = document.createElement('div');
  agentBubble.className = 'agent-bubble streaming loading-bubble';
  agentBubble.innerHTML = '<span class="loading-dots"><span>.</span><span>.</span><span>.</span></span>';
  agentDiv.appendChild(agentBubble);
  thread.appendChild(agentDiv);
  thread.scrollTop = thread.scrollHeight;

  currentAgentText = '';

  try {
    // POST with SSE using fetch + ReadableStream
    const response = await fetch(`/api/sessions/${sessionId}/turns`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ user_text: userText, history })
    });

    if (!response.ok) throw new Error(`Server error: ${response.status}`);

    const reader = response.body.getReader();
    const decoder = new TextDecoder();
    let buffer = '';

    while (true) {
      const { done, value } = await reader.read();
      if (done) break;

      buffer += decoder.decode(value, { stream: true });
      const lines = buffer.split('\n');
      buffer = lines.pop(); // keep incomplete line

      for (const line of lines) {
        if (!line.startsWith('data: ')) continue;
        const jsonStr = line.slice(6);
        if (jsonStr === '[DONE]') break;
        try {
          const event = JSON.parse(jsonStr);
          if (event.type === 'chunk') {
            if (!currentAgentText) agentBubble.classList.remove('loading-bubble');
            currentAgentText += event.text;
            agentBubble.innerHTML = marked.parse(currentAgentText);
            thread.scrollTop = thread.scrollHeight;
          } else if (event.type === 'corrections') {
            agentBubble.classList.remove('streaming');
            renderCorrections(agentDiv, event.corrections);
          } else if (event.type === 'done') {
            agentBubble.classList.remove('streaming');
            history.push({ role: 'assistant', content: currentAgentText });
            document.getElementById('submit-btn').disabled = false;
            document.getElementById('user-input').focus();
          }
        } catch (e) {
          // Ignore malformed SSE lines
        }
      }
    }
  } catch (err) {
    console.error('Turn failed:', err);
    agentBubble.classList.remove('streaming');
    agentBubble.textContent = '[Error: could not get response. Please try again.]';
    agentBubble.classList.add('error-bubble');
    document.getElementById('submit-btn').disabled = false;
    document.getElementById('user-input').focus();
  }
}

// Render corrections panel below agent message
function renderCorrections(parentDiv, corrections) {
  if (!corrections || corrections.length === 0) return;

  const panel = document.createElement('div');
  panel.className = 'corrections-panel';

  const heading = document.createElement('div');
  heading.className = 'corrections-heading';
  heading.textContent = corrections.length === 1
    ? '1 correction'
    : `${corrections.length} corrections`;
  panel.appendChild(heading);

  corrections.forEach(c => {
    const item = document.createElement('div');
    item.className = 'correction-item';
    item.innerHTML = `
      <div class="correction-row">
        <span class="correction-original">${escapeHtml(c.original)}</span>
        <span class="correction-arrow">→</span>
        <span class="correction-corrected">${escapeHtml(c.corrected)}</span>
        <span class="correction-badge category-${escapeHtml(c.category)}">${escapeHtml(c.category)}</span>
      </div>
      <p class="correction-explanation">${escapeHtml(c.explanation)}</p>
    `;
    panel.appendChild(item);
  });

  parentDiv.appendChild(panel);
}

function escapeHtml(str) {
  const div = document.createElement('div');
  div.textContent = str;
  return div.innerHTML;
}

// End session
async function endSession() {
  if (!sessionId) return;

  const btn = document.getElementById('end-session-btn');
  btn.disabled = true;
  btn.textContent = 'Ending…';

  try {
    await fetch(`/api/sessions/${sessionId}/end`, { method: 'POST' });

    const res = await fetch(`/api/sessions/${sessionId}/summary`);
    if (!res.ok) throw new Error(`Server error: ${res.status}`);
    const data = await res.json();

    document.getElementById('summary-text').innerHTML = marked.parse(data.summary || 'No summary available.');
    document.getElementById('summary-turns').textContent = data.turn_count ?? '—';
    document.getElementById('summary-corrections').textContent = data.correction_count ?? '—';

    showView('view-summary');
  } catch (err) {
    console.error('Failed to end session:', err);
    btn.disabled = false;
    btn.textContent = 'End Session';
    showError('Failed to end session. Please try again.');
  }
}

// Load vocabulary
async function loadVocab() {
  try {
    const res = await fetch('/api/vocab?limit=50');
    if (!res.ok) throw new Error(`Server error: ${res.status}`);
    const data = await res.json();

    const tbody = document.getElementById('vocab-tbody');
    const empty = document.getElementById('vocab-empty');
    tbody.innerHTML = '';

    const entries = data.entries || [];

    if (entries.length === 0) {
      empty.classList.remove('hidden');
    } else {
      empty.classList.add('hidden');
      entries.forEach(entry => {
        const tr = document.createElement('tr');
        tr.className = 'vocab-row';
        tr.innerHTML = `
          <td class="py-3 px-4 text-gray-300">${escapeHtml(entry.original)}</td>
          <td class="py-3 px-4 text-green-400">${escapeHtml(entry.corrected)}</td>
          <td class="py-3 px-4">
            <span class="correction-badge category-${escapeHtml(entry.category)}">${escapeHtml(entry.category)}</span>
          </td>
          <td class="py-3 px-4 text-right text-gray-500">${entry.seen_count}</td>
        `;
        tbody.appendChild(tr);
      });
    }

    showView('view-vocab');
  } catch (err) {
    console.error('Failed to load vocab:', err);
    alert('Failed to load vocabulary. Is the server running?');
  }
}

// Event listeners
document.addEventListener('DOMContentLoaded', () => {
  // Start session button
  document.getElementById('start-btn').addEventListener('click', () => {
    const topic = document.getElementById('topic-input').value.trim();
    if (topic) startSession(topic);
  });

  // Topic input enter key
  document.getElementById('topic-input').addEventListener('keydown', e => {
    if (e.key === 'Enter') {
      const topic = e.target.value.trim();
      if (topic) startSession(topic);
    }
  });

  // Submit button
  document.getElementById('submit-btn').addEventListener('click', () => {
    submitTurn(document.getElementById('user-input').value);
  });

  // Cmd/Ctrl+Enter to submit
  document.getElementById('user-input').addEventListener('keydown', e => {
    if ((e.metaKey || e.ctrlKey) && e.key === 'Enter') {
      submitTurn(e.target.value);
    }
  });

  // End session button
  document.getElementById('end-session-btn').addEventListener('click', endSession);

  // New session button
  document.getElementById('new-session-btn').addEventListener('click', () => {
    sessionId = null;
    history = [];
    document.getElementById('topic-input').value = '';
    showView('view-start');
  });

  // Vocabulary nav link
  document.getElementById('vocab-btn').addEventListener('click', loadVocab);

  // Back from vocab
  document.getElementById('back-from-vocab').addEventListener('click', () => {
    showView(sessionId ? 'view-conversation' : 'view-start');
  });
});
