const state = { powerStatus: null, tasks: [], activeTaskId: null, eventSource: null, recoveryPoller: null, activeRefreshPoller: null, selectedForests: {}, selectedFileTree: {}, selectedFilePath: {}, selectedFileManual: {}, fileListFingerprint: {}, lastFileRefreshAt: {}, treeStatusExpanded: {}, usage: {}, usageFetchedAt: {}, usagePending: {}, renderCache: {}, pendingTaskRender: null, pendingTaskRenderFrame: 0, lastTaskListSig: '', lastHomeSig: '', activeReportText: '', fileViewerToken: 0, previewToken: 0, overviewCollapsed: loadOverviewCollapsed(), editingTitle: false, settings: loadSettings() };
const $ = (id) => document.getElementById(id);

const I18N = {
  'zh-CN': {
    newTask:'新建任务', taskLabel:'任务', homeTitle:'你想完成什么？', garden:'工作台', taskPlaceholder:'告诉 Gardener 你的目标、要求和交付物', saveLocation:'保存位置', defaultSave:'默认保存', create:'创建', tasks:'任务', refresh:'刷新', back:'返回', messagePlaceholder:'给 Gardener 发消息', send:'发送', taskPlan:'任务安排', workRecord:'工作记录', stop:'停止', workProcess:'工作过程', viewResult:'查看报告', settings:'设置', close:'关闭', defaultSaveLocation:'默认保存位置', autoSave:'留空则自动保存', saveLocationHelp:'不设置也可以正常使用。', showSaveLocation:'创建任务时显示保存位置', showPlanRecord:'在任务中显示安排和记录', language:'语言', logDetail:'记录详细程度', logQuiet:'简洁', logNormal:'标准', logDetailed:'详细', logHelp:'普通使用建议选择“简洁”。需要排查问题时再切换为“详细”。', save:'保存', copy:'复制', result:'报告', noTasks:'暂无任务', newTaskShort:'新任务', genericTask:'任务', inProgress:'进行中', done:'已完成', waitingForest:'等待阶段', noForest:'无阶段', gardenerWillContinue:'我会继续处理。', resultNotReady:'报告尚未生成。', openingResult:'正在打开报告', emptyResult:'内容为空', openFailed:'无法打开：', stopConfirm:'停止当前任务？', team:'子任务', validationTeam:'验证任务', files:'文件', recentForests:'已有任务', noRecent:'还没有任务', openForest:'打开', allFiles:'全部文件', allTreeFiles:'全部子任务', noFiles:'暂无可查看文件', loadingFiles:'正在读取文件', selectFile:'选择文件查看内容', fileTooLarge:'文件无法预览', treeStatus:'子任务状态', noTreesInForest:'当前阶段暂无子任务', browse:'选择', chooseFolder:'选择保存位置', parentFolder:'上一级', useFolder:'使用此目录', folderEmpty:'没有可选择的子目录', tokenUsage:'Token 消耗', tokenEstimate:'Token 消耗', tokenMaxEstimate:'', tokenNoData:'暂无 token 记录', delete:'删除', deleteConfirm:'删除这个任务并清除它的数据？', deleteFailed:'删除失败：', viewStatus:'查看状态', hideStatus:'收起状态', rename:'重命名', renamePrompt:'输入新的任务名称', renameFailed:'重命名失败：', model:'模型', modelDefault:'CLI 默认模型', cliEngine:'底层 CLI', cliCodex:'Codex CLI', cliClaude:'Claude Code', cliHelp:'创建任务后会固定使用所选 CLI。', modelToken:'Token', modelTokenPlaceholder:'输入当前模型的 token', gardenerProgress:'工作进展', gardenerWorking:'正在工作', gardenerProgressEmpty:'等待下一步进展', stage:'阶段', subtask:'子任务', file:'文件', resumeTask:'继续任务', resumeTaskHint:'任务已暂停。如未完成，可点击“继续任务”，Gardener 会检查当前进度后接着处理。', resumeFailed:'继续失败：', fileEncodingHint:'已自动尝试文本编码识别。', binaryFile:'文件可能不是文本，无法预览', noOutputYet:'正在等待产出文件或报告。', noOutputStale:'长时间没有新输出，底层 CLI 可能仍在处理。你可以直接询问进度，不会中断任务。', statusQuerySafe:'查看进度不会中断任务。', noOutputMinutes:'%dm 无新输出', collapseOverview:'收起概览', expandOverview:'展开概览', overview:'概览', recentMessagesOnly:'仅显示最近 %d 条消息。', previewTruncated:'文件较大，已仅预览前 %d 个字符。', downloadFile:'下载文件', powerWarningTitle:'远程访问提醒', powerWarningPrefix:'这台电脑的电源设置可能导致 Gardener 离线：', dashboard:'任务驾驶舱', duration:'运行时长', idle:'无输出', askProgressSafe:'询问进度不会中断任务', diagnosis:'诊断提示'
  },
  en: {
    newTask:'New task', taskLabel:'Task', homeTitle:'What do you want to get done?', garden:'Workspace', taskPlaceholder:'Tell Gardener your goal, requirements, and deliverables', saveLocation:'Save location', defaultSave:'Default save location', create:'Create', tasks:'Tasks', refresh:'Refresh', back:'Back', messagePlaceholder:'Message Gardener', send:'Send', taskPlan:'Plan', workRecord:'Activity', stop:'Stop', workProcess:'Activity', viewResult:'View report', settings:'Settings', close:'Close', defaultSaveLocation:'Default save location', autoSave:'Leave blank to save automatically', saveLocationHelp:'You can use Gardener without setting this.', showSaveLocation:'Show save location when creating a task', showPlanRecord:'Show plan and activity inside a task', language:'Language', logDetail:'Activity detail', logQuiet:'Simple', logNormal:'Standard', logDetailed:'Detailed', logHelp:'Simple is recommended. Use Detailed only when troubleshooting.', save:'Save', copy:'Copy', result:'Report', noTasks:'No tasks', newTaskShort:'New task', genericTask:'Task', inProgress:'Running', done:'Done', waitingForest:'Waiting for stage', noForest:'No stage', gardenerWillContinue:'I will keep working on it.', resultNotReady:'Report is not ready yet.', openingResult:'Opening report', emptyResult:'Empty content', openFailed:'Unable to open: ', stopConfirm:'Stop this task?', team:'Subtask', validationTeam:'Validation', files:'Files', recentForests:'Tasks', noRecent:'No tasks yet', openForest:'Open', allFiles:'All files', allTreeFiles:'All subtasks', noFiles:'No files', loadingFiles:'Loading files', selectFile:'Select a file to preview', fileTooLarge:'File cannot be previewed', treeStatus:'Subtask status', noTreesInForest:'No subtasks in this stage', browse:'Choose', chooseFolder:'Choose folder', parentFolder:'Parent', useFolder:'Use this folder', folderEmpty:'No folders', tokenUsage:'Token usage', tokenEstimate:'Token usage', tokenMaxEstimate:'', tokenNoData:'No token records yet', delete:'Delete', deleteConfirm:'Delete this task and clear its data?', deleteFailed:'Delete failed: ', viewStatus:'View status', hideStatus:'Hide status', rename:'Rename', renamePrompt:'Enter a new task name', renameFailed:'Rename failed: ', model:'Model', modelDefault:'CLI default model', cliEngine:'Base CLI', cliCodex:'Codex CLI', cliClaude:'Claude Code', cliHelp:'A task keeps the selected CLI after creation.', modelToken:'Token', modelTokenPlaceholder:'Enter the token for the selected model', gardenerProgress:'Work progress', gardenerWorking:'Working', gardenerProgressEmpty:'Waiting for updates', stage:'Stage', subtask:'Subtask', file:'File', resumeTask:'Continue task', resumeTaskHint:'This task is paused. If it is not done, click Continue task and Gardener will inspect the current progress before continuing.', resumeFailed:'Continue failed: ', fileEncodingHint:'Text encoding was detected automatically.', binaryFile:'This file may not be text and cannot be previewed', noOutputYet:'Waiting for files or reports.', noOutputStale:'No new output for a while. The base CLI may still be working. You can ask for progress without interrupting the task.', statusQuerySafe:'Checking progress will not interrupt the task.', noOutputMinutes:'No output for %dm', collapseOverview:'Collapse', expandOverview:'Expand', overview:'Overview', recentMessagesOnly:'Showing latest %d messages only.', previewTruncated:'Large file: only first %d characters are shown.', downloadFile:'Download file', powerWarningTitle:'Remote access warning', powerWarningPrefix:'This computer may go offline because of its power settings: ', dashboard:'Task dashboard', duration:'Duration', idle:'Idle', askProgressSafe:'Asking progress will not interrupt the task', diagnosis:'Diagnostic cue'
  }
};

function loadOverviewCollapsed() {
  try { return localStorage.getItem('gardenerOverviewCollapsed') === '1'; }
  catch { return false; }
}
function saveOverviewCollapsed() {
  try { localStorage.setItem('gardenerOverviewCollapsed', state.overviewCollapsed ? '1' : '0'); }
  catch {}
}

function t(key) { return (I18N[state.settings?.language || 'zh-CN'] || I18N['zh-CN'])[key] || I18N['zh-CN'][key] || key; }
function applyI18n() {
  document.documentElement.lang = state.settings.language || 'zh-CN';
  document.querySelectorAll('[data-i18n]').forEach(el => { el.textContent = t(el.dataset.i18n); });
  document.querySelectorAll('[data-i18n-placeholder]').forEach(el => { el.placeholder = t(el.dataset.i18nPlaceholder); });
  document.querySelectorAll('[data-i18n-title]').forEach(el => { el.title = t(el.dataset.i18nTitle); });
}

async function api(path, options = {}) {
  const res = await fetch(path, { headers: { 'Content-Type': 'application/json', ...(options.headers || {}) }, ...options });
  if (!res.ok) {
    let msg = `HTTP ${res.status}`;
    try { msg = (await res.json()).error || msg; } catch {}
    throw new Error(msg);
  }
  return res.json();
}

async function fetchText(path) {
  const res = await fetch(path);
  if (!res.ok) throw new Error(`HTTP ${res.status}`);
  const buf = await res.arrayBuffer();
  return decodeTextBuffer(buf);
}

function decodeTextBuffer(buf) {
  const bytes = new Uint8Array(buf || new ArrayBuffer(0));
  if (!bytes.length) return '';
  if (bytes.length >= 3 && bytes[0] === 0xef && bytes[1] === 0xbb && bytes[2] === 0xbf) {
    return new TextDecoder('utf-8').decode(bytes.subarray(3));
  }
  if (bytes.length >= 2 && bytes[0] === 0xff && bytes[1] === 0xfe) {
    return new TextDecoder('utf-16le').decode(bytes.subarray(2));
  }
  if (bytes.length >= 2 && bytes[0] === 0xfe && bytes[1] === 0xff) {
    return decodeUTF16BE(bytes.subarray(2));
  }
  const utf16 = guessUTF16(bytes);
  if (utf16 === 'utf-16le') return new TextDecoder('utf-16le').decode(bytes);
  if (utf16 === 'utf-16be') return decodeUTF16BE(bytes);
  try {
    const text = new TextDecoder('utf-8', { fatal: true }).decode(bytes);
    if (!looksBinaryText(text)) return text;
  } catch {}
  const candidates = [];
  for (const label of ['gb18030', 'gbk', 'gb2312', 'big5', 'windows-1252']) {
    try {
      const text = new TextDecoder(label, { fatal: false }).decode(bytes);
      candidates.push({ label, text, score: textDecodeScore(text) });
    } catch {}
  }
  candidates.sort((a, b) => a.score - b.score);
  if (candidates.length && candidates[0].score < 80) return candidates[0].text;
  const fallback = new TextDecoder('utf-8', { fatal: false }).decode(bytes);
  if (looksBinaryText(fallback)) throw new Error(t('binaryFile'));
  return fallback;
}

function decodeUTF16BE(bytes) {
  const swapped = new Uint8Array(bytes.length);
  for (let i = 0; i + 1 < bytes.length; i += 2) {
    swapped[i] = bytes[i + 1];
    swapped[i + 1] = bytes[i];
  }
  if (bytes.length % 2) swapped[bytes.length - 1] = bytes[bytes.length - 1];
  return new TextDecoder('utf-16le').decode(swapped);
}

function guessUTF16(bytes) {
  const n = Math.min(bytes.length, 2000);
  if (n < 16) return '';
  let evenZero = 0, oddZero = 0, pairs = 0;
  for (let i = 0; i + 1 < n; i += 2) {
    if (bytes[i] === 0) evenZero++;
    if (bytes[i + 1] === 0) oddZero++;
    pairs++;
  }
  if (!pairs) return '';
  if (oddZero / pairs > 0.35 && evenZero / pairs < 0.08) return 'utf-16le';
  if (evenZero / pairs > 0.35 && oddZero / pairs < 0.08) return 'utf-16be';
  return '';
}

function textDecodeScore(text) {
  let score = 0;
  for (const ch of String(text || '')) {
    const code = ch.codePointAt(0);
    if (ch === '�') score += 20;
    else if (code === 0) score += 30;
    else if (code < 32 && ch !== '\n' && ch !== '\r' && ch !== '\t' && ch !== '\f') score += 10;
  }
  return score;
}

function looksBinaryText(text) {
  const s = String(text || '');
  if (!s) return false;
  return textDecodeScore(s) > Math.max(60, s.length * 0.03);
}

function setConnected() {}

function taskURL(taskId) {
  return `/forests/${encodeURIComponent(taskId)}`;
}

function routeTaskId() {
  const match = window.location.pathname.match(/^\/forests\/([^/]+)\/?$/);
  return match ? decodeURIComponent(match[1]) : '';
}

function setRoute(path, replace = false) {
  if (window.location.pathname === path) return;
  const method = replace ? 'replaceState' : 'pushState';
  window.history[method]({}, '', path);
}

async function syncFromRoute() {
  const taskId = routeTaskId();
  if (taskId) {
    await selectTask(taskId, { fromRoute: true, replaceRoute: true });
  } else if (state.activeTaskId) {
    backToList({ fromRoute: true });
  }
}


async function loadHealthStatus() {
  try {
    const data = await api('/api/health');
    state.powerStatus = data.power || null;
    renderPowerBanner();
  } catch (err) {
    console.error(err);
  }
}

function renderPowerBanner() {
  const el = $('powerBanner');
  if (!el) return;
  const ps = state.powerStatus;
  if (!ps || ps.ok || !Array.isArray(ps.warnings) || !ps.warnings.length) {
    el.classList.add('hidden');
    el.innerHTML = '';
    return;
  }
  const items = [...(ps.warnings || []), ...(ps.advice || [])].slice(0, 5);
  el.classList.remove('hidden');
  el.innerHTML = `<strong>${escapeHTML(t('powerWarningTitle'))}</strong><div>${escapeHTML(t('powerWarningPrefix'))}</div><ul>${items.map(x => `<li>${escapeHTML(x)}</li>`).join('')}</ul>`;
}

function normalizeCLIEngineValue(value) {
  const v = String(value || '').trim().toLowerCase().replace(/_/g, '-');
  if (['claude', 'claude-code', 'claude-cli', 'anthropic', 'cloud'].includes(v)) return 'claude';
  return 'codex';
}
function compatibleCLIEngineValue(engine, mode) {
  const cli = normalizeCLIEngineValue(engine);
  return mode === 'kimik2.6' && cli === 'codex' ? 'claude' : cli;
}
function normalizeSettingsCompatibility() {
  state.settings.cliEngine = compatibleCLIEngineValue(state.settings.cliEngine || 'codex', state.settings.modelMode || 'default');
}

function loadSettings() {
  try {
    return { defaultWorkspace: '', showSavePath: false, showWorkRecord: false, logLevel: 'quiet', language: 'zh-CN', cliEngine: 'codex', modelMode: 'default', minimaxToken: '', kimiToken: '', ...JSON.parse(localStorage.getItem('autoGardenerSettings') || '{}') };
  } catch {
    return { defaultWorkspace: '', showSavePath: false, showWorkRecord: false, logLevel: 'quiet', language: 'zh-CN', cliEngine: 'codex', modelMode: 'default', minimaxToken: '', kimiToken: '' };
  }
}

async function loadServerSettings() {
  try {
    const data = await api('/api/settings');
    state.settings.logLevel = data.settings?.logLevel || state.settings.logLevel || 'quiet';
    state.settings.cliEngine = normalizeCLIEngineValue(data.settings?.cliEngine || state.settings.cliEngine || 'codex');
    state.settings.modelMode = data.settings?.modelMode || state.settings.modelMode || 'default';
    state.settings.minimaxToken = data.settings?.minimaxToken || state.settings.minimaxToken || '';
    state.settings.kimiToken = data.settings?.kimiToken || state.settings.kimiToken || '';
    applySettings();
  } catch (err) { console.error(err); }
}

async function saveSettings() {
  normalizeSettingsCompatibility();
  localStorage.setItem('autoGardenerSettings', JSON.stringify(state.settings));
  applySettings();
  try { await api('/api/settings', { method: 'PUT', body: JSON.stringify({ logLevel: state.settings.logLevel || 'quiet', cliEngine: normalizeCLIEngineValue(state.settings.cliEngine || 'codex'), modelMode: state.settings.modelMode || 'default', minimaxToken: state.settings.minimaxToken || '', kimiToken: state.settings.kimiToken || '' }) }); } catch (err) { console.error(err); }
}

function applySettings() {
  invalidateRenderCache();
  state.lastTaskListSig = '';
  state.lastHomeSig = '';
  $('defaultWorkspaceInput').value = state.settings.defaultWorkspace || '';
  $('showSavePathToggle').checked = !!state.settings.showSavePath;
  $('showWorkRecordToggle').checked = !!state.settings.showWorkRecord;
  $('languageSelect').value = state.settings.language || 'zh-CN';
  $('logLevelSelect').value = state.settings.logLevel || 'quiet';
  normalizeSettingsCompatibility();
  $('cliEngineSelect').value = compatibleCLIEngineValue(state.settings.cliEngine || 'codex', state.settings.modelMode || 'default');
  $('modelModeSelect').value = state.settings.modelMode || 'default';
  applyModelTokenField();
  document.body.classList.toggle('hide-save-path', false);
  document.body.classList.toggle('hide-work-record', !state.settings.showWorkRecord);
  if (!$('workspaceInput').value.trim()) $('workspaceInput').value = state.settings.defaultWorkspace || '';
  applyI18n();
  renderPowerBanner();
  if (Array.isArray(state.tasks)) {
    renderTaskList();
    renderHomeGarden();
    const active = state.tasks.find(task => task.id === state.activeTaskId);
    if (active) renderTask(active);
  }
}

function currentModelToken() {
  const mode = state.settings.modelMode || 'default';
  if (mode === 'minimaxm2.7') return state.settings.minimaxToken || '';
  if (mode === 'kimik2.6') return state.settings.kimiToken || '';
  return '';
}

function setCurrentModelToken(token) {
  const mode = state.settings.modelMode || 'default';
  if (mode === 'minimaxm2.7') state.settings.minimaxToken = token;
  if (mode === 'kimik2.6') state.settings.kimiToken = token;
}

function applyModelTokenField() {
  const mode = state.settings.modelMode || 'default';
  const section = $('modelTokenSection');
  const input = $('modelTokenInput');
  if (!section || !input) return;
  section.classList.toggle('hidden', mode === 'default');
  input.value = currentModelToken();
}


async function loadTasks(options = {}) {
  try {
    const data = await api('/api/tasks?compact=1');
    state.tasks = data.tasks || [];
    setConnected(true);
    renderTaskList();
    renderHomeGarden();
    if (options.syncRoute) await syncFromRoute();
    else if (state.activeTaskId) await loadActiveTask(false);
  } catch (err) { setConnected(false); console.error(err); }
}

async function loadActiveTask(render = true) {
  if (!state.activeTaskId) return false;
  try {
    const data = await api(`/api/tasks/${state.activeTaskId}`);
    const previous = state.tasks.find(t => t.id === data.task.id);
    upsertTask(data.task);
    if (render) renderTask(data.task, { skipFileViewer: shouldSkipFileViewer(previous, data.task) });
    if (data.task.status === 'Finished') stopActiveRefreshPoller();
    return true;
  } catch (err) { console.error(err); return false; }
}

function stopActiveRefreshPoller() {
  if (state.activeRefreshPoller) {
    clearInterval(state.activeRefreshPoller);
    state.activeRefreshPoller = null;
  }
}

function upsertTask(task) {
  const idx = state.tasks.findIndex(t => t.id === task.id);
  if (idx >= 0) state.tasks[idx] = task; else state.tasks.unshift(task);
  renderTaskList();
  renderHomeGarden();
}


function viewportRefreshMs() {
  return isCompactViewport() ? 12000 : 8000;
}

function taskListSignature(tasks) {
  return (tasks || []).map(t => [t.id, t.title || '', t.status || '', (t.trees || []).length].join(':')).join('|');
}

function visibleMessagesForViewport(messages) {
  const maxMessages = isCompactViewport() ? 50 : 140;
  const all = Array.isArray(messages) ? messages : [];
  return { all, visibleMessages: all.length > maxMessages ? all.slice(-maxMessages) : all, maxMessages };
}

function messageSignature(messages, task) {
  const { all, visibleMessages, maxMessages } = visibleMessagesForViewport(messages);
  const rows = visibleMessages.map(m => [m.id || '', m.role || '', m.createdAt || '', String(m.content || '').length, String(m.content || '').slice(0, 80)].join('~')).join('|');
  return [task?.status || '', all.length, maxMessages, rows].join('::');
}

function progressSignature(task) {
  const raw = Array.isArray(task?.gardenerProgress) ? task.gardenerProgress : [];
  return [task?.status || '', task?.gardenerStatus || '', task?.lastProgressAt || '', raw.slice(-10).join('|')].join('::');
}

function forestSignature(task) {
  const selected = state.selectedForests[task.id] || '';
  const trees = (task.trees || []).map(tree => [tree.id, tree.forest || 1, tree.status || '', tree.fruitPath || '', !!tree.isValidation, tree.updatedAt || ''].join(':')).join('|');
  return [selected, trees].join('::');
}

function overviewSignature(task) {
  const usage = state.usage[task.id];
  return [state.overviewCollapsed ? 1 : 0, task.status || '', forestSignature(task), usage?.totalTokens || 0, task.runtime?.phase || '', task.runtime?.severity || '', task.runtime?.idleSeconds || 0].join('::');
}

function scheduleTaskRender(task, options = {}) {
  state.pendingTaskRender = { task, options: { ...(state.pendingTaskRender?.options || {}), ...options } };
  if (state.pendingTaskRenderFrame) return;
  const run = () => {
    state.pendingTaskRenderFrame = 0;
    const pending = state.pendingTaskRender;
    state.pendingTaskRender = null;
    if (pending?.task && pending.task.id === state.activeTaskId) renderTask(pending.task, pending.options || {});
  };
  state.pendingTaskRenderFrame = window.requestAnimationFrame ? requestAnimationFrame(run) : setTimeout(run, 16);
}

function invalidateRenderCache(taskId = '') {
  if (taskId) delete state.renderCache[taskId];
  else state.renderCache = {};
}

function renderTaskList() {
  const list = $('taskList');
  const sig = `${state.activeTaskId || ''}::${state.settings.language || ''}::${taskListSignature(state.tasks)}`;
  if (state.lastTaskListSig === sig && list.childNodes.length) return;
  state.lastTaskListSig = sig;
  list.innerHTML = '';
  if (!state.tasks.length) { list.innerHTML = `<div class="empty-list">${t('noTasks')}</div>`; return; }
  state.tasks.forEach(task => {
    const item = document.createElement('button');
    item.className = 'task-item' + (task.id === state.activeTaskId ? ' active' : '');
    item.innerHTML = `<span class="task-title">${escapeHTML(task.title)}</span><span class="task-meta">${statusText(task.status)}</span>`;
    item.onclick = () => selectTask(task.id);
    list.appendChild(item);
  });
}

async function selectTask(taskId, options = {}) {
  state.activeTaskId = taskId;
  invalidateRenderCache(taskId);
  resetFileViewerForTask(taskId);
  $('appShell').classList.add('focused');
  $('backBtn').classList.remove('hidden');
  $('renameTaskBtn').classList.remove('hidden');
  renderTaskList();
  if (!options.fromRoute) setRoute(taskURL(taskId), !!options.replaceRoute);
  const ok = await loadActiveTask(true);
  if (!ok) {
    backToList({ replaceRoute: true });
    return;
  }
  connectEvents(taskId);
}

function backToList(options = {}) {
  state.activeTaskId = null;
  if (state.pendingTaskRenderFrame) { try { cancelAnimationFrame(state.pendingTaskRenderFrame); } catch {} state.pendingTaskRenderFrame = 0; state.pendingTaskRender = null; }
  $('appShell').classList.remove('focused');
  $('backBtn').classList.add('hidden');
  $('renameTaskBtn').classList.add('hidden');
  cancelTitleEdit(false);
  $('forestView').classList.add('hidden');
  $('emptyState').classList.remove('hidden');
  $('pageTitle').textContent = 'Gardener';
  if (state.eventSource) state.eventSource.close();
  if (state.recoveryPoller) { clearInterval(state.recoveryPoller); state.recoveryPoller = null; }
  if (state.activeRefreshPoller) { clearInterval(state.activeRefreshPoller); state.activeRefreshPoller = null; }
  if (!options.fromRoute) setRoute('/', !!options.replaceRoute);
  renderTaskList();
  renderHomeGarden();
}

function connectEvents(taskId) {
  if (state.eventSource) state.eventSource.close();
  if (state.recoveryPoller) { clearInterval(state.recoveryPoller); state.recoveryPoller = null; }
  if (state.activeRefreshPoller) { clearInterval(state.activeRefreshPoller); state.activeRefreshPoller = null; }
  if (!window.EventSource) { state.recoveryPoller = setInterval(() => { if (!document.hidden) loadActiveTask(true); }, isCompactViewport() ? 4000 : 2000); return; }
  state.activeRefreshPoller = setInterval(() => {
    if (state.activeTaskId === taskId && !document.hidden) loadActiveTask(true);
  }, viewportRefreshMs());
  const es = new EventSource(`/api/tasks/${taskId}/events`);
  state.eventSource = es;
  es.addEventListener('open', () => setConnected(true));
  es.addEventListener('task', (ev) => {
    const task = JSON.parse(ev.data);
    const previous = state.tasks.find(t => t.id === task.id);
    upsertTask(task);
    if (task.id === state.activeTaskId) {
      scheduleTaskRender(task, { skipFileViewer: shouldSkipFileViewer(previous, task) });
      if (task.status === 'Finished') stopActiveRefreshPoller();
    }
  });
  es.onerror = () => setConnected(false);
}



function treeSpriteHTML(seed, size = 'mini') {
  const variants = ['round', 'pine', 'bloom', 'gold', 'sprout'];
  const variant = variants[Math.abs(seed) % variants.length];
  return `<span class="forest-tree ${size} ${variant}"><span class="tree-shadow"></span><span class="tree-trunk"></span><span class="tree-crown c1"></span><span class="tree-crown c2"></span><span class="tree-crown c3"></span></span>`;
}

function renderHomeGarden() {
  const list = $('homeForestList');
  if (!list) return;
  const tasks = state.tasks || [];
  const sig = `${state.settings.language || ''}::${taskListSignature(tasks.slice(0, 12))}`;
  if (state.lastHomeSig === sig && list.childNodes.length) return;
  state.lastHomeSig = sig;
  list.innerHTML = '';
  if (!tasks.length) {
    list.innerHTML = `<div class="home-forest-empty">${t('noRecent')}</div>`;
    return;
  }
  tasks.slice(0, 12).forEach(task => {
    const item = document.createElement('div');
    item.className = 'home-forest-item';
    item.setAttribute('role', 'button');
    item.tabIndex = 0;
    const forests = getForests(task.trees || []);
    item.innerHTML = `
      <button type="button" class="home-forest-delete" title="${t('delete')}" aria-label="${t('delete')}">×</button>
      <span class="home-forest-title">${escapeHTML(task.title || t('genericTask'))}</span>
      <span class="home-forest-meta"><b>${statusText(task.status)}</b>${forests.length ? ` · ${t('stage')} ${forests.length}` : ''}</span>
    `;
    item.onclick = () => selectTask(task.id);
    item.onkeydown = (e) => { if (e.key === 'Enter' || e.key === ' ') { e.preventDefault(); selectTask(task.id); } };
    item.querySelector('.home-forest-delete').onclick = (e) => { e.stopPropagation(); deleteTask(task.id); };
    list.appendChild(item);
  });
}

function renderTask(task, options = {}) {
  $('emptyState').classList.add('hidden');
  $('forestView').classList.remove('hidden');
  ensureSelectedForest(task);
  const cache = state.renderCache[task.id] || (state.renderCache[task.id] = {});
  const chromeSig = [task.title || '', task.status || '', state.settings.language || ''].join('::');
  if (cache.chromeSig !== chromeSig) {
    cache.chromeSig = chromeSig;
    if (!state.editingTitle) $('pageTitle').textContent = task.title;
    $('forestStatus').textContent = statusText(task.status);
    $('forestStatus').className = 'status-pill ' + task.status;
    setTaskReportLink($('scheduleLink'), `/api/tasks/${task.id}/gardener/schedule.md`, t('taskPlan'));
    setTaskReportLink($('logLink'), `/api/tasks/${task.id}/gardener/log.md`, t('workRecord'));
    $('stopTaskBtn').disabled = task.status === 'Finished';
    const resumeBtn = $('resumeTaskBtn');
    if (resumeBtn) {
      const resumable = task.status === 'Finished';
      resumeBtn.classList.toggle('hidden', !resumable);
      resumeBtn.disabled = !resumable;
      resumeBtn.title = t('resumeTaskHint');
    }
  }
  const forestSig = forestSignature(task);
  if (cache.forestSig !== forestSig) {
    cache.forestSig = forestSig;
    renderForest(task);
    renderTreeStatus(task);
  }
  renderTaskDashboard(task);
  const progressSig = progressSignature(task);
  if (cache.progressSig !== progressSig) {
    cache.progressSig = progressSig;
    renderGardenerProgress(task);
  }
  const msgSig = messageSignature(task.messages || [], task);
  if (cache.msgSig !== msgSig) {
    cache.msgSig = msgSig;
    renderMessages(task.messages || [], task);
  }
  const overviewSig = overviewSignature(task);
  if (cache.overviewSig !== overviewSig) {
    cache.overviewSig = overviewSig;
    applyOverviewCollapsed(task);
  }
  renderUsage(task);
  if (!options.skipFileViewer && !document.hidden) renderFileViewer(task);
}

function shouldSkipFileViewer(previous, task) {
  if (!previous || !task || previous.id !== task.id) return false;
  const sameTaskShape = fileRefreshSignature(previous) === fileRefreshSignature(task);
  if (!sameTaskShape) return false;
  const last = state.lastFileRefreshAt[task.id] || 0;
  return Date.now() - last < (isCompactViewport() ? 15000 : 6000);
}

function fileRefreshSignature(task) {
  const trees = (task.trees || []).map(tree => [tree.id, tree.forest || 1, tree.status || '', tree.fruitPath || '', !!tree.isValidation].join(':')).join('|');
  return [task.id, task.workspacePath || '', task.status || '', trees].join('::');
}


function applyOverviewCollapsed(task) {
  const card = document.querySelector('.forest-summary');
  const btn = $('toggleOverviewBtn');
  if (card) card.classList.toggle('overview-collapsed', !!state.overviewCollapsed);
  if (btn) {
    btn.textContent = state.overviewCollapsed ? t('expandOverview') : t('collapseOverview');
    btn.setAttribute('aria-expanded', state.overviewCollapsed ? 'false' : 'true');
  }
  renderOverviewMini(task);
}

function renderTaskDashboard(task) {
  const panel = $('taskDashboardPanel');
  if (!panel || !task) return;
  const rt = task.runtime || {};
  const forests = getForests(task.trees || []);
  const totalTrees = Number(rt.totalTrees ?? (task.trees || []).length);
  const runningTrees = Number(rt.runningTrees ?? (task.trees || []).filter(tr => tr.status !== 'Finished').length);
  const finishedTrees = Number(rt.finishedTrees ?? Math.max(0, totalTrees - runningTrees));
  const severity = String(rt.severity || 'ok');
  const cue = String(rt.cue || '').trim();
  const phase = humanizePhase(rt.phase || (task.status === 'Finished' ? 'finished' : 'running'));
  panel.className = `task-dashboard-panel ${severity}`;
  panel.innerHTML = `
    <div class="task-dashboard-head">
      <strong>${t('dashboard')}</strong>
      <span class="dashboard-cue-pill ${severity}">${escapeHTML(phase)}</span>
    </div>
    <div class="dashboard-grid">
      <div class="dashboard-metric"><span>${t('duration')}</span><b>${formatDuration(rt.durationSeconds || 0)}</b></div>
      <div class="dashboard-metric"><span>${t('idle')}</span><b>${formatDuration(rt.idleSeconds || 0)}</b></div>
      <div class="dashboard-metric"><span>${t('stage')}</span><b>${forests.length || task.forest || 0}</b></div>
      <div class="dashboard-metric"><span>${t('subtask')}</span><b>${runningTrees}/${finishedTrees}/${totalTrees}</b></div>
    </div>
    <div class="dashboard-cue ${severity}">
      <span>${t('diagnosis')}</span>
      <p>${escapeHTML(cue || t('askProgressSafe'))}</p>
    </div>
  `;
}

function humanizePhase(phase) {
  const lang = state.settings?.language || 'zh-CN';
  const zh = {
    planning:'规划中', running_subtasks:'子任务执行中', validating:'验证中', deciding:'判断下一步', running:'运行中', finished:'已完成', stopped:'已停止', unknown:'未知'
  };
  const en = {
    planning:'Planning', running_subtasks:'Running subtasks', validating:'Validating', deciding:'Deciding', running:'Running', finished:'Finished', stopped:'Stopped', unknown:'Unknown'
  };
  return (lang === 'en' ? en : zh)[phase] || phase;
}

function formatDuration(seconds) {
  seconds = Math.max(0, Number(seconds || 0));
  const m = Math.floor(seconds / 60);
  const h = Math.floor(m / 60);
  const d = Math.floor(h / 24);
  if (d > 0) return `${d}d ${h % 24}h`;
  if (h > 0) return `${h}h ${m % 60}m`;
  if (m > 0) return `${m}m`;
  return `${Math.floor(seconds)}s`;
}

function renderOverviewMini(task) {

  const panel = $('overviewMiniPanel');
  if (!panel || !task) return;
  panel.classList.toggle('hidden', !state.overviewCollapsed);
  if (!state.overviewCollapsed) {
    panel.innerHTML = '';
    return;
  }
  const forests = getForests(task.trees || []);
  const selected = ensureSelectedForest(task);
  const forest = forests.find(o => o.no === selected) || forests[forests.length - 1];
  const items = forest?.items || [];
  const running = items.filter(tree => tree.status !== 'Finished').length;
  const finished = items.length - running;
  const usage = state.usage[task.id];
  const tokens = Number(usage?.totalTokens || 0);
  const progressLabel = task.status === 'Finished' ? statusText('Finished') : t('gardenerWorking');
  const tokenLabel = tokens ? formatTokenCount(tokens) : '—';
  panel.innerHTML = `
    <span class="overview-mini-title">${t('overview')}</span>
    <span class="overview-mini-chip ${task.status || 'Running'}">${t('gardenerProgress')} · ${progressLabel}</span>
    <span class="overview-mini-chip">${t('tokenUsage')} · ${tokenLabel}</span>
    <span class="overview-mini-chip">${t('treeStatus')} · ${statusText('Running')} ${running} / ${statusText('Finished')} ${finished}</span>
  `;
}

function toggleOverviewCollapsed() {
  state.overviewCollapsed = !state.overviewCollapsed;
  saveOverviewCollapsed();
  const active = state.tasks.find(task => task.id === state.activeTaskId);
  if (active) applyOverviewCollapsed(active);
}

function beginTitleEdit() {
  const task = state.tasks.find(t => t.id === state.activeTaskId);
  if (!task) return;
  state.editingTitle = true;
  const wrap = document.querySelector('.title-editor');
  const input = $('titleEditInput');
  if (wrap) wrap.classList.add('editing');
  input.value = task.title || '';
  input.classList.remove('hidden');
  $('renameTaskBtn').textContent = '✓';
  $('renameTaskBtn').setAttribute('aria-label', t('save'));
  $('renameTaskBtn').title = t('save');
  requestAnimationFrame(() => { input.focus(); input.select(); });
}

function cancelTitleEdit(restoreTitle = true) {
  if (!state.editingTitle && restoreTitle) return;
  state.editingTitle = false;
  const wrap = document.querySelector('.title-editor');
  const input = $('titleEditInput');
  if (wrap) wrap.classList.remove('editing');
  if (input) input.classList.add('hidden');
  const btn = $('renameTaskBtn');
  if (btn) {
    btn.textContent = '✎';
    btn.setAttribute('aria-label', t('rename'));
    btn.title = t('rename');
    btn.disabled = false;
  }
  if (restoreTitle) {
    const task = state.tasks.find(t => t.id === state.activeTaskId);
    if (task) $('pageTitle').textContent = task.title || t('genericTask');
  }
}

function isValidRenameTaskPayload(task) {
  return !!task && typeof task === 'object' && typeof task.id === 'string' && task.id.trim() !== '';
}

async function commitTitleEdit() {
  if (!state.editingTitle) return;
  const task = state.tasks.find(t => t.id === state.activeTaskId);
  if (!task) return cancelTitleEdit(false);
  const input = $('titleEditInput');
  const next = input.value.trim();
  if (!next || next === task.title) return cancelTitleEdit(true);
  $('renameTaskBtn').disabled = true;
  try {
    const data = await api(`/api/tasks/${task.id}`, { method:'PATCH', body: JSON.stringify({ title: next }) });
    if (!isValidRenameTaskPayload(data.task)) throw new Error('Invalid task response');
    state.editingTitle = false;
    cancelTitleEdit(false);
    upsertTask(data.task);
    renderTask(data.task);
  } catch(err) {
    alert((t('renameFailed') || 'Rename failed: ') + err.message);
    $('renameTaskBtn').disabled = false;
    input.focus();
  }
}

function getForests(trees) {
  const groups = new Map();
  trees.forEach(tree => {
    const no = Number(tree.forest || 1);
    if (!groups.has(no)) groups.set(no, []);
    groups.get(no).push(tree);
  });
  return [...groups.entries()].sort((a, b) => a[0] - b[0]).map(([no, items]) => {
    const finished = items.filter(t => t.status === 'Finished').length;
    return { no, items, finished, fruit: items.filter(t => t.fruitPath).length, status: finished === items.length ? 'Finished' : 'Running' };
  });
}

function ensureSelectedForest(task) {
  const forests = getForests(task.trees || []);
  if (!forests.length) { delete state.selectedForests[task.id]; return null; }
  const selected = state.selectedForests[task.id];
  if (!forests.some(o => o.no === selected)) state.selectedForests[task.id] = forests[forests.length - 1].no;
  return state.selectedForests[task.id];
}

function renderForest(task) {
  const select = $('forestSelect');
  const prev = $('prevForestBtn');
  const next = $('nextForestBtn');
  if (!select) return;
  const forests = getForests(task.trees || []);
  const selected = ensureSelectedForest(task);
  select.innerHTML = '';
  if (!forests.length) {
    const opt = document.createElement('option');
    opt.value = '';
    opt.textContent = t('waitingForest');
    select.appendChild(opt);
    select.disabled = true;
    if (prev) prev.disabled = true;
    if (next) next.disabled = true;
    $('treeSummaryText').textContent = '';
    return;
  }
  select.disabled = false;
  forests.forEach(forest => {
    const opt = document.createElement('option');
    opt.value = String(forest.no);
    opt.textContent = forests.length > 8 ? `${t('stage')} ${forest.no} / ${forests.length}` : `${t('stage')} ${forest.no}`;
    select.appendChild(opt);
  });
  select.value = String(selected || forests[forests.length - 1].no);
  const currentIndex = Math.max(0, forests.findIndex(o => String(o.no) === select.value));
  if (prev) {
    prev.disabled = currentIndex <= 0;
    prev.onclick = () => setSelectedForest(task, forests[Math.max(0, currentIndex - 1)].no);
  }
  if (next) {
    next.disabled = currentIndex >= forests.length - 1;
    next.onclick = () => setSelectedForest(task, forests[Math.min(forests.length - 1, currentIndex + 1)].no);
  }
  $('treeSummaryText').textContent = `${t('stage')} ${select.value}`;
  select.onchange = () => setSelectedForest(task, Number(select.value));
}

function setSelectedForest(task, forestNo) {
  state.selectedForests[task.id] = Number(forestNo);
  delete state.selectedFilePath[task.id];
  state.selectedFileTree[task.id] = '';
  renderForest(task);
  renderUsage(task);
  renderTreeStatus(task);
  renderFileViewer(task);
}


async function deleteTask(taskId) {
  if (!confirm(t('deleteConfirm'))) return;
  try {
    await api(`/api/tasks/${taskId}`, { method:'DELETE' });
    state.tasks = state.tasks.filter(task => task.id !== taskId);
    if (state.activeTaskId === taskId) backToList({ replaceRoute: true });
    renderTaskList();
    renderHomeGarden();
  } catch (err) {
    alert((t('deleteFailed') || 'Delete failed: ') + err.message);
  }
}

async function renderUsage(task) {
  const panel = $('usagePanel');
  if (!panel || !task?.id) return;
  const cached = state.usage[task.id];
  if (cached) paintUsage(panel, cached);
  const now = Date.now();
  const freshFor = task.status === 'Running' ? 12000 : 60000;
  if (state.usagePending[task.id] || (cached && now - (state.usageFetchedAt[task.id] || 0) < freshFor)) return;
  state.usagePending[task.id] = true;
  try {
    const data = await api(`/api/tasks/${task.id}/usage`);
    if (task.id !== state.activeTaskId) return;
    state.usage[task.id] = data.usage;
    state.usageFetchedAt[task.id] = Date.now();
    paintUsage(panel, data.usage);
    const active = state.tasks.find(t => t.id === task.id);
    if (active) renderOverviewMini(active);
  } catch (err) {
    console.error(err);
    panel.classList.add('hidden');
  } finally {
    delete state.usagePending[task.id];
  }
}

function paintUsage(panel, usage) {
  const total = Number(usage?.totalTokens || 0);
  if (!total) {
    panel.classList.add('hidden');
    panel.innerHTML = '';
    return;
  }
  panel.classList.remove('hidden');
  const models = Array.isArray(usage.models) ? usage.models.slice(0, 3) : [];
  const modelHTML = models.map(m => `
    <span class="usage-model">
      <span class="usage-model-name">${escapeHTML(m.model || 'unknown')}</span>
      <span>${formatTokenCount(m.totalTokens)}</span>
    </span>`).join('');
  panel.innerHTML = `
    <div class="usage-total">
      <span>${t('tokenUsage')}</span>
      <strong>${formatTokenCount(total)}</strong>
    </div>
    <div class="usage-models">${modelHTML}</div>
  `;
  panel.title = usage.pricingNote || '';
}

function formatTokenCount(n) {
  n = Number(n || 0);
  if (n >= 1000000) return `${(n / 1000000).toFixed(n >= 10000000 ? 1 : 2)}M`;
  if (n >= 1000) return `${(n / 1000).toFixed(n >= 100000 ? 0 : 1)}K`;
  return n.toLocaleString();
}




function renderGardenerProgress(task) {
  const panel = $('gardenerProgressPanel');
  if (!panel) return;
  const raw = Array.isArray(task?.gardenerProgress) ? task.gardenerProgress : [];
  const rows = compactDisplayProgress(raw).slice(-6);
  const running = task?.gardenerStatus !== 'Finished' && task?.status !== 'Finished';
  if (!rows.length && !running) {
    panel.classList.add('hidden');
    panel.innerHTML = '';
    return;
  }
  panel.className = `gardener-progress-panel${running ? ' running' : ''}`;
  const idleMinutes = running ? inactiveMinutes(task) : 0;
  const stale = running && idleMinutes >= 5;
  const staleLabel = t('noOutputMinutes').replace('%d', idleMinutes);
  const status = running ? `<span class="gardener-progress-live${stale ? ' stale' : ''}"><span></span>${stale ? staleLabel : t('gardenerWorking')}</span>` : `<span class="status-pill Finished">${statusText('Finished')}</span>`;
  const items = rows.length
    ? rows.map(line => {
        const parsed = parseProgressDisplayLine(line);
        return `<li><time>${escapeHTML(parsed.time)}</time><span>${escapeHTML(humanizeText(parsed.text))}</span></li>`;
      }).join('')
    : `<li class="empty-progress"><span>${t('gardenerProgressEmpty')}</span></li>`;
  const staleHint = stale ? `<li class="empty-progress progress-stale-hint"><span>${escapeHTML(t('noOutputStale'))}</span></li>` : '';
  panel.innerHTML = `
    <div class="gardener-progress-head">
      <strong>${t('gardenerProgress')}</strong>
      ${status}
    </div>
    <ol class="gardener-progress-list">${items}${staleHint}</ol>
  `;
}

function inactiveMinutes(task) {
  const times = [];
  if (task?.lastProgressAt) times.push(new Date(task.lastProgressAt).getTime());
  (task?.trees || []).forEach(tr => {
    if (tr?.updatedAt) times.push(new Date(tr.updatedAt).getTime());
    if (tr?.completedAt) times.push(new Date(tr.completedAt).getTime());
    if (tr?.startedAt) times.push(new Date(tr.startedAt).getTime());
  });
  const latest = Math.max(...times.filter(n => Number.isFinite(n) && n > 0));
  if (!Number.isFinite(latest)) return 0;
  return Math.max(0, Math.floor((Date.now() - latest) / 60000));
}

function compactDisplayProgress(lines) {
  const out = [];
  let last = '';
  (lines || []).forEach(line => {
    const parsed = parseProgressDisplayLine(line);
    let text = normalizeGardenerProgressText(parsed.text);
    if (!text || isLowValueGardenerProgress(text)) return;
    text = summarizeGardenerProgressText(text);
    if (!text || text === last) return;
    last = text;
    out.push(parsed.time ? `${parsed.time} ${text}` : text);
  });
  return out;
}

function parseProgressDisplayLine(line) {
  const s = String(line || '').trim();
  const m = s.match(/^(\d{2}:\d{2})(?::\d{2})?\s+(.+)$/);
  if (m) return { time: m[1], text: m[2] };
  return { time: '', text: s };
}

function normalizeGardenerProgressText(text) {
  return String(text || '')
    .replace(/\[[0-9]{4}-[0-9]{2}-[0-9]{2}T[^\]]+\]\s*/g, '')
    .replace(/^(初始化|规划|判断)：\s*\1：\s*/, '$1：')
    .trim();
}

function summarizeGardenerProgressText(text) {
  const s = normalizeGardenerProgressText(text);
  if (s.startsWith('初始化：Gardener Git 初始化输出摘要') || s.startsWith('Gardener Git 初始化输出摘要')) return '初始化：保存位置已准备';
  if (s.startsWith('规划：本次用户指令')) return '规划：已收到新的用户指令';
  if (s.startsWith('判断：本次用户指令')) return '判断：正在结合新的用户指令评估结果';
  const stage = s.match(/^(初始化|规划|判断)：/);
  if (stage && /"(prompt|objective|scope|trees|message_to_user|forest_finished)"\s*:/.test(s)) {
    if (stage[1] === '规划') return '规划：正在整理子任务范围';
    if (stage[1] === '判断') return '判断：正在评估成果并决定下一步';
    return '初始化：正在准备保存位置';
  }
  if (/^(规划|判断)：\s*\{/.test(s)) return s.startsWith('规划') ? '规划：正在形成任务安排' : '判断：正在评估任务结果';
  const runes = Array.from(s);
  return runes.length > 180 ? runes.slice(0, 180).join('') + '…' : s;
}

function isLowValueGardenerProgress(text) {
  const s = String(text || '').trim();
  const lower = s.toLowerCase();
  const lowValueStarts = [
    '你是 gardener', '规则：', '- 你的工作目录', '- 如果 workspacepath', '- 如果目录中已有文件',
    '- 如果 commit', '- 如果目录为空', '- 允许你自主', 'workspacepath:', '请在当前 workspace',
    '初始化：你是 gardener', '初始化：规则：', '初始化：- 你的工作目录', '初始化：- 如果 workspacepath',
    '初始化：- 如果目录中已有文件', '初始化：- 如果 commit', '初始化：- 如果目录为空', '初始化：- 允许你自主',
    '初始化：workspacepath:', '规划：你是 gardener', '判断：你是 gardener'
  ];
  if (lowValueStarts.some(prefix => lower.startsWith(prefix))) return true;
  return /^初始化：[-•]\s/.test(s) || /^规划：[-•]\s/.test(s) || /^判断：[-•]\s/.test(s);
}

function renderTreeStatus(task) {
  const panel = $('treeStatusPanel');
  if (!panel) return;
  const forests = getForests(task.trees || []);
  const selected = ensureSelectedForest(task);
  const forest = forests.find(o => o.no === selected);
  if (!forest || !forest.items.length) {
    panel.classList.add('hidden');
    panel.innerHTML = '';
    return;
  }
  const key = `${task.id}:${forest.no}`;
  const expanded = !!state.treeStatusExpanded[key];
  const running = forest.items.filter(tree => tree.status !== 'Finished').length;
  const finished = forest.items.length - running;
  const dots = forest.items.slice(0, 24).map(tree => `<span class="tree-mini-dot ${tree.status || 'Running'}${tree.isValidation ? ' validation' : ''}" title="${escapeHTML(humanizeText(tree.name || t('subtask')))}"></span>`).join('');
  panel.className = `tree-status-panel compact${expanded ? ' expanded' : ''}`;
  panel.innerHTML = `
    <div class="tree-status-main">
      <span class="tree-status-title">${t('treeStatus')}</span>
      <span class="tree-status-count">${statusText('Running')} ${running} · ${statusText('Finished')} ${finished}</span>
      <span class="tree-status-dots">${dots}</span>
      <button type="button" class="tree-status-toggle">${expanded ? t('hideStatus') : t('viewStatus')}</button>
    </div>
    <div class="tree-status-list"></div>`;
  panel.querySelector('.tree-status-toggle').onclick = () => { state.treeStatusExpanded[key] = !expanded; renderTreeStatus(task); };
  const list = panel.querySelector('.tree-status-list');
  if (!expanded) return;
  forest.items.forEach(tree => {
    const item = document.createElement('div');
    item.className = `tree-status-chip ${tree.status || 'Running'}${tree.isValidation ? ' validation' : ''}`;
    const name = humanizeText(tree.name || (tree.isValidation ? t('validationTeam') : t('subtask')));
    item.innerHTML = `<span class="tree-dot"></span><span class="tree-status-name">${escapeHTML(name)}</span><span class="tree-status-pill">${statusText(tree.status)}</span>`;
    list.appendChild(item);
  });
}

function resetFileViewerForTask(taskId) {
  state.fileViewerToken++;
  state.previewToken++;
  const fileSelect = $('fileSelect');
  const preview = $('filePreview');
  if (fileSelect) {
    fileSelect.innerHTML = `<option value="">${t('loadingFiles')}</option>`;
    fileSelect.disabled = true;
  }
  if (preview) {
    preview.className = 'file-preview plain-preview';
    preview.textContent = t('selectFile');
  }
}

function isActiveFileRender(taskId, token) {
  return state.activeTaskId === taskId && (!token || token === state.fileViewerToken);
}

function fileListFingerprint(files) {
  return (files || []).map(f => `${f.path}:${f.size || 0}:${f.modTime || ''}`).join('|');
}

function newestFile(files) {
  return (files || []).slice().sort((a, b) => new Date(b.modTime || 0) - new Date(a.modTime || 0) || String(a.path).localeCompare(String(b.path)))[0] || null;
}

function latestFruitReport(task) {
  const trees = (task?.trees || []).filter(tr => tr && tr.fruitPath);
  trees.sort((a, b) => new Date(b.completedAt || b.updatedAt || 0) - new Date(a.completedAt || a.updatedAt || 0));
  return trees[0] || null;
}

async function previewLatestReport(task, token) {
  const preview = $('filePreview');
  const fileSelect = $('fileSelect');
  const tr = latestFruitReport(task);
  if (!tr) {
    if (preview && isActiveFileRender(task.id, token)) preview.textContent = task.status === 'Running' ? t('noOutputYet') : t('noFiles');
    return;
  }
  if (fileSelect && isActiveFileRender(task.id, token)) {
    fileSelect.innerHTML = `<option value="__report__">${t('result')}</option>`;
    fileSelect.disabled = true;
  }
  if (preview && isActiveFileRender(task.id, token)) {
    preview.className = 'file-preview markdown-preview';
    preview.textContent = t('loadingFiles');
  }
  try {
    const text = await fetchText(`/api/tasks/${task.id}/trees/${tr.id}/fruit.md`);
    if (!isActiveFileRender(task.id, token)) return;
    preview.className = 'file-preview markdown-preview';
    preview.innerHTML = renderMarkdown(text || t('emptyResult'));
  } catch (err) {
    if (!isActiveFileRender(task.id, token)) return;
    preview.className = 'file-preview plain-preview';
    preview.textContent = `${t('openFailed')}${err.message}`;
  }
}

async function renderFileViewer(task) {
  const filter = $('fileTreeFilter');
  const fileSelect = $('fileSelect');
  const list = $('fileList');
  const preview = $('filePreview');
  if (!filter || !fileSelect || !preview || !task) return;
  const token = ++state.fileViewerToken;
  state.lastFileRefreshAt[task.id] = Date.now();

  const allTrees = (task.trees || []).filter(tree => !tree.isValidation);
  const currentFilter = state.selectedFileTree[task.id] || '';
  const prevValue = filter.value;

  filter.innerHTML = `<option value="">${t('allTreeFiles')}</option>`;
  allTrees.forEach(tree => {
    const opt = document.createElement('option');
    opt.value = tree.id;
    opt.textContent = humanizeText(tree.name || tree.id);
    filter.appendChild(opt);
  });
  const nextFilter = currentFilter || prevValue || '';
  filter.value = allTrees.some(t => t.id === nextFilter) ? nextFilter : '';
  state.selectedFileTree[task.id] = filter.value;
  filter.disabled = allTrees.length === 0;
  filter.onchange = () => {
    state.selectedFileTree[task.id] = filter.value;
    delete state.selectedFilePath[task.id];
    state.selectedFileManual[task.id] = false;
    renderFileViewer(task);
  };

  if (list) list.innerHTML = '';
  fileSelect.innerHTML = `<option value="">${t('loadingFiles')}</option>`;
  fileSelect.disabled = true;
  preview.className = 'file-preview plain-preview';
  preview.textContent = task.status === 'Running' ? t('noOutputYet') : t('selectFile');
  try {
    const params = new URLSearchParams();
    if (filter.value) params.set('treeId', filter.value);
    const qs = params.toString() ? `?${params.toString()}` : '';
    const data = await api(`/api/tasks/${task.id}/files${qs}`);
    if (!isActiveFileRender(task.id, token)) return;
    const files = data.files || [];
    fileSelect.innerHTML = '';
    if (!files.length) {
      const opt = document.createElement('option');
      opt.value = '';
      opt.textContent = t('noFiles');
      fileSelect.appendChild(opt);
      fileSelect.disabled = true;
      await previewLatestReport(task, token);
      return;
    }
    const fingerprint = fileListFingerprint(files);
    const previousFingerprint = state.fileListFingerprint[task.id] || '';
    state.fileListFingerprint[task.id] = fingerprint;
    const manual = !!state.selectedFileManual[task.id];
    let selected = state.selectedFilePath[task.id];
    const exists = selected && files.some(f => f.path === selected);
    if (!exists || (!manual && fingerprint !== previousFingerprint)) {
      const latest = newestFile(files);
      selected = latest ? latest.path : files[0].path;
    }
    state.selectedFilePath[task.id] = selected;
    files.forEach(file => {
      const opt = document.createElement('option');
      opt.value = file.path;
      opt.textContent = displayFilePath(file.path);
      opt.title = `${file.path} · ${formatBytes(file.size || 0)}`;
      fileSelect.appendChild(opt);
    });
    fileSelect.value = selected;
    fileSelect.disabled = files.length <= 1;
    fileSelect.onchange = () => {
      state.selectedFilePath[task.id] = fileSelect.value;
      state.selectedFileManual[task.id] = true;
      previewFile(task.id, fileSelect.value, token);
    };
    await previewFile(task.id, selected, token);
  } catch (err) {
    if (!isActiveFileRender(task.id, token)) return;
    fileSelect.innerHTML = `<option value="">${t('openFailed')}${escapeHTML(err.message)}</option>`;
    fileSelect.disabled = true;
    preview.textContent = `${t('openFailed')}${err.message}`;
  }
}

async function previewFile(taskId, path, renderToken = 0) {
  const preview = $('filePreview');
  const previewToken = ++state.previewToken;
  if (!preview || state.activeTaskId !== taskId) return;
  preview.className = 'file-preview plain-preview';
  preview.textContent = t('loadingFiles');
  try {
    if (isBinaryPreviewPath(path)) {
      const href = `/api/tasks/${taskId}/files?path=${encodeURIComponent(path)}&download=1`;
      if (state.activeTaskId !== taskId || previewToken !== state.previewToken || (renderToken && renderToken !== state.fileViewerToken)) return;
      preview.className = 'file-preview plain-preview';
      preview.innerHTML = `<div class="file-empty">${escapeHTML(t('binaryFile'))}<br><br><a class="primary small" href="${href}" target="_blank" rel="noopener">${escapeHTML(t('downloadFile'))}</a></div>`;
      return;
    }
    const rawText = await fetchText(`/api/tasks/${taskId}/files?path=${encodeURIComponent(path)}`);
    const { text, notice } = trimPreviewText(rawText, path);
    const withNotice = html => notice ? `<div class="preview-note">${escapeHTML(notice)}</div>${html}` : html;
    if (state.activeTaskId !== taskId || previewToken !== state.previewToken || (renderToken && renderToken !== state.fileViewerToken)) return;
    if (isMarkdownPath(path)) {
      preview.className = 'file-preview markdown-preview';
      preview.innerHTML = withNotice(renderMarkdown(text));
    } else if (isCSVPath(path)) {
      preview.className = 'file-preview csv-preview';
      preview.innerHTML = withNotice(renderCSV(text));
    } else if (isJSONPath(path)) {
      preview.className = 'file-preview code-preview json-preview';
      preview.innerHTML = withNotice(renderJSON(text));
    } else if (isHTMLPath(path)) {
      preview.className = 'file-preview code-preview html-preview';
      preview.innerHTML = withNotice(renderCode(text, 'html'));
    } else if (isPythonPath(path)) {
      preview.className = 'file-preview code-preview python-preview';
      preview.innerHTML = withNotice(renderCode(text, 'python'));
    } else {
      preview.className = 'file-preview plain-preview';
      preview.innerHTML = withNotice(`<pre>${escapeHTML(text)}</pre>`);
    }
  } catch (err) {
    if (state.activeTaskId !== taskId || previewToken !== state.previewToken || (renderToken && renderToken !== state.fileViewerToken)) return;
    preview.className = 'file-preview plain-preview';
    preview.textContent = `${t('fileTooLarge')}：${err.message}`;
  }
}


function isCompactViewport() {
  return typeof window !== 'undefined' && window.matchMedia && window.matchMedia('(max-width: 820px)').matches;
}

function trimPreviewText(text, path = '') {
  const raw = String(text || '');
  const isStructured = isMarkdownPath(path) || isJSONPath(path) || isHTMLPath(path) || isPythonPath(path) || isCSVPath(path);
  const limit = isCompactViewport() ? (isStructured ? 180000 : 260000) : (isStructured ? 420000 : 700000);
  if (raw.length <= limit) return { text: raw, notice: '' };
  return { text: raw.slice(0, limit), notice: t('previewTruncated').replace('%d', limit) };
}

function isBinaryPreviewPath(path) {
  return /(^|\/)[^/]+\.(pdf|zip|7z|rar|gz|tar|tgz|png|jpg|jpeg|gif|webp|doc|docx|xls|xlsx|ppt|pptx)$/i.test(String(path || ''));
}

function isMarkdownPath(path) {
  return /(^|\/)[^/]+\.(md|markdown|mdown|mkd)$/i.test(String(path || ''));
}

function isCSVPath(path) {
  return /(^|\/)[^/]+\.csv$/i.test(String(path || ''));
}

function isJSONPath(path) {
  return /(^|\/)[^/]+\.(json|jsonc|jsonl)$/i.test(String(path || ''));
}

function isHTMLPath(path) {
  return /(^|\/)[^/]+\.(html|htm|xhtml)$/i.test(String(path || ''));
}

function isPythonPath(path) {
  return /(^|\/)[^/]+\.(py|pyw)$/i.test(String(path || ''));
}

function displayFilePath(path) {
  let out = String(path || '').replace(/^\/+/, '');
  const removablePrefixes = ['tree_outputs/', 'outputs/', 'output/', 'reports/', 'report/'];
  for (const prefix of removablePrefixes) {
    if (out.startsWith(prefix)) {
      out = out.slice(prefix.length);
      break;
    }
  }
  if (/^fruit\.md$/i.test(out) || /\/fruit\.md$/i.test(out)) return t('result');
  return out || String(path || '');
}

function formatBytes(n) {
  if (n < 1024) return `${n} B`;
  if (n < 1024 * 1024) return `${(n / 1024).toFixed(1)} KB`;
  return `${(n / 1024 / 1024).toFixed(1)} MB`;
}

function renderMessages(messages, task) {
  const box = $('messages'); box.innerHTML = '';
  const { all, visibleMessages, maxMessages } = visibleMessagesForViewport(messages);
  if (all.length > visibleMessages.length) {
    const note = document.createElement('div');
    note.className = 'chat-message system';
    note.innerHTML = `<div class="bubble"><div>${escapeHTML(t('recentMessagesOnly').replace('%d', maxMessages))}</div></div>`;
    box.appendChild(note);
  }
  visibleMessages.forEach(m => {
    const isUser = m.role === 'user';
    const isSystem = m.role === 'system';
    const el = document.createElement('div');
    el.className = `chat-message ${isUser ? 'user' : (isSystem ? 'system' : 'gardener')}`;
    const bubble = document.createElement('div');
    bubble.className = 'bubble';
    bubble.innerHTML = `<div>${escapeHTML(humanizeText(m.content))}</div>`;
    if (!isUser && m.createdAt) {
      const timeEl = document.createElement('time');
      timeEl.className = 'message-time';
      timeEl.dateTime = m.createdAt;
      timeEl.textContent = formatMessageDateTime(m.createdAt);
      bubble.appendChild(timeEl);
    }
    if (isSystem) {
      el.appendChild(bubble);
    } else {
      const avatar = document.createElement('div');
      avatar.className = `avatar ${isUser ? 'user-avatar' : 'gardener'}`;
      avatar.textContent = isUser ? (state.settings.language === 'en' ? 'U' : '我') : 'G';
      if (isUser) { el.appendChild(bubble); el.appendChild(avatar); }
      else { el.appendChild(avatar); el.appendChild(bubble); }
    }
    box.appendChild(el);
  });
  if (task?.status === 'Finished') {
    const hint = document.createElement('div');
    hint.className = 'chat-message system resume-hint-row';
    hint.innerHTML = `<div class="bubble"><div>${escapeHTML(t('resumeTaskHint'))}</div><button type="button" class="soft-btn inline-resume-btn">${escapeHTML(t('resumeTask'))}</button></div>`;
    const btn = hint.querySelector('.inline-resume-btn');
    if (btn) btn.onclick = () => resumeActiveTask();
    box.appendChild(hint);
  }
  if (task?.status === 'Running') {
    const typing = document.createElement('div');
    typing.className = 'chat-message gardener typing-row';
    typing.innerHTML = '<div class="avatar gardener">G</div><div class="typing-bubble"><span></span><span></span><span></span></div>';
    box.appendChild(typing);
  }
  box.scrollTop = box.scrollHeight;
}

function formatMessageDateTime(value) {
  const d = new Date(value);
  if (Number.isNaN(d.getTime())) return '';
  const pad = n => String(n).padStart(2, '0');
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())} ${pad(d.getHours())}:${pad(d.getMinutes())}`;
}

function renderTrees(task) {
  const list = $('treeList'); const template = $('treeTemplate'); list.innerHTML = '';
  const forests = getForests(task.trees || []);
  const selected = ensureSelectedForest(task);
  const forest = forests.find(o => o.no === selected);
  if (!forest) { $('treeSummaryText').textContent = ''; $('treePanelMeta').textContent = ''; list.innerHTML = `<div class="empty-list large">${t('waitingForest')}</div>`; return; }

  $('treeSummaryText').textContent = `${t('stage')} ${forest.no}`;
  $('treePanelTitle').textContent = `${t('stage')} ${forest.no}`;
  $('treePanelMeta').textContent = '';

  forest.items.forEach(tree => {
    const node = template.content.firstElementChild.cloneNode(true);
    node.dataset.treeId = tree.id;
    node.classList.toggle('validation', !!tree.isValidation);
    node.querySelector('.tree-badge').textContent = tree.isValidation ? 'V' : 'T';
    node.querySelector('h4').textContent = humanizeText(tree.name);
    node.querySelector('p').textContent = tree.isValidation ? t('validationTeam') : t('team');
    const pill = node.querySelector('.status-pill'); pill.textContent = statusText(tree.status); pill.className = 'status-pill ' + tree.status;
    const scopeText = [(tree.scope || []).join(' / '), tree.objective || ''].filter(Boolean).join('\n');
    node.querySelector('.scope').textContent = humanizeText(scopeText || '等待 Gardener 分配范围');
    const ul = node.querySelector('ul');
    (tree.progress || []).slice(-8).forEach(p => { const li = document.createElement('li'); li.textContent = humanizeText(p); ul.appendChild(li); });
    setFruitLink(node.querySelector('.fruit-btn'), task.id, tree);
    list.appendChild(node);
  });
}

function setTaskReportLink(anchor, url, title) {
  anchor.href = url;
  anchor.onclick = e => { e.preventDefault(); openReport(url, title); };
}

function setFruitLink(anchor, taskId, tree) {
  const url = `/api/tasks/${taskId}/trees/${tree.id}/fruit.md`;
  anchor.href = url;
  anchor.removeAttribute('target');
  anchor.removeAttribute('rel');
  if (!tree.fruitPath) {
    anchor.classList.add('disabled');
    anchor.setAttribute('aria-disabled', 'true');
    anchor.onclick = e => { e.preventDefault(); alert(t('resultNotReady')); };
  } else {
    anchor.classList.remove('disabled');
    anchor.removeAttribute('aria-disabled');
    anchor.onclick = e => { e.preventDefault(); openReport(url, humanizeText(tree.name || t('result'))); };
  }
}

async function openReport(url, title) {
  const overlay = $('reportOverlay');
  $('reportTitle').textContent = title;
  $('reportBody').innerHTML = `<div class="report-loading">${t('openingResult')}</div>`;
  overlay.classList.remove('hidden');
  overlay.setAttribute('aria-hidden', 'false');
  try {
    const text = await fetchText(url);
    state.activeReportText = text;
    $('reportBody').innerHTML = renderMarkdown(text);
  } catch (err) {
    $('reportBody').innerHTML = `<div class="report-loading">${t('openFailed')}${escapeHTML(err.message)}</div>`;
  }
}

function closeReport() {
  $('reportOverlay').classList.add('hidden');
  $('reportOverlay').setAttribute('aria-hidden', 'true');
}

function renderJSON(text) {
  const raw = String(text || '');
  if (!raw.trim()) return `<div class="report-loading">${t('emptyResult')}</div>`;
  let formatted = raw;
  try {
    if (/\.jsonl$/i.test(state.selectedFilePath[state.activeTaskId] || '')) {
      formatted = raw.split(/\r?\n/).filter(Boolean).map(line => JSON.stringify(JSON.parse(line), null, 2)).join('\n');
    } else {
      formatted = JSON.stringify(JSON.parse(raw), null, 2);
    }
  } catch {
    formatted = raw;
  }
  return renderCode(formatted, 'json');
}

function renderCode(text, lang = '') {
  const rawLines = String(text || '').replace(/\t/g, '  ').split(/\r?\n/);
  const maxLines = 2500;
  const lines = rawLines.slice(0, maxLines);
  const truncated = rawLines.length > maxLines ? `<div class="preview-note">${escapeHTML(t('previewTruncated').replace('%d', String(text || '').length))}</div>` : '';
  const body = lines.map((line, idx) => `
    <div class="code-line">
      <span class="line-no">${idx + 1}</span>
      <code>${highlightCodeLine(line, lang) || '&nbsp;'}</code>
    </div>`).join('');
  return `${truncated}<div class="code-wrap ${escapeHTML(lang)}">${body}</div>`;
}

function highlightCodeLine(line, lang) {
  let html = escapeHTML(line);
  if (lang === 'json') {
    html = html
      .replace(/(&quot;[^&]*?&quot;)(\s*:)/g, '<span class="tok-key">$1</span>$2')
      .replace(/:\s*(&quot;[^&]*?&quot;)/g, ': <span class="tok-string">$1</span>')
      .replace(/\b(true|false|null)\b/g, '<span class="tok-literal">$1</span>')
      .replace(/(-?\b\d+(?:\.\d+)?\b)/g, '<span class="tok-number">$1</span>');
  } else if (lang === 'python') {
    html = html
      .replace(/(#.*)$/g, '<span class="tok-comment">$1</span>')
      .replace(/(&quot;.*?&quot;|&#39;.*?&#39;)/g, '<span class="tok-string">$1</span>')
      .replace(/\b(def|class|return|if|elif|else|for|while|try|except|finally|with|as|import|from|pass|break|continue|in|is|not|and|or|lambda|yield|True|False|None)\b/g, '<span class="tok-keyword">$1</span>')
      .replace(/(-?\b\d+(?:\.\d+)?\b)/g, '<span class="tok-number">$1</span>');
  } else if (lang === 'html') {
    html = html
      .replace(/(&lt;\/?[a-zA-Z][^&]*?&gt;)/g, '<span class="tok-tag">$1</span>')
      .replace(/(&lt;!--.*?--&gt;)/g, '<span class="tok-comment">$1</span>');
  }
  return html;
}

function renderCSV(text) {
  const rows = parseCSV(text);
  if (!rows.length) return `<div class="report-loading">${t('emptyResult')}</div>`;
  const maxRows = 500;
  const visibleRows = rows.slice(0, maxRows);
  const colCount = Math.max(...visibleRows.map(r => r.length));
  const normalized = visibleRows.map(r => Array.from({ length: colCount }, (_, i) => r[i] ?? ''));
  const [head, ...body] = normalized;
  const tableHead = `<thead><tr>${head.map(c => `<th>${escapeHTML(c)}</th>`).join('')}</tr></thead>`;
  const tableBody = `<tbody>${body.map(r => `<tr>${r.map(c => `<td>${escapeHTML(c)}</td>`).join('')}</tr>`).join('')}</tbody>`;
  const note = rows.length > maxRows ? `<div class="csv-note">Only showing first ${maxRows} rows.</div>` : '';
  return `${note}<div class="csv-table-wrap"><table>${tableHead}${tableBody}</table></div>`;
}

function parseCSV(text) {
  const rows = [];
  let row = [], field = '', inQuotes = false;
  const input = String(text || '');
  for (let i = 0; i < input.length; i++) {
    const ch = input[i];
    if (inQuotes) {
      if (ch === '"') {
        if (input[i + 1] === '"') { field += '"'; i++; }
        else inQuotes = false;
      } else field += ch;
      continue;
    }
    if (ch === '"') { inQuotes = true; continue; }
    if (ch === ',') { row.push(field); field = ''; continue; }
    if (ch === '\n') { row.push(field); rows.push(row); row = []; field = ''; continue; }
    if (ch === '\r') continue;
    field += ch;
  }
  if (field !== '' || row.length) { row.push(field); rows.push(row); }
  return rows.filter(r => r.some(c => String(c).trim() !== ''));
}

function renderMarkdown(md) {
  const lines = String(md || '').split(/\r?\n/);
  const out = [];
  let inCode = false, code = [], para = [], list = [], table = [];
  const flushPara = () => { if (para.length) { out.push(`<p>${inline(para.join(' '))}</p>`); para = []; } };
  const flushList = () => { if (list.length) { out.push(`<ul>${list.map(x => `<li>${inline(x)}</li>`).join('')}</ul>`); list = []; } };
  const flushCode = () => { out.push(`<pre><code>${escapeHTML(code.join('\n'))}</code></pre>`); code = []; };
  const splitTable = (line) => line.trim().replace(/^\|/, '').replace(/\|$/, '').split('|').map(c => c.trim());
  const isTableDivider = (line) => /^\s*\|?\s*:?-{3,}:?\s*(\|\s*:?-{3,}:?\s*)+\|?\s*$/.test(line);
  const flushTable = () => {
    if (!table.length) return;
    const [head, ...rows] = table;
    out.push(`<div class="md-table-wrap"><table><thead><tr>${head.map(c => `<th>${inline(c)}</th>`).join('')}</tr></thead><tbody>${rows.map(r => `<tr>${r.map(c => `<td>${inline(c)}</td>`).join('')}</tr>`).join('')}</tbody></table></div>`);
    table = [];
  };
  for (let i = 0; i < lines.length; i++) {
    const raw = lines[i];
    const line = raw.replace(/\s+$/, '');
    if (line.startsWith('```')) {
      if (inCode) { inCode = false; flushCode(); } else { flushPara(); flushList(); flushTable(); inCode = true; code = []; }
      continue;
    }
    if (inCode) { code.push(line); continue; }
    if (!line.trim()) { flushPara(); flushList(); flushTable(); continue; }
    if (/^\s*---+\s*$/.test(line)) { flushPara(); flushList(); flushTable(); out.push('<hr />'); continue; }
    if (line.includes('|') && i + 1 < lines.length && isTableDivider(lines[i + 1])) {
      flushPara(); flushList(); flushTable();
      table.push(splitTable(line));
      i += 1;
      while (i + 1 < lines.length && lines[i + 1].includes('|') && lines[i + 1].trim()) {
        i += 1;
        table.push(splitTable(lines[i]));
      }
      flushTable();
      continue;
    }
    const h = line.match(/^(#{1,6})\s+(.+)$/);
    if (h) { flushPara(); flushList(); flushTable(); const level = Math.min(6, Math.max(2, h[1].length + 1)); out.push(`<h${level}>${inline(h[2])}</h${level}>`); continue; }
    const quote = line.match(/^>\s*(.+)$/);
    if (quote) { flushPara(); flushList(); flushTable(); out.push(`<blockquote>${inline(quote[1])}</blockquote>`); continue; }
    const li = line.match(/^\s*[-*]\s+(.+)$/) || line.match(/^\s*\d+\.\s+(.+)$/);
    if (li) { flushPara(); flushTable(); list.push(li[1]); continue; }
    para.push(line);
  }
  flushPara(); flushList(); flushTable(); if (inCode) flushCode();
  return out.join('') || `<div class="report-loading">${t('emptyResult')}</div>`;
}

function inline(s) {
  return escapeHTML(s)
    .replace(/`([^`]+)`/g, '<code>$1</code>')
    .replace(/\*\*([^*]+)\*\*/g, '<strong>$1</strong>');
}

function statusText(status) { return status === 'Finished' ? t('done') : t('inProgress'); }

function humanizeText(s) {
  const words = state.settings.language === 'en'
    ? { validation:'Validation', tree:'Subtask', forest:'Task', stage:'Stage', workspace:'save location', fruit:'report', log:'activity log', schedule:'plan' }
    : { validation:'验证任务', tree:'子任务', forest:'任务', stage:'阶段', workspace:'保存位置', fruit:'报告', log:'工作记录', schedule:'任务安排' };
  return String(s || '')
    .replace(/执行小队/g, words.tree)
    .replace(/检查小队|测试小队/g, words.validation)
    .replace(/小队/g, words.tree)
    .replace(/Validation Tree W(\d+)/g, `${words.validation} $1`)
    .replace(/Validation Tree O(\d+)/g, `${words.validation} $1`)
    .replace(/Validation Tree/g, words.validation)
    .replace(/Research Tree|Builder Tree|Writer Tree|Repair Tree|Integration Tree/g, words.tree)
    .replace(/\bTree\b/g, words.tree)
    .replace(new RegExp(`\\b${['Or', 'chard'].join('')}\\b`, 'g'), words.stage)
    .replace(/\bWave\b/g, words.stage)
    .replace(/Forest (\d+)/g, `${words.stage} $1`)
    .replace(/Forest ID/g, state.settings.language === 'en' ? 'Task ID' : '任务 ID')
    .replace(/\bForest\b/g, words.forest)
    .replace(/workspacePath|workspace/g, words.workspace)
    .replace(/fruit\.md/g, words.fruit)
    .replace(/\bFruit\b/g, words.fruit)
    .replace(/\bfruit\b/g, words.fruit)
    .replace(/log\.md/g, words.log)
    .replace(/schedule\.md/g, words.schedule);
}
function escapeHTML(s) { return String(s || '').replace(/[&<>'"]/g, c => ({'&':'&amp;','<':'&lt;','>':'&gt;',"'":'&#39;','"':'&quot;'}[c])); }


let currentDirectoryPath = '';

async function openDirectoryPicker() {
  $('directoryOverlay').classList.remove('hidden');
  $('directoryOverlay').setAttribute('aria-hidden', 'false');
  await loadDirectory($('workspaceInput').value.trim() || state.settings.defaultWorkspace || '');
}

function closeDirectoryPicker() {
  $('directoryOverlay').classList.add('hidden');
  $('directoryOverlay').setAttribute('aria-hidden', 'true');
}

async function loadDirectory(path = '') {
  const qs = path ? `?path=${encodeURIComponent(path)}` : '';
  const data = await api(`/api/fs/dirs${qs}`);
  currentDirectoryPath = data.path || '';
  $('directoryPath').textContent = currentDirectoryPath;
  $('parentDirectoryBtn').disabled = !data.parent;
  $('parentDirectoryBtn').onclick = () => data.parent && loadDirectory(data.parent);
  const list = $('directoryList');
  list.innerHTML = '';
  const entries = data.entries || [];
  if (!entries.length) {
    list.innerHTML = `<div class="file-empty">${t('folderEmpty')}</div>`;
    return;
  }
  entries.forEach(entry => {
    const btn = document.createElement('button');
    btn.type = 'button';
    btn.className = 'directory-item';
    btn.innerHTML = `<span>📁</span><strong>${escapeHTML(entry.name)}</strong>`;
    btn.onclick = () => loadDirectory(entry.path);
    list.appendChild(btn);
  });
}

async function resumeActiveTask() {
  if (!state.activeTaskId) return;
  const taskId = state.activeTaskId;
  const originalTask = state.tasks.find(t => t.id === taskId);
  const resumeBtn = $('resumeTaskBtn');
  if (resumeBtn) resumeBtn.disabled = true;
  if (originalTask) {
    const optimistic = {
      ...originalTask,
      status: 'Running',
      gardenerStatus: 'Running',
      stopRequested: false,
      messages: [...(originalTask.messages || []), { id: `local_resume_${Date.now()}`, role: 'user', content: t('resumeTask'), createdAt: new Date().toISOString() }]
    };
    upsertTask(optimistic);
    renderTask(optimistic, { skipFileViewer: true });
  }
  try {
    const data = await api(`/api/tasks/${taskId}/resume`, { method:'POST', body:'{}' });
    upsertTask(data.task);
    if (state.activeTaskId === taskId) renderTask(data.task, { skipFileViewer: true });
  } catch (err) {
    if (originalTask && state.activeTaskId === taskId) renderTask(originalTask, { skipFileViewer: true });
    alert((t('resumeFailed') || 'Continue failed: ') + err.message);
  } finally {
    if (resumeBtn) resumeBtn.disabled = false;
  }
}

$('createTaskBtn').onclick = async () => {
  const prompt = $('taskInput').value.trim();
  const workspacePath = $('workspaceInput').value.trim() || state.settings.defaultWorkspace || '';
  if (!prompt) return alert(t('taskPlaceholder'));
  $('createTaskBtn').disabled = true;
  try { const data = await api('/api/tasks', { method:'POST', body: JSON.stringify({ prompt, workspacePath }) }); $('taskInput').value=''; $('workspaceInput').value = state.settings.defaultWorkspace || ''; await loadTasks(); await selectTask(data.task.id); }
  catch (err) { alert(err.message); } finally { $('createTaskBtn').disabled = false; }
};
if ($('toggleOverviewBtn')) $('toggleOverviewBtn').onclick = toggleOverviewCollapsed;
$('sendMessageBtn').onclick = async () => {
  const input = $('messageInput');
  const content = input.value.trim();
  if (!state.activeTaskId || !content) return;
  const taskId = state.activeTaskId;
  const originalTask = state.tasks.find(t => t.id === taskId);
  $('sendMessageBtn').disabled = true;
  input.value = '';
  autoResizeMessageInput();
  if (originalTask) {
    const optimistic = {
      ...originalTask,
      status: 'Running',
      gardenerStatus: 'Running',
      messages: [...(originalTask.messages || []), { id: `local_${Date.now()}`, role: 'user', content, createdAt: new Date().toISOString() }]
    };
    upsertTask(optimistic);
    renderTask(optimistic, { skipFileViewer: true });
  }
  try {
    const data = await api(`/api/tasks/${taskId}/messages`, { method:'POST', body: JSON.stringify({ content }) });
    upsertTask(data.task);
    if (state.activeTaskId === taskId) renderTask(data.task, { skipFileViewer: true });
  } catch(err){
    input.value = content;
    autoResizeMessageInput();
    if (originalTask && state.activeTaskId === taskId) renderTask(originalTask, { skipFileViewer: true });
    alert(err.message);
  } finally {
    $('sendMessageBtn').disabled=false;
  }
};
$('resumeTaskBtn').onclick = resumeActiveTask;
$('stopTaskBtn').onclick = async () => { if (!state.activeTaskId) return; if (!confirm(t('stopConfirm'))) return; $('stopTaskBtn').disabled = true; try { await api(`/api/tasks/${state.activeTaskId}/stop`, { method:'POST', body:'{}' }); } catch(err){ alert(err.message); } };
$('deleteTaskBtn').onclick = async () => { if (!state.activeTaskId) return; if (!confirm(t('deleteConfirm'))) return; const deleted = state.activeTaskId; $('deleteTaskBtn').disabled = true; try { await api(`/api/tasks/${deleted}`, { method:'DELETE' }); state.tasks = state.tasks.filter(t => t.id !== deleted); backToList({ replaceRoute: true }); renderTaskList(); renderHomeGarden(); } catch(err){ alert((t('deleteFailed') || 'Delete failed: ') + err.message); } finally { $('deleteTaskBtn').disabled = false; } };
$('renameTaskBtn').addEventListener('mousedown', e => { if (state.editingTitle) e.preventDefault(); });
$('renameTaskBtn').onclick = () => { state.editingTitle ? commitTitleEdit() : beginTitleEdit(); };
$('pageTitle').ondblclick = beginTitleEdit;
$('titleEditInput').addEventListener('keydown', e => {
  if (e.key === 'Enter') { e.preventDefault(); commitTitleEdit(); }
  if (e.key === 'Escape') { e.preventDefault(); cancelTitleEdit(true); }
});
$('titleEditInput').addEventListener('blur', () => { if (state.editingTitle) commitTitleEdit(); });
$('messageInput').addEventListener('input', autoResizeMessageInput);
$('messageInput').addEventListener('keydown', e => { if (e.key === 'Enter' && !e.shiftKey) { e.preventDefault(); $('sendMessageBtn').click(); } });
function autoResizeMessageInput() {
  const el = $('messageInput');
  if (!el) return;
  el.style.height = 'auto';
  const max = Math.max(96, Math.floor(window.innerHeight * 0.28));
  const next = Math.min(el.scrollHeight, max);
  el.style.height = `${next}px`;
  el.style.overflowY = el.scrollHeight > max ? 'auto' : 'hidden';
}
autoResizeMessageInput();
$('refreshBtn').onclick = loadTasks;
if ($('homeRefreshBtn')) $('homeRefreshBtn').onclick = loadTasks;
$('backBtn').onclick = backToList;
$('settingsBtn').onclick = openSettings;
if ($('homeSettingsBtn')) $('homeSettingsBtn').onclick = openSettings;
if ($('detailSettingsBtn')) $('detailSettingsBtn').onclick = openSettings;
$('browseWorkspaceBtn').onclick = openDirectoryPicker;
$('closeDirectoryBtn').onclick = closeDirectoryPicker;
$('useDirectoryBtn').onclick = () => { $('workspaceInput').value = currentDirectoryPath; closeDirectoryPicker(); };
$('directoryOverlay').onclick = e => { if (e.target === $('directoryOverlay')) closeDirectoryPicker(); };
$('closeSettingsBtn').onclick = closeSettings;
$('saveSettingsBtn').onclick = async () => {
  state.settings.defaultWorkspace = $('defaultWorkspaceInput').value.trim();
  state.settings.showSavePath = $('showSavePathToggle').checked;
  state.settings.showWorkRecord = $('showWorkRecordToggle').checked;
  state.settings.language = $('languageSelect').value;
  state.settings.logLevel = $('logLevelSelect').value;
  state.settings.cliEngine = normalizeCLIEngineValue($('cliEngineSelect').value);
  state.settings.modelMode = $('modelModeSelect').value;
  state.settings.cliEngine = compatibleCLIEngineValue($('cliEngineSelect').value, state.settings.modelMode);
  setCurrentModelToken($('modelTokenInput').value.trim());
  await saveSettings();
  closeSettings();
};
$('modelModeSelect').onchange = () => {
  state.settings.modelMode = $('modelModeSelect').value;
  state.settings.cliEngine = compatibleCLIEngineValue($('cliEngineSelect').value, state.settings.modelMode);
  $('cliEngineSelect').value = state.settings.cliEngine;
  applyModelTokenField();
};
$('cliEngineSelect').onchange = () => {
  state.settings.cliEngine = compatibleCLIEngineValue($('cliEngineSelect').value, state.settings.modelMode || 'default');
  $('cliEngineSelect').value = state.settings.cliEngine;
};
$('modelTokenInput').oninput = () => setCurrentModelToken($('modelTokenInput').value.trim());
$('closeReportBtn').onclick = closeReport;
$('reportOverlay').onclick = e => { if (e.target === $('reportOverlay')) closeReport(); };
$('settingsOverlay').onclick = e => { if (e.target === $('settingsOverlay')) closeSettings(); };
$('copyReportBtn').onclick = async () => { if (!state.activeReportText) return; try { await navigator.clipboard.writeText(state.activeReportText); } catch {} };
document.addEventListener('keydown', e => { if (e.key === 'Escape') { closeReport(); closeSettings(); closeDirectoryPicker(); } });
window.addEventListener('popstate', () => syncFromRoute());
document.addEventListener('visibilitychange', () => {
  if (!document.hidden && state.activeTaskId) loadActiveTask(true);
});
let resizeTimer = 0;
window.addEventListener('resize', () => {
  clearTimeout(resizeTimer);
  resizeTimer = setTimeout(() => {
    autoResizeMessageInput();
    invalidateRenderCache(state.activeTaskId || '');
    const active = state.tasks.find(task => task.id === state.activeTaskId);
    if (active) renderTask(active, { skipFileViewer: true });
  }, 160);
}, { passive: true });
function openSettings() { $('settingsOverlay').classList.remove('hidden'); $('settingsOverlay').setAttribute('aria-hidden', 'false'); }
function closeSettings() { $('settingsOverlay').classList.add('hidden'); $('settingsOverlay').setAttribute('aria-hidden', 'true'); }
applySettings();
loadHealthStatus();
setInterval(loadHealthStatus, 5 * 60 * 1000);
loadServerSettings().finally(() => loadTasks({ syncRoute: true }));
