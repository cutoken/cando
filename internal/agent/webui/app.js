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
  // Project UI elements (formerly workspace)
  projectPickerBtn: null,
  projectMenu: null,
  projectMenuList: null,
  currentProjectName: null,
  newProjectMenuBtn: null,
  openFolderMenuBtn: null,
  projectSettingsMenuBtn: null,
  // Chat UI elements (formerly session)
  chatPickerBtn: null,
  chatMenu: null,
  chatMenuList: null,
  currentChatLabel: null,
  newChatBtn: null,
  helpBtn: null,
  analyticsToggle: null,
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
  // Project UI elements
  ui.projectPickerBtn = document.getElementById('projectPickerBtn');
  ui.projectMenu = document.getElementById('projectMenu');
  ui.projectMenuList = document.getElementById('projectMenuList');
  ui.currentProjectName = document.getElementById('currentProjectName');
  ui.newProjectMenuBtn = document.getElementById('newProjectMenuBtn');
  ui.openFolderMenuBtn = document.getElementById('openFolderMenuBtn');
  ui.projectSettingsMenuBtn = document.getElementById('projectSettingsMenuBtn');
  // Chat UI elements
  ui.chatPickerBtn = document.getElementById('chatPickerBtn');
  ui.chatMenu = document.getElementById('chatMenu');
  ui.chatMenuList = document.getElementById('chatMenuList');
  ui.currentChatLabel = document.getElementById('currentChatLabel');
  ui.newChatBtn = document.getElementById('newChatBtn');
  ui.helpBtn = document.getElementById('helpBtn');
  ui.analyticsToggle = document.getElementById('analyticsToggle');

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
  if (ui.analyticsToggle) {
    ui.analyticsToggle.addEventListener('change', toggleAnalytics);
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

  // Project dropdown in toolbar
  if (ui.projectPickerBtn) {
    ui.projectPickerBtn.addEventListener('click', (e) => {
      e.stopPropagation();
      toggleProjectDropdown();
    });
  }

  // Close project dropdown when clicking outside
  document.addEventListener('click', (e) => {
    if (!ui.projectMenu || ui.projectMenu.classList.contains('hidden')) return;
    if (!ui.projectMenu.contains(e.target) && !ui.projectPickerBtn.contains(e.target)) {
      hideProjectDropdown();
    }
  });

  // New Project button in dropdown menu
  if (ui.newProjectMenuBtn) {
    ui.newProjectMenuBtn.addEventListener('click', () => {
      hideProjectDropdown();
      showNewProjectDialog();
    });
  }

  // Open Folder button in dropdown menu
  if (ui.openFolderMenuBtn) {
    ui.openFolderMenuBtn.addEventListener('click', () => {
      hideProjectDropdown();
      showFolderPicker();
    });
  }

  // Project Settings button in dropdown menu
  if (ui.projectSettingsMenuBtn) {
    ui.projectSettingsMenuBtn.addEventListener('click', () => {
      hideProjectDropdown();
      showProjectSettingsDialog();
    });
  }

  // Chat button - switches to chat view (no dropdown)
  if (ui.chatPickerBtn) {
    ui.chatPickerBtn.addEventListener('click', (e) => {
      e.stopPropagation();
      switchToChatView();
    });
  }

  // Chat dropdown button - opens chat switcher
  const chatDropdownBtn = document.getElementById('chatDropdownBtn');
  if (chatDropdownBtn) {
    chatDropdownBtn.addEventListener('click', (e) => {
      e.stopPropagation();
      toggleChatDropdown();
    });
  }

  // Close chat dropdown when clicking outside
  document.addEventListener('click', (e) => {
    if (!ui.chatMenu || ui.chatMenu.classList.contains('hidden')) return;
    const chatDropdownBtn = document.getElementById('chatDropdownBtn');
    if (!ui.chatMenu.contains(e.target) &&
        !ui.chatPickerBtn?.contains(e.target) &&
        !chatDropdownBtn?.contains(e.target)) {
      hideChatDropdown();
    }
  });

  // New Chat button in toolbar
  if (ui.newChatBtn) {
    ui.newChatBtn.addEventListener('click', () => {
      createNewChat();
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

  const providerSelect = document.getElementById('providerSelect');
  if (providerSelect) {
    providerSelect.addEventListener('change', (e) => {
      const provider = e.target.value;
      if (provider === 'zai') {
        ui.apiKeyHelp.innerHTML = 'Get your Z.AI key at: <a href="https://z.ai" target="_blank">z.ai</a>';
      } else if (provider === 'openrouter') {
        ui.apiKeyHelp.innerHTML = 'Get your OpenRouter key at: <a href="https://openrouter.ai/keys" target="_blank">openrouter.ai/keys</a>';
      }
    });
  }

  // Check credentials on load
  await checkCredentials();

  await refreshSession();

  // Initialize additional components
  initSettings();
  initAutocomplete();
  initFileDragDrop();
  initProjects();
  initUpdateChecker();
  initFileExplorer();
  sendTelemetry();
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

    // Refresh file explorer
    if (typeof loadFileTree === 'function') {
      loadFileTree();
    }
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
  updateProjectUI();
  updateChatUI();
  const hasProject = !!appState.data.workspace;
  // Show/hide chat controls based on project selection
  if (ui.chatPickerBtn) {
    ui.chatPickerBtn.classList.toggle('hidden', !hasProject);
  }
  if (ui.newChatBtn) {
    ui.newChatBtn.classList.toggle('hidden', !hasProject);
  }
  if (ui.promptInput) {
    if (hasProject) {
      ui.promptInput.disabled = false;
      ui.promptInput.readOnly = !!appState.busy;
      ui.promptInput.placeholder = 'Ask Cando anything… (Enter to send, Shift+Enter for new line)';
    } else {
      ui.promptInput.value = '';
      ui.promptInput.disabled = true;
      ui.promptInput.readOnly = true;
      ui.promptInput.placeholder = 'Select a project to get started';
    }
  }
  if (ui.sendBtn && !appState.busy) {
    ui.sendBtn.disabled = !hasProject;
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
  ui.cancelBtn.disabled = !appState.data.running || !hasProject;
}

function renderMessages() {
  ui.messages.innerHTML = '';
  if (!appState.data?.workspace) {
    renderProjectEmptyState();
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

function renderProjectEmptyState() {
  ui.messages.innerHTML = getProjectGuideHTML();
  wireProjectGuideActions(ui.messages);
}

function getProjectGuideHTML() {
  return `
    <div class="project-empty">
      <div class="project-empty-card project-empty-minimal">
        <div class="project-empty-icon">📂</div>
        <h2>Open or create a project to start chatting</h2>
        <div class="project-empty-actions">
          <button class="primary" data-help-action="new-project">New Project</button>
          <button class="ghost" data-help-action="select-project">Open Project</button>
        </div>
      </div>
    </div>
  `;
}

function getHelpGuideHTML() {
  return `
    <div class="help-guide">
      <div class="help-section">
        <h3>Toolbar</h3>
        <div class="help-row">
          <span class="help-icon-inline">📁</span>
          <span class="help-label">Project ▼</span>
          <span class="help-desc">Switch projects, open settings</span>
        </div>
        <div class="help-row">
          <span class="help-icon-inline">💬</span>
          <span class="help-label">Chat ▼ | +</span>
          <span class="help-desc">Switch or create chats</span>
        </div>
        <div class="help-row">
          <span class="help-icon-inline">⚙</span>
          <span class="help-label">Settings</span>
          <span class="help-desc">API keys, models, preferences</span>
        </div>
      </div>
      <div class="help-section">
        <h3>How It Works</h3>
        <ul class="help-list">
          <li><strong>Project</strong> = a folder on your computer</li>
          <li><strong>Chat</strong> = a conversation within a project</li>
          <li>Each project keeps its own chat history</li>
          <li><strong>Project Settings</strong> = per-project instructions sent with every message</li>
        </ul>
      </div>
      <div class="help-section">
        <h3>Shortcuts</h3>
        <div class="help-row">
          <span class="help-key">Enter</span>
          <span class="help-desc">Send message</span>
        </div>
        <div class="help-row">
          <span class="help-key">Shift + Enter</span>
          <span class="help-desc">New line</span>
        </div>
      </div>
    </div>
  `;
}

function wireProjectGuideActions(root, afterAction) {
  if (!root) return;
  const selectBtn = root.querySelector('[data-help-action="select-project"]');
  if (selectBtn) {
    selectBtn.addEventListener('click', () => {
      showFolderPicker();
      if (typeof afterAction === 'function') afterAction();
    });
  }
  const newProjectBtn = root.querySelector('[data-help-action="new-project"]');
  if (newProjectBtn) {
    newProjectBtn.addEventListener('click', () => {
      showNewProjectDialog();
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
  const providerSelect = document.getElementById('providerSelect');
  const provider = providerSelect ? providerSelect.value : '';
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
    // Configure renderer to open links in new tab
    const renderer = new marked.Renderer();
    // marked 4.x+ passes object {href, title, text}, older versions pass separate params
    renderer.link = function(hrefOrObj, title, text) {
      let href, linkTitle, linkText;
      if (typeof hrefOrObj === 'object' && hrefOrObj !== null) {
        // New API (marked 4.x+)
        href = hrefOrObj.href;
        linkTitle = hrefOrObj.title;
        linkText = hrefOrObj.text;
      } else {
        // Old API
        href = hrefOrObj;
        linkTitle = title;
        linkText = text;
      }
      const titleAttr = linkTitle ? ` title="${linkTitle}"` : '';
      return `<a href="${href}"${titleAttr} target="_blank" rel="noopener noreferrer">${linkText}</a>`;
    };
    const html = window.marked.parse(source, { breaks: true, renderer: renderer });
    if (window.DOMPurify) {
      return window.DOMPurify.sanitize(html, { ADD_ATTR: ['target'] });
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

async function toggleAnalytics() {
  if (!ui.analyticsToggle) return;
  const enabled = ui.analyticsToggle.checked;
  const res = await fetch('/api/config', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ analytics_enabled: enabled }),
  });
  if (!res.ok) {
    setStatus('Analytics toggle failed');
    ui.analyticsToggle.checked = !enabled;
    return;
  }
  appState.data = await res.json();
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

// Sequential chat name generator (chat-1, chat-2, etc.)
function generateChatName() {
  const sessions = appState.data?.sessions || [];
  // Find highest existing chat number
  let maxNum = 0;
  sessions.forEach(s => {
    const match = s.key.match(/^chat-(\d+)$/);
    if (match) {
      const num = parseInt(match[1], 10);
      if (num > maxNum) maxNum = num;
    }
  });
  return `chat-${maxNum + 1}`;
}

// Settings dialog management
let settingsDialog, closeSettingsBtn, settingsBtn;
let chatList, currentChatName, newChatDialogBtn, clearChatBtn, deleteChatBtn;
let tabBtns = [];
let helpDialog, helpDialogContent, helpBtn, closeHelpBtn;

function initSettings() {
  settingsDialog = document.getElementById('settingsDialog');
  closeSettingsBtn = document.getElementById('closeSettingsDialog');
  settingsBtn = document.getElementById('settingsBtn');
  chatList = document.getElementById('chatListContainer');
  currentChatName = document.getElementById('currentChatNameDisplay');
  newChatDialogBtn = document.getElementById('newChatDialogBtn');
  clearChatBtn = document.getElementById('clearChatBtn');
  deleteChatBtn = document.getElementById('deleteChatBtn');
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

  // Chat actions (in chats dialog)
  if (newChatDialogBtn) {
    newChatDialogBtn.addEventListener('click', createNewChat);
  }
  if (clearChatBtn) {
    clearChatBtn.addEventListener('click', clearState);
  }
  if (deleteChatBtn) {
    deleteChatBtn.addEventListener('click', deleteState);
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

  // Check for updates button
  const checkForUpdatesBtn = document.getElementById('checkForUpdatesBtn');
  if (checkForUpdatesBtn) {
    checkForUpdatesBtn.addEventListener('click', manualCheckForUpdates);
  }

  initHelpDialog();
}

async function openSettingsDialog() {
  if (settingsDialog) {
    settingsDialog.style.display = 'flex';
    refreshChatList();
    refreshCompactionInfo();
    loadApiKeyStatus();
    await loadOpenRouterModels();
    updateProviderStatus();
    initializeProviderAccordions();
    populateSystemPrompt();
    populateAnalyticsToggle();
  }
}

function populateAnalyticsToggle() {
  if (!ui.analyticsToggle || !appState.data) return;
  ui.analyticsToggle.checked = appState.data.analytics_enabled !== false;
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
  helpDialogContent.innerHTML = getHelpGuideHTML();
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

async function createNewChat() {
  const name = generateChatName();
  try {
    const res = await fetchWithWorkspace('/api/state', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ action: 'new', key: name }),
    });
    if (res.ok) {
      await refreshSession();
      refreshChatList();
      // Refresh chats dialog if open
      loadChatsDialogData();
    }
  } catch (err) {
    console.error('Create chat failed:', err);
  }
}

function refreshChatList() {
  if (!chatList || !appState.data) return;

  const sessions = appState.data.sessions || [];
  const currentKey = appState.data.current_key || '';

  if (currentChatName) {
    currentChatName.textContent = currentKey;
  }

  chatList.innerHTML = '';

  sessions.forEach(session => {
    const item = document.createElement('div');
    item.className = 'chat-item';
    if (session.key === currentKey) {
      item.classList.add('active');
    }

    const name = document.createElement('div');
    name.className = 'chat-item-name';
    name.textContent = session.key;

    const meta = document.createElement('div');
    meta.className = 'chat-item-meta';
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
          refreshChatList();
        }
      } catch (err) {
        console.error('Switch chat failed:', err);
      }
    });

    chatList.appendChild(item);
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
let modelSearchSetupDone = { main: false, summary: false, vision: false };

// Consolidated model configuration - single source of truth
const MODEL_CONFIG = {
  main: {
    configKey: 'provider_models',
    elementPrefix: 'openrouterModel',      // openrouterModelSelect, openrouterModelSearch, etc.
    defaults: { zai: 'glm-4.6', openrouter: 'deepseek/deepseek-chat-v3-0324' },
    filter: null  // no capability filter
  },
  summary: {
    configKey: 'provider_summary_models',
    elementPrefix: 'openrouterSummaryModel',
    defaults: { zai: 'glm-4.5-air', openrouter: 'qwen/qwen3-30b-a3b-instruct-2507' },
    filter: null
  },
  vision: {
    configKey: 'provider_vl_models',
    elementPrefix: 'openrouterVisionModel',
    defaults: { zai: 'glm-4.5v', openrouter: 'qwen/qwen2.5-vl-32b-instruct' },
    filter: (model) => model.capabilities && model.capabilities.includes('image')
  }
};

// Get DOM elements for a model type
function getModelElements(modelType) {
  const cfg = MODEL_CONFIG[modelType];
  const prefix = cfg.elementPrefix;
  return {
    hidden: document.getElementById(prefix === 'openrouterModel' ? 'openrouterModelSelect' : prefix),
    search: document.getElementById(prefix + 'Search'),
    dropdown: document.getElementById(prefix + 'Dropdown'),
    info: document.getElementById(prefix + 'Info')
  };
}

// Display model info as a compact single line (only for paid models - free is already in name)
function displayModelInfo(model, infoElement) {
  if (!model || !infoElement) return;

  const isFree = model.pricing && model.pricing.prompt === "0" && model.pricing.completion === "0";

  if (isFree) {
    // Free is already shown in model name, no need to repeat
    infoElement.style.display = 'none';
  } else if (model.pricing) {
    const promptPrice = model.pricing.prompt ? (parseFloat(model.pricing.prompt) * 1000000).toFixed(2) : '—';
    const completionPrice = model.pricing.completion ? (parseFloat(model.pricing.completion) * 1000000).toFixed(2) : '—';
    infoElement.textContent = `$${promptPrice}/$${completionPrice} per 1M tokens (in/out)`;
    infoElement.style.display = 'inline';
  } else {
    infoElement.style.display = 'none';
  }
}

// Get current model from config (works even before provider is active)
function getCurrentModel(provider, modelType) {
  const cfg = MODEL_CONFIG[modelType];
  const configModels = appState.data?.[cfg.configKey] || {};
  const providers = appState.data?.providers || [];
  const activeProvider = providers.find(p => p.key === provider);

  // Prefer active provider's model, fall back to config, then default
  return activeProvider?.model || configModels[provider] || cfg.defaults[provider];
}

// Check if model is already saved (guard against unnecessary saves)
function isModelAlreadySaved(provider, modelType, modelId) {
  const cfg = MODEL_CONFIG[modelType];
  const configModels = appState.data?.[cfg.configKey] || {};
  return configModels[provider] === modelId;
}

// Save provider model (generic for all model types)
async function saveProviderModelGeneric(provider, modelType, modelId) {
  if (!modelId) return;

  // Guard: don't save if already the same
  if (isModelAlreadySaved(provider, modelType, modelId)) {
    console.log(`${modelType} model ${modelId} already configured for ${provider}, skipping save`);
    return;
  }

  try {
    setStatus(`Saving ${provider} ${modelType} model…`);
    const res = await fetch('/api/provider/model', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ provider, model_type: modelType, model: modelId })
    });

    if (res.ok) {
      setStatus(`Model updated: ${modelId}`);

      // Show saved indicator
      const elements = getModelElements(modelType);
      const indicatorId = elements.hidden?.id + 'Saved';
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

// Select model (display + save) - called when user picks from dropdown
function selectProviderModel(provider, modelType, model) {
  const elements = getModelElements(modelType);
  if (!elements.hidden || !elements.search) return;

  elements.hidden.value = model.id;
  elements.search.value = model.name;
  if (modelType === 'main') selectedOpenRouterModel = model;

  displayModelInfo(model, elements.info);
  elements.dropdown.style.display = 'none';

  saveProviderModelGeneric(provider, modelType, model.id);
}

// Populate model display from config (no save triggered)
function populateModelFromConfig(provider, modelType) {
  const elements = getModelElements(modelType);
  if (!elements.hidden) return;

  const modelId = getCurrentModel(provider, modelType);
  if (!modelId) return;

  elements.hidden.value = modelId;

  const model = openrouterModels.find(m => m.id === modelId);
  if (model && elements.search) {
    elements.search.value = model.name;
    if (modelType === 'main') selectedOpenRouterModel = model;
    displayModelInfo(model, elements.info);
  }
}

// Filter and show models in dropdown
function filterAndShowModelDropdown(modelType, query) {
  const cfg = MODEL_CONFIG[modelType];
  const elements = getModelElements(modelType);
  if (!elements.dropdown || !elements.search) return;

  const lowerQuery = query.toLowerCase();

  let filtered = openrouterModels.filter(model => {
    // Apply capability filter if defined (e.g., vision models)
    if (cfg.filter && !cfg.filter(model)) return false;

    // Text search filter
    const matchesSearch = model.name.toLowerCase().includes(lowerQuery) ||
      (model.id && model.id.toLowerCase().includes(lowerQuery));

    return matchesSearch;
  });

  elements.dropdown.innerHTML = '';

  if (filtered.length === 0) {
    const empty = document.createElement('div');
    empty.className = 'model-dropdown-item';
    empty.textContent = 'No models found';
    empty.style.color = '#888';
    elements.dropdown.appendChild(empty);
  } else {
    filtered.slice(0, 50).forEach(model => {
      const item = document.createElement('div');
      item.className = 'model-dropdown-item';

      const isFree = model.pricing && model.pricing.prompt === "0" && model.pricing.completion === "0";

      item.innerHTML = `
        <div class="model-dropdown-name">${model.name}${isFree ? ' <span style="color: #10b981; font-size: 0.75rem;">FREE</span>' : ''}</div>
        <div class="model-dropdown-id">${model.id}</div>
      `;
      item.addEventListener('click', () => selectProviderModel('openrouter', modelType, model));
      elements.dropdown.appendChild(item);
    });

    if (filtered.length > 50) {
      const more = document.createElement('div');
      more.className = 'model-dropdown-item';
      more.textContent = `... and ${filtered.length - 50} more. Type to filter.`;
      more.style.color = '#888';
      more.style.fontStyle = 'italic';
      elements.dropdown.appendChild(more);
    }
  }

  // Position dropdown below the search input
  const inputRect = elements.search.getBoundingClientRect();
  elements.dropdown.style.top = (inputRect.bottom + 4) + 'px';
  elements.dropdown.style.left = inputRect.left + 'px';
  elements.dropdown.style.width = Math.max(inputRect.width, 400) + 'px';
  elements.dropdown.style.display = 'block';
}

// Setup model search for a model type (generic)
function setupModelSearchGeneric(modelType) {
  if (modelSearchSetupDone[modelType]) return;

  const elements = getModelElements(modelType);
  if (!elements.search || !elements.dropdown) return;

  modelSearchSetupDone[modelType] = true;

  elements.search.addEventListener('focus', () => {
    filterAndShowModelDropdown(modelType, elements.search.value);
  });

  elements.search.addEventListener('input', (e) => {
    filterAndShowModelDropdown(modelType, e.target.value);
  });

  document.addEventListener('click', (e) => {
    if (!elements.search.contains(e.target) && !elements.dropdown.contains(e.target)) {
      elements.dropdown.style.display = 'none';
    }
  });
}

// Load OpenRouter models and setup searchable dropdowns
async function loadOpenRouterModels() {
  try {
    const res = await fetch('/openrouter-models.json');
    if (!res.ok) throw new Error('Failed to load models');
    openrouterModels = await res.json();

    // Setup all model type searches
    setupModelSearchGeneric('main');
    setupModelSearchGeneric('summary');
    setupModelSearchGeneric('vision');

    // Setup Free Mode
    setupFreeMode();
  } catch (err) {
    console.error('Failed to load OpenRouter models:', err);
    openrouterModels = [];
  }
}

// Find the top free model matching a filter (models are already sorted by popularity)
function findTopFreeModel(filter = null) {
  return openrouterModels.find(model => {
    const isFree = model.pricing && model.pricing.prompt === "0" && model.pricing.completion === "0";
    if (!isFree) return false;
    if (filter && !filter(model)) return false;
    return true;
  });
}

// Apply Free Mode - auto-select top free models for all categories
function applyFreeMode() {
  // Find top free text model (for main and summary)
  const topFreeText = findTopFreeModel();
  // Find top free vision model
  const topFreeVision = findTopFreeModel(model => model.capabilities && model.capabilities.includes('image'));

  if (topFreeText) {
    // Select for main model
    selectProviderModel('openrouter', 'main', topFreeText);
    // Select for summary model
    selectProviderModel('openrouter', 'summary', topFreeText);
  }

  if (topFreeVision) {
    selectProviderModel('openrouter', 'vision', topFreeVision);
  } else if (topFreeText) {
    // Fallback to text model if no vision model available
    selectProviderModel('openrouter', 'vision', topFreeText);
  }
}

// Save Free Mode state to config
async function saveFreeMode(enabled) {
  try {
    await fetch('/api/config', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ openrouter_free_mode: enabled })
    });
  } catch (err) {
    console.error('Failed to save free mode:', err);
  }
}

// Setup Free Mode toggle
function setupFreeMode() {
  const freeModeToggle = document.getElementById('openrouterFreeMode');
  if (!freeModeToggle) return;

  // Load saved state
  const freeModeEnabled = appState.data?.openrouter_free_mode || false;
  freeModeToggle.checked = freeModeEnabled;

  // If free mode is enabled on load, apply it
  if (freeModeEnabled) {
    applyFreeMode();
  }

  // Handle toggle changes
  freeModeToggle.addEventListener('change', async () => {
    const enabled = freeModeToggle.checked;
    await saveFreeMode(enabled);

    if (enabled) {
      applyFreeMode();
    }
    // When disabled, keep current selections (don't revert)
  });
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

  // Set Z.AI model selections
  if (zaiProvider) {
    const zaiSelect = document.getElementById('zaiModelSelect');
    if (zaiSelect && zaiProvider.model) {
      zaiSelect.value = zaiProvider.model;
    }
    const zaiSummarySelect = document.getElementById('zaiSummaryModelSelect');
    if (zaiSummarySelect) {
      const summaryModels = appState.data?.provider_summary_models || {};
      zaiSummarySelect.value = summaryModels['zai'] || 'glm-4.5-air';
    }
  }

  // Set OpenRouter model selections (works even before API key is set)
  populateModelFromConfig('openrouter', 'main');
  populateModelFromConfig('openrouter', 'summary');
  populateModelFromConfig('openrouter', 'vision');
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
        displayModelInfo(model, openrouterVisionInfo);
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

// ========== FILE DRAG-DROP & ATTACH ==========

function initFileDragDrop() {
  if (!ui.promptInput) return;

  const textarea = ui.promptInput;
  const wrapper = textarea.closest('.input-wrapper');

  // Prevent default drag behaviors on wrapper
  ['dragenter', 'dragover', 'dragleave', 'drop'].forEach(eventName => {
    wrapper.addEventListener(eventName, (e) => {
      e.preventDefault();
      e.stopPropagation();
    });
  });

  // Visual feedback
  ['dragenter', 'dragover'].forEach(eventName => {
    wrapper.addEventListener(eventName, () => {
      wrapper.classList.add('drag-over');
    });
  });

  ['dragleave', 'drop'].forEach(eventName => {
    wrapper.addEventListener(eventName, () => {
      wrapper.classList.remove('drag-over');
    });
  });

  // Handle drop
  wrapper.addEventListener('drop', (e) => {
    const files = e.dataTransfer.files;
    if (files.length > 0) {
      insertFilePaths(files);
    }
  });

  // Attach button - open custom file browser instead of native picker
  const attachBtn = document.getElementById('attachBtn');
  if (attachBtn) {
    attachBtn.addEventListener('click', () => {
      showFileBrowser();
    });
  }
}

function insertFilePaths(files) {
  const textarea = ui.promptInput;
  if (!textarea) return;

  // Build paths string
  const paths = Array.from(files).map(f => `"${f.path || f.name}"`).join(' ');

  // Insert at cursor position
  const start = textarea.selectionStart;
  const end = textarea.selectionEnd;
  const text = textarea.value;

  // Add space before if needed
  const before = text.substring(0, start);
  const after = text.substring(end);
  const needsSpaceBefore = before.length > 0 && !/\s$/.test(before);
  const needsSpaceAfter = after.length > 0 && !/^\s/.test(after);

  const insert = (needsSpaceBefore ? ' ' : '') + paths + (needsSpaceAfter ? ' ' : '');

  textarea.value = before + insert + after;
  textarea.selectionStart = textarea.selectionEnd = start + insert.length;
  textarea.focus();
}

function insertFilePathStrings(paths) {
  const textarea = ui.promptInput;
  if (!textarea || !paths.length) return;

  const pathsStr = paths.map(p => `"${p}"`).join(' ');

  const start = textarea.selectionStart;
  const end = textarea.selectionEnd;
  const text = textarea.value;

  const before = text.substring(0, start);
  const after = text.substring(end);
  const needsSpaceBefore = before.length > 0 && !/\s$/.test(before);
  const needsSpaceAfter = after.length > 0 && !/^\s/.test(after);

  const insert = (needsSpaceBefore ? ' ' : '') + pathsStr + (needsSpaceAfter ? ' ' : '');

  textarea.value = before + insert + after;
  textarea.selectionStart = textarea.selectionEnd = start + insert.length;
  textarea.focus();
}

function showFileBrowser() {
  const dialog = document.getElementById('fileBrowserDialog');
  const fileList = document.getElementById('fileBrowserList');
  const breadcrumb = document.getElementById('fileBrowserBreadcrumb');
  const selectedDisplay = document.getElementById('selectedFilesDisplay');
  const confirmBtn = document.getElementById('confirmFileBrowser');
  const cancelBtn = document.getElementById('cancelFileBrowser');
  const closeBtn = document.getElementById('closeFileBrowser');

  if (!dialog) return;

  let currentPath = getCurrentWorkspacePath() || '';
  let selectedFiles = [];

  // Reset display
  if (selectedDisplay) selectedDisplay.textContent = 'None';

  // Show dialog
  dialog.style.display = 'flex';

  // Load folder contents with files
  const loadFolder = async (path = '') => {
    if (!fileList) return;

    fileList.innerHTML = '<div class="loading">Loading...</div>';

    try {
      let url = '/api/browse?includeFiles=true';
      if (path) {
        url += `&path=${encodeURIComponent(path)}`;
      }
      const res = await fetchWithWorkspace(url);
      if (!res.ok) {
        const errorText = await res.text();
        throw new Error(errorText || 'Failed to browse directory');
      }

      const data = await res.json();
      currentPath = data.current;

      // Update breadcrumb
      renderBreadcrumb(data.current, data.parent);

      // Render file list
      renderFileList(data.directories, data.files, data.parent);

    } catch (err) {
      console.error('Browse error:', err);
      if (fileList) {
        const errorMsg = err.message || 'Failed to load directory';
        fileList.innerHTML = `<div class="error-message">${errorMsg}</div>`;
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

  // Render file list
  const renderFileList = (directories, files, parent) => {
    if (!fileList) return;

    fileList.innerHTML = '';

    // Parent directory (..)
    if (parent && parent !== currentPath) {
      const parentItem = document.createElement('div');
      parentItem.className = 'file-browser-item parent';
      parentItem.innerHTML = '<i data-lucide="corner-up-left"></i><span>..</span>';
      parentItem.addEventListener('click', () => loadFolder(parent));
      fileList.appendChild(parentItem);
    }

    // Directories
    const dirs = directories || [];
    dirs.forEach(dir => {
      const item = document.createElement('div');
      item.className = 'file-browser-item folder';
      item.innerHTML = `<i data-lucide="folder"></i><span>${dir.name}</span>`;
      item.addEventListener('click', () => loadFolder(dir.path));
      fileList.appendChild(item);
    });

    // Files
    const fileEntries = files || [];
    if (fileEntries.length === 0 && dirs.length === 0) {
      const emptyItem = document.createElement('div');
      emptyItem.className = 'file-browser-empty';
      emptyItem.textContent = 'Empty folder';
      fileList.appendChild(emptyItem);
    } else {
      fileEntries.forEach(file => {
        const item = document.createElement('div');
        const isSelected = selectedFiles.includes(file.path);
        item.className = 'file-browser-item file' + (isSelected ? ' selected' : '');
        item.innerHTML = `<i data-lucide="file"></i><span>${file.name}</span>`;
        item.addEventListener('click', () => {
          toggleFileSelection(file.path);
          item.classList.toggle('selected');
        });
        fileList.appendChild(item);
      });
    }

    // Reinitialize Lucide icons
    if (window.lucide) {
      lucide.createIcons();
    }
  };

  const toggleFileSelection = (path) => {
    const idx = selectedFiles.indexOf(path);
    if (idx === -1) {
      selectedFiles.push(path);
    } else {
      selectedFiles.splice(idx, 1);
    }
    updateSelectedDisplay();
  };

  const updateSelectedDisplay = () => {
    if (!selectedDisplay) return;
    if (selectedFiles.length === 0) {
      selectedDisplay.textContent = 'None';
    } else {
      selectedDisplay.textContent = selectedFiles.map(p => p.split('/').pop()).join(', ');
    }
  };

  // Event handlers
  const handleConfirm = () => {
    if (selectedFiles.length > 0) {
      insertFilePathStrings(selectedFiles);
    }
    closeDialog();
  };

  const closeDialog = () => {
    dialog.style.display = 'none';
    // Remove event listeners
    if (confirmBtn) confirmBtn.removeEventListener('click', handleConfirm);
    if (cancelBtn) cancelBtn.removeEventListener('click', closeDialog);
    if (closeBtn) closeBtn.removeEventListener('click', closeDialog);
  };

  // Attach event listeners
  if (confirmBtn) confirmBtn.addEventListener('click', handleConfirm);
  if (cancelBtn) cancelBtn.addEventListener('click', closeDialog);
  if (closeBtn) closeBtn.addEventListener('click', closeDialog);

  // Start at workspace folder
  loadFolder(currentPath);
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

async function initProjects() {
  // Initialize project/chat data from session payload
  updateProjectUI();
  updateChatUI();
}

function updateProjectUI() {
  if (!appState.data) return;

  workspaceState.workspaces = appState.data.workspaces || [];
  workspaceState.currentWorkspace = appState.data.workspace || null;
  workspaceState.recentWorkspaces = appState.data.recent_workspaces || [];

  // Update project name in toolbar
  if (ui.currentProjectName && workspaceState.currentWorkspace) {
    const path = workspaceState.currentWorkspace.path || '';
    const name = workspaceState.currentWorkspace.name || path.split('/').pop() || 'Project';
    ui.currentProjectName.textContent = name;
    ui.currentProjectName.title = path;
  } else if (ui.currentProjectName) {
    ui.currentProjectName.textContent = 'No Project';
    ui.currentProjectName.title = '';
  }

  // Render project dropdown menu
  renderProjectMenu();
}

function updateChatUI() {
  if (!appState.data) return;

  const currentKey = appState.data.current_key || '';

  // Update chat label in toolbar
  if (ui.currentChatLabel) {
    ui.currentChatLabel.textContent = currentKey || 'chat-1';
  }

  // Render chat dropdown menu
  renderChatMenu();
}

function toggleProjectDropdown() {
  if (ui.projectMenu) {
    ui.projectMenu.classList.toggle('hidden');
    if (!ui.projectMenu.classList.contains('hidden')) {
      renderProjectMenu();
    }
  }
}

function hideProjectDropdown() {
  if (ui.projectMenu) {
    ui.projectMenu.classList.add('hidden');
  }
}

function toggleChatDropdown() {
  if (ui.chatMenu) {
    ui.chatMenu.classList.toggle('hidden');
    if (!ui.chatMenu.classList.contains('hidden')) {
      renderChatMenu();
    }
  }
}

function hideChatDropdown() {
  if (ui.chatMenu) {
    ui.chatMenu.classList.add('hidden');
  }
}

function renderProjectMenu() {
  if (!ui.projectMenuList) return;

  ui.projectMenuList.innerHTML = '';

  // Combine open and recent projects, deduped by path
  const openProjects = workspaceState.workspaces || [];
  const recentProjects = workspaceState.recentWorkspaces || [];

  const seenPaths = new Set();
  const allProjects = [];

  // Add open projects first
  openProjects.forEach(p => {
    if (!seenPaths.has(p.path)) {
      seenPaths.add(p.path);
      allProjects.push(p);
    }
  });

  // Add recent projects that aren't already in the list
  recentProjects.forEach(p => {
    if (!seenPaths.has(p.path)) {
      seenPaths.add(p.path);
      allProjects.push(p);
    }
  });

  if (allProjects.length === 0) {
    ui.projectMenuList.innerHTML = '<div class="project-menu-empty">No projects yet</div>';
  } else {
    allProjects.forEach(project => {
      const item = document.createElement('div');
      item.className = 'project-menu-item';
      if (workspaceState.currentWorkspace && project.path === workspaceState.currentWorkspace.path) {
        item.classList.add('active');
      }

      const name = project.name || project.path.split('/').pop() || project.path;
      item.innerHTML = `
        <div class="project-item-info">
          <div class="project-item-name">${name}</div>
          <div class="project-item-path">${project.path}</div>
        </div>
      `;

      item.addEventListener('click', () => {
        if (project.path !== workspaceState.currentWorkspace?.path) {
          switchWorkspace(project.path);
        }
        hideProjectDropdown();
      });

      ui.projectMenuList.appendChild(item);
    });
  }
}

function renderChatMenu() {
  if (!ui.chatMenuList) return;

  ui.chatMenuList.innerHTML = '';
  const sessions = appState.data?.sessions || [];
  const currentKey = appState.data?.current_key || '';

  if (sessions.length === 0) {
    ui.chatMenuList.innerHTML = '<div class="chat-menu-empty">No chats yet</div>';
    return;
  }

  sessions.forEach(session => {
    const item = document.createElement('div');
    item.className = 'chat-menu-item';
    if (session.key === currentKey) {
      item.classList.add('current');
    }

    item.innerHTML = `
      <span class="chat-menu-item-name">${session.key}</span>
      <span class="chat-menu-item-meta">${session.message_count || 0} messages</span>
    `;

    item.addEventListener('click', async () => {
      if (session.key !== currentKey) {
        await switchChat(session.key);
      }
      hideChatDropdown();
    });

    ui.chatMenuList.appendChild(item);
  });
}

async function switchChat(key) {
  try {
    const res = await fetchWithWorkspace('/api/state', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ action: 'switch', key }),
    });
    if (res.ok) {
      await refreshSession();
    }
  } catch (err) {
    console.error('Switch chat failed:', err);
  }
}

async function reopenProject(path) {
  try {
    const res = await fetch('/api/workspace/add', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ path }),
    });
    if (!res.ok) {
      const error = await res.text();
      throw new Error(error || 'Failed to reopen project');
    }
    await switchWorkspace(path);
  } catch (err) {
    console.error('Reopen project error:', err);
    alert('Failed to reopen project: ' + err.message);
  }
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
    // Close open editor tabs before switching
    if (typeof closeAllTabs === 'function') {
      closeAllTabs();
    }

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
  showFolderPickerWithCallback(null);
}

function showFolderPickerWithCallback(callback) {
  const dialog = document.getElementById('folderPickerDialog');
  const folderList = document.getElementById('folderList');
  const breadcrumb = document.getElementById('folderBreadcrumb');
  const selectedFolderDiv = document.getElementById('selectedFolder');
  const errorDiv = document.getElementById('folderPickerError');
  const confirmBtn = document.getElementById('confirmAddWorkspace');
  const cancelBtn = document.getElementById('cancelFolderPicker');
  const closeBtn = document.getElementById('closeFolderPicker');
  const newFolderBtn = document.getElementById('createNewFolderBtn');

  if (!dialog) return;

  let currentPath = '';

  // Reset
  if (errorDiv) errorDiv.style.display = 'none';
  if (selectedFolderDiv) selectedFolderDiv.textContent = '—';

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

      // Update selected folder display to show current folder
      if (selectedFolderDiv) {
        selectedFolderDiv.textContent = currentPath;
      }

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

    // Subdirectories (handle null/undefined as empty array)
    const dirs = directories || [];
    if (dirs.length === 0) {
      const emptyItem = document.createElement('div');
      emptyItem.className = 'folder-empty-message';
      emptyItem.textContent = 'No subfolders';
      folderList.appendChild(emptyItem);
    } else {
      dirs.forEach(dir => {
        const item = document.createElement('div');
        item.className = 'folder-item';
        item.innerHTML = `<i data-lucide="folder"></i><span>${dir.name}</span>`;
        item.addEventListener('click', () => loadFolder(dir.path));
        folderList.appendChild(item);
      });
    }

    // Reinitialize Lucide icons
    if (window.lucide) {
      lucide.createIcons();
    }
  };

  // Handle new folder creation
  const handleNewFolder = async () => {
    const name = prompt('Enter new folder name:');
    if (!name || !name.trim()) return;

    // Validate name
    const trimmedName = name.trim();
    if (!/^[a-zA-Z0-9_.-]+$/.test(trimmedName)) {
      alert('Folder name can only contain letters, numbers, dots, hyphens, and underscores');
      return;
    }

    const newPath = `${currentPath}/${trimmedName}`;

    try {
      const res = await fetch('/api/folder/create', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ path: newPath })
      });

      if (!res.ok) {
        const text = await res.text();
        throw new Error(text || 'Failed to create folder');
      }

      // Reload current folder to show new folder
      await loadFolder(currentPath);
    } catch (err) {
      console.error('Create folder error:', err);
      alert('Failed to create folder: ' + err.message);
    }
  };

  // Initial load - determine default path
  let defaultPath = '';
  if (!workspaceState.currentWorkspace && workspaceState.recentWorkspaces.length > 0) {
    defaultPath = workspaceState.recentWorkspaces[0].path;
  }
  loadFolder(defaultPath);

  // Event handlers
  const handleConfirm = async () => {
    // Use currentPath directly - no need to explicitly select "."
    if (!currentPath) {
      if (errorDiv) {
        errorDiv.textContent = 'Please navigate to a folder';
        errorDiv.style.display = 'block';
      }
      return;
    }

    // If callback provided, use it instead of adding workspace
    if (callback) {
      callback(currentPath);
      dialog.style.display = 'none';
      cleanup();
      return;
    }

    try {
      const res = await fetch('/api/workspace/add', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ path: currentPath })
      });

      if (!res.ok) {
        const text = await res.text();
        throw new Error(text || 'Failed to add project');
      }

      const data = await res.json();

      // Refresh project list
      await refreshSession();

      // Close dialog
      dialog.style.display = 'none';
      cleanup();

      // Switch to new project
      if (data.workspace) {
        await switchWorkspace(data.workspace.path);
      }
    } catch (err) {
      console.error('Add project error:', err);
      if (errorDiv) {
        errorDiv.textContent = 'Failed to add project: ' + err.message;
        errorDiv.style.display = 'block';
      }
    }
  };

  const handleCancel = () => {
    dialog.style.display = 'none';
    cleanup();
    // Call callback with null to signal cancellation
    if (callback) {
      callback(null);
    }
  };

  const cleanup = () => {
    if (confirmBtn) confirmBtn.removeEventListener('click', handleConfirm);
    if (cancelBtn) cancelBtn.removeEventListener('click', handleCancel);
    if (closeBtn) closeBtn.removeEventListener('click', handleCancel);
    if (newFolderBtn) newFolderBtn.removeEventListener('click', handleNewFolder);
  };

  if (confirmBtn) confirmBtn.addEventListener('click', handleConfirm);
  if (cancelBtn) cancelBtn.addEventListener('click', handleCancel);
  if (closeBtn) closeBtn.addEventListener('click', handleCancel);
  if (newFolderBtn) newFolderBtn.addEventListener('click', handleNewFolder);
}

function showCloseProjectDialog(project) {
  const dialog = document.getElementById('closeProjectDialog');
  const nameSpan = document.getElementById('closeProjectName');
  const confirmBtn = document.getElementById('confirmCloseProject');
  const cancelBtn1 = document.getElementById('cancelCloseProjectBtn');
  const cancelBtn2 = document.getElementById('cancelCloseProject');

  if (!dialog) return;

  // Set project name
  if (nameSpan) {
    nameSpan.textContent = project.name || project.path.split('/').pop() || project.path;
  }

  // Show dialog
  dialog.style.display = 'flex';

  const handleConfirm = async () => {
    try {
      const res = await fetch('/api/workspace/remove', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ path: project.path })
      });

      if (!res.ok) {
        const text = await res.text();
        throw new Error(text || 'Failed to remove project');
      }

      // Refresh session (backend auto-switches to new current)
      await refreshSession();

      // Close dialog
      dialog.style.display = 'none';
      cleanup();
    } catch (err) {
      console.error('Remove project error:', err);
      alert('Failed to close project: ' + err.message);
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

// ========== CHATS DIALOG ==========

function showChatsDialog() {
  const dialog = document.getElementById('chatsDialog');
  const closeBtn = document.getElementById('closeChatsDialog');
  const newChatBtn = document.getElementById('newChatDialogBtn');
  const clearBtn = document.getElementById('clearChatBtn');
  const cancelBtn = document.getElementById('cancelChatsDialog');

  if (!dialog) return;

  // Show dialog
  dialog.style.display = 'flex';

  // Load chat data
  loadChatsDialogData();

  // Event handlers
  const handleClose = () => {
    dialog.style.display = 'none';
    cleanup();
  };

  const cleanup = () => {
    if (closeBtn) closeBtn.removeEventListener('click', handleClose);
    if (cancelBtn) cancelBtn.removeEventListener('click', handleClose);
    if (newChatBtn) newChatBtn.removeEventListener('click', createNewChat);
    if (clearBtn) clearBtn.removeEventListener('click', clearCurrentChat);
  };

  if (closeBtn) closeBtn.addEventListener('click', handleClose);
  if (cancelBtn) cancelBtn.addEventListener('click', handleClose);
  if (newChatBtn) newChatBtn.addEventListener('click', createNewChat);
  if (clearBtn) clearBtn.addEventListener('click', clearCurrentChat);
}

// For backwards compatibility
window.initChatsPage = async function() {
  const backBtn = document.getElementById('backToChat');
  if (backBtn) {
    backBtn.addEventListener('click', () => {
      window.location.href = '/';
    });
  }
  await loadChatsPageData();
  const newChatBtn = document.getElementById('newChatDialogBtn');
  if (newChatBtn) {
    newChatBtn.addEventListener('click', createNewChat);
  }
  const clearBtn = document.getElementById('clearChatBtn');
  if (clearBtn) {
    clearBtn.addEventListener('click', clearCurrentChat);
  }
};

async function loadChatsDialogData() {
  try {
    const res = await fetchWithWorkspace('/api/session');
    if (!res.ok) throw new Error('Failed to load chat data');

    const data = await res.json();

    // Update current chat
    const currentNameEl = document.getElementById('currentChatNameDisplay');
    const currentMessagesEl = document.getElementById('currentChatMessages');
    const currentUpdatedEl = document.getElementById('currentChatUpdated');

    if (currentNameEl) currentNameEl.textContent = data.current_key || '—';
    if (currentMessagesEl) {
      const count = data.messages?.length || 0;
      currentMessagesEl.textContent = `${count} message${count !== 1 ? 's' : ''}`;
    }
    if (currentUpdatedEl) {
      // Find current chat in summaries for updated time
      const current = data.sessions?.find(s => s.key === data.current_key);
      if (current && current.updated) {
        const date = new Date(current.updated);
        currentUpdatedEl.textContent = `Updated: ${date.toLocaleString()}`;
      } else {
        currentUpdatedEl.textContent = '—';
      }
    }

    // Render all chats
    renderChatsList(data.sessions || [], data.current_key);
  } catch (err) {
    console.error('Load chats error:', err);
  }
}

function clearCurrentChat() {
  clearState();
}

function renderChatsList(sessions, currentKey) {
  const container = document.getElementById('chatListContainer');
  if (!container) return;

  container.innerHTML = '';

  // Filter out current chat from list
  const otherChats = sessions.filter(s => s.key !== currentKey);

  if (otherChats.length === 0) {
    container.innerHTML = '<p class="help-text">No other chats. Create a new chat to get started.</p>';
    return;
  }

  otherChats.forEach(chat => {
    const card = document.createElement('div');
    card.className = 'chat-card';

    const info = document.createElement('div');
    info.className = 'chat-info';

    const name = document.createElement('div');
    name.className = 'chat-name';
    name.textContent = chat.key;

    const meta = document.createElement('div');
    meta.className = 'chat-meta';

    const messagesSpan = document.createElement('span');
    messagesSpan.textContent = `${chat.message_count || 0} messages`;

    const updatedSpan = document.createElement('span');
    if (chat.updated) {
      const date = new Date(chat.updated);
      updatedSpan.textContent = `Updated: ${date.toLocaleString()}`;
    }

    meta.appendChild(messagesSpan);
    if (chat.updated) meta.appendChild(updatedSpan);

    info.appendChild(name);
    info.appendChild(meta);

    const actions = document.createElement('div');
    actions.className = 'chat-actions';

    const deleteBtn = document.createElement('button');
    deleteBtn.className = 'ghost danger';
    deleteBtn.innerHTML = '<i data-lucide="trash-2"></i>';
    deleteBtn.title = 'Delete chat';
    deleteBtn.addEventListener('click', (e) => {
      e.stopPropagation();
      deleteChat(chat.key);
    });

    actions.appendChild(deleteBtn);

    card.appendChild(info);
    card.appendChild(actions);

    // Click to switch
    card.addEventListener('click', () => {
      switchChat(chat.key);
    });

    container.appendChild(card);
  });

  // Reinitialize Lucide icons
  if (window.lucide) {
    lucide.createIcons();
  }
}

async function deleteChat(key) {
  if (!confirm(`Delete chat "${key}" permanently?`)) return;

  try {
    const res = await fetchWithWorkspace('/api/state', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ action: 'delete', key })
    });

    if (!res.ok) {
      const text = await res.text();
      throw new Error(text || 'Failed to delete chat');
    }

    await refreshSession();
    loadChatsDialogData();
  } catch (err) {
    console.error('Delete chat error:', err);
    alert('Failed to delete chat: ' + err.message);
  }
}

async function loadChatsPageData() {
  // Wrapper for backwards compatibility
  return loadChatsDialogData();
}

// ========== NEW PROJECT DIALOG ==========

let newProjectParentPath = '';

function showNewProjectDialog() {
  const dialog = document.getElementById('newProjectDialog');
  const nameInput = document.getElementById('newProjectName');
  const pathDisplay = document.getElementById('newProjectPath');
  const changeBtn = document.getElementById('changeProjectLocation');
  const confirmBtn = document.getElementById('confirmNewProject');
  const cancelBtn = document.getElementById('cancelNewProject');
  const closeBtn = document.getElementById('closeNewProjectDialog');
  const errorDiv = document.getElementById('newProjectError');

  if (!dialog) return;

  // Reset
  newProjectParentPath = '';
  if (nameInput) nameInput.value = '';
  if (pathDisplay) pathDisplay.textContent = '';
  if (errorDiv) errorDiv.style.display = 'none';

  // Get user home directory from current workspace or default
  fetch('/api/browse')
    .then(res => res.json())
    .then(data => {
      // Try to determine home from current path
      const current = data.current || '';
      const homeParts = current.split('/').slice(0, 3);
      newProjectParentPath = homeParts.join('/') || '/home';
      updateProjectPathDisplay();
    })
    .catch(() => {
      newProjectParentPath = '/home';
      updateProjectPathDisplay();
    });

  // Show dialog
  dialog.style.display = 'flex';
  if (nameInput) nameInput.focus();

  const updateProjectPathDisplay = () => {
    if (!pathDisplay || !nameInput) return;
    const name = nameInput.value.trim() || 'my-project';
    pathDisplay.textContent = `${newProjectParentPath}/${name}`;
  };

  const handleNameInput = () => {
    updateProjectPathDisplay();
    if (errorDiv) errorDiv.style.display = 'none';
  };

  const handleChangeLocation = () => {
    // Show folder picker in "select parent" mode
    showFolderPickerForNewProject((selectedPath) => {
      newProjectParentPath = selectedPath;
      updateProjectPathDisplay();
    });
  };

  const handleConfirm = async () => {
    const name = nameInput?.value.trim();
    if (!name) {
      if (errorDiv) {
        errorDiv.textContent = 'Please enter a project name';
        errorDiv.style.display = 'block';
      }
      return;
    }

    // Validate name (no special chars except hyphen/underscore)
    if (!/^[a-zA-Z0-9_-]+$/.test(name)) {
      if (errorDiv) {
        errorDiv.textContent = 'Project name can only contain letters, numbers, hyphens, and underscores';
        errorDiv.style.display = 'block';
      }
      return;
    }

    const fullPath = `${newProjectParentPath}/${name}`;

    try {
      // Create folder via API
      const createRes = await fetch('/api/folder/create', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ path: fullPath })
      });

      if (!createRes.ok) {
        const text = await createRes.text();
        throw new Error(text || 'Failed to create folder');
      }

      // Add as workspace
      const addRes = await fetch('/api/workspace/add', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ path: fullPath })
      });

      if (!addRes.ok) {
        const text = await addRes.text();
        throw new Error(text || 'Failed to add project');
      }

      // Refresh and switch
      await refreshSession();
      await switchWorkspace(fullPath);

      // Close dialog
      dialog.style.display = 'none';
      cleanup();
    } catch (err) {
      console.error('Create project error:', err);
      if (errorDiv) {
        errorDiv.textContent = err.message || 'Failed to create project';
        errorDiv.style.display = 'block';
      }
    }
  };

  const handleCancel = () => {
    dialog.style.display = 'none';
    cleanup();
  };

  const cleanup = () => {
    if (nameInput) nameInput.removeEventListener('input', handleNameInput);
    if (changeBtn) changeBtn.removeEventListener('click', handleChangeLocation);
    if (confirmBtn) confirmBtn.removeEventListener('click', handleConfirm);
    if (cancelBtn) cancelBtn.removeEventListener('click', handleCancel);
    if (closeBtn) closeBtn.removeEventListener('click', handleCancel);
  };

  if (nameInput) nameInput.addEventListener('input', handleNameInput);
  if (changeBtn) changeBtn.addEventListener('click', handleChangeLocation);
  if (confirmBtn) confirmBtn.addEventListener('click', handleConfirm);
  if (cancelBtn) cancelBtn.addEventListener('click', handleCancel);
  if (closeBtn) closeBtn.addEventListener('click', handleCancel);
}

function showFolderPickerForNewProject(callback) {
  // Hide the new project dialog while folder picker is open
  const newProjectDialog = document.getElementById('newProjectDialog');
  if (newProjectDialog) {
    newProjectDialog.style.display = 'none';
  }

  showFolderPickerWithCallback((selectedPath) => {
    // Show the new project dialog again
    if (newProjectDialog) {
      newProjectDialog.style.display = 'flex';
    }
    if (selectedPath) {
      callback(selectedPath);
    }
  });
}

// ========== PROJECT SETTINGS DIALOG ==========

async function showProjectSettingsDialog() {
  const dialog = document.getElementById('projectSettingsDialog');
  const instructionsInput = document.getElementById('projectInstructionsInput');
  const saveBtn = document.getElementById('saveProjectSettings');
  const cancelBtn = document.getElementById('cancelProjectSettings');
  const closeBtn = document.getElementById('closeProjectSettings');

  if (!dialog || !instructionsInput) return;

  // Check if we have a project selected
  if (!workspaceState.currentWorkspace) {
    alert('Please select a project first');
    return;
  }

  // Load current instructions
  try {
    const res = await fetchWithWorkspace('/api/project/instructions');
    if (res.ok) {
      const data = await res.json();
      instructionsInput.value = data.instructions || '';
    }
  } catch (err) {
    console.error('Failed to load project instructions:', err);
  }

  // Show dialog
  dialog.style.display = 'flex';

  // Handle save
  const handleSave = async () => {
    try {
      const res = await fetchWithWorkspace('/api/project/instructions', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ instructions: instructionsInput.value }),
      });
      if (res.ok) {
        closeDialog();
      } else {
        const errorText = await res.text();
        alert('Failed to save: ' + errorText);
      }
    } catch (err) {
      console.error('Failed to save project instructions:', err);
      alert('Failed to save project instructions');
    }
  };

  const closeDialog = () => {
    dialog.style.display = 'none';
    if (saveBtn) saveBtn.removeEventListener('click', handleSave);
    if (cancelBtn) cancelBtn.removeEventListener('click', closeDialog);
    if (closeBtn) closeBtn.removeEventListener('click', closeDialog);
  };

  // Attach event listeners
  if (saveBtn) saveBtn.addEventListener('click', handleSave);
  if (cancelBtn) cancelBtn.addEventListener('click', closeDialog);
  if (closeBtn) closeBtn.addEventListener('click', closeDialog);

  // Handle tab switching within project settings (for future tabs)
  const tabBtns = dialog.querySelectorAll('.tab-btn');
  tabBtns.forEach(btn => {
    btn.addEventListener('click', () => {
      const tabName = btn.dataset.tab;
      // Update active tab button
      tabBtns.forEach(b => b.classList.remove('active'));
      btn.classList.add('active');
      // Update active pane
      dialog.querySelectorAll('.tab-pane').forEach(pane => {
        pane.classList.toggle('active', pane.id === `tab-${tabName}`);
      });
    });
  });
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

// ============================================
// Update Checker
// ============================================

const updateUI = {};

function initUpdateChecker() {
  // Cache DOM elements
  updateUI.dialog = document.getElementById('updateDialog');
  updateUI.closeBtn = document.getElementById('closeUpdateDialog');
  updateUI.currentVersion = document.getElementById('updateCurrentVersion');
  updateUI.latestVersion = document.getElementById('updateLatestVersion');
  updateUI.updateNowBtn = document.getElementById('updateNowBtn');
  updateUI.dismissBtn = document.getElementById('updateDismissBtn');
  updateUI.restartBtn = document.getElementById('updateRestartBtn');
  updateUI.refreshBtn = document.getElementById('updateRefreshBtn');
  updateUI.copyBtn = document.getElementById('updateCopyBtn');
  updateUI.retryBtn = document.getElementById('updateRetryBtn');
  updateUI.closeErrorBtn = document.getElementById('updateCloseErrorBtn');
  updateUI.manualCommand = document.getElementById('updateManualCommand');
  updateUI.errorMessage = document.getElementById('updateErrorMessage');

  // Phases
  updateUI.checkPhase = document.getElementById('updateCheckPhase');
  updateUI.downloadPhase = document.getElementById('updateDownloadPhase');
  updateUI.readyPhase = document.getElementById('updateReadyPhase');
  updateUI.restartingPhase = document.getElementById('updateRestartingPhase');
  updateUI.manualPhase = document.getElementById('updateManualPhase');
  updateUI.errorPhase = document.getElementById('updateErrorPhase');

  if (!updateUI.dialog) return;

  // Event listeners
  updateUI.closeBtn.addEventListener('click', closeUpdateDialog);
  updateUI.dialog.addEventListener('click', (e) => {
    if (e.target === updateUI.dialog) closeUpdateDialog();
  });
  updateUI.updateNowBtn.addEventListener('click', startUpdate);
  updateUI.dismissBtn.addEventListener('click', dismissUpdate);
  updateUI.restartBtn.addEventListener('click', restartApp);
  updateUI.refreshBtn.addEventListener('click', () => window.location.reload());
  updateUI.copyBtn.addEventListener('click', copyUpdateCommand);
  updateUI.retryBtn.addEventListener('click', startUpdate);
  updateUI.closeErrorBtn.addEventListener('click', closeUpdateDialog);

  // Check for updates on load
  checkForUpdates();
}

async function checkForUpdates() {
  try {
    const res = await fetch('/api/update-check');
    if (!res.ok) return;

    const data = await res.json();
    if (data.isDev) {
      console.log('Running dev version, skipping update check');
      return;
    }

    if (data.updateAvailable && !data.dismissed) {
      showUpdateDialog(data.current, data.latest);
    }
  } catch (err) {
    console.error('Update check failed:', err);
  }
}

function showUpdateDialog(current, latest) {
  updateUI.currentVersion.textContent = current || '—';
  updateUI.latestVersion.textContent = latest || '—';
  showUpdatePhase('check');
  updateUI.dialog.style.display = 'flex';
}

function closeUpdateDialog() {
  updateUI.dialog.style.display = 'none';
}

function showUpdatePhase(phase) {
  const phases = ['check', 'download', 'ready', 'restarting', 'manual', 'error'];
  phases.forEach(p => {
    const el = updateUI[p + 'Phase'];
    if (el) el.style.display = p === phase ? 'block' : 'none';
  });
}

async function dismissUpdate() {
  try {
    await fetch('/api/update/dismiss', { method: 'POST' });
    closeUpdateDialog();
  } catch (err) {
    console.error('Dismiss failed:', err);
    closeUpdateDialog();
  }
}

async function startUpdate() {
  showUpdatePhase('download');

  try {
    const res = await fetch('/api/update', { method: 'POST' });

    // Handle non-JSON error responses
    const contentType = res.headers.get('content-type') || '';
    if (!res.ok) {
      if (contentType.includes('application/json')) {
        const data = await res.json();
        throw new Error(data.message || 'Update failed');
      } else {
        const text = await res.text();
        throw new Error(text || 'Update failed');
      }
    }

    const data = await res.json();

    if (data.needsManual) {
      updateUI.manualCommand.textContent = data.command;
      showUpdatePhase('manual');
      return;
    }

    if (data.success) {
      showUpdatePhase('ready');
    } else {
      throw new Error(data.message || 'Update failed');
    }
  } catch (err) {
    console.error('Update error:', err);
    updateUI.errorMessage.textContent = err.message;
    showUpdatePhase('error');
  }
}

async function restartApp() {
  showUpdatePhase('restarting');

  try {
    await fetch('/api/restart', { method: 'POST' });

    // Start polling for server to come back
    setTimeout(pollForRestart, 2000);
  } catch (err) {
    // Expected - server is restarting
    setTimeout(pollForRestart, 2000);
  }
}

async function pollForRestart() {
  const maxAttempts = 30;
  let attempts = 0;

  const poll = async () => {
    try {
      const res = await fetch('/api/health');
      if (res.ok) {
        // Server is back, reload page
        window.location.reload();
        return;
      }
    } catch (err) {
      // Server not ready yet
    }

    attempts++;
    if (attempts < maxAttempts) {
      setTimeout(poll, 1500);
    } else {
      // Give up, show refresh button
      console.log('Restart poll timeout, user can manually refresh');
    }
  };

  poll();
}

function copyUpdateCommand() {
  const command = updateUI.manualCommand.textContent;
  navigator.clipboard.writeText(command).then(() => {
    updateUI.copyBtn.textContent = 'Copied!';
    setTimeout(() => {
      updateUI.copyBtn.textContent = 'Copy Command';
    }, 2000);
  });
}

// Manual check for updates from settings
async function manualCheckForUpdates() {
  const btn = document.getElementById('checkForUpdatesBtn');
  const status = document.getElementById('updateCheckStatus');

  if (btn) btn.disabled = true;
  if (status) status.textContent = 'Checking...';

  try {
    const res = await fetch('/api/update-check?force=true');
    if (!res.ok) throw new Error('Check failed');

    const data = await res.json();

    if (data.isDev) {
      if (status) status.textContent = 'Running development version';
      return;
    }

    if (data.updateAvailable) {
      if (status) status.textContent = `Update available: ${data.latest}`;
      // Close settings and show update dialog
      closeSettingsDialog();
      showUpdateDialog(data.current, data.latest);
    } else {
      if (status) status.textContent = `You're up to date (${data.current})`;
    }
  } catch (err) {
    console.error('Update check failed:', err);
    if (status) status.textContent = 'Failed to check for updates';
  } finally {
    if (btn) btn.disabled = false;
  }
}

// Send telemetry with browser context
function sendTelemetry() {
  // Only send if telemetry is enabled
  if (appState.data && appState.data.analytics_enabled === false) {
    return;
  }

  const screenSize = `${window.screen.width}x${window.screen.height}`;
  const userAgent = navigator.userAgent;

  // Send to backend for server-side tracking
  fetch('/api/telemetry', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({
      user_agent: userAgent,
      screen_size: screenSize,
    }),
  }).catch(() => {
    // Silently ignore telemetry errors
  });

  // Dynamically load GoatCounter only if telemetry enabled
  if (!window.goatcounter) {
    window.goatcounter = {
      endpoint: 'https://cando.goatcounter.com/count',
      no_onload: true
    };
    const script = document.createElement('script');
    script.async = true;
    script.src = '//gc.zgo.at/count.js';
    script.onload = () => {
      if (window.goatcounter && window.goatcounter.count) {
        window.goatcounter.count({ path: '/app/view', event: true });
      }
    };
    document.body.appendChild(script);
  } else if (window.goatcounter.count) {
    window.goatcounter.count({ path: '/app/view', event: true });
  }
}

// Test function - call from browser console: testUpdateDialog()
window.testUpdateDialog = function(current = 'v1.0.0', latest = 'v1.1.0') {
  showUpdateDialog(current, latest);
};

// Simulate full update flow - call: testUpdateFlow()
window.testUpdateFlow = async function() {
  showUpdateDialog('v1.0.0', 'v1.1.0');

  // Wait for user to click Update Now, then simulate phases
  const originalStartUpdate = startUpdate;
  window.startUpdate = async function() {
    showUpdatePhase('download');
    await new Promise(r => setTimeout(r, 2000)); // Simulate download
    showUpdatePhase('ready');
  };

  // Restore after test
  setTimeout(() => { window.startUpdate = originalStartUpdate; }, 30000);
  console.log('Test mode active for 30s - click "Update Now" to simulate');
};

// Test manual update phase
window.testManualUpdate = function() {
  showUpdateDialog('v1.0.0', 'v1.1.0');
  updateUI.manualCommand.textContent = 'curl -fsSL https://raw.githubusercontent.com/cutoken/cando/main/install.sh | bash';
  showUpdatePhase('manual');
};

// Test restart UI flow (uses real /api/restart endpoint)
window.testRealRestart = function() {
  showUpdateDialog('v1.0.0', 'v1.1.0');
  showUpdatePhase('ready'); // Skip to ready phase
  console.log('Click "Restart Now" to trigger restart (uses real endpoint)');
};

// ============================================
// File Explorer & Editor
// ============================================

const fileExplorer = {
  sidebar: null,
  fileTree: null,
  // Split pane elements
  editorPane: null,
  chatPane: null,
  paneResizeHandle: null,
  editorTabsSection: null,
  editorTabs: null,
  editorContainer: null,
  // Sidebar controls
  collapseSidebarBtn: null,
  expandSidebarBtn: null,
  sidebarResizeHandle: null,
  // Pane controls
  collapseEditorBtn: null,
  expandEditorBtn: null,
  collapseChatBtn: null,
  expandChatBtn: null,
  // Editor state
  editor: null,
  openTabs: [],      // Array of { path, name, content, dirty, modTime, workspacePath }
  activeTabPath: null,
  saveTimeout: null,
  AUTOSAVE_DELAY: 1500,
  FILE_WATCH_INTERVAL: 3000,  // Check for external changes every 3s
  TREE_REFRESH_INTERVAL: 5000, // Refresh tree every 5s
  fileWatchTimer: null,
  treeRefreshTimer: null,
  isPaneResizing: false,
  isSidebarResizing: false,
};

function initFileExplorer() {
  // File sidebar elements
  fileExplorer.sidebar = document.getElementById('fileSidebar');
  fileExplorer.fileTree = document.getElementById('fileTree');
  fileExplorer.collapseSidebarBtn = document.getElementById('collapseSidebarBtn');
  fileExplorer.expandSidebarBtn = document.getElementById('expandSidebarBtn');
  fileExplorer.sidebarResizeHandle = document.getElementById('sidebarResizeHandle');

  // Split pane elements
  fileExplorer.editorPane = document.getElementById('editorPane');
  fileExplorer.chatPane = document.getElementById('chatPane');
  fileExplorer.paneResizeHandle = document.getElementById('paneResizeHandle');
  fileExplorer.editorTabsSection = document.getElementById('editorTabsSection');
  fileExplorer.editorTabs = document.getElementById('editorTabs');
  fileExplorer.editorContainer = document.getElementById('editorContainer');

  // Pane collapse/expand buttons
  fileExplorer.collapseEditorBtn = document.getElementById('collapseEditorBtn');
  fileExplorer.expandEditorBtn = document.getElementById('expandEditorBtn');
  fileExplorer.collapseChatBtn = document.getElementById('collapseChatBtn');
  fileExplorer.expandChatBtn = document.getElementById('expandChatBtn');

  // Explorer toolbar buttons
  fileExplorer.newFileBtn = document.getElementById('newFileBtn');
  fileExplorer.newFolderBtn = document.getElementById('newFolderBtn');
  fileExplorer.revealInExplorerBtn = document.getElementById('revealInExplorerBtn');

  if (!fileExplorer.sidebar) return;

  // Sidebar toggle handlers
  if (fileExplorer.collapseSidebarBtn) {
    fileExplorer.collapseSidebarBtn.addEventListener('click', () => {
      fileExplorer.sidebar.classList.add('collapsed');
      if (fileExplorer.expandSidebarBtn) {
        fileExplorer.expandSidebarBtn.classList.remove('hidden');
      }
    });
  }

  if (fileExplorer.expandSidebarBtn) {
    fileExplorer.expandSidebarBtn.addEventListener('click', () => {
      fileExplorer.sidebar.classList.remove('collapsed');
      fileExplorer.expandSidebarBtn.classList.add('hidden');
    });
  }

  // Explorer toolbar handlers
  if (fileExplorer.newFileBtn) {
    fileExplorer.newFileBtn.addEventListener('click', handleNewFile);
  }
  if (fileExplorer.newFolderBtn) {
    fileExplorer.newFolderBtn.addEventListener('click', handleNewFolder);
  }
  if (fileExplorer.revealInExplorerBtn) {
    fileExplorer.revealInExplorerBtn.addEventListener('click', handleRevealInExplorer);
  }

  // Sidebar resize handle
  if (fileExplorer.sidebarResizeHandle) {
    fileExplorer.sidebarResizeHandle.addEventListener('mousedown', (e) => {
      e.preventDefault();
      fileExplorer.isSidebarResizing = true;
      fileExplorer.sidebarResizeHandle.classList.add('dragging');
      document.body.style.cursor = 'col-resize';
      document.body.style.userSelect = 'none';
    });
  }

  // Pane resize handle
  if (fileExplorer.paneResizeHandle) {
    fileExplorer.paneResizeHandle.addEventListener('mousedown', (e) => {
      e.preventDefault();
      fileExplorer.isPaneResizing = true;
      fileExplorer.paneResizeHandle.classList.add('dragging');
      document.body.style.cursor = 'col-resize';
      document.body.style.userSelect = 'none';
    });
  }

  // Global mouse handlers for resizing
  document.addEventListener('mousemove', (e) => {
    if (fileExplorer.isSidebarResizing) {
      const newWidth = e.clientX;
      if (newWidth >= 150 && newWidth <= 500) {
        fileExplorer.sidebar.style.width = newWidth + 'px';
      }
    }
    if (fileExplorer.isPaneResizing && fileExplorer.editorPane && fileExplorer.chatPane) {
      const container = document.getElementById('splitPaneContainer');
      if (container) {
        const containerRect = container.getBoundingClientRect();
        const offsetX = e.clientX - containerRect.left;
        const containerWidth = containerRect.width;
        const minWidth = 250;
        const maxWidth = containerWidth - 300;
        if (offsetX >= minWidth && offsetX <= maxWidth) {
          fileExplorer.editorPane.style.flex = 'none';
          fileExplorer.editorPane.style.width = offsetX + 'px';
        }
      }
    }
  });

  document.addEventListener('mouseup', () => {
    if (fileExplorer.isSidebarResizing) {
      fileExplorer.isSidebarResizing = false;
      fileExplorer.sidebarResizeHandle?.classList.remove('dragging');
      document.body.style.cursor = '';
      document.body.style.userSelect = '';
    }
    if (fileExplorer.isPaneResizing) {
      fileExplorer.isPaneResizing = false;
      fileExplorer.paneResizeHandle?.classList.remove('dragging');
      document.body.style.cursor = '';
      document.body.style.userSelect = '';
    }
  });

  // Editor pane collapse/expand
  if (fileExplorer.collapseEditorBtn) {
    fileExplorer.collapseEditorBtn.addEventListener('click', () => {
      collapseEditorPane();
    });
  }

  if (fileExplorer.expandEditorBtn) {
    fileExplorer.expandEditorBtn.addEventListener('click', () => {
      expandEditorPane();
    });
  }

  // Chat pane collapse/expand
  if (fileExplorer.collapseChatBtn) {
    fileExplorer.collapseChatBtn.addEventListener('click', () => {
      collapseChatPane();
    });
  }

  if (fileExplorer.expandChatBtn) {
    fileExplorer.expandChatBtn.addEventListener('click', () => {
      expandChatPane();
    });
  }

  // Load file tree when workspace changes
  loadFileTree();
}

async function loadFileTree() {
  // Reset tree hash on load to ensure fresh render
  lastTreeHash = '';

  if (!fileExplorer.fileTree || !appState.data?.workspace?.path) {
    if (fileExplorer.fileTree) {
      fileExplorer.fileTree.innerHTML = '<div class="file-tree-empty">No project selected</div>';
    }
    stopFileWatching();
    stopTreeRefresh();
    return;
  }

  const workspacePath = appState.data.workspace.path;

  try {
    const res = await fetch(`/api/files/tree?workspace=${encodeURIComponent(workspacePath)}`);
    if (!res.ok) throw new Error('Failed to load files');

    const tree = await res.json();
    renderFileTree(tree, workspacePath);

    // Restore previously open tabs for this workspace
    await restoreOpenTabs();

    // Start file watching and tree refresh
    startFileWatching();
    startTreeRefresh();
  } catch (err) {
    console.error('Failed to load file tree:', err);
    fileExplorer.fileTree.innerHTML = '<div class="file-tree-empty">Failed to load files</div>';
  }
}

function renderFileTree(entries, workspacePath) {
  fileExplorer.fileTree.innerHTML = '';

  // Extract project name from workspace path
  const projectName = workspacePath.split('/').filter(p => p).pop() || 'Project';

  // Create root project node (VS Code style)
  const rootItem = document.createElement('div');
  rootItem.className = 'file-tree-item file-tree-root expanded';
  rootItem.innerHTML = `
    <i data-lucide="chevron-down" class="expand-icon"></i>
    <span class="item-name root-name">${escapeHtml(projectName.toUpperCase())}</span>
  `;

  // Create container for project contents
  const rootContents = document.createElement('div');
  rootContents.className = 'file-tree-folder-contents expanded';

  // Root toggle handler
  rootItem.addEventListener('click', (e) => {
    e.stopPropagation();
    const isExpanded = rootItem.classList.toggle('expanded');
    rootContents.classList.toggle('expanded', isExpanded);
    // Update chevron icon
    const chevron = rootItem.querySelector('.expand-icon');
    if (chevron) {
      chevron.setAttribute('data-lucide', isExpanded ? 'chevron-down' : 'chevron-right');
      lucide.createIcons({ nodes: [chevron.parentElement] });
    }
  });

  fileExplorer.fileTree.appendChild(rootItem);
  fileExplorer.fileTree.appendChild(rootContents);

  // Handle empty folder
  if (!entries || entries.length === 0) {
    rootContents.innerHTML = '<div class="file-tree-empty">Empty folder</div>';
    lucide.createIcons({ nodes: [fileExplorer.fileTree] });
    return;
  }

  // Render all entries under root
  for (const entry of entries) {
    renderFileTreeItem(entry, rootContents, workspacePath, 0);
  }

  lucide.createIcons({ nodes: [fileExplorer.fileTree] });
}

function renderFileTreeItem(entry, container, workspacePath, depth) {
  const item = document.createElement('div');
  item.className = 'file-tree-item';
  item.dataset.path = entry.path;
  item.dataset.depth = depth;
  item.dataset.isDir = entry.isDir;

  if (entry.isDir) {
    item.innerHTML = `
      <i data-lucide="chevron-right" class="expand-icon"></i>
      <i data-lucide="folder"></i>
      <span class="item-name">${escapeHtml(entry.name)}</span>
    `;

    const childContainer = document.createElement('div');
    childContainer.className = 'file-tree-folder-contents';

    item.addEventListener('click', (e) => {
      e.stopPropagation();
      const isExpanded = item.classList.toggle('expanded');
      childContainer.classList.toggle('expanded', isExpanded);

      // Toggle chevron icon
      const chevron = item.querySelector('.expand-icon');
      if (chevron) {
        chevron.setAttribute('data-lucide', isExpanded ? 'chevron-down' : 'chevron-right');
        lucide.createIcons({ nodes: [item] });
      }

      if (isExpanded && childContainer.children.length === 0 && entry.children) {
        for (const child of entry.children) {
          renderFileTreeItem(child, childContainer, workspacePath, depth + 1);
        }
        lucide.createIcons({ nodes: [childContainer] });
      }
    });

    container.appendChild(item);
    container.appendChild(childContainer);
  } else {
    const icon = getFileIcon(entry.name);
    item.innerHTML = `
      <i data-lucide="${icon}"></i>
      <span class="item-name">${escapeHtml(entry.name)}</span>
    `;

    item.addEventListener('dblclick', (e) => {
      e.stopPropagation();
      openFile(entry.path, entry.name, workspacePath);
    });

    container.appendChild(item);
  }
}

// Reveal and highlight a file in the tree (VS Code behavior)
function revealFileInTree(filePath) {
  if (!fileExplorer.fileTree || !filePath) return;

  // Remove previous selection
  const prevSelected = fileExplorer.fileTree.querySelector('.file-tree-item.selected');
  if (prevSelected) {
    prevSelected.classList.remove('selected');
  }

  // Get workspace path to calculate relative path
  const workspacePath = appState.data?.workspace?.path;
  if (!workspacePath) return;

  // Calculate relative path (tree uses relative paths from workspace root)
  let relativePath = filePath;
  if (filePath.startsWith(workspacePath)) {
    relativePath = filePath.slice(workspacePath.length);
    if (relativePath.startsWith('/')) relativePath = relativePath.slice(1);
  }

  const pathParts = relativePath.split('/').filter(p => p);
  if (pathParts.length === 0) return;

  // First, make sure root is expanded
  const rootItem = fileExplorer.fileTree.querySelector('.file-tree-root');
  const rootContents = fileExplorer.fileTree.querySelector('.file-tree-folder-contents');
  if (rootItem && rootContents && !rootItem.classList.contains('expanded')) {
    rootItem.classList.add('expanded');
    rootContents.classList.add('expanded');
    const chevron = rootItem.querySelector('.expand-icon');
    if (chevron) {
      chevron.setAttribute('data-lucide', 'chevron-down');
      lucide.createIcons({ nodes: [rootItem] });
    }
  }

  // Build RELATIVE path incrementally (tree items use relative paths)
  let currentRelPath = '';
  let currentContainer = rootContents;

  for (let i = 0; i < pathParts.length; i++) {
    const part = pathParts[i];
    currentRelPath = currentRelPath ? currentRelPath + '/' + part : part;
    const isLastPart = i === pathParts.length - 1;

    // Find the item in current container by relative path
    const item = currentContainer?.querySelector(`.file-tree-item[data-path="${currentRelPath}"]`);

    if (!item) {
      // Item not found - might be in unexpanded folder
      break;
    }

    if (isLastPart) {
      // This is the file - select and scroll to it
      item.classList.add('selected');
      item.scrollIntoView({ behavior: 'smooth', block: 'nearest' });
    } else {
      // This is a folder - expand it if not already
      if (!item.classList.contains('expanded')) {
        item.click(); // Trigger expansion
      }
      // Get the folder contents container (next sibling)
      currentContainer = item.nextElementSibling;
    }
  }
}

// Explorer toolbar handlers
async function handleNewFile() {
  const workspacePath = appState.data?.workspace?.path;
  if (!workspacePath) {
    showToast('No workspace selected', 'error');
    return;
  }

  // Get currently selected folder or use root
  const selectedItem = fileExplorer.fileTree?.querySelector('.file-tree-item.selected');
  let basePath = '';
  if (selectedItem) {
    const itemPath = selectedItem.getAttribute('data-path');
    const isDir = selectedItem.getAttribute('data-is-dir') === 'true';
    basePath = isDir ? itemPath : itemPath.substring(0, itemPath.lastIndexOf('/'));
  }

  const fileName = prompt('Enter file name:', 'newfile.txt');
  if (!fileName || !fileName.trim()) return;

  const filePath = basePath ? `${basePath}/${fileName.trim()}` : fileName.trim();

  try {
    const resp = await fetch('/api/files/create', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ workspace: workspacePath, path: filePath })
    });

    if (!resp.ok) {
      const err = await resp.json();
      showToast(err.error || 'Failed to create file', 'error');
      return;
    }

    // Refresh tree and open the new file
    await refreshFileTree();
    openFile(workspacePath, filePath);
    showToast(`Created ${fileName}`, 'success');
  } catch (err) {
    showToast('Failed to create file', 'error');
  }
}

async function handleNewFolder() {
  const workspacePath = appState.data?.workspace?.path;
  if (!workspacePath) {
    showToast('No workspace selected', 'error');
    return;
  }

  // Get currently selected folder or use root
  const selectedItem = fileExplorer.fileTree?.querySelector('.file-tree-item.selected');
  let basePath = '';
  if (selectedItem) {
    const itemPath = selectedItem.getAttribute('data-path');
    const isDir = selectedItem.getAttribute('data-is-dir') === 'true';
    basePath = isDir ? itemPath : itemPath.substring(0, itemPath.lastIndexOf('/'));
  }

  const folderName = prompt('Enter folder name:', 'newfolder');
  if (!folderName || !folderName.trim()) return;

  const folderPath = basePath ? `${basePath}/${folderName.trim()}` : folderName.trim();

  try {
    const resp = await fetch('/api/files/mkdir', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ workspace: workspacePath, path: folderPath })
    });

    if (!resp.ok) {
      const err = await resp.json();
      showToast(err.error || 'Failed to create folder', 'error');
      return;
    }

    // Refresh tree
    await refreshFileTree();
    showToast(`Created ${folderName}`, 'success');
  } catch (err) {
    showToast('Failed to create folder', 'error');
  }
}

async function handleRevealInExplorer() {
  const workspacePath = appState.data?.workspace?.path;
  if (!workspacePath) {
    showToast('No workspace selected', 'error');
    return;
  }

  // Get currently selected item or use workspace root
  const selectedItem = fileExplorer.fileTree?.querySelector('.file-tree-item.selected');
  const filePath = selectedItem?.getAttribute('data-path') || '';

  try {
    const resp = await fetch('/api/files/reveal', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ workspace: workspacePath, path: filePath })
    });

    if (!resp.ok) {
      const err = await resp.json();
      showToast(err.error || 'Failed to open explorer', 'error');
    }
  } catch (err) {
    showToast('Failed to open explorer', 'error');
  }
}

function getFileIcon(filename) {
  const ext = filename.split('.').pop().toLowerCase();
  const iconMap = {
    js: 'file-code',
    ts: 'file-code',
    jsx: 'file-code',
    tsx: 'file-code',
    go: 'file-code',
    py: 'file-code',
    rb: 'file-code',
    rs: 'file-code',
    java: 'file-code',
    c: 'file-code',
    cpp: 'file-code',
    h: 'file-code',
    css: 'file-code',
    scss: 'file-code',
    html: 'file-code',
    vue: 'file-code',
    svelte: 'file-code',
    json: 'file-json',
    yaml: 'file-text',
    yml: 'file-text',
    md: 'file-text',
    txt: 'file-text',
    sql: 'database',
    sh: 'terminal',
    bash: 'terminal',
    zsh: 'terminal',
    png: 'image',
    jpg: 'image',
    jpeg: 'image',
    gif: 'image',
    svg: 'image',
    pdf: 'file',
  };
  return iconMap[ext] || 'file';
}

async function openFile(filePath, fileName, workspacePath) {
  // Check if already open
  const existingTab = fileExplorer.openTabs.find(t => t.path === filePath);
  if (existingTab) {
    switchToTab(filePath);
    return;
  }

  try {
    const res = await fetch(`/api/files/read?workspace=${encodeURIComponent(workspacePath)}&path=${encodeURIComponent(filePath)}`);
    if (!res.ok) {
      const err = await res.json();
      alert(err.error || 'Failed to open file');
      return;
    }

    const data = await res.json();

    if (data.isBinary) {
      alert('Cannot edit binary files');
      return;
    }

    // Add to tabs
    const tab = {
      path: filePath,
      name: fileName,
      content: data.content,
      originalContent: data.content,
      dirty: false,
      modTime: data.modTime,
      workspacePath: workspacePath,
    };

    fileExplorer.openTabs.push(tab);

    // Show editor pane and tabs section
    showEditorPane();

    renderTabs();
    switchToTab(filePath);

    // Refresh CodeMirror after container is visible (fixes blank editor issue)
    if (fileExplorer.editor) {
      setTimeout(() => fileExplorer.editor.refresh(), 0);
    }

    // Persist open tabs
    saveOpenTabs();
  } catch (err) {
    console.error('Failed to open file:', err);
    alert('Failed to open file');
  }
}

function renderTabs() {
  if (!fileExplorer.editorTabs) return;

  fileExplorer.editorTabs.innerHTML = '';

  for (const tab of fileExplorer.openTabs) {
    const tabEl = document.createElement('button');
    tabEl.className = 'editor-tab' + (tab.path === fileExplorer.activeTabPath ? ' active' : '') + (tab.dirty ? ' dirty' : '');
    tabEl.dataset.path = tab.path;

    const icon = getFileIcon(tab.name);
    tabEl.innerHTML = `
      <i data-lucide="${icon}" class="tab-icon"></i>
      <span class="tab-name">${escapeHtml(tab.name)}</span>
      <span class="tab-dirty"></span>
      <button class="tab-close" title="Close">
        <i data-lucide="x"></i>
      </button>
    `;

    tabEl.addEventListener('click', (e) => {
      if (!e.target.closest('.tab-close')) {
        switchToTab(tab.path);
      }
    });

    tabEl.querySelector('.tab-close').addEventListener('click', (e) => {
      e.stopPropagation();
      closeTab(tab.path);
    });

    fileExplorer.editorTabs.appendChild(tabEl);
  }

  lucide.createIcons({ nodes: [fileExplorer.editorTabs] });
}

function switchToTab(path) {
  const tab = fileExplorer.openTabs.find(t => t.path === path);
  if (!tab) return;

  fileExplorer.activeTabPath = path;

  // Update tab UI
  renderTabs();

  // Update editor content
  if (fileExplorer.editor) {
    fileExplorer.editor.setValue(tab.content);
    fileExplorer.editor.clearHistory();

    // Set mode based on file extension
    const mode = getModeForFile(tab.name);
    fileExplorer.editor.setOption('mode', mode);
  } else {
    // Initialize CodeMirror
    initEditor(tab);
  }

  // Reveal file in explorer tree (VS Code behavior)
  revealFileInTree(path);

  // Save active tab
  saveOpenTabs();
}

function initEditor(tab) {
  if (!fileExplorer.editorContainer || !window.CodeMirror) return;

  fileExplorer.editorContainer.innerHTML = '';

  const mode = getModeForFile(tab.name);

  fileExplorer.editor = CodeMirror(fileExplorer.editorContainer, {
    value: tab.content,
    mode: mode,
    theme: 'material-darker',
    lineNumbers: true,
    indentUnit: 2,
    tabSize: 2,
    indentWithTabs: false,
    lineWrapping: true,
    matchBrackets: true,
    autoCloseBrackets: true,
    extraKeys: {
      'Ctrl-S': () => saveCurrentFile(),
      'Cmd-S': () => saveCurrentFile(),
    },
  });

  // Handle changes
  fileExplorer.editor.on('change', () => {
    const currentTab = fileExplorer.openTabs.find(t => t.path === fileExplorer.activeTabPath);
    if (!currentTab) return;

    const newContent = fileExplorer.editor.getValue();
    currentTab.content = newContent;
    currentTab.dirty = newContent !== currentTab.originalContent;

    // Update tab UI to show dirty state
    const tabEl = fileExplorer.editorTabs.querySelector(`[data-path="${currentTab.path}"]`);
    if (tabEl) {
      tabEl.classList.toggle('dirty', currentTab.dirty);
    }

    // Auto-save with debounce
    if (fileExplorer.saveTimeout) {
      clearTimeout(fileExplorer.saveTimeout);
    }
    if (currentTab.dirty) {
      fileExplorer.saveTimeout = setTimeout(() => {
        saveCurrentFile();
      }, fileExplorer.AUTOSAVE_DELAY);
    }
  });
}

function getModeForFile(filename) {
  const ext = filename.split('.').pop().toLowerCase();
  const modeMap = {
    js: 'javascript',
    jsx: 'javascript',
    ts: 'javascript',
    tsx: 'javascript',
    json: { name: 'javascript', json: true },
    html: 'htmlmixed',
    htm: 'htmlmixed',
    vue: 'htmlmixed',
    svelte: 'htmlmixed',
    xml: 'xml',
    css: 'css',
    scss: 'css',
    less: 'css',
    md: 'markdown',
    markdown: 'markdown',
    py: 'python',
    go: 'go',
    sh: 'shell',
    bash: 'shell',
    zsh: 'shell',
    yaml: 'yaml',
    yml: 'yaml',
    sql: 'sql',
    rs: 'rust',
    c: 'text/x-csrc',
    cpp: 'text/x-c++src',
    h: 'text/x-csrc',
    hpp: 'text/x-c++src',
    java: 'text/x-java',
  };
  return modeMap[ext] || 'text/plain';
}

async function saveCurrentFile() {
  const tab = fileExplorer.openTabs.find(t => t.path === fileExplorer.activeTabPath);
  if (!tab || !tab.dirty) return;

  try {
    const res = await fetch('/api/files/save', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        workspace: tab.workspacePath,
        path: tab.path,
        content: tab.content,
      }),
    });

    if (!res.ok) {
      const err = await res.json();
      console.error('Failed to save file:', err.error);
      return;
    }

    const data = await res.json();
    tab.originalContent = tab.content;
    tab.dirty = false;
    tab.modTime = data.modTime;

    // Update tab UI
    const tabEl = fileExplorer.editorTabs.querySelector(`[data-path="${tab.path}"]`);
    if (tabEl) {
      tabEl.classList.remove('dirty');
    }
  } catch (err) {
    console.error('Failed to save file:', err);
  }
}

function closeTab(path) {
  const tabIndex = fileExplorer.openTabs.findIndex(t => t.path === path);
  if (tabIndex === -1) return;

  const tab = fileExplorer.openTabs[tabIndex];

  // Check for unsaved changes
  if (tab.dirty) {
    if (!confirm(`"${tab.name}" has unsaved changes. Close anyway?`)) {
      return;
    }
  }

  // Remove tab
  fileExplorer.openTabs.splice(tabIndex, 1);

  // Switch to another tab or hide editor pane
  if (fileExplorer.openTabs.length === 0) {
    fileExplorer.activeTabPath = null;
    hideEditorPane();
    if (fileExplorer.editor) {
      fileExplorer.editor = null;
      if (fileExplorer.editorContainer) {
        fileExplorer.editorContainer.innerHTML = '';
      }
    }
  } else {
    // Switch to previous or next tab
    const newIndex = Math.min(tabIndex, fileExplorer.openTabs.length - 1);
    switchToTab(fileExplorer.openTabs[newIndex].path);
  }

  renderTabs();
  saveOpenTabs();
}

// Close all tabs when switching projects
function closeAllTabs() {
  stopFileWatching();
  fileExplorer.openTabs = [];
  fileExplorer.activeTabPath = null;
  hideEditorPane();
  if (fileExplorer.editor) {
    fileExplorer.editor = null;
    if (fileExplorer.editorContainer) {
      fileExplorer.editorContainer.innerHTML = '';
    }
  }
  if (fileExplorer.editorTabs) {
    fileExplorer.editorTabs.innerHTML = '';
  }
  saveOpenTabs();
}

// File watching - detect external changes
function startFileWatching() {
  if (fileExplorer.fileWatchTimer) return;

  fileExplorer.fileWatchTimer = setInterval(async () => {
    if (fileExplorer.openTabs.length === 0) return;

    const dirtyChanges = [];  // Files with local changes that were modified externally
    const cleanChanges = [];  // Files without local changes that were modified externally

    for (const tab of fileExplorer.openTabs) {
      try {
        const res = await fetch(`/api/files/read?workspace=${encodeURIComponent(tab.workspacePath)}&path=${encodeURIComponent(tab.path)}`);
        if (!res.ok) continue;

        const data = await res.json();

        // Check if file was modified externally
        if (data.modTime && data.modTime !== tab.modTime) {
          if (tab.dirty) {
            dirtyChanges.push({ tab, data });
          } else {
            cleanChanges.push({ tab, data });
          }
        }
      } catch (err) {
        // File might have been deleted - ignore
      }
    }

    // Silently reload files without local changes (no flicker - only update if content differs)
    for (const { tab, data } of cleanChanges) {
      if (tab.content !== data.content) {
        tab.content = data.content;
        tab.originalContent = data.content;
        tab.modTime = data.modTime;

        // Only update editor if this is the active tab
        if (fileExplorer.activeTabPath === tab.path && fileExplorer.editor) {
          const cursor = fileExplorer.editor.getCursor();
          const scrollInfo = fileExplorer.editor.getScrollInfo();
          fileExplorer.editor.setValue(data.content);
          fileExplorer.editor.setCursor(cursor);
          fileExplorer.editor.scrollTo(scrollInfo.left, scrollInfo.top);
        }
      } else {
        // Content same, just update modTime
        tab.modTime = data.modTime;
      }
    }

    // Batch prompt for files with local changes
    if (dirtyChanges.length > 0) {
      const fileNames = dirtyChanges.map(c => c.tab.name).join(', ');
      const msg = dirtyChanges.length === 1
        ? `"${dirtyChanges[0].tab.name}" was modified externally. Reload and lose local changes?`
        : `${dirtyChanges.length} files were modified externally (${fileNames}). Reload all and lose local changes?`;

      if (confirm(msg)) {
        for (const { tab, data } of dirtyChanges) {
          tab.content = data.content;
          tab.originalContent = data.content;
          tab.modTime = data.modTime;
          tab.dirty = false;

          if (fileExplorer.activeTabPath === tab.path && fileExplorer.editor) {
            fileExplorer.editor.setValue(data.content);
            fileExplorer.editor.clearHistory();
          }
        }
        renderTabs();
      } else {
        // User declined - update modTime to avoid repeated prompts
        for (const { tab, data } of dirtyChanges) {
          tab.modTime = data.modTime;
        }
      }
    }
  }, fileExplorer.FILE_WATCH_INTERVAL);
}

function stopFileWatching() {
  if (fileExplorer.fileWatchTimer) {
    clearInterval(fileExplorer.fileWatchTimer);
    fileExplorer.fileWatchTimer = null;
  }
}

// Explorer tree auto-refresh
function startTreeRefresh() {
  if (fileExplorer.treeRefreshTimer) return;

  fileExplorer.treeRefreshTimer = setInterval(() => {
    refreshFileTree();
  }, fileExplorer.TREE_REFRESH_INTERVAL);
}

function stopTreeRefresh() {
  if (fileExplorer.treeRefreshTimer) {
    clearInterval(fileExplorer.treeRefreshTimer);
    fileExplorer.treeRefreshTimer = null;
  }
}

// Cache for tree structure comparison
let lastTreeHash = '';

function hashTree(entries) {
  // Create a simple hash of the tree structure (paths only, not content)
  const paths = [];
  function collectPaths(items, prefix = '') {
    for (const item of items) {
      paths.push(prefix + item.name + (item.isDir ? '/' : ''));
      if (item.children) {
        collectPaths(item.children, prefix + item.name + '/');
      }
    }
  }
  collectPaths(entries);
  return paths.sort().join('|');
}

async function refreshFileTree() {
  const workspacePath = appState.data?.workspace?.path;
  if (!workspacePath || !fileExplorer.fileTree) return;

  try {
    const res = await fetch(`/api/files/tree?workspace=${encodeURIComponent(workspacePath)}`);
    if (!res.ok) return;

    const entries = await res.json();

    // Check if tree structure actually changed
    const newHash = hashTree(entries);
    if (newHash === lastTreeHash) {
      return; // No changes, skip re-render to avoid flicker
    }
    lastTreeHash = newHash;

    // Remember expanded folders
    const expandedPaths = new Set();
    fileExplorer.fileTree.querySelectorAll('.file-tree-item.expanded').forEach(el => {
      if (el.dataset.path) expandedPaths.add(el.dataset.path);
    });
    const rootExpanded = fileExplorer.fileTree.querySelector('.file-tree-root.expanded') !== null;

    // Re-render tree
    renderFileTree(entries, workspacePath);

    // Restore expanded state
    if (rootExpanded) {
      const rootItem = fileExplorer.fileTree.querySelector('.file-tree-root');
      const rootContents = fileExplorer.fileTree.querySelector('.file-tree-folder-contents');
      if (rootItem && rootContents) {
        rootItem.classList.add('expanded');
        rootContents.classList.add('expanded');
        const chevron = rootItem.querySelector('.expand-icon');
        if (chevron) {
          chevron.setAttribute('data-lucide', 'chevron-down');
        }
      }
    }

    // Restore folder expansions
    expandedPaths.forEach(path => {
      const item = fileExplorer.fileTree.querySelector(`.file-tree-item[data-path="${path}"]`);
      if (item && !item.classList.contains('expanded')) {
        item.click();
      }
    });

    // Re-select active file
    if (fileExplorer.activeTabPath) {
      revealFileInTree(fileExplorer.activeTabPath);
    }

    lucide.createIcons({ nodes: [fileExplorer.fileTree] });
  } catch (err) {
    console.error('Failed to refresh file tree:', err);
  }
}

// Show editor pane (when files are opened)
function showEditorPane() {
  if (fileExplorer.editorPane) {
    fileExplorer.editorPane.classList.remove('hidden');
  }
  if (fileExplorer.paneResizeHandle) {
    fileExplorer.paneResizeHandle.classList.remove('hidden');
  }
  if (fileExplorer.editorTabsSection) {
    fileExplorer.editorTabsSection.classList.remove('hidden');
  }
  // Show expand button in chat pane header if it was hidden
  if (fileExplorer.expandEditorBtn) {
    fileExplorer.expandEditorBtn.classList.add('hidden');
  }
}

// Hide editor pane (when all files closed)
function hideEditorPane() {
  if (fileExplorer.editorPane) {
    fileExplorer.editorPane.classList.add('hidden');
  }
  if (fileExplorer.paneResizeHandle) {
    fileExplorer.paneResizeHandle.classList.add('hidden');
  }
  if (fileExplorer.editorTabsSection) {
    fileExplorer.editorTabsSection.classList.add('hidden');
  }
}

// Collapse editor pane (user action - keeps tabs in toolbar)
function collapseEditorPane() {
  if (fileExplorer.editorPane) {
    fileExplorer.editorPane.classList.add('collapsed');
  }
  if (fileExplorer.paneResizeHandle) {
    fileExplorer.paneResizeHandle.classList.add('hidden');
  }
  if (fileExplorer.expandEditorBtn) {
    fileExplorer.expandEditorBtn.classList.remove('hidden');
  }
}

// Expand editor pane (user action)
function expandEditorPane() {
  if (fileExplorer.editorPane) {
    fileExplorer.editorPane.classList.remove('collapsed');
  }
  if (fileExplorer.paneResizeHandle) {
    fileExplorer.paneResizeHandle.classList.remove('hidden');
  }
  if (fileExplorer.expandEditorBtn) {
    fileExplorer.expandEditorBtn.classList.add('hidden');
  }
  // Refresh editor if exists
  if (fileExplorer.editor) {
    setTimeout(() => fileExplorer.editor.refresh(), 0);
  }
}

// Collapse chat pane (user action)
function collapseChatPane() {
  if (fileExplorer.chatPane) {
    fileExplorer.chatPane.classList.add('collapsed');
  }
  if (fileExplorer.paneResizeHandle) {
    fileExplorer.paneResizeHandle.classList.add('hidden');
  }
  if (fileExplorer.expandChatBtn) {
    fileExplorer.expandChatBtn.classList.remove('hidden');
  }
  // Clear any inline width so editor pane expands to fill space
  if (fileExplorer.editorPane) {
    fileExplorer.editorPane.style.width = '';
  }
}

// Expand chat pane (user action)
function expandChatPane() {
  if (fileExplorer.chatPane) {
    fileExplorer.chatPane.classList.remove('collapsed');
  }
  // Only show resize handle if editor pane is also visible
  if (fileExplorer.paneResizeHandle && fileExplorer.editorPane &&
      !fileExplorer.editorPane.classList.contains('hidden') &&
      !fileExplorer.editorPane.classList.contains('collapsed')) {
    fileExplorer.paneResizeHandle.classList.remove('hidden');
  }
  if (fileExplorer.expandChatBtn) {
    fileExplorer.expandChatBtn.classList.add('hidden');
  }
}

// Save open tabs to localStorage
function saveOpenTabs() {
  if (!appState.data?.workspace?.path) return;
  const workspacePath = appState.data.workspace.path;
  const tabsData = fileExplorer.openTabs.map(t => ({
    path: t.path,
    name: t.name,
  }));
  const key = `cando_tabs_${workspacePath}`;
  localStorage.setItem(key, JSON.stringify({
    tabs: tabsData,
    activeTab: fileExplorer.activeTabPath,
  }));
}

// Restore open tabs from localStorage
async function restoreOpenTabs() {
  if (!appState.data?.workspace?.path) return;
  const workspacePath = appState.data.workspace.path;
  const key = `cando_tabs_${workspacePath}`;
  const saved = localStorage.getItem(key);
  if (!saved) return;

  try {
    const data = JSON.parse(saved);
    if (!data.tabs || !Array.isArray(data.tabs)) return;

    // Open each tab
    for (const tab of data.tabs) {
      await openFile(tab.path, tab.name, workspacePath);
    }

    // Switch to active tab if specified
    if (data.activeTab && fileExplorer.openTabs.some(t => t.path === data.activeTab)) {
      switchToTab(data.activeTab);
    }
  } catch (err) {
    console.error('Failed to restore tabs:', err);
  }
}

// For backward compatibility - now just focuses chat pane
function switchToChatView() {
  // With split panes, both are visible - just ensure chat is not collapsed
  expandChatPane();
}

// Hook into project switching to reload file tree
const originalSwitchWorkspace = typeof switchWorkspace === 'function' ? switchWorkspace : null;
