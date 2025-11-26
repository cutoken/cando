const MAX_VISIBLE_MESSAGES = 120;

const ui = {
  messages: null,
  planSummaryText: null,
  promptForm: null,
  promptInput: null,
  sendBtn: null,
  cancelBtn: null,
  thinkingToggle: null,
  forceThinkingToggle: null,
  systemPromptInput: null,
  statusText: null,
  statusMeta: null,
  thinkingStatus: null,
  contextProgressBar: null,
  contextProgressFill: null,
  compactionHistoryBtn: null,
  compactionDialog: null,
  closeCompactionDialog: null,
  compactionHistoryContent: null,
  messageBanner: null,
  thinkingIndicator: null,
  thinkingPlan: null,
  modelSelect: null,
  onboardingDialog: null,
  saveCredentialsBtn: null,
  apiKeyInput: null,
  apiKeyHelp: null,
  onboardingError: null,
  onboardingSuccess: null,
  systemPromptInput: null,
  planDropdown: null,
  closePlanDropdown: null,
  planDropdownTitle: null,
  planDropdownSteps: null,
  thinkingModelInfo: null,
  workspaceBtn: null,
  workspaceLabel: null,
  workspaceMenu: null,
  workspaceMenuOpen: null,
  workspaceMenuRecent: null,
  addWorkspaceMenuBtn: null,
  helpBtn: null,
};

const appState = {
  data: null,
  busy: false,
  showAllMessages: false,
  currentAbortController: null,
};

async function initUI() {
  ui.messages = document.getElementById('messages');
  ui.promptForm = document.getElementById('promptForm');
  ui.promptInput = document.getElementById('promptInput');
  ui.sendBtn = document.getElementById('sendBtn');
  ui.cancelBtn = document.getElementById('cancelBtn');
  ui.thinkingToggle = document.getElementById('thinkingToggle');
  ui.forceThinkingToggle = document.getElementById('forceThinkingToggle');
  ui.systemPromptInput = document.getElementById('systemPromptInput');
  ui.thinkingStatus = document.getElementById('thinkingStatus');
  ui.statusText = document.getElementById('statusText');
  ui.statusMeta = document.getElementById('statusMeta');
  ui.contextProgressBar = document.getElementById('contextProgressBar');
  ui.contextProgressFill = document.getElementById('contextProgressFill');
  ui.compactionHistoryBtn = document.getElementById('compactionHistoryBtn');
  ui.compactionDialog = document.getElementById('compactionDialog');
  ui.closeCompactionDialog = document.getElementById('closeCompactionDialog');
  ui.compactionHistoryContent = document.getElementById('compactionHistoryContent');
  ui.messageBanner = document.getElementById('messageBanner');
  ui.thinkingIndicator = document.getElementById('thinkingIndicator');
  ui.thinkingPlan = document.getElementById('thinkingPlan');
  ui.planSummaryText = document.getElementById('planSummaryText');
  ui.modelSelect = document.getElementById('modelSelect');
  ui.onboardingDialog = document.getElementById('onboardingDialog');
  ui.saveCredentialsBtn = document.getElementById('saveCredentialsBtn');
  ui.apiKeyInput = document.getElementById('apiKeyInput');
  ui.apiKeyHelp = document.getElementById('apiKeyHelp');
  ui.onboardingError = document.getElementById('onboardingError');
  ui.onboardingSuccess = document.getElementById('onboardingSuccess');
  ui.systemPromptInput = document.getElementById('systemPromptInput');
  ui.planDropdown = document.getElementById('planDropdown');
  ui.closePlanDropdown = document.getElementById('closePlanDropdown');
  ui.planDropdownTitle = document.getElementById('planDropdownTitle');
  ui.planDropdownSteps = document.getElementById('planDropdownSteps');
  ui.thinkingModelInfo = document.getElementById('thinkingModelInfo');
  ui.workspaceBtn = document.getElementById('workspaceBtn');
  ui.workspaceLabel = document.getElementById('workspaceLabel');
  ui.workspaceMenu = document.getElementById('workspaceMenu');
  ui.workspaceMenuOpen = document.getElementById('workspaceMenuOpen');
  ui.workspaceMenuRecent = document.getElementById('workspaceMenuRecent');
  ui.addWorkspaceMenuBtn = document.getElementById('addWorkspaceMenuBtn');
  ui.helpBtn = document.getElementById('helpBtn');

  ui.promptForm.addEventListener('submit', async (e) => {
    e.preventDefault();
    await submitPrompt();
  });
  ui.promptInput.addEventListener('keydown', (e) => {
    if (e.key === 'Enter' && !e.shiftKey && !autocompleteActive) {
      e.preventDefault();
      submitPrompt();
    }
  });

  // Auto-grow textarea
  ui.promptInput.addEventListener('input', (e) => {
    e.target.style.height = 'auto';
    e.target.style.height = Math.min(e.target.scrollHeight, 200) + 'px';
  });

  ui.cancelBtn.addEventListener('click', cancelRequest);
  ui.thinkingToggle.addEventListener('click', toggleThinking);
  if (ui.forceThinkingToggle) {
    ui.forceThinkingToggle.addEventListener('click', toggleForceThinking);
  }
  if (ui.systemPromptInput) {
    ui.systemPromptInput.addEventListener('blur', updateSystemPrompt);
  }
  ui.compactionHistoryBtn.addEventListener('click', showCompactionHistory);
  ui.closeCompactionDialog.addEventListener('click', closeCompactionHistory);
  ui.compactionDialog.addEventListener('click', (e) => {
    if (e.target === ui.compactionDialog) {
      closeCompactionHistory();
    }
  });

  // Plan dropdown
  if (ui.planSummaryText) {
    ui.planSummaryText.addEventListener('click', togglePlanDropdown);
  }
  if (ui.closePlanDropdown) {
    ui.closePlanDropdown.addEventListener('click', hidePlanDropdown);
  }

  // Close dropdown when clicking outside
  document.addEventListener('click', (e) => {
    if (!ui.planDropdown || ui.planDropdown.classList.contains('hidden')) return;
    if (!ui.planDropdown.contains(e.target) && !ui.planSummaryText.contains(e.target)) {
      hidePlanDropdown();
    }
  });

  // Workspace dropdown in CANDO bar
  if (ui.workspaceBtn) {
    ui.workspaceBtn.addEventListener('click', (e) => {
      e.stopPropagation();
      toggleWorkspaceDropdown();
    });
  }

  // Close workspace dropdown when clicking outside
  document.addEventListener('click', (e) => {
    if (!ui.workspaceMenu || ui.workspaceMenu.classList.contains('hidden')) return;
    if (!ui.workspaceMenu.contains(e.target) && !ui.workspaceBtn.contains(e.target)) {
      hideWorkspaceDropdown();
    }
  });

  // Add workspace button in dropdown menu
  if (ui.addWorkspaceMenuBtn) {
    ui.addWorkspaceMenuBtn.addEventListener('click', () => {
      hideWorkspaceDropdown();
      showFolderPicker();
    });
  }

  // Theme selector
  const themeSelect = document.getElementById('themeSelect');
  const savedTheme = localStorage.getItem('theme') || 'dark';
  document.documentElement.setAttribute('data-theme', savedTheme);
  themeSelect.value = savedTheme;
  themeSelect.addEventListener('change', (e) => {
    const theme = e.target.value;
    document.documentElement.setAttribute('data-theme', theme);
    localStorage.setItem('theme', theme);
  });

  if (ui.modelSelect) {
    ui.modelSelect.addEventListener('change', (e) => {
      const key = e.target.value;
      if (!key) return;
      switchProvider(key);
    });
  }

  // Onboarding event listeners
  ui.saveCredentialsBtn.addEventListener('click', saveCredentials);

  const providerRadios = document.querySelectorAll('input[name="provider"]');
  providerRadios.forEach(radio => {
    radio.addEventListener('change', (e) => {
      const provider = e.target.value;
      if (provider === 'zai') {
        ui.apiKeyHelp.innerHTML = 'Get your Z.AI key at: <a href="https://z.ai" target="_blank">z.ai</a>';
      } else if (provider === 'openrouter') {
        ui.apiKeyHelp.innerHTML = 'Get your OpenRouter key at: <a href="https://openrouter.ai/keys" target="_blank">openrouter.ai/keys</a>';
      }
    });
  });

  // Check credentials on load
  await checkCredentials();

  await refreshSession();

  // Initialize additional components
  initSettings();
  initAutocomplete();
  initWorkspaces();
  updateStatusBar();

  document.addEventListener('keydown', handleGlobalKeydown);
}

async function refreshSession() {
  setBusy(true, 'Syncing context…');
  try {
    const res = await fetchWithWorkspace('/api/session');
    if (!res.ok) throw new Error('Session fetch failed');
    appState.data = await res.json();
    appState.showAllMessages = false;

    // Update localStorage if workspace changed
    if (appState.data.workspace && appState.data.workspace.path) {
      setCurrentWorkspacePath(appState.data.workspace.path);
    } else {
      localStorage.removeItem('currentWorkspace');
    }

    render();
    setStatus(appState.data.running ? 'Sublimating… (Esc to cancel)' : 'Ready.');
  } catch (err) {
    console.error(err);
    setStatus(err.message || 'Failed to load session');
  } finally {
    setBusy(false);
  }
}

function render() {
  if (!appState.data) return;
  renderMessages();
  renderPlan();
  renderModelSelector();
  updateStatusMeta();
  updateThinkingModelInfo();
  updateStatusBar();
  updateWorkspaceUI();
  const sessionBtn = document.getElementById('sessionPickerBtn');
  const hasWorkspace = !!appState.data.workspace;
  if (sessionBtn) {
    sessionBtn.classList.toggle('hidden', !hasWorkspace);
  }
  if (ui.promptInput) {
    if (hasWorkspace) {
      ui.promptInput.disabled = false;
      ui.promptInput.readOnly = !!appState.busy;
      ui.promptInput.placeholder = 'Ask Cando anything… (Enter to send, Shift+Enter for new line)';
    } else {
      ui.promptInput.value = '';
      ui.promptInput.disabled = true;
      ui.promptInput.readOnly = true;
      ui.promptInput.placeholder = 'Select a workspace to get started';
    }
  }
  if (ui.sendBtn && !appState.busy) {
    ui.sendBtn.disabled = !hasWorkspace;
  }
  ui.thinkingToggle.textContent = appState.data.thinking ? 'On' : 'Off';
  ui.thinkingToggle.classList.toggle('active', appState.data.thinking);
  if (ui.forceThinkingToggle) {
    ui.forceThinkingToggle.textContent = appState.data.force_thinking ? 'On' : 'Off';
    ui.forceThinkingToggle.classList.toggle('active', appState.data.force_thinking);
  }
  if (ui.systemPromptInput && ui.systemPromptInput.value !== appState.data.system_prompt) {
    ui.systemPromptInput.value = appState.data.system_prompt || '';
  }
  ui.cancelBtn.disabled = !appState.data.running || !hasWorkspace;
}

function renderMessages() {
  ui.messages.innerHTML = '';
  if (!appState.data?.workspace) {
    renderWorkspaceEmptyState();
    return;
  }
  const total = appState.data.messages.length;
  let messages = appState.data.messages;
  let truncated = false;
  let offset = 0;
  if (!appState.showAllMessages && total > MAX_VISIBLE_MESSAGES) {
    offset = Math.max(total - MAX_VISIBLE_MESSAGES, 0);
    messages = messages.slice(offset);
    truncated = true;
  }

  renderMessageBanner(truncated, total, messages.length);

  const lastPrimaryIndex = findLastPrimaryMessageIndex(messages);
  let previousRole = null;

  for (let i = 0; i < messages.length; i++) {
    const msg = messages[i];
    if (msg.role === 'tool') {
      continue;
    }
    const attached = [];
    let peek = i + 1;
    while (peek < messages.length && messages[peek].role === 'tool') {
      attached.push(messages[peek]);
      peek++;
    }
    i = peek - 1;

    const isLatest = offset + i === findLastPrimaryMessageIndex(appState.data.messages);
    const showRole = msg.role !== previousRole;
    const node = createMessageElement(msg, attached, isLatest, showRole);
    if (node) {
      ui.messages.appendChild(node);
    }
    previousRole = msg.role;
  }
  scrollMessagesToBottom();
}

function scrollMessagesToBottom() {
  if (!ui.messages) return;

  // Use multiple strategies to ensure scroll works after content loads
  const doScroll = () => {
    ui.messages.scrollTop = ui.messages.scrollHeight;
  };

  // Immediate scroll
  doScroll();

  // After next frame
  requestAnimationFrame(() => {
    doScroll();
    // And one more after paint
    requestAnimationFrame(doScroll);
  });

  // Also after a small delay to catch late-loading content
  setTimeout(doScroll, 50);
  setTimeout(doScroll, 150);
}

function renderWorkspaceEmptyState() {
  ui.messages.innerHTML = getWorkspaceGuideHTML();
  wireWorkspaceGuideActions(ui.messages);
}

function getWorkspaceGuideHTML() {
  return `
    <div class="workspace-empty">
      <div class="workspace-empty-card">
        <div class="workspace-guide">
          <div class="workspace-guide-main">
            <div class="workspace-empty-icon">📂</div>
            <h2>Select a workspace to get started</h2>
            <p>Workspaces keep your conversations, files, and sessions organized per project.</p>
            <button class="primary" data-help-action="select-workspace">Select Workspace</button>
          </div>
          <div class="workspace-empty-help">
            <div class="help-item">
              <div class="help-icon">+</div>
              <div class="help-body">
                <h3>Add Workspace</h3>
                <p>Click the + button to register a new folder or reopen a recent project.</p>
              </div>
            </div>
            <div class="help-item">
              <div class="help-icon">⌄</div>
              <div class="help-body">
                <h3>Workspaces Menu</h3>
                <p>Use the “Workspaces” dropdown to jump between open or recently closed projects.</p>
              </div>
            </div>
            <div class="help-item">
              <div class="help-icon">⚙️</div>
              <div class="help-body">
                <h3>Settings</h3>
                <p>Manage API keys, providers, and compaction settings from the gear icon.</p>
                <button class="help-link" data-help-action="open-settings">Open Settings</button>
              </div>
            </div>
            <div class="help-item">
              <div class="help-icon">🧠</div>
              <div class="help-body">
                <h3>Model & Tokens</h3>
                <p>The toolbar shows the active model and how much context budget remains.</p>
              </div>
            </div>
            <div class="help-item">
              <div class="help-icon">?</div>
              <div class="help-body">
                <h3>Need Help?</h3>
                <p>Use the ? button in the status bar to view this guide anytime.</p>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  `;
}

function wireWorkspaceGuideActions(root, afterAction) {
  if (!root) return;
  const selectBtn = root.querySelector('[data-help-action="select-workspace"]');
  if (selectBtn) {
    selectBtn.addEventListener('click', () => {
      showFolderPicker();
      if (typeof afterAction === 'function') afterAction();
    });
  }
  root.querySelectorAll('[data-help-action="open-settings"]').forEach(btn => {
    btn.addEventListener('click', () => {
      openSettingsDialog();
      if (typeof afterAction === 'function') afterAction();
    });
  });
}

function renderModelSelector() {
  if (!ui.modelSelect) return;
  const providers = Array.isArray(appState.data?.providers) ? appState.data.providers : [];
  const active = appState.data?.current_provider || '';
  ui.modelSelect.innerHTML = '';

  if (providers.length === 0) {
    const option = document.createElement('option');
    option.value = '';
    option.textContent = appState.data?.model ? `Model: ${appState.data.model}` : 'Model locked';
    ui.modelSelect.appendChild(option);
    ui.modelSelect.disabled = true;
    return;
  }

  providers.forEach((opt) => {
    const option = document.createElement('option');
    option.value = opt.key;
    option.textContent = opt.label || opt.model || opt.key;
    ui.modelSelect.appendChild(option);
  });

  ui.modelSelect.disabled = providers.length <= 1;
  if (active) {
    ui.modelSelect.value = active;
  }
}

function findLastPrimaryMessageIndex(messages) {
  for (let idx = messages.length - 1; idx >= 0; idx--) {
    if (messages[idx].role !== 'tool') {
      return idx;
    }
  }
  return -1;
}

function renderPlan() {
  const plan = appState.data?.plan;
  if (!plan || !Array.isArray(plan.steps) || plan.steps.length === 0) {
    if (ui.planSummaryText) {
      ui.planSummaryText.textContent = '';
      ui.planSummaryText.classList.remove('completed');
    }
    return;
  }

  updatePlanSummary(plan.steps);
}

function renderMessageBanner(truncated, total, visible) {
  if (!ui.messageBanner) return;
  ui.messageBanner.innerHTML = '';
  if (!truncated && !appState.showAllMessages && total <= MAX_VISIBLE_MESSAGES) {
    ui.messageBanner.classList.add('hidden');
    return;
  }
  ui.messageBanner.classList.remove('hidden');

  const text = document.createElement('span');
  if (appState.showAllMessages) {
    text.textContent = `Showing all ${total} messages.`;
  } else if (truncated) {
    text.textContent = `Showing the latest ${visible} of ${total} messages.`;
  } else {
    text.textContent = `Showing ${visible} messages.`;
  }
  ui.messageBanner.appendChild(text);

  const btn = document.createElement('button');
  btn.className = 'ghost';
  if (appState.showAllMessages) {
    btn.textContent = 'Collapse history';
    btn.addEventListener('click', () => {
      appState.showAllMessages = false;
      renderMessages();
    });
  } else {
    btn.textContent = truncated ? 'Show entire history' : 'Collapse history';
    btn.addEventListener('click', () => {
      appState.showAllMessages = !appState.showAllMessages;
      renderMessages();
    });
  }
  ui.messageBanner.appendChild(btn);
}

function formatSessionMeta(summary) {
  if (!summary) return '';
  const count = summary.message_count ?? summary.messages ?? '';
  const stamp = summary.updated_at || summary.created_at;
  if (!stamp) {
    return count ? `${count} messages` : '';
  }
  const date = new Date(stamp);
  const dateStr = date.toLocaleDateString(undefined, { month: 'short', day: 'numeric' });
  const timeStr = date.toLocaleTimeString(undefined, { hour: '2-digit', minute: '2-digit' });
  const msgs = count ? ` · ${count} msgs` : '';
  return `${dateStr} · ${timeStr}${msgs}`;
}

function roleLabel(msg) {
  switch (msg.role) {
    case 'assistant':
      return 'Cando';
    case 'user':
      return 'You';
    case 'system':
      return 'System';
    case 'tool':
      return msg.name ? `Tool · ${msg.name}` : 'Tool';
    default:
      return msg.role || 'Unknown';
  }
}

function createMessageElement(msg, attachedTools = [], isLatest = false, showRole = true) {
  const wrapper = document.createElement('article');
  wrapper.className = `message ${msg.role}`;

  const body = document.createElement('div');
  body.className = 'message-body';

  if (msg.role === 'tool') {
    body.appendChild(buildToolCard(msg));
  } else {
    if (showRole) {
      const role = document.createElement('div');
      role.className = 'message-role';
      role.textContent = roleLabel(msg);
      wrapper.appendChild(role);
    }

    // Add action buttons (copy for assistant, edit for user)
    if (msg.role === 'assistant' || msg.role === 'user') {
      const actions = document.createElement('div');
      actions.className = 'message-actions';

      if (msg.role === 'assistant') {
        const copyBtn = document.createElement('button');
        copyBtn.className = 'message-action-btn copy-btn';
        copyBtn.title = 'Copy message';
        copyBtn.innerHTML = '📋';
        copyBtn.onclick = () => copyMessageContent(msg.content, copyBtn);
        actions.appendChild(copyBtn);
      }

      if (msg.role === 'user') {
        const editBtn = document.createElement('button');
        editBtn.className = 'message-action-btn edit-btn';
        editBtn.title = 'Edit and branch';
        editBtn.innerHTML = '✏️';
        editBtn.onclick = () => editUserMessage(wrapper, msg);
        actions.appendChild(editBtn);
      }

      wrapper.appendChild(actions);
    }

    // Add thinking block if present
    if (msg.thinking && msg.thinking.trim()) {
      body.appendChild(buildThinkingBlock(msg.thinking));
    }

    // Add main content in a wrapper div to avoid innerHTML overwriting thinking block
    const contentWrapper = document.createElement('div');
    contentWrapper.className = 'message-content';
    contentWrapper.innerHTML = renderMarkdown(msg.content || '');
    body.appendChild(contentWrapper);

    const toolCalls = msg.tool_calls || [];
    if (toolCalls.length > 0 || attachedTools.length > 0) {
      body.appendChild(buildToolGroup(toolCalls, attachedTools, isLatest));
    }
  }

  wrapper.appendChild(body);
  return wrapper;
}

async function copyMessageContent(content, button) {
  try {
    await navigator.clipboard.writeText(content);
    const originalText = button.innerHTML;
    button.innerHTML = '✓';
    button.style.color = 'var(--success)';
    setTimeout(() => {
      button.innerHTML = originalText;
      button.style.color = '';
    }, 2000);
  } catch (err) {
    console.error('Failed to copy:', err);
    button.innerHTML = '✗';
    button.style.color = 'var(--danger)';
    setTimeout(() => {
      button.innerHTML = '📋';
      button.style.color = '';
    }, 2000);
  }
}

function editUserMessage(wrapper, msg) {
  // Find message index in current conversation
  const messages = appState.data?.messages || [];
  const msgIndex = messages.findIndex(m => m.content === msg.content && m.role === 'user');

  if (msgIndex === -1) {
    setStatus('Could not find message to edit');
    return;
  }

  // Replace content with editable textarea
  const contentWrapper = wrapper.querySelector('.message-content');
  if (!contentWrapper) return;

  const originalContent = msg.content;
  const textarea = document.createElement('textarea');
  textarea.className = 'message-edit-textarea';
  textarea.value = originalContent;
  textarea.rows = Math.max(3, originalContent.split('\n').length);

  const actions = document.createElement('div');
  actions.className = 'message-edit-actions';

  const saveBtn = document.createElement('button');
  saveBtn.textContent = 'Submit Edit';
  saveBtn.className = 'primary';
  saveBtn.onclick = async () => {
    const newContent = textarea.value.trim();
    if (!newContent) {
      setStatus('Message cannot be empty');
      return;
    }

    if (newContent === originalContent) {
      // No change, just cancel edit mode
      contentWrapper.innerHTML = renderMarkdown(originalContent);
      return;
    }

    // Show warning dialog
    const confirmed = confirm('This will create a new session from this edit. Continue?');
    if (!confirmed) {
      contentWrapper.innerHTML = renderMarkdown(originalContent);
      return;
    }

    // Create new session and branch
    await createBranchFromEdit(msgIndex, newContent);
  };

  const cancelBtn = document.createElement('button');
  cancelBtn.textContent = 'Cancel';
  cancelBtn.onclick = () => {
    contentWrapper.innerHTML = renderMarkdown(originalContent);
  };

  actions.appendChild(saveBtn);
  actions.appendChild(cancelBtn);

  contentWrapper.innerHTML = '';
  contentWrapper.appendChild(textarea);
  contentWrapper.appendChild(actions);
  textarea.focus();
}

async function createBranchFromEdit(editIndex, newContent) {
  setBusy(true, 'Creating new session...');

  try {
    // Call backend API to create branch
    const branchRes = await fetchWithWorkspace('/api/branch', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        edit_index: editIndex,
        new_content: newContent
      }),
    });

    if (!branchRes.ok) {
      const text = await branchRes.text();
      throw new Error(text || 'Failed to create branch');
    }

    const branchData = await branchRes.json();
    const newSessionKey = branchData.new_session_key;

    // Switch to new session
    await mutateState({ action: 'switch', key: newSessionKey });

    // Submit the edited message for fresh LLM response (don't refresh yet - streaming will update UI)
    setStatus('Working...');

    // Show user message immediately
    appendUserMessage(newContent);
    appendThinkingPlaceholder();
    startThinkingIndicator();

    const streamRes = await fetchWithWorkspace('/api/stream', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ content: newContent }),
    });

    if (!streamRes.ok) {
      throw new Error('Failed to submit edited message');
    }

    // Handle streaming response
    const reader = streamRes.body.getReader();
    const decoder = new TextDecoder();
    let buffer = '';

    while (true) {
      const { done, value } = await reader.read();
      if (done) break;

      buffer += decoder.decode(value, { stream: true });
      const lines = buffer.split('\n');
      buffer = lines.pop() || '';

      for (const line of lines) {
        if (!line.trim() || !line.startsWith('data: ')) continue;
        const data = line.slice(6);
        try {
          const event = JSON.parse(data);
          handleStreamEvent(event);
        } catch (e) {
          console.error('Failed to parse SSE event:', e, data);
        }
      }
    }

    removeThinkingPlaceholder();
    setStatus('Ready.');
  } catch (err) {
    console.error('Branch creation failed:', err);
    setStatus(err.message || 'Failed to create branch');
    removeThinkingPlaceholder();
  } finally {
    setBusy(false);
  }
}

function buildToolCard(msg) {
  const card = document.createElement('div');
  card.className = 'tool-card';
  const name = msg.name || 'Tool';
  const summary = document.createElement('details');

  // Collapse all tools by default
  summary.open = false;

  const heading = document.createElement('summary');
  const cmd = formatToolCommand(msg);
  heading.textContent = cmd ? `${name}: ${cmd}` : `${name} output`;
  summary.appendChild(heading);
  if (msg.content) {
    const pre = document.createElement('pre');
    pre.textContent = msg.content;
    summary.appendChild(pre);
  }
  card.appendChild(summary);
  return card;
}

function buildToolCallCard(toolCall) {
  const card = document.createElement('div');
  card.className = 'tool-card tool-call';
  const funcName = toolCall.function?.name || 'unknown';
  const args = toolCall.function?.arguments || '{}';

  const summary = document.createElement('details');
  summary.open = false;

  const heading = document.createElement('summary');
  heading.innerHTML = `<span class="tool-name">${escapeHtml(funcName)}</span> <span class="tool-status">calling...</span>`;
  summary.appendChild(heading);

  const argsContainer = document.createElement('div');
  argsContainer.className = 'tool-arguments';
  try {
    const parsed = JSON.parse(args);
    const pre = document.createElement('pre');
    pre.textContent = JSON.stringify(parsed, null, 2);
    argsContainer.appendChild(pre);
  } catch (e) {
    const pre = document.createElement('pre');
    pre.textContent = args;
    argsContainer.appendChild(pre);
  }

  summary.appendChild(argsContainer);
  card.appendChild(summary);
  return card;
}

function buildToolGroup(toolCalls, toolMessages, isLatest) {
  const container = document.createElement('details');
  container.open = false;
  container.className = 'tool-group';

  const summary = document.createElement('summary');
  summary.textContent = formatToolGroupLabel(toolCalls, toolMessages);
  container.appendChild(summary);

  const stack = document.createElement('div');
  stack.className = 'tool-stack';

  toolCalls.forEach((toolCall) => {
    stack.appendChild(buildToolCallCard(toolCall));
  });

  toolMessages.forEach((toolMsg) => {
    stack.appendChild(buildToolCard(toolMsg));
  });

  container.appendChild(stack);
  return container;
}

function formatToolGroupLabel(toolCalls, toolMessages) {
  const callCount = (toolCalls || []).length;
  const responseCount = (toolMessages || []).length;
  const total = callCount + responseCount;

  if (total === 0) {
    return 'Tool operations';
  }

  const names = [];
  (toolCalls || []).forEach((tc) => {
    if (tc.function && tc.function.name) {
      names.push(tc.function.name);
    }
  });
  (toolMessages || []).forEach((msg) => {
    if (msg.name && !names.includes(msg.name)) {
      names.push(msg.name);
    }
  });

  if (names.length === 0) {
    return `${total} tool operation${total !== 1 ? 's' : ''}`;
  }

  if (names.length === 1) {
    return `${names[0]}`;
  }

  const maxPreview = 2;
  const preview = names.slice(0, maxPreview).join(', ');
  if (names.length <= maxPreview) {
    return `${names.length} tools: ${preview}`;
  }

  const remaining = names.length - maxPreview;
  return `${names.length} tools: ${preview} +${remaining} more`;
}

function updatePlanSummary(steps) {
  if (!ui.thinkingPlan || !ui.planSummaryText) return;
  const summary = summarizePlanSteps(steps);
  if (!summary) {
    ui.planSummaryText.textContent = '';
    ui.planSummaryText.classList.remove('completed');
    updateThinkingIndicatorVisibility();
    return;
  }
  ui.planSummaryText.textContent = `${summary.text} ${summary.icon}`;
  ui.planSummaryText.classList.toggle('completed', summary.state === 'completed');
  updateThinkingIndicatorVisibility();
}

function summarizePlanSteps(steps = []) {
  if (!steps.length) return null;

  const inProgress = steps.find((s) => s.status === 'in_progress');
  if (inProgress) {
    return { state: 'in_progress', icon: '▶', text: inProgress.step };
  }

  const pending = steps.filter((s) => s.status === 'pending');
  if (pending.length > 0) {
    const firstPending = pending[0];
    return { state: 'pending', icon: '○', text: firstPending.step };
  }

  // All steps are completed - show completion status
  const completed = steps.filter((s) => s.status === 'completed');
  if (completed.length === steps.length) {
    return { state: 'completed', icon: '✓', text: `Plan completed (${completed.length} steps)` };
  }

  // Should not reach here, but return last completed as fallback
  if (completed.length > 0) {
    const lastDone = completed[completed.length - 1];
    return { state: 'completed', icon: '✓', text: lastDone.step };
  }

  return null;
}

function updateThinkingIndicatorVisibility() {
  if (!ui.thinkingIndicator) return;

  // Always show the entire indicator (workspace, CANDO, status)
  ui.thinkingIndicator.classList.remove('hidden');

  // Update status text based on state
  if (ui.thinkingStatus) {
    ui.thinkingStatus.textContent = appState.busy ? ui.thinkingStatus.textContent : 'Ready';
  }

  // Only animate (pulse) when busy
  ui.thinkingIndicator.classList.toggle('busy', appState.busy);
}

function buildThinkingBlock(thinkingContent) {
  const block = document.createElement('div');
  block.className = 'thinking-block';

  const header = document.createElement('div');
  header.className = 'thinking-header';

  const label = document.createElement('span');
  label.className = 'thinking-label';
  label.textContent = '💭 Reasoning / Thinking';
  header.appendChild(label);

  const toggle = document.createElement('button');
  toggle.className = 'thinking-toggle';
  toggle.textContent = '▶';
  toggle.addEventListener('click', () => {
    const content = block.querySelector('.thinking-content');
    if (content.style.display === 'none') {
      content.style.display = 'block';
      toggle.textContent = '▼';
    } else {
      content.style.display = 'none';
      toggle.textContent = '▶';
    }
  });
  header.appendChild(toggle);

  const content = document.createElement('div');
  content.className = 'thinking-content';
  content.style.display = 'none';  // Collapsed by default
  content.innerHTML = renderMarkdown(thinkingContent);

  block.appendChild(header);
  block.appendChild(content);
  return block;
}

function startThinkingIndicator() {
  setStatus('Working...');
}

function stopThinkingIndicator() {
  // No-op, status will be updated by refresh
}

async function submitPrompt() {
  if (!appState.data?.workspace) {
    setStatus('Select a workspace to get started.');
    return;
  }
  if (appState.busy) return;
  const content = ui.promptInput.value.trim();
  if (!content) {
    setStatus('Enter a prompt.');
    return;
  }

  // Immediately show user's message in the feed
  appendUserMessage(content);

  // Show thinking indicator
  appendThinkingPlaceholder();

  // Create abort controller for this request
  appState.currentAbortController = new AbortController();

  setBusy(true);
  startThinkingIndicator();
  ui.promptInput.value = '';
  ui.promptInput.style.height = 'auto';

  try {
    const res = await fetchWithWorkspace('/api/stream', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ content }),
      signal: appState.currentAbortController.signal,
    });

    if (!res.ok) {
      const text = await res.text();
      throw new Error(text || 'Stream failed');
    }

    const reader = res.body.getReader();
    const decoder = new TextDecoder();
    let buffer = '';

    while (true) {
      const { done, value } = await reader.read();
      if (done) break;

      buffer += decoder.decode(value, { stream: true });
      const lines = buffer.split('\n');
      buffer = lines.pop() || '';

      for (const line of lines) {
        if (!line.trim() || !line.startsWith('data: ')) continue;
        const data = line.slice(6);
        try {
          const event = JSON.parse(data);
          handleStreamEvent(event);
        } catch (e) {
          console.error('Failed to parse SSE event:', e, data);
        }
      }
    }

    removeThinkingPlaceholder();
    // No need to refresh - stream events already updated state
    setStatus('Ready.');
  } catch (err) {
    console.error(err);
    removeThinkingPlaceholder();

    // Check if it was aborted
    if (err.name === 'AbortError') {
      setStatus('Request cancelled.');
    } else {
      setStatus(err.message || 'Request failed.');
    }

    // Refresh on error to get correct state from server
    await refreshSession();
  } finally {
    stopThinkingIndicator();
    setBusy(false);
    appState.currentAbortController = null;
  }
}

function appendUserMessage(content) {
  const wrapper = document.createElement('article');
  wrapper.className = 'message user';

  const role = document.createElement('div');
  role.className = 'message-role';
  role.textContent = 'You';

  const body = document.createElement('div');
  body.className = 'message-body';
  body.innerHTML = renderMarkdown(content);

  wrapper.append(role, body);
  ui.messages.appendChild(wrapper);
  scrollMessagesToBottom();
}

function appendThinkingPlaceholder() {
  if (appState.busy && ui.thinkingStatus && !ui.thinkingStatus.textContent) {
    ui.thinkingStatus.textContent = 'Working...';
  }
  updateThinkingIndicatorVisibility();
}

function removeThinkingPlaceholder() {
  if (ui.thinkingStatus && !appState.busy) {
    ui.thinkingStatus.textContent = '';
  }
  updateThinkingIndicatorVisibility();
}

function handleStreamEvent(event) {
  switch (event.type) {
    case 'tool_call_started':
      console.log('Tool call started:', event.data);
      setStatus(`Running ${event.data.function}...`);
      appendStreamingToolCall(event.data);
      break;
    case 'tool_call_completed':
      console.log('Tool call completed:', event.data);
      setStatus(`Completed ${event.data.function}`);
      updateStreamingToolResult(event.data);
      break;
    case 'request_retry': {
      const data = event.data || {};
      const next = data.next_attempt || data.attempt + 1 || 2;
      const max = data.max_attempts || 5;
      const delayMs = Number(data.delay_ms || 0);
      const seconds = (delayMs / 1000).toFixed(1);
      const message = data.error ? ` after error: ${data.error}` : '';
      setStatus(`Retrying request (attempt ${next}/${max}) in ${seconds}s${message}`);
      break;
    }
    case 'assistant_message':
      console.log('Assistant message:', event.data);
      setStatus('Ready.');

      // Construct message object
      const assistantMsg = {
        role: 'assistant',
        content: event.data.content || ''
      };

      // Include thinking if present
      if (event.data.thinking && event.data.thinking.trim()) {
        assistantMsg.thinking = event.data.thinking;
      }

      // Add to app state
      appState.data.messages.push(assistantMsg);

      // Update token counts in real-time
      if (event.data.context_chars !== undefined) {
        appState.data.context_chars = event.data.context_chars;
      }
      if (event.data.total_tokens !== undefined) {
        appState.data.total_tokens = event.data.total_tokens;
      }
      if (event.data.context_limit_tokens !== undefined) {
        appState.data.context_limit_tokens = event.data.context_limit_tokens;
      }
      updateStatusMeta();
      updateThinkingModelInfo();

      // Create and append DOM element
      const msgElement = createMessageElement(assistantMsg, [], true, true);
      if (msgElement) {
        ui.messages.appendChild(msgElement);
        scrollMessagesToBottom();
      }
      break;
    case 'compaction_start':
      console.log('Compaction started:', event.data);
      setStatus(`Compacting context (${event.data.chars_before?.toLocaleString()} chars)...`);
      break;
    case 'compaction_complete':
      console.log('Compaction complete:', event.data);
      const savedChars = Math.max(0, (event.data.chars_before || 0) - (event.data.chars_after || 0));
      const savedTokens = Math.round(savedChars / 3);
      setStatus(`Context compacted: saved ${savedTokens.toLocaleString()} tokens (${event.data.messages_compacted} messages)`);
      // After showing compaction result briefly, transition to "Working..." while waiting for LLM response
      setTimeout(() => {
        // Only set "Working..." if request is still in flight
        if (appState.busy) {
          setStatus('Working...');
        }
      }, 1500);
      break;
    case 'plan_update':
      console.log('Plan updated:', event.data);
      try {
        const planData = JSON.parse(event.data.plan);
        if (!appState.data) {
          appState.data = {};
        }
        appState.data.plan = planData;
        renderPlan();
      } catch (err) {
        console.error('Failed to parse plan update:', err);
      }
      break;
    case 'context_update':
      // Update context values in app state
      if (event.data.context_chars !== undefined) {
        appState.data.context_chars = event.data.context_chars;
      }
      if (event.data.total_tokens !== undefined) {
        appState.data.total_tokens = event.data.total_tokens;
      }
      if (event.data.context_limit_tokens !== undefined) {
        appState.data.context_limit_tokens = event.data.context_limit_tokens;
      }
      // Refresh display
      updateStatusMeta();
      updateThinkingModelInfo();
      break;
    case 'complete':
      console.log('Stream complete');
      break;
    case 'error':
      console.error('Stream error:', event.data);
      setStatus(`Error: ${event.data.message}`);
      break;
  }
}

function appendStreamingToolCall(data) {
  const messages = ui.messages;
  let lastMessage = messages.lastElementChild;

  // Check if last message is an assistant message (could be from assistant_message event)
  if (!lastMessage || !lastMessage.classList.contains('assistant')) {
    // No assistant message exists, create one
    lastMessage = document.createElement('div');
    lastMessage.className = 'message assistant streaming-tools';

    // Add role badge ONLY when creating new message
    const role = document.createElement('div');
    role.className = 'message-role';
    role.textContent = 'Cando';

    const body = document.createElement('div');
    body.className = 'message-body';

    lastMessage.append(role, body);
    messages.appendChild(lastMessage);
  }

  const body = lastMessage.querySelector('.message-body');
  if (!body) return;

  // Find or create tool-group details wrapper (matches static rendering)
  let toolGroup = body.querySelector('.tool-group');
  if (!toolGroup) {
    toolGroup = document.createElement('details');
    toolGroup.className = 'tool-group';
    toolGroup.open = false;

    const summary = document.createElement('summary');
    summary.textContent = 'Tool operations';
    toolGroup.appendChild(summary);

    const toolStack = document.createElement('div');
    toolStack.className = 'tool-stack';
    toolGroup.appendChild(toolStack);

    body.appendChild(toolGroup);
  }

  const toolStack = toolGroup.querySelector('.tool-stack');
  if (!toolStack) return;

  // Create tool card
  const toolCard = document.createElement('div');
  toolCard.className = 'tool-card tool-call';
  toolCard.dataset.toolId = data.id;
  toolCard.dataset.toolName = data.function;

  const details = document.createElement('details');
  details.open = false;

  const summary = document.createElement('summary');
  summary.innerHTML = `<span class="tool-name">${escapeHtml(data.function)}</span> <span class="tool-status">running...</span>`;
  details.appendChild(summary);

  const argsContainer = document.createElement('div');
  argsContainer.className = 'tool-arguments';
  try {
    const parsed = JSON.parse(data.arguments);
    const pre = document.createElement('pre');
    pre.textContent = JSON.stringify(parsed, null, 2);
    argsContainer.appendChild(pre);
  } catch (e) {
    const pre = document.createElement('pre');
    pre.textContent = data.arguments;
    argsContainer.appendChild(pre);
  }
  details.appendChild(argsContainer);
  toolCard.appendChild(details);
  toolStack.appendChild(toolCard);

  // Update tool-group summary with current tool names
  updateToolGroupSummary(toolGroup);

  scrollMessagesToBottom();
}

// Helper to update tool-group summary dynamically as tools stream in
function updateToolGroupSummary(toolGroup) {
  const toolStack = toolGroup.querySelector('.tool-stack');
  if (!toolStack) return;

  const toolCards = toolStack.querySelectorAll('.tool-card');
  const toolNames = [];

  toolCards.forEach(card => {
    const name = card.dataset.toolName;
    if (name && !toolNames.includes(name)) {
      toolNames.push(name);
    }
  });

  const summary = toolGroup.querySelector('summary');
  if (!summary) return;

  if (toolNames.length === 0) {
    summary.textContent = 'Tool operations';
  } else if (toolNames.length === 1) {
    summary.textContent = toolNames[0];
  } else {
    const maxPreview = 2;
    const preview = toolNames.slice(0, maxPreview).join(', ');
    if (toolNames.length <= maxPreview) {
      summary.textContent = `${toolNames.length} tools: ${preview}`;
    } else {
      const remaining = toolNames.length - maxPreview;
      summary.textContent = `${toolNames.length} tools: ${preview} +${remaining} more`;
    }
  }
}

function updateStreamingToolResult(data) {
  const toolCard = document.querySelector(`[data-tool-id="${data.id}"]`);
  if (!toolCard) return;

  const status = toolCard.querySelector('.tool-status');
  if (status) {
    status.textContent = data.error ? 'failed' : 'completed';
    status.style.color = data.error ? 'var(--danger)' : 'var(--muted)';
  }
}

async function switchState(key) {
  setBusy(true, `Switching to ${key}…`);
  try {
    await mutateState({ action: 'switch', key });
  } finally {
    setBusy(false);
  }
}

async function createState() {
  const key = prompt('Name for the new session:');
  if (!key) return;
  setBusy(true, 'Creating session…');
  try {
    await mutateState({ action: 'new', key });
  } finally {
    setBusy(false);
  }
}

async function clearState() {
  if (!confirm('Clear the current conversation history?')) return;
  setBusy(true, 'Clearing history…');
  try {
    await mutateState({ action: 'clear' });
  } finally {
    setBusy(false);
  }
}

async function deleteState() {
  const current = appState.data?.current_key;
  if (!current) return;
  if (!confirm(`Delete session "${current}" permanently?`)) return;
  setBusy(true, 'Deleting session…');
  try {
    await mutateState({ action: 'delete', key: current });
  } finally {
    setBusy(false);
  }
}

async function showCompactionHistory() {
  ui.compactionDialog.style.display = 'flex';
  ui.compactionHistoryContent.innerHTML = '<p>Loading...</p>';

  try {
    const res = await fetchWithWorkspace('/api/compaction-history');
    if (!res.ok) throw new Error('Failed to fetch compaction history');

    const data = await res.json();
    const history = data.history || [];

    if (history.length === 0) {
      ui.compactionHistoryContent.innerHTML = '<div class="no-compaction-history">No compaction events recorded yet.</div>';
      return;
    }

    // Sort by timestamp descending (newest first)
    history.sort((a, b) => new Date(b.timestamp) - new Date(a.timestamp));

    const html = history.map(event => {
      const date = new Date(event.timestamp);
      const saved = event.chars_before - event.chars_after;
      const savingsPercent = ((saved / event.chars_before) * 100).toFixed(1);

      return `
        <div class="compaction-entry">
          <div class="compaction-entry-header">
            <span>${event.messages_compacted} messages compacted</span>
            <span class="compaction-timestamp">${date.toLocaleString()}</span>
          </div>
          <div class="compaction-stats">
            <div class="compaction-stat">
              <div class="compaction-stat-label">Before</div>
              <div class="compaction-stat-value">${event.chars_before.toLocaleString()} chars</div>
            </div>
            <div class="compaction-stat">
              <div class="compaction-stat-label">After</div>
              <div class="compaction-stat-value">${event.chars_after.toLocaleString()} chars</div>
            </div>
            <div class="compaction-stat">
              <div class="compaction-stat-label">Saved</div>
              <div class="compaction-stat-value positive">${saved.toLocaleString()} chars (${savingsPercent}%)</div>
            </div>
            <div class="compaction-stat">
              <div class="compaction-stat-label">Duration</div>
              <div class="compaction-stat-value">${event.duration_ms}ms</div>
            </div>
            <div class="compaction-stat">
              <div class="compaction-stat-label">Considered</div>
              <div class="compaction-stat-value">${event.messages_considered} messages</div>
            </div>
          </div>
        </div>
      `;
    }).join('');

    ui.compactionHistoryContent.innerHTML = html;
  } catch (err) {
    console.error('Error loading compaction history:', err);
    ui.compactionHistoryContent.innerHTML = `<div class="no-compaction-history">Error loading history: ${err.message}</div>`;
  }
}

function closeCompactionHistory() {
  ui.compactionDialog.style.display = 'none';
}

function togglePlanDropdown() {
  if (!ui.planDropdown) return;

  if (ui.planDropdown.classList.contains('hidden')) {
    showPlanDropdown();
  } else {
    hidePlanDropdown();
  }
}

function showPlanDropdown() {
  if (!appState.data?.plan || !appState.data.plan.steps || appState.data.plan.steps.length === 0) {
    return;
  }

  const plan = appState.data.plan;
  const total = plan.steps.length;
  const completed = plan.steps.filter((s) => s.status === 'completed').length;

  // Update title
  if (ui.planDropdownTitle) {
    ui.planDropdownTitle.textContent = `Plan · ${completed}/${total}`;
  }

  // Populate steps
  if (ui.planDropdownSteps) {
    ui.planDropdownSteps.innerHTML = '';
    plan.steps.forEach((step) => {
      const stepItem = document.createElement('div');
      stepItem.className = 'plan-step-item';

      const icon = document.createElement('div');
      icon.className = `plan-step-icon ${step.status}`;
      icon.textContent = step.status === 'in_progress' ? '▶' : step.status === 'completed' ? '✓' : '○';

      const content = document.createElement('div');
      content.className = 'plan-step-content';
      content.textContent = step.step;

      stepItem.appendChild(icon);
      stepItem.appendChild(content);
      ui.planDropdownSteps.appendChild(stepItem);
    });
  }

  // Hide just the summary text (not the whole container) when dropdown is open
  if (ui.planSummaryText) {
    ui.planSummaryText.style.visibility = 'hidden';
  }

  // Show dropdown
  if (ui.planDropdown) {
    ui.planDropdown.classList.remove('hidden');
  }
}

function hidePlanDropdown() {
  if (ui.planDropdown) {
    ui.planDropdown.classList.add('hidden');
  }
  // Restore summary text visibility
  if (ui.planSummaryText) {
    ui.planSummaryText.style.visibility = 'visible';
  }
}

async function checkCredentials() {
  try {
    const res = await fetch('/api/credentials');
    if (!res.ok) throw new Error('Failed to check credentials');
    const data = await res.json();

    if (!data.configured) {
      // Show onboarding modal
      ui.onboardingDialog.style.display = 'flex';
    }
  } catch (err) {
    console.error('Credential check failed:', err);
    // On error, assume credentials not configured and show onboarding
    ui.onboardingDialog.style.display = 'flex';
  }
}

async function saveCredentials() {
  // Get selected provider
  const providerRadio = document.querySelector('input[name="provider"]:checked');
  const provider = providerRadio ? providerRadio.value : '';
  const apiKey = ui.apiKeyInput.value.trim();

  // Clear previous messages
  ui.onboardingError.style.display = 'none';
  ui.onboardingSuccess.style.display = 'none';

  // Validation
  if (!provider) {
    ui.onboardingError.textContent = 'Please select a provider';
    ui.onboardingError.style.display = 'block';
    return;
  }

  if (!apiKey) {
    ui.onboardingError.textContent = 'Please enter your API key';
    ui.onboardingError.style.display = 'block';
    return;
  }

  // Disable button during submission
  ui.saveCredentialsBtn.disabled = true;
  ui.saveCredentialsBtn.textContent = 'Saving...';

  try {
    const res = await fetch('/api/credentials', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        provider: provider,
        api_key: apiKey,
      }),
    });

    if (!res.ok) {
      const errorText = await res.text();
      throw new Error(errorText || 'Failed to save credentials');
    }

    const data = await res.json();

    // Show success message
    ui.onboardingSuccess.textContent = data.message || 'Credentials saved successfully!';
    ui.onboardingSuccess.style.display = 'block';

    // Close modal after 2 seconds and reload page
    setTimeout(() => {
      window.location.reload();
    }, 2000);

  } catch (err) {
    console.error('Failed to save credentials:', err);
    ui.onboardingError.textContent = err.message || 'Failed to save credentials';
    ui.onboardingError.style.display = 'block';
    ui.saveCredentialsBtn.disabled = false;
    ui.saveCredentialsBtn.textContent = 'Save & Continue';
  }
}

async function mutateState(payload) {
  const res = await fetchWithWorkspace('/api/state', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(payload),
  });
  if (!res.ok) {
    const text = await res.text();
    throw new Error(text || 'State update failed');
  }
  appState.data = await res.json();
  appState.showAllMessages = false;
  render();
}

function renderMarkdown(text) {
  const source = text || '';
  if (window.marked) {
    const html = window.marked.parse(source, { breaks: true });
    if (window.DOMPurify) {
      return window.DOMPurify.sanitize(html);
    }
    return html;
  }
  return escapeHtml(source).replace(/\n/g, '<br/>');
}

function escapeHtml(str) {
  return (str || '')
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;')
    .replace(/'/g, '&#39;');
}

async function toggleThinking() {
  if (!appState.data) return;
  const next = !appState.data.thinking;
  const res = await fetch('/api/thinking', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ enabled: next }),
  });
  if (!res.ok) {
    setStatus('Thinking toggle failed');
    return;
  }
  appState.data = await res.json();
  render();
}

async function toggleForceThinking() {
  if (!appState.data) return;
  const next = !appState.data.force_thinking;
  const res = await fetch('/api/force-thinking', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ enabled: next }),
  });
  if (!res.ok) {
    setStatus('Force thinking toggle failed');
    return;
  }
  appState.data = await res.json();
  render();
}

async function updateSystemPrompt() {
  if (!appState.data || !ui.systemPromptInput) return;
  const prompt = ui.systemPromptInput.value.trim();
  const res = await fetch('/api/system-prompt', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ prompt }),
  });
  if (!res.ok) {
    setStatus('System prompt update failed');
    return;
  }
  appState.data = await res.json();
}

async function switchProvider(key) {
  if (!key || !appState.data) return;
  if (key === appState.data.current_provider) return;
  try {
    setStatus('Switching model…');
    if (ui.modelSelect) ui.modelSelect.disabled = true;
    const res = await fetch('/api/provider', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ provider: key }),
    });
    if (!res.ok) {
      const message = await res.text();
      throw new Error(message || 'Provider switch failed');
    }
    appState.data = await res.json();
    render();
    setStatus(`Using ${appState.data.model}`);
  } catch (err) {
    console.error(err);
    setStatus(err.message || 'Failed to switch model');
  } finally {
    if (ui.modelSelect) ui.modelSelect.disabled = false;
  }
}

async function cancelRequest() {
  // Abort the current fetch request if it exists
  if (appState.currentAbortController) {
    appState.currentAbortController.abort();
    appState.currentAbortController = null;
  }

  // Also notify backend to cancel
  try {
    await fetch('/api/cancel', { method: 'POST' });
  } catch (e) {
    console.error('Failed to notify backend of cancellation:', e);
  }
}

function setBusy(flag, message) {
  appState.busy = flag;
  ui.sendBtn.disabled = flag;
  ui.promptInput.readOnly = flag;
  ui.cancelBtn.disabled = !flag;

  // Change button text and style based on state
  if (flag) {
    ui.cancelBtn.textContent = 'Stop';
    ui.cancelBtn.classList.add('danger');
    ui.sendBtn.style.display = 'none';
    ui.cancelBtn.style.display = 'block';
  } else {
    ui.cancelBtn.textContent = 'Cancel';
    ui.cancelBtn.classList.remove('danger');
    ui.sendBtn.style.display = 'block';
    ui.cancelBtn.style.display = 'none';
  }

  if (ui.statusText) {
    ui.statusText.classList.toggle('hidden', flag);
  }
  if (!flag && ui.thinkingStatus) {
    ui.thinkingStatus.textContent = '';
  }

  if (message) setStatus(message);
  updateThinkingIndicatorVisibility();
}

function setStatus(message) {
  const indicatorActive = appState.busy && ui.thinkingIndicator && !ui.thinkingIndicator.classList.contains('hidden');
  if (indicatorActive && ui.thinkingStatus) {
    ui.thinkingStatus.textContent = message || '';
  } else if (ui.statusText) {
    ui.statusText.textContent = message;
  }
}

function updateThinkingModelInfo() {
  if (!appState.data || !ui.thinkingModelInfo) return;
  const usedChars = Math.max(0, appState.data.context_chars || 0);
  const usedTokens = Math.max(0, Math.round(usedChars / 3));
  const limitTokens = Math.max(0, appState.data.context_limit_tokens || 0);
  const model = appState.data.model || '';

  if (limitTokens > 0) {
    const percentUsed = Math.min(100, Math.round((usedTokens / limitTokens) * 100));
    const percentLeft = Math.max(0, 100 - percentUsed);
    ui.thinkingModelInfo.textContent = model ? `${model} • ${percentLeft}% left` : `${percentLeft}% left`;
  } else {
    ui.thinkingModelInfo.textContent = model || '';
  }
}

function updateStatusMeta() {
  if (!appState.data || !ui.statusMeta) {
    return;
  }
  const usedChars = Math.max(0, appState.data.context_chars || 0);
  const usedTokens = Math.max(0, Math.round(usedChars / 3));
  const limitTokens = Math.max(0, appState.data.context_limit_tokens || 0);
  const model = appState.data.model || '';

  if (limitTokens > 0) {
    const percentUsed = Math.min(100, Math.round((usedTokens / limitTokens) * 100));
    const remainingTokens = Math.max(limitTokens - usedTokens, 0);
    if (ui.contextProgressFill) {
      ui.contextProgressFill.style.width = `${percentUsed}%`;
      ui.contextProgressFill.classList.toggle('warning', percentUsed >= 80 && percentUsed < 95);
      ui.contextProgressFill.classList.toggle('danger', percentUsed >= 95);
    }
    if (ui.contextProgressBar) {
      ui.contextProgressBar.classList.remove('hidden');
    }
    const leftLabel = `${remainingTokens.toLocaleString()} tokens left`;
    ui.statusMeta.textContent = model ? `${model} · ${leftLabel}` : leftLabel;
  } else {
    if (ui.contextProgressFill) {
      ui.contextProgressFill.style.width = '0%';
      ui.contextProgressFill.classList.remove('warning', 'danger');
    }
    if (ui.contextProgressBar) {
      ui.contextProgressBar.classList.add('hidden');
    }
    const usedLabel = `${usedTokens.toLocaleString()} tokens used`;
    ui.statusMeta.textContent = model ? `${model} · ${usedLabel}` : usedLabel;
  }
}

function updateContextChars(chars) {
  if (!appState.data) return;
  appState.data.context_chars = chars;
  updateStatusMeta();
  updateThinkingModelInfo();
}


document.addEventListener('DOMContentLoaded', initUI);

function formatToolCommand(msg) {
  if (!msg || !msg.arguments) {
    return '';
  }
  try {
    const data = JSON.parse(msg.arguments);
    if (Array.isArray(data.command)) {
      return data.command.join(' ');
    }
  } catch (err) {
    return '';
  }
  return '';
}

// Random name generator for sessions
const adjectives = ['jungle', 'cosmic', 'mystic', 'golden', 'silver', 'crystal', 'amber', 'ruby', 'emerald', 'sapphire', 'thunder', 'lightning', 'ocean', 'mountain', 'forest', 'desert', 'arctic', 'tropical'];
const nouns = ['safari', 'voyage', 'quest', 'journey', 'odyssey', 'adventure', 'expedition', 'discovery', 'exploration', 'mission', 'trail', 'path', 'route', 'horizon', 'peak', 'valley', 'canyon', 'ridge'];

function generateRandomSessionName() {
  const adj = adjectives[Math.floor(Math.random() * adjectives.length)];
  const noun = nouns[Math.floor(Math.random() * nouns.length)];
  return `${adj}-${noun}`;
}

// Settings dialog management
let settingsDialog, closeSettingsBtn, settingsBtn;
let sessionList, currentSessionName, newSessionBtn, clearSessionBtn, deleteSessionBtn;
let tabBtns = [];
let helpDialog, helpDialogContent, helpBtn, closeHelpBtn;

function initSettings() {
  settingsDialog = document.getElementById('settingsDialog');
  closeSettingsBtn = document.getElementById('closeSettingsDialog');
  settingsBtn = document.getElementById('settingsBtn');
  sessionList = document.getElementById('sessionList');
  currentSessionName = document.getElementById('currentSessionName');
  newSessionBtn = document.getElementById('newSessionBtn');
  clearSessionBtn = document.getElementById('clearSessionBtn');
  deleteSessionBtn = document.getElementById('deleteSessionBtn');
  tabBtns = Array.from(document.querySelectorAll('.tab-btn'));

  if (settingsBtn) {
    settingsBtn.addEventListener('click', openSettingsDialog);
  }
  if (closeSettingsBtn) {
    closeSettingsBtn.addEventListener('click', closeSettingsDialog);
  }
  if (settingsDialog) {
    settingsDialog.addEventListener('click', (e) => {
      if (e.target === settingsDialog) {
        closeSettingsDialog();
      }
    });
  }

  // Tab switching
  tabBtns.forEach(btn => {
    btn.addEventListener('click', () => {
      const tabName = btn.dataset.tab;
      switchTab(tabName);
    });
  });

  // Session actions
  if (newSessionBtn) {
    newSessionBtn.addEventListener('click', createRandomSession);
  }
  if (clearSessionBtn) {
    clearSessionBtn.addEventListener('click', clearState);
  }
  if (deleteSessionBtn) {
    deleteSessionBtn.addEventListener('click', deleteState);
  }

  // API key management
  const saveZaiKey = document.getElementById('saveZaiKey');
  const saveOpenrouterKey = document.getElementById('saveOpenrouterKey');
  if (saveZaiKey) {
    saveZaiKey.addEventListener('click', () => saveApiKey('zai'));
  }
  if (saveOpenrouterKey) {
    saveOpenrouterKey.addEventListener('click', () => saveApiKey('openrouter'));
  }

  // Auto-save on model selection change
  const zaiModelSelect = document.getElementById('zaiModelSelect');
  if (zaiModelSelect) {
    zaiModelSelect.addEventListener('change', () => saveProviderModel('zai'));
  }

  // Auto-save on summary model selection change
  const zaiSummaryModelSelect = document.getElementById('zaiSummaryModelSelect');
  if (zaiSummaryModelSelect) {
    zaiSummaryModelSelect.addEventListener('change', () => saveProviderSummaryModel('zai'));
  }

  // Note: OpenRouter summary model auto-saves in selectSummaryModel() function

  // Compaction settings
  const saveCompactionSettings = document.getElementById('saveCompactionSettings');
  if (saveCompactionSettings) {
    saveCompactionSettings.addEventListener('click', saveCompactionConfig);
  }

  const saveSystemPromptBtn = document.getElementById('saveSystemPrompt');
  if (saveSystemPromptBtn) {
    saveSystemPromptBtn.addEventListener('click', saveSystemPrompt);
  }

  initHelpDialog();
}

async function openSettingsDialog() {
  if (settingsDialog) {
    settingsDialog.style.display = 'flex';
    refreshSessionList();
    refreshCompactionInfo();
    loadApiKeyStatus();
    await loadOpenRouterModels();
    updateProviderStatus();
    initializeProviderAccordions();
    populateSystemPrompt();
  }
}

function closeSettingsDialog() {
  if (settingsDialog) {
    settingsDialog.style.display = 'none';
  }
}

function populateSystemPrompt() {
  if (!ui.systemPromptInput || !appState.data?.config) return;
  ui.systemPromptInput.value = appState.data.config.system_prompt || '';
}

async function saveSystemPrompt() {
  if (!ui.systemPromptInput) return;
  const prompt = ui.systemPromptInput.value;
  try {
    setStatus('Saving system prompt…');
    const res = await fetch('/api/system-prompt', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ prompt }),
    });
    if (!res.ok) {
      const text = await res.text();
      throw new Error(text || 'Failed to save system prompt');
    }
    const data = await res.json();
    if (appState.data?.config) {
      appState.data.config.system_prompt = data.system_prompt || prompt;
    }
    setStatus('System prompt saved.');
  } catch (err) {
    console.error(err);
    alert(err.message || 'Failed to save system prompt.');
  }
}

function initHelpDialog() {
  helpDialog = document.getElementById('helpDialog');
  helpDialogContent = document.getElementById('helpDialogContent');
  helpBtn = document.getElementById('helpBtn');
  closeHelpBtn = document.getElementById('closeHelpDialog');

  if (helpBtn) {
    helpBtn.addEventListener('click', openHelpDialog);
  }
  if (closeHelpBtn) {
    closeHelpBtn.addEventListener('click', closeHelpDialog);
  }
  if (helpDialog) {
    helpDialog.addEventListener('click', (e) => {
      if (e.target === helpDialog) {
        closeHelpDialog();
      }
    });
  }
}

function openHelpDialog() {
  if (!helpDialog || !helpDialogContent) return;
  helpDialog.style.display = 'flex';
  helpDialogContent.innerHTML = getWorkspaceGuideHTML();
  wireWorkspaceGuideActions(helpDialogContent, closeHelpDialog);
}

function closeHelpDialog() {
  if (helpDialog) {
    helpDialog.style.display = 'none';
  }
}

function switchTab(tabName) {
  // Update tab buttons
  tabBtns.forEach(btn => {
    if (btn.dataset.tab === tabName) {
      btn.classList.add('active');
    } else {
      btn.classList.remove('active');
    }
  });

  // Update tab panes
  const panes = document.querySelectorAll('.tab-pane');
  panes.forEach(pane => {
    if (pane.id === `tab-${tabName}`) {
      pane.classList.add('active');
    } else {
      pane.classList.remove('active');
    }
  });
}

async function createRandomSession() {
  const name = generateRandomSessionName();
  try {
    const res = await fetchWithWorkspace('/api/state', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ action: 'new', key: name }),
    });
    if (res.ok) {
      await refreshSession();
      refreshSessionList();
      // Refresh sessions dialog if open
      loadSessionsDialogData();
    }
  } catch (err) {
    console.error('Create session failed:', err);
  }
}

function refreshSessionList() {
  if (!sessionList || !appState.data) return;

  const sessions = appState.data.sessions || [];
  const currentKey = appState.data.current_key || '';

  if (currentSessionName) {
    currentSessionName.textContent = currentKey;
  }

  sessionList.innerHTML = '';

  sessions.forEach(session => {
    const item = document.createElement('div');
    item.className = 'session-item';
    if (session.key === currentKey) {
      item.classList.add('active');
    }

    const name = document.createElement('div');
    name.className = 'session-item-name';
    name.textContent = session.key;

    const meta = document.createElement('div');
    meta.className = 'session-item-meta';
    meta.textContent = `${session.message_count || 0} messages`;

    item.appendChild(name);
    item.appendChild(meta);

    item.addEventListener('click', async () => {
      try {
        const res = await fetch('/api/prompt', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ action: 'use', key: session.key }),
        });
        if (res.ok) {
          await refreshSession();
          refreshSessionList();
        }
      } catch (err) {
        console.error('Switch session failed:', err);
      }
    });

    sessionList.appendChild(item);
  });
}

function refreshProviderList() {
  const providerList = document.getElementById('providerList');
  if (!providerList || !appState.data) return;

  const providers = Array.isArray(appState.data?.providers) ? appState.data.providers : [];
  const activeProvider = appState.data?.current_provider || '';

  providerList.innerHTML = '';

  if (providers.length === 0) {
    providerList.innerHTML = '<p>No providers configured.</p>';
    return;
  }

  providers.forEach(provider => {
    const card = document.createElement('div');
    card.className = 'provider-card';
    if (provider.key === activeProvider) {
      card.classList.add('active');
    }

    const name = document.createElement('div');
    name.className = 'provider-name';
    name.textContent = provider.label || provider.key;

    const model = document.createElement('div');
    model.className = 'provider-model';
    model.textContent = provider.model || '';

    card.appendChild(name);
    card.appendChild(model);

    if (provider.key !== activeProvider) {
      card.style.cursor = 'pointer';
      card.addEventListener('click', async () => {
        try {
          await switchProvider(provider.key);
          refreshProviderList();
        } catch (err) {
          console.error('Switch provider failed:', err);
        }
      });
    }

    providerList.appendChild(card);
  });
}

// OpenRouter model search state
let openrouterModels = [];
let selectedOpenRouterModel = null;
let modelSearchSetup = false;
let summaryModelSearchSetup = false;
let visionModelSearchSetup = false;

// Load OpenRouter models and setup searchable dropdown
async function loadOpenRouterModels() {
  try {
    const res = await fetch('/openrouter-models.json');
    if (!res.ok) throw new Error('Failed to load models');
    openrouterModels = await res.json();

    // Only setup event listeners once
    if (!modelSearchSetup) {
      setupModelSearch();
    }
    if (!summaryModelSearchSetup) {
      setupSummaryModelSearch();
    }
    if (!visionModelSearchSetup) {
      setupVisionModelSearch();
    }
  } catch (err) {
    console.error('Failed to load OpenRouter models:', err);
    openrouterModels = [];
  }
}

function setupModelSearch() {
  const searchInput = document.getElementById('openrouterModelSearch');
  const dropdown = document.getElementById('openrouterModelDropdown');
  const hiddenSelect = document.getElementById('openrouterModelSelect');
  const modelInfo = document.getElementById('openrouterModelInfo');
  const freeOnlyCheckbox = document.getElementById('freeModelsOnly');

  if (!searchInput || !dropdown) return;

  modelSearchSetup = true;

  // Show dropdown on focus
  searchInput.addEventListener('focus', () => {
    filterAndShowModels(searchInput.value);
  });

  // Filter on input
  searchInput.addEventListener('input', (e) => {
    filterAndShowModels(e.target.value);
  });

  // Filter when free-only checkbox changes
  if (freeOnlyCheckbox) {
    freeOnlyCheckbox.addEventListener('change', () => {
      filterAndShowModels(searchInput.value);
    });
  }

  // Hide dropdown when clicking outside
  document.addEventListener('click', (e) => {
    if (!searchInput.contains(e.target) && !dropdown.contains(e.target)) {
      dropdown.style.display = 'none';
    }
  });
}

function filterAndShowModels(query) {
  const dropdown = document.getElementById('openrouterModelDropdown');
  const searchInput = document.getElementById('openrouterModelSearch');
  const freeOnlyCheckbox = document.getElementById('freeModelsOnly');
  if (!dropdown || !searchInput) return;

  const lowerQuery = query.toLowerCase();
  const freeOnly = freeOnlyCheckbox ? freeOnlyCheckbox.checked : false;

  let filtered = openrouterModels.filter(model => {
    // Text search filter
    const matchesSearch = model.name.toLowerCase().includes(lowerQuery) ||
      (model.id && model.id.toLowerCase().includes(lowerQuery));

    // Free models filter (models with $0 pricing)
    const isFree = model.pricing && model.pricing.prompt === "0" && model.pricing.completion === "0";
    const matchesFreeFilter = !freeOnly || isFree;

    return matchesSearch && matchesFreeFilter;
  });

  dropdown.innerHTML = '';

  if (filtered.length === 0) {
    dropdown.innerHTML = '<div class="model-dropdown-item">No models found</div>';
    positionDropdown(searchInput, dropdown);
    dropdown.style.display = 'block';
    return;
  }

  // Show max 50 results
  filtered.slice(0, 50).forEach(model => {
    const item = document.createElement('div');
    item.className = 'model-dropdown-item';

    // Add "FREE" badge if it's a free model
    const isFree = model.pricing && model.pricing.prompt === "0" && model.pricing.completion === "0";
    const freeBadge = isFree ? ' <span style="color: #10b981; font-weight: 600;">[FREE]</span>' : '';

    item.innerHTML = `
      <div class="model-dropdown-item-name">${model.name}${freeBadge}</div>
      <div class="model-dropdown-item-id">${model.id || 'Unknown ID'}</div>
    `;

    item.addEventListener('click', () => {
      selectModel(model);
    });

    dropdown.appendChild(item);
  });

  positionDropdown(searchInput, dropdown);
  dropdown.style.display = 'block';
}

// Position dropdown below input field
function positionDropdown(inputElement, dropdownElement) {
  const rect = inputElement.getBoundingClientRect();
  dropdownElement.style.top = `${rect.bottom + 4}px`;
  dropdownElement.style.left = `${rect.left}px`;
  dropdownElement.style.width = `${rect.width}px`;
}

function selectModel(model) {
  const searchInput = document.getElementById('openrouterModelSearch');
  const dropdown = document.getElementById('openrouterModelDropdown');
  const hiddenSelect = document.getElementById('openrouterModelSelect');
  const modelInfo = document.getElementById('openrouterModelInfo');

  if (!searchInput || !dropdown || !hiddenSelect || !modelInfo) return;

  // Update input and hidden select
  searchInput.value = model.name;
  hiddenSelect.value = model.id;
  selectedOpenRouterModel = model;

  // Hide dropdown
  dropdown.style.display = 'none';

  // Show model info
  const nameEl = modelInfo.querySelector('.model-info-name');
  const pricingEl = modelInfo.querySelector('.model-info-pricing');
  const capabilitiesEl = modelInfo.querySelector('.model-info-capabilities');

  if (nameEl) nameEl.textContent = model.name;

  if (pricingEl) {
    if (model.pricing) {
      // Check if it's a free model
      const isFree = model.pricing.prompt === "0" && model.pricing.completion === "0";

      if (isFree) {
        pricingEl.innerHTML = '<span style="color: #10b981; font-weight: 600; font-size: 1.1rem;">FREE</span>';
      } else {
        // Prices are per token, multiply by 1M for per-million-tokens
        const promptPrice = model.pricing.prompt ? (parseFloat(model.pricing.prompt) * 1000000).toFixed(2) : '—';
        const completionPrice = model.pricing.completion ? (parseFloat(model.pricing.completion) * 1000000).toFixed(2) : '—';
        pricingEl.textContent = `Pricing: $${promptPrice}/1M input tokens, $${completionPrice}/1M output tokens`;
      }
    } else {
      pricingEl.textContent = 'Pricing: Not available';
    }
  }

  if (capabilitiesEl && model.capabilities) {
    capabilitiesEl.textContent = `Capabilities: ${model.capabilities.join(', ')}`;
  }

  modelInfo.style.display = 'block';

  // Auto-save the selected model
  saveProviderModel('openrouter');
}

// Summary model search functions (duplicated for summary model)
function setupSummaryModelSearch() {
  const searchInput = document.getElementById('openrouterSummaryModelSearch');
  const dropdown = document.getElementById('openrouterSummaryModelDropdown');
  const hiddenSelect = document.getElementById('openrouterSummaryModel');
  const modelInfo = document.getElementById('openrouterSummaryModelInfo');
  const freeOnlyCheckbox = document.getElementById('freeModelsOnlySummary');

  if (!searchInput || !dropdown) return;

  summaryModelSearchSetup = true;

  // Show dropdown on focus
  searchInput.addEventListener('focus', () => {
    filterAndShowSummaryModels(searchInput.value);
  });

  // Filter on input
  searchInput.addEventListener('input', (e) => {
    filterAndShowSummaryModels(e.target.value);
  });

  // Filter when free-only checkbox changes
  if (freeOnlyCheckbox) {
    freeOnlyCheckbox.addEventListener('change', () => {
      filterAndShowSummaryModels(searchInput.value);
    });
  }

  // Hide dropdown when clicking outside
  document.addEventListener('click', (e) => {
    if (!searchInput.contains(e.target) && !dropdown.contains(e.target)) {
      dropdown.style.display = 'none';
    }
  });
}

function filterAndShowSummaryModels(query) {
  const dropdown = document.getElementById('openrouterSummaryModelDropdown');
  const searchInput = document.getElementById('openrouterSummaryModelSearch');
  const freeOnlyCheckbox = document.getElementById('freeModelsOnlySummary');
  if (!dropdown || !searchInput) return;

  const lowerQuery = query.toLowerCase();
  const freeOnly = freeOnlyCheckbox ? freeOnlyCheckbox.checked : false;

  let filtered = openrouterModels.filter(model => {
    // Text search filter
    const matchesSearch = model.name.toLowerCase().includes(lowerQuery) ||
      (model.id && model.id.toLowerCase().includes(lowerQuery));

    // Free models filter (models with $0 pricing)
    const isFree = model.pricing && model.pricing.prompt === "0" && model.pricing.completion === "0";
    const matchesFreeFilter = !freeOnly || isFree;

    return matchesSearch && matchesFreeFilter;
  });

  dropdown.innerHTML = '';

  if (filtered.length === 0) {
    dropdown.innerHTML = '<div class="model-dropdown-item">No models found</div>';
    positionDropdown(searchInput, dropdown);
    dropdown.style.display = 'block';
    return;
  }

  // Show max 50 results
  filtered.slice(0, 50).forEach(model => {
    const item = document.createElement('div');
    item.className = 'model-dropdown-item';

    // Add "FREE" badge if it's a free model
    const isFree = model.pricing && model.pricing.prompt === "0" && model.pricing.completion === "0";
    const freeBadge = isFree ? ' <span style="color: #10b981; font-weight: 600;">[FREE]</span>' : '';

    item.innerHTML = `
      <div class="model-dropdown-item-name">${model.name}${freeBadge}</div>
      <div class="model-dropdown-item-id">${model.id || 'Unknown ID'}</div>
    `;

    item.addEventListener('click', () => {
      selectSummaryModel(model);
    });

    dropdown.appendChild(item);
  });

  positionDropdown(searchInput, dropdown);
  dropdown.style.display = 'block';
}

function selectSummaryModel(model) {
  const searchInput = document.getElementById('openrouterSummaryModelSearch');
  const dropdown = document.getElementById('openrouterSummaryModelDropdown');
  const hiddenSelect = document.getElementById('openrouterSummaryModel');
  const modelInfo = document.getElementById('openrouterSummaryModelInfo');

  if (!searchInput || !dropdown || !hiddenSelect || !modelInfo) return;

  // Update input and hidden select
  searchInput.value = model.name;
  hiddenSelect.value = model.id;

  // Hide dropdown
  dropdown.style.display = 'none';

  // Show model info
  const nameEl = modelInfo.querySelector('.model-info-name');
  const pricingEl = modelInfo.querySelector('.model-info-pricing');
  const capabilitiesEl = modelInfo.querySelector('.model-info-capabilities');

  if (nameEl) nameEl.textContent = model.name;

  if (pricingEl) {
    if (model.pricing) {
      // Check if it's a free model
      const isFree = model.pricing.prompt === "0" && model.pricing.completion === "0";

      if (isFree) {
        pricingEl.innerHTML = '<span style="color: #10b981; font-weight: 600; font-size: 1.1rem;">FREE</span>';
      } else {
        // Prices are per token, multiply by 1M for per-million-tokens
        const promptPrice = model.pricing.prompt ? (parseFloat(model.pricing.prompt) * 1000000).toFixed(2) : '—';
        const completionPrice = model.pricing.completion ? (parseFloat(model.pricing.completion) * 1000000).toFixed(2) : '—';
        pricingEl.textContent = `Pricing: $${promptPrice}/1M input tokens, $${completionPrice}/1M output tokens`;
      }
    } else {
      pricingEl.textContent = 'Pricing: Not available';
    }
  }

  if (capabilitiesEl && model.capabilities) {
    capabilitiesEl.textContent = `Capabilities: ${model.capabilities.join(', ')}`;
  }

  modelInfo.style.display = 'block';

  // Auto-save the selected summary model
  saveProviderSummaryModel('openrouter');
}

// Vision model search functions (filtered by image capability)
function setupVisionModelSearch() {
  const searchInput = document.getElementById('openrouterVisionModelSearch');
  const dropdown = document.getElementById('openrouterVisionModelDropdown');
  const hiddenSelect = document.getElementById('openrouterVisionModel');
  const modelInfo = document.getElementById('openrouterVisionModelInfo');
  const freeOnlyCheckbox = document.getElementById('freeModelsOnlyVision');

  if (!searchInput || !dropdown) return;

  visionModelSearchSetup = true;

  // Show dropdown on focus
  searchInput.addEventListener('focus', () => {
    filterAndShowVisionModels(searchInput.value);
  });

  // Filter on input
  searchInput.addEventListener('input', (e) => {
    filterAndShowVisionModels(e.target.value);
  });

  // Filter when free-only checkbox changes
  if (freeOnlyCheckbox) {
    freeOnlyCheckbox.addEventListener('change', () => {
      filterAndShowVisionModels(searchInput.value);
    });
  }

  // Hide dropdown when clicking outside
  document.addEventListener('click', (e) => {
    if (!searchInput.contains(e.target) && !dropdown.contains(e.target)) {
      dropdown.style.display = 'none';
    }
  });
}

function filterAndShowVisionModels(query) {
  const dropdown = document.getElementById('openrouterVisionModelDropdown');
  const searchInput = document.getElementById('openrouterVisionModelSearch');
  const freeOnlyCheckbox = document.getElementById('freeModelsOnlyVision');
  if (!dropdown || !searchInput) return;

  const lowerQuery = query.toLowerCase();
  const freeOnly = freeOnlyCheckbox ? freeOnlyCheckbox.checked : false;

  let filtered = openrouterModels.filter(model => {
    // Must have image capability
    const hasImageCapability = model.capabilities && model.capabilities.includes('image');
    if (!hasImageCapability) return false;

    // Text search filter
    const matchesSearch = model.name.toLowerCase().includes(lowerQuery) ||
      (model.id && model.id.toLowerCase().includes(lowerQuery));

    // Free models filter (models with $0 pricing)
    const isFree = model.pricing && model.pricing.prompt === "0" && model.pricing.completion === "0";
    const matchesFreeFilter = !freeOnly || isFree;

    return matchesSearch && matchesFreeFilter;
  });

  dropdown.innerHTML = '';

  if (filtered.length === 0) {
    dropdown.innerHTML = '<div class="model-dropdown-item">No vision models found</div>';
    positionDropdown(searchInput, dropdown);
    dropdown.style.display = 'block';
    return;
  }

  // Show max 50 results
  filtered.slice(0, 50).forEach(model => {
    const item = document.createElement('div');
    item.className = 'model-dropdown-item';

    // Add "FREE" badge if it's a free model
    const isFree = model.pricing && model.pricing.prompt === "0" && model.pricing.completion === "0";
    const freeBadge = isFree ? ' <span style="color: #10b981; font-weight: 600;">[FREE]</span>' : '';

    item.innerHTML = `
      <div class="model-dropdown-item-name">${model.name}${freeBadge}</div>
      <div class="model-dropdown-item-id">${model.id || 'Unknown ID'}</div>
    `;

    item.addEventListener('click', () => {
      selectVisionModel(model);
    });

    dropdown.appendChild(item);
  });

  positionDropdown(searchInput, dropdown);
  dropdown.style.display = 'block';
}

function selectVisionModel(model) {
  const searchInput = document.getElementById('openrouterVisionModelSearch');
  const dropdown = document.getElementById('openrouterVisionModelDropdown');
  const hiddenSelect = document.getElementById('openrouterVisionModel');
  const modelInfo = document.getElementById('openrouterVisionModelInfo');

  if (!searchInput || !dropdown || !hiddenSelect || !modelInfo) return;

  // Update input and hidden select
  searchInput.value = model.name;
  hiddenSelect.value = model.id;

  // Hide dropdown
  dropdown.style.display = 'none';

  // Show model info
  const nameEl = modelInfo.querySelector('.model-info-name');
  const pricingEl = modelInfo.querySelector('.model-info-pricing');
  const capabilitiesEl = modelInfo.querySelector('.model-info-capabilities');

  if (nameEl) nameEl.textContent = model.name;

  if (pricingEl) {
    if (model.pricing) {
      // Check if it's a free model
      const isFree = model.pricing.prompt === "0" && model.pricing.completion === "0";

      if (isFree) {
        pricingEl.innerHTML = '<span style="color: #10b981; font-weight: 600; font-size: 1.1rem;">FREE</span>';
      } else {
        // Prices are per token, multiply by 1M for per-million-tokens
        const promptPrice = model.pricing.prompt ? (parseFloat(model.pricing.prompt) * 1000000).toFixed(2) : '—';
        const completionPrice = model.pricing.completion ? (parseFloat(model.pricing.completion) * 1000000).toFixed(2) : '—';
        pricingEl.textContent = `Pricing: $${promptPrice}/1M input tokens, $${completionPrice}/1M output tokens`;
      }
    } else {
      pricingEl.textContent = 'Pricing: Not available';
    }
  }

  if (capabilitiesEl && model.capabilities) {
    capabilitiesEl.textContent = `Capabilities: ${model.capabilities.join(', ')}`;
  }

  modelInfo.style.display = 'block';

  // Auto-save the selected vision model
  saveProviderVisionModel('openrouter');
}

function updateProviderStatus() {
  if (!appState.data) return;

  const providers = Array.isArray(appState.data?.providers) ? appState.data.providers : [];
  const activeKey = appState.data?.current_provider || '';

  // Update Z.AI status
  const zaiStatus = document.getElementById('zaiStatus');
  const zaiProvider = providers.find(p => p.key === 'zai');
  const zaiRadio = document.getElementById('radio-zai');

  if (zaiStatus) {
    if (zaiProvider) {
      zaiStatus.textContent = 'Configured';
      zaiStatus.classList.add('configured');
    } else {
      zaiStatus.textContent = 'Not configured';
      zaiStatus.classList.remove('configured');
    }
  }

  if (zaiRadio) {
    zaiRadio.checked = (activeKey === 'zai');
    zaiRadio.disabled = !zaiProvider; // Disable if not configured
  }

  // Update OpenRouter status
  const openrouterStatus = document.getElementById('openrouterStatus');
  const openrouterProvider = providers.find(p => p.key === 'openrouter');
  const openrouterRadio = document.getElementById('radio-openrouter');

  if (openrouterStatus) {
    if (openrouterProvider) {
      openrouterStatus.textContent = 'Configured';
      openrouterStatus.classList.add('configured');
    } else {
      openrouterStatus.textContent = 'Not configured';
      openrouterStatus.classList.remove('configured');
    }
  }

  if (openrouterRadio) {
    openrouterRadio.checked = (activeKey === 'openrouter');
    openrouterRadio.disabled = !openrouterProvider; // Disable if not configured
  }

  // Set current model selections
  if (zaiProvider) {
    const zaiSelect = document.getElementById('zaiModelSelect');
    if (zaiSelect && zaiProvider.model) {
      zaiSelect.value = zaiProvider.model;
    }
  }

  if (openrouterProvider) {
    const openrouterSelect = document.getElementById('openrouterModelSelect');
    const openrouterSearch = document.getElementById('openrouterModelSearch');
    if (openrouterSelect && openrouterProvider.model) {
      openrouterSelect.value = openrouterProvider.model;

      // Find and display the model info
      const model = openrouterModels.find(m => m.id === openrouterProvider.model);
      if (model && openrouterSearch) {
        openrouterSearch.value = model.name;
        selectModel(model);
      }
    }
  }

  // Set summary model selections
  const summaryModels = appState.data?.provider_summary_models || {};

  const zaiSummarySelect = document.getElementById('zaiSummaryModelSelect');
  if (zaiSummarySelect) {
    const zaiSummaryModel = summaryModels['zai'] || appState.data?.summary_model || 'glm-4.5-air';
    zaiSummarySelect.value = zaiSummaryModel;
  }

  const openrouterSummaryHidden = document.getElementById('openrouterSummaryModel');
  const openrouterSummarySearch = document.getElementById('openrouterSummaryModelSearch');
  const openrouterSummaryInfo = document.getElementById('openrouterSummaryModelInfo');

  if (openrouterSummaryHidden) {
    const orSummaryModel = summaryModels['openrouter'] || 'qwen/qwen3-next-80b-a3b-instruct';
    openrouterSummaryHidden.value = orSummaryModel;

    // Find and display the model info
    const model = openrouterModels.find(m => m.id === orSummaryModel);
    if (model && openrouterSummarySearch && openrouterSummaryInfo) {
      openrouterSummarySearch.value = model.name;

      // Populate model info
      const nameEl = openrouterSummaryInfo.querySelector('.model-info-name');
      const pricingEl = openrouterSummaryInfo.querySelector('.model-info-pricing');
      const capabilitiesEl = openrouterSummaryInfo.querySelector('.model-info-capabilities');

      if (nameEl) nameEl.textContent = model.name;

      if (pricingEl && model.pricing) {
        const isFree = model.pricing.prompt === "0" && model.pricing.completion === "0";
        if (isFree) {
          pricingEl.innerHTML = '<span style="color: #10b981; font-weight: 600; font-size: 1.1rem;">FREE</span>';
        } else {
          const promptPrice = model.pricing.prompt ? (parseFloat(model.pricing.prompt) * 1000000).toFixed(2) : '—';
          const completionPrice = model.pricing.completion ? (parseFloat(model.pricing.completion) * 1000000).toFixed(2) : '—';
          pricingEl.textContent = `Pricing: $${promptPrice}/1M input tokens, $${completionPrice}/1M output tokens`;
        }
      }

      if (capabilitiesEl && model.capabilities) {
        capabilitiesEl.textContent = `Capabilities: ${model.capabilities.join(', ')}`;
      }

      openrouterSummaryInfo.style.display = 'block';
    }
  }
}

// Initialize provider accordions - expand active provider by default
function initializeProviderAccordions() {
  if (!appState.data) return;

  const activeKey = appState.data?.current_provider || '';

  // Expand the active provider, collapse others
  const providers = ['zai', 'openrouter'];
  providers.forEach(key => {
    const section = document.querySelector(`.provider-section[data-provider="${key}"]`);
    if (section) {
      if (key === activeKey) {
        section.classList.add('expanded');
      } else {
        section.classList.remove('expanded');
      }
    }
  });
}

// Toggle provider accordion - auto-collapse others
function toggleProviderAccordion(providerKey) {
  const section = document.querySelector(`.provider-section[data-provider="${providerKey}"]`);
  if (!section) return;

  const isCurrentlyExpanded = section.classList.contains('expanded');

  // Collapse all provider sections
  document.querySelectorAll('.provider-section').forEach(s => {
    s.classList.remove('expanded');
  });

  // If it wasn't expanded, expand it now (toggle behavior)
  if (!isCurrentlyExpanded) {
    section.classList.add('expanded');
  }
}

// Toggle compaction advanced settings accordion
function toggleCompactionAdvanced() {
  const section = document.querySelector('.compaction-advanced-section');
  if (!section) return;

  const body = section.querySelector('.compaction-advanced-body');
  const icon = section.querySelector('.accordion-icon');

  if (!body || !icon) return;

  const isExpanded = body.style.display !== 'none';

  if (isExpanded) {
    body.style.display = 'none';
    icon.textContent = '▶';
  } else {
    body.style.display = 'block';
    icon.textContent = '▼';
  }
}

// Switch active provider
async function switchActiveProvider(providerKey) {
  try {
    const res = await fetch('/api/provider', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ provider: providerKey })
    });

    if (res.ok) {
      await refreshSession();
      updateProviderStatus();
      initializeProviderAccordions();
    } else {
      const error = await res.text();
      alert(`Failed to switch provider: ${error}`);
    }
  } catch (err) {
    console.error('Failed to switch provider:', err);
    alert('Failed to switch provider');
  }
}

// Save provider model selection
// Show saved indicator temporarily
function showSavedIndicator(indicatorId) {
  const indicator = document.getElementById(indicatorId);
  if (!indicator) return;

  indicator.style.display = 'block';
  indicator.classList.add('show');

  setTimeout(() => {
    indicator.classList.remove('show');
    setTimeout(() => {
      indicator.style.display = 'none';
    }, 300); // Wait for fade-out transition
  }, 2000); // Show for 2 seconds
}

async function saveProviderModel(providerKey) {
  const selectId = providerKey === 'zai' ? 'zaiModelSelect' : 'openrouterModelSelect';
  const select = document.getElementById(selectId);

  if (!select || !select.value) {
    return;
  }

  const selectedModel = select.value;

  // Check if the selected model is already the current one - avoid unnecessary saves
  const providers = Array.isArray(appState.data?.providers) ? appState.data.providers : [];
  const provider = providers.find(p => p.key === providerKey);
  if (provider && provider.model === selectedModel) {
    console.log(`Model ${selectedModel} already configured for ${providerKey}, skipping save`);
    return;
  }

  try {
    setStatus(`Saving ${providerKey} model…`);
    const res = await fetch('/api/provider/model', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        provider: providerKey,
        model: selectedModel
      })
    });

    if (res.ok) {
      const data = await res.json();
      setStatus(`Model updated: ${selectedModel}`);

      // Show saved indicator
      const indicatorId = providerKey === 'zai' ? 'zaiModelSaved' : 'openrouterModelSaved';
      showSavedIndicator(indicatorId);

      await refreshSession();
      updateProviderStatus();
    } else {
      const error = await res.text();
      setStatus(`Failed to save model: ${error}`);
    }
  } catch (err) {
    console.error('Save model failed:', err);
    setStatus('Failed to save model');
  }
}

// Save provider summary model selection
async function saveProviderSummaryModel(providerKey) {
  const selectId = providerKey === 'zai' ? 'zaiSummaryModelSelect' : 'openrouterSummaryModel';
  const input = document.getElementById(selectId);

  if (!input || !input.value) {
    return;
  }

  const selectedModel = input.value.trim();

  // Check if the selected model is already the current one - avoid unnecessary saves
  const summaryModels = appState.data?.provider_summary_models || {};
  const currentModel = summaryModels[providerKey];
  if (currentModel === selectedModel) {
    console.log(`Summary model ${selectedModel} already configured for ${providerKey}, skipping save`);
    return;
  }

  try {
    setStatus(`Saving ${providerKey} summary model…`);
    const res = await fetch('/api/provider/summary-model', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        provider: providerKey,
        model: selectedModel
      })
    });

    if (res.ok) {
      const data = await res.json();
      setStatus(`Summary model updated: ${selectedModel}`);

      // Show saved indicator
      const indicatorId = providerKey === 'zai' ? 'zaiSummaryModelSaved' : 'openrouterSummaryModelSaved';
      showSavedIndicator(indicatorId);

      await refreshSession();
      updateProviderStatus();
    } else {
      const error = await res.text();
      setStatus(`Failed to save summary model: ${error}`);
    }
  } catch (err) {
    console.error('Save summary model failed:', err);
    setStatus('Failed to save summary model');
  }
}

async function saveProviderVisionModel(providerKey) {
  const selectId = 'openrouterVisionModel';
  const input = document.getElementById(selectId);

  if (!input || !input.value) {
    return;
  }

  const selectedModel = input.value.trim();

  try {
    setStatus(`Saving ${providerKey} vision model…`);
    const res = await fetch('/api/provider/vision-model', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        provider: providerKey,
        model: selectedModel
      })
    });

    if (res.ok) {
      const data = await res.json();
      setStatus(`Vision model updated: ${selectedModel}`);
      await loadApiKeyStatus();
    } else {
      const error = await res.text();
      setStatus(`Failed to save vision model: ${error}`);
    }
  } catch (err) {
    console.error('Save vision model failed:', err);
    setStatus('Failed to save vision model');
  }
}

async function saveApiKey(provider) {
  const inputId = provider === 'zai' ? 'zaiApiKey' : 'openrouterApiKey';
  const input = document.getElementById(inputId);
  if (!input || !input.value.trim()) {
    alert('Please enter a valid API key');
    return;
  }

  // Get vision model if set
  const visionModelId = provider === 'zai' ? 'zaiVisionModel' : 'openrouterVisionModel';
  const visionModelInput = document.getElementById(visionModelId);
  const visionModel = visionModelInput ? visionModelInput.value.trim() : '';

  try {
    const payload = {
      provider: provider,
      api_key: input.value.trim()
    };
    if (visionModel) {
      payload.vision_model = visionModel;
    }

    const res = await fetch('/api/credentials', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(payload)
    });

    if (res.ok) {
      // Parse response
      const data = await res.json();

      // Clear input and update placeholder
      input.value = '';
      input.placeholder = 'API key configured ✓';

      // Update button text temporarily
      const saveBtn = document.getElementById(provider === 'zai' ? 'saveZaiKey' : 'saveOpenrouterKey');
      if (saveBtn) {
        const originalText = saveBtn.textContent;
        saveBtn.textContent = 'Saved ✓';
        saveBtn.disabled = true;
        setTimeout(() => {
          saveBtn.textContent = originalText;
          saveBtn.disabled = false;
        }, 2000);
      }

      // Show success message from server
      alert(data.message || `${provider.toUpperCase()} provider configured successfully!`);

      // Refresh session to get updated provider list
      await refreshSession();
      await loadApiKeyStatus();
      updateProviderStatus();
    } else {
      const error = await res.text();
      alert(`Failed to save API key: ${error}`);
    }
  } catch (err) {
    console.error('Save API key failed:', err);
    alert('Failed to save API key. Check console for details.');
  }
}

async function loadApiKeyStatus() {
  try {
    const res = await fetch('/api/credentials');
    if (!res.ok) return;

    const data = await res.json();
    const zaiInput = document.getElementById('zaiApiKey');
    const openrouterInput = document.getElementById('openrouterApiKey');
    const zaiVisionModel = document.getElementById('zaiVisionModel');
    const openrouterVisionModel = document.getElementById('openrouterVisionModel');

    if (zaiInput) {
      if (data.zai_configured) {
        zaiInput.placeholder = '••••••••••••••••••••••••';
        zaiInput.setAttribute('data-configured', 'true');
      } else {
        zaiInput.placeholder = 'Enter Z.AI API key';
        zaiInput.removeAttribute('data-configured');
      }
    }
    if (zaiVisionModel && data.zai_vision_model) {
      zaiVisionModel.value = data.zai_vision_model;
    }
    if (openrouterInput) {
      if (data.openrouter_configured) {
        openrouterInput.placeholder = '••••••••••••••••••••••••';
        openrouterInput.setAttribute('data-configured', 'true');
      } else {
        openrouterInput.placeholder = 'Enter OpenRouter API key';
        openrouterInput.removeAttribute('data-configured');
      }
    }
    // Handle OpenRouter vision model (with default)
    const openrouterVisionSearch = document.getElementById('openrouterVisionModelSearch');
    const openrouterVisionInfo = document.getElementById('openrouterVisionModelInfo');
    const defaultVisionModelId = 'qwen/qwen2.5-vl-32b-instruct';

    if (openrouterVisionModel && openrouterVisionSearch && openrouterModels.length > 0) {
      const savedModelId = data.openrouter_vision_model;
      const modelId = savedModelId || defaultVisionModelId;

      // Find the paid version (non-free) of the model
      let model = openrouterModels.find(m =>
        m.id === modelId &&
        m.pricing &&
        !(m.pricing.prompt === "0" && m.pricing.completion === "0")
      );

      // Fall back to any version if paid not found
      if (!model) {
        model = openrouterModels.find(m => m.id === modelId);
      }

      if (model) {
        openrouterVisionModel.value = model.id;
        openrouterVisionSearch.value = model.name;

        // Show model info
        if (openrouterVisionInfo) {
          const nameEl = openrouterVisionInfo.querySelector('.model-info-name');
          const pricingEl = openrouterVisionInfo.querySelector('.model-info-pricing');
          const capabilitiesEl = openrouterVisionInfo.querySelector('.model-info-capabilities');

          if (nameEl) nameEl.textContent = model.name;
          if (pricingEl) {
            if (model.pricing) {
              const isFree = model.pricing.prompt === "0" && model.pricing.completion === "0";
              if (isFree) {
                pricingEl.innerHTML = '<span style="color: #10b981; font-weight: 600; font-size: 1.1rem;">FREE</span>';
              } else {
                const promptPrice = model.pricing.prompt ? (parseFloat(model.pricing.prompt) * 1000000).toFixed(2) : '—';
                const completionPrice = model.pricing.completion ? (parseFloat(model.pricing.completion) * 1000000).toFixed(2) : '—';
                pricingEl.textContent = `Pricing: $${promptPrice}/1M input tokens, $${completionPrice}/1M output tokens`;
              }
            }
          }
          if (capabilitiesEl && model.capabilities) {
            capabilitiesEl.textContent = `Capabilities: ${model.capabilities.join(', ')}`;
          }
          openrouterVisionInfo.style.display = 'block';
        }
      }
    }
  } catch (err) {
    console.error('Load API key status failed:', err);
  }
}

function refreshCompactionInfo() {
  if (!appState.data || !appState.data.config) return;

  const config = appState.data.config || {};

  const profileEl = document.getElementById('compactionProfile');
  const messagePercentEl = document.getElementById('compactionMessagePercent');
  const conversationPercentEl = document.getElementById('compactionConversationPercent');
  const protectRecentEl = document.getElementById('compactionProtectRecent');
  const messagePercentValueEl = document.getElementById('messagePercentValue');
  const conversationPercentValueEl = document.getElementById('conversationPercentValue');

  if (profileEl) {
    const profile = config.context_profile || 'default';
    profileEl.textContent = profile === 'memory' ? 'Perpetual' : profile;
  }

  // Convert decimal percentages (0.02, 0.50) to integers (2, 50) for sliders
  if (messagePercentEl && messagePercentValueEl) {
    const messagePercent = Math.round((config.context_message_percent || 0.02) * 100);
    messagePercentEl.value = messagePercent;
    messagePercentValueEl.textContent = messagePercent;
  }
  if (conversationPercentEl && conversationPercentValueEl) {
    const conversationPercent = Math.round((config.context_conversation_percent || 0.50) * 100);
    conversationPercentEl.value = conversationPercent;
    conversationPercentValueEl.textContent = conversationPercent;
  }
  if (protectRecentEl) {
    protectRecentEl.value = config.context_protect_recent || '';
  }

  // Set up slider value updates
  if (messagePercentEl) {
    messagePercentEl.addEventListener('input', (e) => {
      if (messagePercentValueEl) {
        messagePercentValueEl.textContent = e.target.value;
      }
    });
  }
  if (conversationPercentEl) {
    conversationPercentEl.addEventListener('input', (e) => {
      if (conversationPercentValueEl) {
        conversationPercentValueEl.textContent = e.target.value;
      }
    });
  }
}

async function saveCompactionConfig() {
  const messagePercentInt = parseInt(document.getElementById('compactionMessagePercent').value);
  const conversationPercentInt = parseInt(document.getElementById('compactionConversationPercent').value);
  const protectRecent = parseInt(document.getElementById('compactionProtectRecent').value);

  if (isNaN(messagePercentInt) || messagePercentInt <= 0 || messagePercentInt > 10) {
    alert('Message threshold must be between 1% and 10%');
    return;
  }
  if (isNaN(conversationPercentInt) || conversationPercentInt <= 0 || conversationPercentInt > 80) {
    alert('Conversation threshold must be between 1% and 80%');
    return;
  }
  if (isNaN(protectRecent) || protectRecent < 0) {
    alert('Protected recent must be 0 or greater');
    return;
  }

  // Convert integer percentages to decimal (2 -> 0.02, 50 -> 0.50)
  const messagePercent = messagePercentInt / 100;
  const conversationPercent = conversationPercentInt / 100;

  try {
    const res = await fetch('/api/config', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        context_message_percent: messagePercent,
        context_conversation_percent: conversationPercent,
        context_protect_recent: protectRecent
      })
    });

    if (res.ok) {
      alert('Compaction settings saved successfully');
      await refreshSession();
      refreshCompactionInfo();
    } else {
      const error = await res.text();
      alert(`Failed to save settings: ${error}`);
    }
  } catch (err) {
    console.error('Save compaction settings failed:', err);
    alert('Failed to save settings. Check console for details.');
  }
}

// Update status bar with working directory
function updateStatusBar() {
  // Deprecated placeholder
}

function handleGlobalKeydown(e) {
  if (e.key !== 'Escape') {
    return;
  }

  const closers = [
    ['folderPickerDialog', ['closeFolderPicker', 'cancelFolderPicker']],
    ['closeWorkspaceDialog', ['cancelCloseWorkspaceBtn', 'cancelCloseWorkspace']],
    ['sessionsDialog', ['cancelSessionsDialog', 'closeSessionsDialog']],
    ['settingsDialog', ['closeSettingsDialog'], closeSettingsDialog],
    ['compactionDialog', ['closeCompactionDialog']],
    ['onboardingDialog', ['closeOnboardingDialog']],
    ['helpDialog', ['closeHelpDialog'], closeHelpDialog],
  ];

  for (const [dialogId, buttons, fallback] of closers) {
    if (closeDialogIfOpen(dialogId, buttons, fallback)) {
      e.preventDefault();
      return;
    }
  }
}

function closeDialogIfOpen(dialogId, buttonIds = [], fallback) {
  const dialog = document.getElementById(dialogId);
  if (!dialog) return false;
  const visible = window.getComputedStyle(dialog).display !== 'none';
  if (!visible) return false;

  if (Array.isArray(buttonIds)) {
    for (const id of buttonIds) {
      const btn = document.getElementById(id);
      if (btn) {
        btn.click();
        return true;
      }
    }
  }

  if (typeof fallback === 'function') {
    fallback();
    return true;
  }

  dialog.style.display = 'none';
  return true;
}

// File autocomplete functionality
let autocompleteDropdown = null;
let autocompleteActive = false;
let autocompleteResults = [];
let autocompleteSelectedIndex = -1;
let autocompleteStartPos = -1;
let autocompleteQuery = '';

function initAutocomplete() {
  if (!ui.promptInput) return;

  // Create dropdown element
  autocompleteDropdown = document.createElement('div');
  autocompleteDropdown.className = 'autocomplete-dropdown';
  autocompleteDropdown.style.display = 'none';
  document.body.appendChild(autocompleteDropdown);

  // Listen for input changes
  ui.promptInput.addEventListener('input', handleAutocompleteInput);
  ui.promptInput.addEventListener('keydown', handleAutocompleteKeydown);

  // Close on click outside
  document.addEventListener('click', (e) => {
    if (!autocompleteDropdown.contains(e.target) && e.target !== ui.promptInput) {
      hideAutocomplete();
    }
  });
}

function handleAutocompleteInput(e) {
  const textarea = e.target;
  const text = textarea.value;
  const cursorPos = textarea.selectionStart;

  // Find @ character before cursor
  let atPos = -1;
  for (let i = cursorPos - 1; i >= 0; i--) {
    if (text[i] === '@') {
      // Check if @ is at start or has whitespace before it
      if (i === 0 || /\s/.test(text[i - 1])) {
        atPos = i;
        break;
      }
    }
    if (/\s/.test(text[i])) {
      break; // Stop at whitespace
    }
  }

  if (atPos !== -1) {
    const query = text.substring(atPos + 1, cursorPos);
    autocompleteStartPos = atPos;
    autocompleteQuery = query;
    fetchFileCompletions(query);
  } else {
    hideAutocomplete();
  }
}

function handleAutocompleteKeydown(e) {
  if (!autocompleteActive) return;

  if (e.key === 'ArrowDown') {
    e.preventDefault();
    autocompleteSelectedIndex = Math.min(
      autocompleteSelectedIndex + 1,
      autocompleteResults.length - 1
    );
    renderAutocomplete();
  } else if (e.key === 'ArrowUp') {
    e.preventDefault();
    autocompleteSelectedIndex = Math.max(autocompleteSelectedIndex - 1, 0);
    renderAutocomplete();
  } else if (e.key === 'Enter' && autocompleteActive) {
    e.preventDefault();
    selectAutocompleteItem(autocompleteSelectedIndex);
  } else if (e.key === 'Escape') {
    hideAutocomplete();
  }
}

async function fetchFileCompletions(query) {
  try {
    const res = await fetchWithWorkspace('/api/files?q=' + encodeURIComponent(query));
    if (!res.ok) return;

    const files = await res.json();
    autocompleteResults = files || [];
    autocompleteSelectedIndex = 0;

    if (autocompleteResults.length > 0) {
      showAutocomplete();
      renderAutocomplete();
    } else {
      hideAutocomplete();
    }
  } catch (err) {
    console.error('File completion failed:', err);
  }
}

function showAutocomplete() {
  autocompleteActive = true;
  autocompleteDropdown.style.display = 'block';
  positionAutocomplete();
}

function hideAutocomplete() {
  autocompleteActive = false;
  autocompleteDropdown.style.display = 'none';
  autocompleteResults = [];
  autocompleteSelectedIndex = -1;
}

function positionAutocomplete() {
  if (!ui.promptInput) return;

  const rect = ui.promptInput.getBoundingClientRect();
  autocompleteDropdown.style.left = rect.left + 'px';
  autocompleteDropdown.style.bottom = (window.innerHeight - rect.top + 5) + 'px';
  autocompleteDropdown.style.width = Math.min(400, rect.width) + 'px';
}

function renderAutocomplete() {
  if (!autocompleteDropdown) return;

  autocompleteDropdown.innerHTML = '';

  autocompleteResults.forEach((file, index) => {
    const item = document.createElement('div');
    item.className = 'autocomplete-item';
    if (index === autocompleteSelectedIndex) {
      item.classList.add('selected');
    }

    const icon = document.createElement('span');
    icon.className = 'autocomplete-icon';
    icon.textContent = file.type === 'dir' ? '📁' : '📄';

    const name = document.createElement('span');
    name.className = 'autocomplete-name';
    name.textContent = file.name;

    const path = document.createElement('span');
    path.className = 'autocomplete-path';
    path.textContent = file.path;

    item.appendChild(icon);
    item.appendChild(name);
    item.appendChild(path);

    item.addEventListener('click', () => selectAutocompleteItem(index));

    autocompleteDropdown.appendChild(item);
  });
}

function selectAutocompleteItem(index) {
  if (index < 0 || index >= autocompleteResults.length) return;

  const file = autocompleteResults[index];
  const textarea = ui.promptInput;
  const text = textarea.value;

  // Replace @query with file path
  const before = text.substring(0, autocompleteStartPos);
  const after = text.substring(textarea.selectionStart);

  // Insert full path in quotes to prevent LLM confusion
  const quotedPath = `"${file.path}"`;
  const newText = before + quotedPath + after;

  textarea.value = newText;
  textarea.selectionStart = textarea.selectionEnd = before.length + quotedPath.length;

  hideAutocomplete();
  textarea.focus();
}

// ========== WORKSPACE MANAGEMENT ==========

let workspaceState = {
  workspaces: [],
  currentWorkspace: null,
  recentWorkspaces: []
};

// Workspace localStorage helpers
function getCurrentWorkspacePath() {
  return localStorage.getItem('currentWorkspace') || '';
}

function setCurrentWorkspacePath(path) {
  localStorage.setItem('currentWorkspace', path);
}

// Helper to add workspace header to fetch requests
function fetchWithWorkspace(url, options = {}) {
  const workspace = getCurrentWorkspacePath();
  if (workspace) {
    options.headers = options.headers || {};
    options.headers['X-Workspace'] = workspace;
  }
  return fetch(url, options);
}

async function initWorkspaces() {
  // Session picker button
  const sessionPickerBtn = document.getElementById('sessionPickerBtn');
  if (sessionPickerBtn) {
    sessionPickerBtn.addEventListener('click', showSessionsDialog);
  }

  // Workspace menu button
  const workspaceMenuBtn = document.getElementById('workspaceMenuBtn');
  if (workspaceMenuBtn) {
    workspaceMenuBtn.addEventListener('click', (e) => {
      e.stopPropagation();
      toggleWorkspaceMenu();
    });
  }

  // Close workspace menu when clicking outside
  document.addEventListener('click', (e) => {
    const menu = document.getElementById('workspaceMenu');
    const menuBtn = document.getElementById('workspaceMenuBtn');
    if (menu && menuBtn && menu.style.display !== 'none') {
      if (!menu.contains(e.target) && !menuBtn.contains(e.target)) {
        menu.style.display = 'none';
      }
    }
  });

  // Add workspace button
  const addWorkspaceBtn = document.getElementById('addWorkspaceBtn');
  if (addWorkspaceBtn) {
    addWorkspaceBtn.addEventListener('click', showFolderPicker);
  }

  // Scroll buttons
  const scrollLeft = document.getElementById('workspaceTabScrollLeft');
  const scrollRight = document.getElementById('workspaceTabScrollRight');
  const tabsContainer = document.getElementById('workspaceTabs');

  if (scrollLeft && tabsContainer) {
    scrollLeft.addEventListener('click', () => {
      tabsContainer.scrollLeft -= 150;
    });
  }

  if (scrollRight && tabsContainer) {
    scrollRight.addEventListener('click', () => {
      tabsContainer.scrollLeft += 150;
    });
  }

  // Check scroll buttons visibility on scroll
  if (tabsContainer) {
    tabsContainer.addEventListener('scroll', updateScrollButtons);
  }

  // Initialize workspace data from session payload
  updateWorkspaceUI();
}

function updateWorkspaceUI() {
  if (!appState.data) return;

  workspaceState.workspaces = appState.data.workspaces || [];
  workspaceState.currentWorkspace = appState.data.workspace || null;
  workspaceState.recentWorkspaces = appState.data.recent_workspaces || [];

  // Update workspace button label in input bar
  if (ui.workspaceLabel && workspaceState.currentWorkspace) {
    const path = workspaceState.currentWorkspace.path || '';
    const name = workspaceState.currentWorkspace.name || path.split('/').pop() || 'Workspace';
    ui.workspaceLabel.textContent = name;
  } else if (ui.workspaceLabel) {
    ui.workspaceLabel.textContent = 'No workspace';
  }

  renderWorkspaceTabs();
  updateSessionPickerLabel();
  updateScrollButtons();
}

function renderWorkspaceTabs() {
  const container = document.getElementById('workspaceTabs');
  if (!container) return;

  container.innerHTML = '';

  const ordered = [...workspaceState.workspaces].sort((a, b) => {
    const timeA = a.added ? new Date(a.added).getTime() : 0;
    const timeB = b.added ? new Date(b.added).getTime() : 0;
    return timeB - timeA;
  });

  ordered.forEach(workspace => {
    const tab = document.createElement('div');
    tab.className = 'workspace-tab';
    if (workspaceState.currentWorkspace && workspace.path === workspaceState.currentWorkspace.path) {
      tab.classList.add('active');
    }

    const nameSpan = document.createElement('span');
    nameSpan.className = 'workspace-tab-name';
    const pathParts = workspace.path.split('/');
    const shortPath = pathParts.slice(-2).join('/');
    nameSpan.innerHTML = '📁 ' + shortPath;
    nameSpan.title = workspace.path;

    const closeBtn = document.createElement('button');
    closeBtn.className = 'workspace-tab-close';
    closeBtn.innerHTML = '<i data-lucide="x"></i>';
    closeBtn.title = 'Close workspace';
    closeBtn.addEventListener('click', (e) => {
      e.stopPropagation();
      showCloseWorkspaceDialog(workspace);
    });

    tab.appendChild(nameSpan);
    tab.appendChild(closeBtn);

    tab.addEventListener('click', () => {
      if (workspace.path !== workspaceState.currentWorkspace?.path) {
        switchWorkspace(workspace.path);
      }
    });

    container.appendChild(tab);
  });

  // Reinitialize Lucide icons for new elements
  if (window.lucide) {
    lucide.createIcons();
  }
}

function updateScrollButtons() {
  const container = document.getElementById('workspaceTabs');
  const scrollLeft = document.getElementById('workspaceTabScrollLeft');
  const scrollRight = document.getElementById('workspaceTabScrollRight');

  if (!container || !scrollLeft || !scrollRight) return;

  const isScrollable = container.scrollWidth > container.clientWidth;
  const isAtStart = container.scrollLeft <= 0;
  const isAtEnd = container.scrollLeft >= container.scrollWidth - container.clientWidth - 5;

  if (isScrollable) {
    scrollLeft.classList.toggle('hidden', isAtStart);
    scrollRight.classList.toggle('hidden', isAtEnd);
  } else {
    scrollLeft.classList.add('hidden');
    scrollRight.classList.add('hidden');
  }
}

function renderWorkspaceMenu() {
  const menu = document.getElementById('workspaceMenu');
  if (!menu) return;

  menu.innerHTML = '';

  // Sort workspaces by Added time (most recent first)
  const sorted = [...workspaceState.workspaces].sort((a, b) => {
    const timeA = a.added ? new Date(a.added).getTime() : 0;
    const timeB = b.added ? new Date(b.added).getTime() : 0;
    return timeB - timeA; // Descending order
  });

  sorted.forEach(workspace => {
    const item = document.createElement('div');
    item.className = 'workspace-menu-item';
    if (workspaceState.currentWorkspace && workspace.path === workspaceState.currentWorkspace.path) {
      item.classList.add('current');
    }

    const pathDiv = document.createElement('div');
    pathDiv.className = 'workspace-menu-item-path';
    pathDiv.textContent = workspace.path;
    pathDiv.title = workspace.path;

    item.appendChild(pathDiv);

    // Add checkmark for current workspace
    if (workspaceState.currentWorkspace && workspace.path === workspaceState.currentWorkspace.path) {
      const check = document.createElement('i');
      check.className = 'workspace-menu-item-check';
      check.setAttribute('data-lucide', 'check');
      item.appendChild(check);
    }

    item.addEventListener('click', () => {
      if (workspace.path !== workspaceState.currentWorkspace?.path) {
        switchWorkspace(workspace.path);
      }
      toggleWorkspaceMenu(); // Close menu after selection
    });

    menu.appendChild(item);
  });

  const history = (workspaceState.recentWorkspaces || []).filter(entry => entry && !workspaceState.workspaces.some(ws => ws.path === entry.path));
  if (history.length > 0) {
    const divider = document.createElement('div');
    divider.className = 'workspace-menu-divider';
    menu.appendChild(divider);

    const recentLabel = document.createElement('div');
    recentLabel.className = 'workspace-menu-section';
    recentLabel.textContent = 'Recently closed';
    menu.appendChild(recentLabel);

    history.slice(0, 5).forEach(workspace => {
      const item = document.createElement('div');
      item.className = 'workspace-menu-item recent';

      const name = workspace.name || workspace.path.split('/').filter(Boolean).pop() || workspace.path;
      const info = document.createElement('div');
      info.className = 'workspace-menu-item-path';
      info.textContent = `${name} · ${workspace.path}`;
      info.title = workspace.path;
      item.appendChild(info);

      item.addEventListener('click', () => {
        reopenWorkspace(workspace.path);
        toggleWorkspaceMenu();
      });

      menu.appendChild(item);
    });
  }

  // Reinitialize Lucide icons
  if (window.lucide) {
    lucide.createIcons();
  }
}

function toggleWorkspaceMenu() {
  const menu = document.getElementById('workspaceMenu');
  if (!menu) return;

  const isVisible = menu.style.display !== 'none';
  if (isVisible) {
    menu.style.display = 'none';
  } else {
    renderWorkspaceMenu();
    menu.style.display = 'block';
  }
}

function toggleWorkspaceDropdown() {
  if (!ui.workspaceMenu) return;

  const isHidden = ui.workspaceMenu.classList.contains('hidden');
  if (isHidden) {
    renderWorkspaceDropdown();
    ui.workspaceMenu.classList.remove('hidden');
  } else {
    ui.workspaceMenu.classList.add('hidden');
  }
}

function hideWorkspaceDropdown() {
  if (ui.workspaceMenu) {
    ui.workspaceMenu.classList.add('hidden');
  }
}

function renderWorkspaceDropdown() {
  if (!ui.workspaceMenuOpen || !ui.workspaceMenuRecent) return;

  // Clear existing items
  ui.workspaceMenuOpen.innerHTML = '';
  ui.workspaceMenuRecent.innerHTML = '';

  // Sort workspaces by added time (most recent first)
  const sorted = [...workspaceState.workspaces].sort((a, b) => {
    const timeA = a.added ? new Date(a.added).getTime() : 0;
    const timeB = b.added ? new Date(b.added).getTime() : 0;
    return timeB - timeA;
  });

  // Render open workspaces
  sorted.forEach(workspace => {
    const item = document.createElement('div');
    item.className = 'workspace-menu-item';
    if (workspaceState.currentWorkspace && workspace.path === workspaceState.currentWorkspace.path) {
      item.classList.add('active');
    }

    const icon = document.createElement('i');
    icon.setAttribute('data-lucide', 'folder');
    item.appendChild(icon);

    const nameDiv = document.createElement('div');
    nameDiv.className = 'workspace-menu-item-name';
    const name = workspace.name || workspace.path.split('/').filter(Boolean).pop() || workspace.path;
    nameDiv.textContent = name;
    nameDiv.title = workspace.path;
    item.appendChild(nameDiv);

    item.addEventListener('click', () => {
      if (workspace.path !== workspaceState.currentWorkspace?.path) {
        switchWorkspace(workspace.path);
      }
      hideWorkspaceDropdown();
    });

    ui.workspaceMenuOpen.appendChild(item);
  });

  // Render recent workspaces (closed ones)
  const history = (workspaceState.recentWorkspaces || []).filter(
    entry => entry && !workspaceState.workspaces.some(ws => ws.path === entry.path)
  );

  if (history.length > 0) {
    history.slice(0, 5).forEach(workspace => {
      const item = document.createElement('div');
      item.className = 'workspace-menu-item';

      const icon = document.createElement('i');
      icon.setAttribute('data-lucide', 'folder');
      item.appendChild(icon);

      const contentDiv = document.createElement('div');
      contentDiv.style.flex = '1';
      contentDiv.style.overflow = 'hidden';

      const nameDiv = document.createElement('div');
      nameDiv.className = 'workspace-menu-item-name';
      const name = workspace.name || workspace.path.split('/').filter(Boolean).pop() || workspace.path;
      nameDiv.textContent = name;
      contentDiv.appendChild(nameDiv);

      const pathDiv = document.createElement('div');
      pathDiv.className = 'workspace-menu-item-path';
      pathDiv.textContent = workspace.path;
      pathDiv.title = workspace.path;
      contentDiv.appendChild(pathDiv);

      item.appendChild(contentDiv);

      item.addEventListener('click', () => {
        reopenWorkspace(workspace.path);
        hideWorkspaceDropdown();
      });

      ui.workspaceMenuRecent.appendChild(item);
    });
  } else {
    // Show "No recent workspaces" message
    const empty = document.createElement('div');
    empty.style.padding = '0.65rem 1rem';
    empty.style.fontSize = '0.85rem';
    empty.style.color = 'var(--muted)';
    empty.style.fontStyle = 'italic';
    empty.textContent = 'No recent workspaces';
    ui.workspaceMenuRecent.appendChild(empty);
  }

  // Reinitialize Lucide icons
  if (window.lucide) {
    lucide.createIcons();
  }
}

function updateSessionPickerLabel() {
  const label = document.getElementById('currentSessionLabel');
  if (!label) return;

  if (appState.data && appState.data.current_key) {
    label.textContent = appState.data.current_key;
  } else {
    label.textContent = 'Session';
  }
}

async function switchWorkspace(path) {
  try {
    // Update localStorage with new workspace
    setCurrentWorkspacePath(path);

    // Update backend's current workspace tracking
    await fetch('/api/workspace/switch', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ path }),
    });

    // Refresh session with new workspace context
    await refreshSession();
  } catch (err) {
    console.error('Switch workspace error:', err);
    alert('Failed to switch workspace: ' + err.message);
  }
}

async function reopenWorkspace(path) {
  try {
    const res = await fetch('/api/workspace/add', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ path }),
    });
    if (!res.ok) {
      const error = await res.text();
      throw new Error(error || 'Failed to reopen workspace');
    }
    await switchWorkspace(path);
  } catch (err) {
    console.error('Reopen workspace error:', err);
    alert('Failed to reopen workspace: ' + err.message);
  }
}

function showFolderPicker() {
  const dialog = document.getElementById('folderPickerDialog');
  const folderList = document.getElementById('folderList');
  const breadcrumb = document.getElementById('folderBreadcrumb');
  const selectedFolderDiv = document.getElementById('selectedFolder');
  const errorDiv = document.getElementById('folderPickerError');
  const confirmBtn = document.getElementById('confirmAddWorkspace');
  const cancelBtn = document.getElementById('cancelFolderPicker');
  const closeBtn = document.getElementById('closeFolderPicker');

  if (!dialog) return;

  let currentPath = '';
  let selectedPath = '';

  // Reset
  if (errorDiv) errorDiv.style.display = 'none';
  if (selectedFolderDiv) selectedFolderDiv.textContent = '—';
  selectedPath = '';

  // Show dialog
  dialog.style.display = 'flex';

  // Load folder contents
  const loadFolder = async (path = '') => {
    if (!folderList) return;

    folderList.innerHTML = '<div class="loading">Loading...</div>';

    try {
      const url = path ? `/api/browse?path=${encodeURIComponent(path)}` : '/api/browse';
      const res = await fetchWithWorkspace(url);
      if (!res.ok) {
        const errorText = await res.text();
        throw new Error(errorText || 'Failed to browse directory');
      }

      const data = await res.json();
      currentPath = data.current;

      // Update breadcrumb
      renderBreadcrumb(data.current, data.parent);

      // Render folder list
      renderFolderList(data.directories, data.parent);

    } catch (err) {
      console.error('Browse error:', err);
      if (folderList) {
        const errorMsg = err.message || 'Failed to load directory';
        folderList.innerHTML = `<div class="error-message">${errorMsg}</div>`;
      }
    }
  };

  // Render breadcrumb navigation
  const renderBreadcrumb = (current, parent) => {
    if (!breadcrumb) return;

    const parts = current.split('/').filter(p => p);
    let path = '';

    breadcrumb.innerHTML = '';

    // Root
    const root = document.createElement('span');
    root.className = 'breadcrumb-item';
    root.textContent = '/';
    root.addEventListener('click', () => loadFolder('/'));
    breadcrumb.appendChild(root);

    // Path parts
    parts.forEach((part, idx) => {
      path += '/' + part;
      const pathCopy = path;

      const sep = document.createElement('span');
      sep.className = 'breadcrumb-sep';
      sep.textContent = '/';
      breadcrumb.appendChild(sep);

      const item = document.createElement('span');
      item.className = 'breadcrumb-item';
      item.textContent = part;
      if (idx < parts.length - 1) {
        item.addEventListener('click', () => loadFolder(pathCopy));
      } else {
        item.classList.add('current');
      }
      breadcrumb.appendChild(item);
    });
  };

  // Render folder list
  const renderFolderList = (directories, parent) => {
    if (!folderList) return;

    folderList.innerHTML = '';

    // Parent directory (..)
    if (parent && parent !== currentPath) {
      const parentItem = document.createElement('div');
      parentItem.className = 'folder-item parent';
      parentItem.innerHTML = '<i data-lucide="corner-up-left"></i><span>..</span>';
      parentItem.addEventListener('click', () => loadFolder(parent));
      folderList.appendChild(parentItem);
    }

    // Current directory (select this folder)
    const currentItem = document.createElement('div');
    currentItem.className = 'folder-item current-dir';
    currentItem.innerHTML = '<i data-lucide="folder"></i><span>. (Select this folder)</span>';
    currentItem.addEventListener('click', () => {
      selectedPath = currentPath;
      if (selectedFolderDiv) {
        selectedFolderDiv.textContent = currentPath;
      }
      if (errorDiv) errorDiv.style.display = 'none';
    });
    folderList.appendChild(currentItem);

    // Subdirectories (handle null/undefined as empty array)
    const dirs = directories || [];
    dirs.forEach(dir => {
      const item = document.createElement('div');
      item.className = 'folder-item';
      item.innerHTML = `<i data-lucide="folder"></i><span>${dir.name}</span>`;
      item.addEventListener('click', () => loadFolder(dir.path));
      folderList.appendChild(item);
    });

    // Reinitialize Lucide icons
    if (window.lucide) {
      lucide.createIcons();
    }
  };

  // Initial load - determine default path
  // Priority: current workspace (backend handles) → recent workspace → home (backend handles)
  let defaultPath = '';
  if (!workspaceState.currentWorkspace && workspaceState.recentWorkspaces.length > 0) {
    defaultPath = workspaceState.recentWorkspaces[0].path;
  }
  loadFolder(defaultPath);

  // Event handlers
  const handleConfirm = async () => {
    if (!selectedPath) {
      if (errorDiv) {
        errorDiv.textContent = 'Please select a folder';
        errorDiv.style.display = 'block';
      }
      return;
    }

    try {
      const res = await fetch('/api/workspace/add', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ path: selectedPath })
      });

      if (!res.ok) {
        const text = await res.text();
        throw new Error(text || 'Failed to add workspace');
      }

      const data = await res.json();

      // Refresh workspace list
      await refreshSession();

      // Close dialog
      dialog.style.display = 'none';
      cleanup();

      // Switch to new workspace
      if (data.workspace) {
        await switchWorkspace(data.workspace.path);
      }
    } catch (err) {
      console.error('Add workspace error:', err);
      if (errorDiv) {
        errorDiv.textContent = 'Failed to add workspace: ' + err.message;
        errorDiv.style.display = 'block';
      }
    }
  };

  const handleCancel = () => {
    dialog.style.display = 'none';
    cleanup();
  };

  const cleanup = () => {
    if (confirmBtn) confirmBtn.removeEventListener('click', handleConfirm);
    if (cancelBtn) cancelBtn.removeEventListener('click', handleCancel);
    if (closeBtn) closeBtn.removeEventListener('click', handleCancel);
  };

  if (confirmBtn) confirmBtn.addEventListener('click', handleConfirm);
  if (cancelBtn) cancelBtn.addEventListener('click', handleCancel);
  if (closeBtn) closeBtn.addEventListener('click', handleCancel);
}

function showCloseWorkspaceDialog(workspace) {
  const dialog = document.getElementById('closeWorkspaceDialog');
  const nameSpan = document.getElementById('closeWorkspaceName');
  const confirmBtn = document.getElementById('confirmCloseWorkspace');
  const cancelBtn1 = document.getElementById('cancelCloseWorkspaceBtn');
  const cancelBtn2 = document.getElementById('cancelCloseWorkspace');

  if (!dialog) return;

  // Set workspace name
  if (nameSpan) {
    nameSpan.textContent = workspace.name || workspace.path;
  }

  // Show dialog
  dialog.style.display = 'flex';

  const handleConfirm = async () => {
    try {
      const res = await fetch('/api/workspace/remove', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ path: workspace.path })
      });

      if (!res.ok) {
        const text = await res.text();
        throw new Error(text || 'Failed to remove workspace');
      }

      // Refresh session (backend auto-switches to new current)
      await refreshSession();

      // Close dialog
      dialog.style.display = 'none';
      cleanup();
    } catch (err) {
      console.error('Remove workspace error:', err);
      alert('Failed to close workspace: ' + err.message);
      dialog.style.display = 'none';
      cleanup();
    }
  };

  const handleCancel = () => {
    dialog.style.display = 'none';
    cleanup();
  };

  const cleanup = () => {
    if (confirmBtn) confirmBtn.removeEventListener('click', handleConfirm);
    if (cancelBtn1) cancelBtn1.removeEventListener('click', handleCancel);
    if (cancelBtn2) cancelBtn2.removeEventListener('click', handleCancel);
  };

  if (confirmBtn) confirmBtn.addEventListener('click', handleConfirm);
  if (cancelBtn1) cancelBtn1.addEventListener('click', handleCancel);
  if (cancelBtn2) cancelBtn2.addEventListener('click', handleCancel);
}

// ========== SESSIONS DIALOG ==========

function showSessionsDialog() {
  const dialog = document.getElementById('sessionsDialog');
  const closeBtn = document.getElementById('closeSessionsDialog');
  const newSessionBtn = document.getElementById('newSessionBtn');
  const clearBtn = document.getElementById('clearSessionBtn');
  const cancelBtn = document.getElementById('cancelSessionsDialog');

  if (!dialog) return;

  // Show dialog
  dialog.style.display = 'flex';

  // Load session data
  loadSessionsDialogData();

  // Event handlers
  const handleClose = () => {
    dialog.style.display = 'none';
    cleanup();
  };

  const cleanup = () => {
    if (closeBtn) closeBtn.removeEventListener('click', handleClose);
    if (cancelBtn) cancelBtn.removeEventListener('click', handleClose);
    if (newSessionBtn) newSessionBtn.removeEventListener('click', createRandomSession);
    if (clearBtn) clearBtn.removeEventListener('click', clearCurrentSession);
  };

  if (closeBtn) closeBtn.addEventListener('click', handleClose);
  if (cancelBtn) cancelBtn.addEventListener('click', handleClose);
  if (newSessionBtn) newSessionBtn.addEventListener('click', createRandomSession);
  if (clearBtn) clearBtn.addEventListener('click', clearCurrentSession);
}

// For backwards compatibility with sessions.tmpl if still used
window.initSessionsPage = async function() {
  const backBtn = document.getElementById('backToChat');
  if (backBtn) {
    backBtn.addEventListener('click', () => {
      window.location.href = '/';
    });
  }
  await loadSessionsPageData();
  const newSessionBtn = document.getElementById('newSessionBtn');
  if (newSessionBtn) {
    newSessionBtn.addEventListener('click', createRandomSession);
  }
  const clearBtn = document.getElementById('clearSessionBtn');
  if (clearBtn) {
    clearBtn.addEventListener('click', clearCurrentSession);
  }
};

async function loadSessionsDialogData() {
  try {
    const res = await fetchWithWorkspace('/api/session');
    if (!res.ok) throw new Error('Failed to load session data');

    const data = await res.json();

    // Update current session
    const currentNameEl = document.getElementById('currentSessionNameDisplay');
    const currentMessagesEl = document.getElementById('currentSessionMessages');
    const currentUpdatedEl = document.getElementById('currentSessionUpdated');

    if (currentNameEl) currentNameEl.textContent = data.current_key || '—';
    if (currentMessagesEl) {
      const count = data.messages?.length || 0;
      currentMessagesEl.textContent = `${count} message${count !== 1 ? 's' : ''}`;
    }
    if (currentUpdatedEl) {
      // Find current session in summaries for updated time
      const current = data.sessions?.find(s => s.key === data.current_key);
      if (current && current.updated) {
        const date = new Date(current.updated);
        currentUpdatedEl.textContent = `Updated: ${date.toLocaleString()}`;
      } else {
        currentUpdatedEl.textContent = '—';
      }
    }

    // Render all sessions
    renderSessionsList(data.sessions || [], data.current_key);
  } catch (err) {
    console.error('Load sessions error:', err);
  }
}

async function loadSessionsPageData() {
  try {
    const res = await fetchWithWorkspace('/api/session');
    if (!res.ok) throw new Error('Failed to load session data');

    const data = await res.json();

    // Update workspace name (for sessions page)
    const workspaceLabel = document.getElementById('sessionWorkspaceName');
    if (workspaceLabel && data.workspace) {
      workspaceLabel.textContent = data.workspace.name || data.workspace.path.split('/').pop();
    }

    // Update current session
    const currentNameEl = document.getElementById('currentSessionNameDisplay');
    const currentMessagesEl = document.getElementById('currentSessionMessages');
    const currentUpdatedEl = document.getElementById('currentSessionUpdated');

    if (currentNameEl) currentNameEl.textContent = data.current_key || '—';
    if (currentMessagesEl) {
      const count = data.messages?.length || 0;
      currentMessagesEl.textContent = `${count} message${count !== 1 ? 's' : ''}`;
    }
    if (currentUpdatedEl) {
      // Find current session in summaries for updated time
      const current = data.sessions?.find(s => s.key === data.current_key);
      if (current && current.updated) {
        const date = new Date(current.updated);
        currentUpdatedEl.textContent = `Updated: ${date.toLocaleString()}`;
      } else {
        currentUpdatedEl.textContent = '—';
      }
    }

    // Render all sessions
    renderSessionsList(data.sessions || [], data.current_key);
  } catch (err) {
    console.error('Load sessions error:', err);
  }
}

function renderSessionsList(sessions, currentKey) {
  const container = document.getElementById('sessionListContainer');
  if (!container) return;

  container.innerHTML = '';

  // Filter out current session from list
  const otherSessions = sessions.filter(s => s.key !== currentKey);

  if (otherSessions.length === 0) {
    container.innerHTML = '<p class="help-text">No other sessions. Create a new session to get started.</p>';
    return;
  }

  otherSessions.forEach(session => {
    const card = document.createElement('div');
    card.className = 'session-card';

    const info = document.createElement('div');
    info.className = 'session-info';

    const name = document.createElement('div');
    name.className = 'session-name';
    name.textContent = session.key;

    const meta = document.createElement('div');
    meta.className = 'session-meta';

    const messagesSpan = document.createElement('span');
    messagesSpan.textContent = `${session.message_count || 0} messages`;

    const updatedSpan = document.createElement('span');
    if (session.updated) {
      const date = new Date(session.updated);
      updatedSpan.textContent = `Updated: ${date.toLocaleString()}`;
    }

    meta.appendChild(messagesSpan);
    if (session.updated) meta.appendChild(updatedSpan);

    info.appendChild(name);
    info.appendChild(meta);

    const actions = document.createElement('div');
    actions.className = 'session-actions';

    const deleteBtn = document.createElement('button');
    deleteBtn.className = 'ghost danger';
    deleteBtn.innerHTML = '<i data-lucide="trash-2"></i>';
    deleteBtn.title = 'Delete session';
    deleteBtn.addEventListener('click', (e) => {
      e.stopPropagation();
      deleteSession(session.key);
    });

    actions.appendChild(deleteBtn);

    card.appendChild(info);
    card.appendChild(actions);

    // Click to switch
    card.addEventListener('click', () => {
      switchSession(session.key);
    });

    container.appendChild(card);
  });

  // Reinitialize Lucide icons
  if (window.lucide) {
    lucide.createIcons();
  }
}

async function createNewSession() {
  const key = prompt('Enter new session name:');
  if (!key || !key.trim()) return;

  try {
    const res = await fetchWithWorkspace('/api/state', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ action: 'new', key: key.trim() })
    });

    if (!res.ok) {
      const text = await res.text();
      throw new Error(text || 'Failed to create session');
    }

    // Reload session data
    await refreshSession();
    await loadSessionsPageData();
    loadSessionsDialogData();
  } catch (err) {
    console.error('Create session error:', err);
    alert('Failed to create session: ' + err.message);
  }
}

async function switchSession(key) {
  try {
    const res = await fetchWithWorkspace('/api/state', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ action: 'switch', key })
    });

    if (!res.ok) {
      const text = await res.text();
      throw new Error(text || 'Failed to switch session');
    }

    // Navigate back to chat
    window.location.href = '/';
  } catch (err) {
    console.error('Switch session error:', err);
    alert('Failed to switch session: ' + err.message);
  }
}

async function clearCurrentSession() {
  if (!confirm('Clear all messages in current session?')) return;

  try {
    const res = await fetchWithWorkspace('/api/state', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ action: 'clear' })
    });

    if (!res.ok) {
      const text = await res.text();
      throw new Error(text || 'Failed to clear session');
    }

    // Reload session data
    await refreshSession();
    await loadSessionsPageData();
    loadSessionsDialogData();
  } catch (err) {
    console.error('Clear session error:', err);
    alert('Failed to clear session: ' + err.message);
  }
}

async function deleteSession(key) {
  if (!confirm(`Delete session "${key}"? This cannot be undone.`)) return;

  try {
    const res = await fetchWithWorkspace('/api/state', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ action: 'delete', key })
    });

    if (!res.ok) {
      const text = await res.text();
      throw new Error(text || 'Failed to delete session');
    }

    // Reload session data
    await refreshSession();
    await loadSessionsPageData();
    loadSessionsDialogData();
  } catch (err) {
    console.error('Delete session error:', err);
    alert('Failed to delete session: ' + err.message);
  }
}
