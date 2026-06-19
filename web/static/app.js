const state = { powerStatus: null, tasks: [], activeTaskId: null, homeTaskSearch: '', eventSource: null, recoveryPoller: null, activeRefreshPoller: null, selectedForests: {}, selectedFileTree: {}, selectedFilePath: {}, selectedFileManual: {}, fileListFingerprint: {}, lastFileRefreshAt: {}, treeStatusExpanded: {}, usage: {}, usageFetchedAt: {}, usagePending: {}, renderCache: {}, pendingTaskRender: null, pendingTaskRenderFrame: 0, lastTaskListSig: '', lastHomeSig: '', activeReportText: '', fileViewerToken: 0, previewToken: 0, overviewCollapsed: loadOverviewCollapsed(), chatCollapsed: loadChatCollapsed(), editingTitle: false, settings: loadSettings() };
const $ = (id) => document.getElementById(id);
const MAX_CSV_PREVIEW_ROWS = 500;
let pendingDeleteResolve = null;
let pendingDeleteTaskId = '';

const I18N = {
  'zh-CN': {
    newTask:'新建任务', taskLabel:'任务', homeTitle:'你想完成什么？', garden:'工作台', taskPlaceholder:'告诉 Gardener 你的目标、要求和交付物', saveLocation:'保存位置', defaultSave:'默认保存', create:'创建', tasks:'任务', refresh:'刷新', back:'返回', messagePlaceholder:'给 Gardener 发消息', clarificationReplyPlaceholder:'请直接回答 Gardener 的问题', send:'发送', taskPlan:'任务安排', workRecord:'工作记录', stop:'停止', workProcess:'工作过程', viewResult:'查看报告', settings:'设置', close:'关闭', defaultSaveLocation:'默认保存位置', autoSave:'留空则自动保存', saveLocationHelp:'不设置也可以正常使用。', showSaveLocation:'创建任务时显示保存位置', showPlanRecord:'在任务中显示安排和记录', language:'语言', logDetail:'记录详细程度', logQuiet:'简洁', logNormal:'标准', logDetailed:'详细', logHelp:'普通使用建议选择“简洁”。需要排查问题时再切换为“详细”。', save:'保存', copy:'复制', result:'报告', noTasks:'暂无任务', newTaskShort:'新任务', genericTask:'任务', inProgress:'进行中', done:'已完成', waitingForest:'等待阶段', noForest:'无阶段', gardenerWillContinue:'我会继续处理。', resultNotReady:'报告尚未生成。', openingResult:'正在打开报告', emptyResult:'内容为空', openFailed:'无法打开：', stopConfirm:'停止当前任务？', team:'子任务', validationTeam:'验证任务', files:'文件', recentForests:'已有任务', taskSearchPlaceholder:'搜索任务标题、ID、状态或模型', noTaskSearchResults:'没有匹配的任务', noRecent:'还没有任务', openForest:'打开', allFiles:'全部文件', allTreeFiles:'全部子任务', noFiles:'暂无可查看文件', loadingFiles:'正在读取文件', selectFile:'选择文件查看内容', fileTooLarge:'文件无法预览', treeStatus:'子任务状态', noTreesInForest:'暂无子任务', browse:'选择', chooseFolder:'选择保存位置', parentFolder:'上一级', useFolder:'使用此目录', folderEmpty:'没有可选择的子目录', tokenUsage:'Token 消耗', tokenEstimate:'Token 消耗', tokenMaxEstimate:'', tokenNoData:'暂无 token 记录', delete:'删除', cancel:'取消', deleteConfirm:'删除这个任务并清除它的数据？', deleteConfirmTitle:'删除这个任务？', deleteConfirmDescription:'任务记录、对话和本地任务数据将被移除。', deleteConfirmNote:'此操作无法撤销。', deleteFailed:'删除失败：', viewStatus:'查看状态', hideStatus:'收起状态', rename:'重命名', renamePrompt:'输入新的任务名称', renameFailed:'重命名失败：', model:'模型', modelDefault:'CLI 默认模型', cliEngine:'底层 CLI', cliCodex:'Codex CLI', cliClaude:'Claude Code', cliHelp:'切换后会同步到已有任务；正在运行的底层进程不会被打断。', modelToken:'Token', modelTokenPlaceholder:'输入当前模型的 token', modelTokenPlaceholderConfigured:'已内置，无需填写；填写会覆盖', modelTokenHelpConfigured:'此模型的 SK 已由安装包内置/服务器预置，不需要再次填写。', modelTokenHelpEmpty:'未检测到内置 SK；如需使用此模型，请填写一次 token。', gardenerProgress:'工作进展', gardenerWorking:'正在工作', gardenerProgressEmpty:'等待下一步进展', stage:'阶段', subtask:'子任务', file:'文件', resumeTask:'继续任务', resumeTaskHint:'任务已暂停。如未完成，可点击“继续任务”，Gardener 会检查当前进度后接着处理。', resumeFailed:'继续失败：', fileEncodingHint:'已自动尝试文本编码识别。', binaryFile:'文件可能不是文本，无法预览', statusQuerySafe:'查看进度不会中断任务。', collapseOverview:'收起概览', expandOverview:'展开概览', overview:'概览', recentMessagesOnly:'仅显示最近 %d 条消息。', previewTruncated:'文件较大，已仅预览前 %d 个字符。', downloadFile:'下载文件', powerWarningTitle:'远程访问提醒', powerWarningPrefix:'这台电脑的电源设置可能导致 Gardener 离线：', dashboard:'任务驾驶舱', askProgressSafe:'询问进度不会中断任务', diagnosis:'诊断提示', collapseChat:'收起对话', expandChat:'展开对话', taskNow:'当前状态', taskNext:'是否需要操作', workingNormally:'正在正常处理，你可以等待；如果想了解进展，直接发消息询问，不会中断任务。', checkingResults:'子任务已返回，Gardener 正在检查结果并决定下一步。', planningTask:'Gardener 正在把目标拆成可执行的小任务。', finishedTaskHint:'任务已完成。如需补充或继续迭代，可以点击继续任务。', awaitingUserInput:'Gardener 正在等你补充需求。请直接在对话框回答它的问题。', jumpToLatest:'跳到最新'
  },
  en: {
    newTask:'New task', taskLabel:'Task', homeTitle:'What do you want to get done?', garden:'Workspace', taskPlaceholder:'Tell Gardener your goal, requirements, and deliverables', saveLocation:'Save location', defaultSave:'Default save location', create:'Create', tasks:'Tasks', refresh:'Refresh', back:'Back', messagePlaceholder:'Message Gardener', clarificationReplyPlaceholder:'Reply to Gardener’s question', send:'Send', taskPlan:'Plan', workRecord:'Activity', stop:'Stop', workProcess:'Activity', viewResult:'View report', settings:'Settings', close:'Close', defaultSaveLocation:'Default save location', autoSave:'Leave blank to save automatically', saveLocationHelp:'You can use Gardener without setting this.', showSaveLocation:'Show save location when creating a task', showPlanRecord:'Show plan and activity inside a task', language:'Language', logDetail:'Activity detail', logQuiet:'Simple', logNormal:'Standard', logDetailed:'Detailed', logHelp:'Simple is recommended. Use Detailed only when troubleshooting.', save:'Save', copy:'Copy', result:'Report', noTasks:'No tasks', newTaskShort:'New task', genericTask:'Task', inProgress:'Running', done:'Done', waitingForest:'Waiting for stage', noForest:'No stage', gardenerWillContinue:'I will keep working on it.', resultNotReady:'Report is not ready yet.', openingResult:'Opening report', emptyResult:'Empty content', openFailed:'Unable to open: ', stopConfirm:'Stop this task?', team:'Subtask', validationTeam:'Validation', files:'Files', recentForests:'Tasks', taskSearchPlaceholder:'Search title, ID, status, or model', noTaskSearchResults:'No matching tasks', noRecent:'No tasks yet', openForest:'Open', allFiles:'All files', allTreeFiles:'All subtasks', noFiles:'No files', loadingFiles:'Loading files', selectFile:'Select a file to preview', fileTooLarge:'File cannot be previewed', treeStatus:'Subtask status', noTreesInForest:'No subtasks', browse:'Choose', chooseFolder:'Choose folder', parentFolder:'Parent', useFolder:'Use this folder', folderEmpty:'No folders', tokenUsage:'Token usage', tokenEstimate:'Token usage', tokenMaxEstimate:'', tokenNoData:'No token records yet', delete:'Delete', cancel:'Cancel', deleteConfirm:'Delete this task and clear its data?', deleteConfirmTitle:'Delete this task?', deleteConfirmDescription:'The task record, conversation, and local task data will be removed.', deleteConfirmNote:'This action cannot be undone.', deleteFailed:'Delete failed: ', viewStatus:'View status', hideStatus:'Hide status', rename:'Rename', renamePrompt:'Enter a new task name', renameFailed:'Rename failed: ', model:'Model', modelDefault:'CLI default model', cliEngine:'Base CLI', cliCodex:'Codex CLI', cliClaude:'Claude Code', cliHelp:'Changes sync to existing tasks; already-running CLI processes are not interrupted.', modelToken:'Token', modelTokenPlaceholder:'Enter the token for the selected model', modelTokenPlaceholderConfigured:'Bundled; optional override', modelTokenHelpConfigured:'This model key is bundled by the installer/server, so no extra token is required.', modelTokenHelpEmpty:'No bundled key was detected; enter a token to use this model.', gardenerProgress:'Work progress', gardenerWorking:'Working', gardenerProgressEmpty:'Waiting for updates', stage:'Stage', subtask:'Subtask', file:'File', resumeTask:'Continue task', resumeTaskHint:'This task is paused. If it is not done, click Continue task and Gardener will inspect the current progress before continuing.', resumeFailed:'Continue failed: ', fileEncodingHint:'Text encoding was detected automatically.', binaryFile:'This file may not be text and cannot be previewed', statusQuerySafe:'Checking progress will not interrupt the task.', collapseOverview:'Collapse', expandOverview:'Expand', overview:'Overview', recentMessagesOnly:'Showing latest %d messages only.', previewTruncated:'Large file: only first %d characters are shown.', downloadFile:'Download file', powerWarningTitle:'Remote access warning', powerWarningPrefix:'This computer may go offline because of its power settings: ', dashboard:'Task dashboard', askProgressSafe:'Asking progress will not interrupt the task', diagnosis:'Diagnostic cue', collapseChat:'Hide chat', expandChat:'Show chat', taskNow:'Current status', taskNext:'Do I need to act?', workingNormally:'Gardener is working normally. You can wait or ask for progress without interrupting the task.', checkingResults:'Subtasks have returned. Gardener is checking results and deciding next steps.', planningTask:'Gardener is breaking the goal into executable subtasks.', finishedTaskHint:'The task is finished. Click Continue task if you want to add more work.', awaitingUserInput:'Gardener is waiting for your clarification. Reply directly in the chat box.', jumpToLatest:'Jump to latest'
  }
};

function loadOverviewCollapsed() {
  try {
    const value = localStorage.getItem('gardenerOverviewCollapsed');
    if (value === null) return true;
    return value === '1';
  }
  catch { return true; }
}
function saveOverviewCollapsed() {
  try { localStorage.setItem('gardenerOverviewCollapsed', state.overviewCollapsed ? '1' : '0'); }
  catch {}
}
function loadChatCollapsed() {
  try { return localStorage.getItem('gardenerChatCollapsed') === '1'; }
  catch { return false; }
}
function saveChatCollapsed() {
  try { localStorage.setItem('gardenerChatCollapsed', state.chatCollapsed ? '1' : '0'); }
  catch {}
}

function t(key) { return (I18N[state.settings?.language || 'zh-CN'] || I18N['zh-CN'])[key] || I18N['zh-CN'][key] || key; }
function isCompactViewport() {
  try { return window.matchMedia && window.matchMedia('(max-width: 820px)').matches; }
  catch { return (window.innerWidth || 0) <= 820; }
}
function applyI18n() {
  document.documentElement.lang = state.settings.language || 'zh-CN';
  document.querySelectorAll('[data-i18n]').forEach(el => { el.textContent = t(el.dataset.i18n); });
  document.querySelectorAll('[data-i18n-placeholder]').forEach(el => { el.placeholder = t(el.dataset.i18nPlaceholder); });
  document.querySelectorAll('[data-i18n-title]').forEach(el => { el.title = t(el.dataset.i18nTitle); });
}

const MAX_API_ERROR_CHARS = 1000;
function apiErrorMessage(message) {
  const chars = Array.from(String(message || ''));
  return chars.length > MAX_API_ERROR_CHARS ? chars.slice(0, MAX_API_ERROR_CHARS).join('') + '…' : chars.join('');
}

async function api(path, options = {}) {
  const res = await fetch(path, { headers: { 'Content-Type': 'application/json', ...(options.headers || {}) }, ...options });
  if (!res.ok) {
    let msg = `HTTP ${res.status}`;
    try { msg = (await res.json()).error || msg; } catch {}
    throw new Error(apiErrorMessage(msg));
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

function taskAPIPath(taskId, suffix = '') {
  return `/api/tasks/${encodeURIComponent(String(taskId || ''))}${suffix}`;
}

function routeTaskId() {
  const match = window.location.pathname.match(/^\/forests\/([^/]+)\/?$/);
  if (!match) return '';
  try {
    return decodeURIComponent(match[1]);
  } catch {
    return '';
  }
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


function isValidHealthResponse(data) {
  if (!data || typeof data !== 'object' || Array.isArray(data)) return false;
  const power = data.power;
  if (power === undefined || power === null) return true;
  if (typeof power !== 'object' || Array.isArray(power)) return false;
  if (typeof power.ok !== 'boolean') return false;
  for (const key of ['platform', 'checkedAt']) {
    if (power[key] !== undefined && typeof power[key] !== 'string') return false;
  }
  if (power.checked !== undefined && typeof power.checked !== 'boolean') return false;
  for (const key of ['warnings', 'advice']) {
    if (power[key] === undefined) continue;
    if (!Array.isArray(power[key]) || power[key].length > 20) return false;
    if (!power[key].every(item => typeof item === 'string')) return false;
  }
  return true;
}

async function loadHealthStatus() {
  try {
    const data = await api('/api/health');
    if (!isValidHealthResponse(data)) throw new Error('Invalid health response');
    state.powerStatus = data.power || null;
    renderPowerBanner();
  } catch (err) {
    console.error(err);
  }
}

const MAX_RENDERED_POWER_WARNING_CHARS = 240;
function renderedPowerWarningText(value) {
  const chars = Array.from(String(value || ''));
  return chars.length > MAX_RENDERED_POWER_WARNING_CHARS ? chars.slice(0, MAX_RENDERED_POWER_WARNING_CHARS).join('') + '…' : chars.join('');
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
  el.innerHTML = `<strong>${escapeHTML(t('powerWarningTitle'))}</strong><div>${escapeHTML(t('powerWarningPrefix'))}</div><ul>${items.map(x => `<li>${escapeHTML(renderedPowerWarningText(x))}</li>`).join('')}</ul>`;
}

function normalizeCLIEngineValue(value) {
  const v = String(value || '').trim().toLowerCase().replace(/_/g, '-');
  if (['claude', 'claude-code', 'claude-cli', 'anthropic', 'cloud'].includes(v)) return 'claude';
  return 'codex';
}

function normalizeModelModeValue(value) {
  const raw = String(value || '').trim();
  const v = raw.toLowerCase();
  if (v === 'minimaxm2.7' || v === 'minimax-m2.7' || v === 'minimaxm3' || v === 'minimax-m3' || raw === 'MiniMax-M3') return 'MiniMax-M3';
  if (v === 'kimi-k2.7' || v === 'kimi-k2.7-code' || v === 'kimik2.7' || v === 'kimik2.7-code' || v === 'kimik2.6' || v === 'kimi-k2.6' || v === 'kimi-coding') return 'kimi-k2.7-code';
  return 'default';
}

function compatibleCLIEngineValue(engine, mode) {
  mode = normalizeModelModeValue(mode);
  const cli = normalizeCLIEngineValue(engine);
  return mode === 'kimi-k2.7-code' && cli === 'codex' ? 'claude' : cli;
}
function normalizeSettingsCompatibility() {
  state.settings.modelMode = normalizeModelModeValue(state.settings.modelMode || 'default');
  state.settings.cliEngine = compatibleCLIEngineValue(state.settings.cliEngine || 'codex', state.settings.modelMode);
}

function loadSettings() {
  try {
    const saved = JSON.parse(localStorage.getItem('autoGardenerSettings') || '{}');
    if (saved.minimaxToken || saved.kimiToken) {
      localStorage.setItem('autoGardenerSettings', JSON.stringify(persistedClientSettings(saved)));
    }
    const settings = { defaultWorkspace: '', showSavePath: false, showWorkRecord: false, logLevel: 'quiet', language: 'zh-CN', cliEngine: 'codex', modelMode: 'default', minimaxToken: '', kimiToken: '', ...persistedClientSettings(saved) };
    settings.modelMode = normalizeModelModeValue(settings.modelMode);
    return settings;
  } catch {
    return { defaultWorkspace: '', showSavePath: false, showWorkRecord: false, logLevel: 'quiet', language: 'zh-CN', cliEngine: 'codex', modelMode: 'default', minimaxToken: '', kimiToken: '' };
  }
}

function persistedClientSettings(settings) {
  const { minimaxToken, kimiToken, ...safeSettings } = settings || {};
  return safeSettings;
}

async function loadServerSettings() {
  try {
    const data = await api('/api/settings');
    state.settings.logLevel = data.settings?.logLevel || state.settings.logLevel || 'quiet';
    state.settings.cliEngine = normalizeCLIEngineValue(data.settings?.cliEngine || state.settings.cliEngine || 'codex');
    state.settings.modelMode = normalizeModelModeValue(data.settings?.modelMode || normalizeModelModeValue(state.settings.modelMode || 'default'));
    state.settings.minimaxToken = data.settings?.minimaxToken || state.settings.minimaxToken || '';
    state.settings.kimiToken = data.settings?.kimiToken || state.settings.kimiToken || '';
    state.settings.minimaxTokenConfigured = !!data.settings?.minimaxTokenConfigured;
    state.settings.kimiTokenConfigured = !!data.settings?.kimiTokenConfigured;
    applySettings();
  } catch (err) { console.error(err); }
}

async function saveSettings() {
  normalizeSettingsCompatibility();
  localStorage.setItem('autoGardenerSettings', JSON.stringify(persistedClientSettings(state.settings)));
  applySettings();
  try {
    await api('/api/settings', { method: 'PUT', body: JSON.stringify({ logLevel: state.settings.logLevel || 'quiet', cliEngine: normalizeCLIEngineValue(state.settings.cliEngine || 'codex'), modelMode: normalizeModelModeValue(state.settings.modelMode || 'default'), minimaxToken: state.settings.minimaxToken || '', kimiToken: state.settings.kimiToken || '' }) });
    await loadTasks();
  } catch (err) { console.error(err); }
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
  $('cliEngineSelect').value = compatibleCLIEngineValue(state.settings.cliEngine || 'codex', normalizeModelModeValue(state.settings.modelMode || 'default'));
  $('modelModeSelect').value = normalizeModelModeValue(state.settings.modelMode || 'default');
  applyModelTokenField();
  document.body.classList.toggle('hide-save-path', !state.settings.showSavePath);
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
  const mode = normalizeModelModeValue(state.settings.modelMode || 'default');
  if (mode === 'MiniMax-M3') return state.settings.minimaxToken || '';
  if (mode === 'kimi-k2.7-code') return state.settings.kimiToken || '';
  return '';
}

function currentModelTokenConfigured() {
  const mode = normalizeModelModeValue(state.settings.modelMode || 'default');
  if (mode === 'MiniMax-M3') return !!state.settings.minimaxTokenConfigured || !!state.settings.minimaxToken;
  if (mode === 'kimi-k2.7-code') return !!state.settings.kimiTokenConfigured || !!state.settings.kimiToken;
  return false;
}

function setCurrentModelToken(token) {
  const mode = normalizeModelModeValue(state.settings.modelMode || 'default');
  if (mode === 'MiniMax-M3') {
    state.settings.minimaxToken = token;
    if (token) state.settings.minimaxTokenConfigured = true;
  }
  if (mode === 'kimi-k2.7-code') {
    state.settings.kimiToken = token;
    if (token) state.settings.kimiTokenConfigured = true;
  }
}

function applyModelTokenField() {
  const mode = normalizeModelModeValue(state.settings.modelMode || 'default');
  const section = $('modelTokenSection');
  const input = $('modelTokenInput');
  const help = $('modelTokenHelp');
  if (!section || !input) return;
  const hidden = mode === 'default';
  const configured = currentModelTokenConfigured();
  section.classList.toggle('hidden', hidden);
  input.value = currentModelToken();
  input.placeholder = configured ? t('modelTokenPlaceholderConfigured') : t('modelTokenPlaceholder');
  if (help) help.textContent = hidden ? '' : (configured ? t('modelTokenHelpConfigured') : t('modelTokenHelpEmpty'));
}


async function loadTasks(options = {}) {
  try {
    const data = await api('/api/tasks?compact=1');
    state.tasks = Array.isArray(data.tasks) ? data.tasks : [];
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
    const data = await api(taskAPIPath(state.activeTaskId));
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

const MAX_SIGNATURE_TEXT_CHARS = 200;
function signatureText(value) {
  const chars = Array.from(String(value || ''));
  return chars.length > MAX_SIGNATURE_TEXT_CHARS ? chars.slice(0, MAX_SIGNATURE_TEXT_CHARS).join('') : chars.join('');
}

function taskListSignature(tasks) {
  return (tasks || []).map(t => [t.id, signatureText(t.title), signatureText(t.workspacePath), t.status || '', (t.trees || []).length].join(':')).join('|');
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

function isMessagesNearBottom(el = $('messages')) {
  if (!el) return true;
  return el.scrollHeight - el.scrollTop - el.clientHeight < 120;
}

function scrollMessagesToBottom(options = {}) {
  const el = $('messages');
  if (!el) return;
  const behavior = options.smooth ? 'smooth' : 'auto';
  try { el.scrollTo({ top: el.scrollHeight, behavior }); }
  catch { el.scrollTop = el.scrollHeight; }
  updateJumpToLatestButton();
}

function updateJumpToLatestButton() {
  const btn = $('jumpToLatestBtn');
  const el = $('messages');
  if (!btn || !el) return;
  btn.classList.toggle('hidden', isMessagesNearBottom(el));
}

function messageRoleClass(role) {
  const value = String(role || '').toLowerCase();
  if (value === 'user') return 'user';
  if (value === 'system') return 'system';
  return 'gardener';
}

function renderMessages(messages, task) {
  const box = $('messages');
  if (!box) return;
  const wasNearBottom = isMessagesNearBottom(box);
  const { all, visibleMessages, maxMessages } = visibleMessagesForViewport(messages);
  const rows = [];
  if (all.length > visibleMessages.length) {
    rows.push(`<div class="chat-message system"><div class="bubble">${escapeHTML(t('recentMessagesOnly').replace('%d', String(maxMessages)))}</div></div>`);
  }
  visibleMessages.forEach(msg => {
    const role = messageRoleClass(msg.role);
    const content = String(msg.content || '').trim();
    if (!content) return;
    if (shouldHideStatusTipMessage(content)) return;
    const displayContent = sanitizeRuntimeConceptText(content);
    const time = msg.createdAt ? formatMessageTime(msg.createdAt) : '';
    rows.push(`
      <div class="chat-message ${role}">
        ${role === 'gardener' ? '<div class="avatar gardener">G</div>' : ''}
        <div class="bubble">${escapeHTML(displayContent)}${time ? `<span class="message-time">${escapeHTML(time)}</span>` : ''}</div>
        ${role === 'user' ? '<div class="avatar user-avatar">我</div>' : ''}
      </div>
    `);
  });
  box.innerHTML = rows.join('');
  if (wasNearBottom || task?.status === 'Running') scrollMessagesToBottom();
  else updateJumpToLatestButton();
}

function shouldHideStatusTipMessage(content) {
  const text = String(content || '');
  return text.includes('【任务状态提示】') ||
    text.includes('没有新的输出。底层 CLI 可能仍在运行') ||
    text.includes('no new output') && text.includes('underlying CLI');
}

function sanitizeRuntimeConceptText(content) {
  return String(content || '').replaceAll('当前阶段', '接下来');
}

function formatMessageTime(value) {
  const date = new Date(value);
  if (!Number.isFinite(date.getTime())) return '';
  return date.toLocaleString(state.settings?.language || 'zh-CN', { month:'2-digit', day:'2-digit', hour:'2-digit', minute:'2-digit' });
}

const MAX_PROGRESS_SIGNATURE_CHARS = 200;
function progressSignatureText(value) {
  const chars = Array.from(String(value || ''));
  return chars.length > MAX_PROGRESS_SIGNATURE_CHARS ? chars.slice(0, MAX_PROGRESS_SIGNATURE_CHARS).join('') : chars.join('');
}

function progressSignature(task) {
  const raw = Array.isArray(task?.gardenerProgress) ? task.gardenerProgress : [];
  return [task?.status || '', task?.gardenerStatus || '', task?.lastProgressAt || '', raw.slice(-10).map(progressSignatureText).join('|')].join('::');
}

const MAX_FOREST_SIGNATURE_PATH_CHARS = 400;
function forestSignaturePath(value) {
  const chars = Array.from(String(value || ''));
  return chars.length > MAX_FOREST_SIGNATURE_PATH_CHARS ? chars.slice(0, MAX_FOREST_SIGNATURE_PATH_CHARS).join('') : chars.join('');
}

function forestSignature(task) {
  const selected = state.selectedForests[task.id] || '';
  const trees = (task.trees || []).map(tree => [tree.id, tree.forest || 1, tree.status || '', forestSignaturePath(tree.fruitPath), !!tree.isValidation, tree.updatedAt || ''].join(':')).join('|');
  return [selected, trees].join('::');
}

function overviewSignature(task) {
  const usage = state.usage[task.id];
  return [state.overviewCollapsed ? 1 : 0, state.chatCollapsed ? 1 : 0, task.status || '', forestSignature(task), usage?.totalTokens || 0, task.runtime?.phase || '', task.runtime?.severity || '', task.runtime?.idleSeconds || 0].join('::');
}

const MAX_TASK_CHROME_SIGNATURE_TITLE_CHARS = 200;
function taskChromeSignatureTitle(value) {
  const chars = Array.from(String(value || ''));
  return chars.length > MAX_TASK_CHROME_SIGNATURE_TITLE_CHARS ? chars.slice(0, MAX_TASK_CHROME_SIGNATURE_TITLE_CHARS).join('') : chars.join('');
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

const MAX_TASK_EVENT_CHARS = 2_000_000;
function isTaskEventTooLarge(data) {
  return String(data || '').length > MAX_TASK_EVENT_CHARS;
}

function connectEvents(taskId) {
  if (state.eventSource) state.eventSource.close();
  if (state.recoveryPoller) { clearInterval(state.recoveryPoller); state.recoveryPoller = null; }
  if (state.activeRefreshPoller) { clearInterval(state.activeRefreshPoller); state.activeRefreshPoller = null; }
  if (!window.EventSource) { state.recoveryPoller = setInterval(() => { if (!document.hidden) loadActiveTask(true); }, isCompactViewport() ? 4000 : 2000); return; }
  state.activeRefreshPoller = setInterval(() => {
    if (state.activeTaskId === taskId && !document.hidden) loadActiveTask(true);
  }, viewportRefreshMs());
  const es = new EventSource(taskAPIPath(taskId, '/events'));
  state.eventSource = es;
  es.addEventListener('open', () => setConnected(true));
  es.addEventListener('task', (ev) => {
    if (isTaskEventTooLarge(ev.data)) {
      console.warn('Ignoring oversized task event');
      loadActiveTask(true);
      return;
    }
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

function normalizeTaskSearchText(value) {
  return String(value || '').normalize('NFKC').toLowerCase();
}

function taskSearchHaystack(task) {
  const forests = getForests(task?.trees || []);
  const treeValues = (task?.trees || []).flatMap(tree => [tree?.id, tree?.name, tree?.status, tree?.forest, tree?.isValidation ? t('validationTeam') : '']);
  return normalizeTaskSearchText([
    task?.id,
    task?.title,
    task?.status,
    statusText(task?.status),
    task?.cliEngine,
    task?.modelMode,
    task?.createdAt,
    task?.updatedAt,
    forests.length ? `${t('stage')} ${forests.length}` : '',
    ...treeValues
  ].filter(Boolean).join(' '));
}

function homeVisibleTasks(tasks) {
  const query = normalizeTaskSearchText(state.homeTaskSearch).trim();
  if (!query) return (tasks || []).slice(0, 12);
  const terms = query.split(/\s+/).filter(Boolean);
  return (tasks || []).filter(task => {
    const haystack = taskSearchHaystack(task);
    return terms.every(term => haystack.includes(term));
  });
}

function renderHomeGarden() {
  const list = $('homeForestList');
  if (!list) return;
  const tasks = state.tasks || [];
  const searchInput = $('homeTaskSearchInput');
  if (searchInput && searchInput.value !== state.homeTaskSearch) searchInput.value = state.homeTaskSearch || '';
  const query = normalizeTaskSearchText(state.homeTaskSearch).trim();
  const visibleTasks = homeVisibleTasks(tasks);
  const sig = `${state.settings.language || ''}::${state.homeTaskSearch || ''}::${tasks.length}::${taskListSignature(visibleTasks)}`;
  if (state.lastHomeSig === sig && list.childNodes.length) return;
  state.lastHomeSig = sig;
  list.innerHTML = '';
  if (!tasks.length) {
    list.innerHTML = `<div class="home-forest-empty">${t('noRecent')}</div>`;
    return;
  }
  if (query && !visibleTasks.length) {
    list.innerHTML = `<div class="home-forest-empty">${t('noTaskSearchResults')}</div>`;
    return;
  }
  visibleTasks.forEach(task => {
    const item = document.createElement('div');
    item.className = 'home-forest-item';
    item.setAttribute('role', 'button');
    item.tabIndex = 0;
    const forests = getForests(task.trees || []);
    const workspacePath = String(task.workspacePath || '').trim();
    item.innerHTML = `
      <span class="home-forest-title">${escapeHTML(task.title || t('genericTask'))}</span>
      <span class="home-forest-path" title="${escapeHTML(workspacePath)}">${escapeHTML(workspacePath)}</span>
      <span class="home-forest-meta"><b>${statusText(task.status)}</b>${forests.length ? ` · ${t('stage')} ${forests.length}` : ''}</span>
      <div class="home-forest-actions">
        <button type="button" class="home-forest-delete" title="${t('delete')}" aria-label="${t('delete')}">${t('delete')}</button>
      </div>
    `;
    item.onclick = () => selectTask(task.id);
    item.onkeydown = (e) => { if (e.key === 'Enter' || e.key === ' ') { e.preventDefault(); selectTask(task.id); } };
    item.querySelector('.home-forest-delete').onclick = (e) => { e.stopPropagation(); deleteTask(task.id); };
    list.appendChild(item);
  });
}

function resetFileViewerForTask(taskId) {
  if (!taskId) return;
  state.fileViewerToken++;
  state.previewToken++;
  delete state.selectedFilePath[taskId];
  state.selectedFileTree[taskId] = '';
  state.selectedFileManual[taskId] = false;
  delete state.fileListFingerprint[taskId];
  delete state.lastFileRefreshAt[taskId];
  const fileList = $('fileList');
  const preview = $('filePreview');
  const fileTitle = $('fileTitle');
  if (fileList) fileList.innerHTML = '';
  if (preview) preview.textContent = t('selectFile');
  if (fileTitle) fileTitle.textContent = t('files');
}

function renderTask(task, options = {}) {
  $('emptyState').classList.add('hidden');
  $('forestView').classList.remove('hidden');
  applyChatCollapsed();
  const messageInput = $('messageInput');
  if (messageInput) messageInput.placeholder = task?.awaitingUserInput ? t('clarificationReplyPlaceholder') : t('messagePlaceholder');
  ensureSelectedForest(task);
  const cache = state.renderCache[task.id] || (state.renderCache[task.id] = {});
  const chromeSig = [taskChromeSignatureTitle(task.title), task.status || '', state.settings.language || ''].join('::');
  if (cache.chromeSig !== chromeSig) {
    cache.chromeSig = chromeSig;
    if (!state.editingTitle) $('pageTitle').textContent = task.title;
    $('forestStatus').textContent = statusText(task.status);
    $('forestStatus').className = 'status-pill ' + task.status;
    setTaskReportLink($('scheduleLink'), taskAPIPath(task.id, '/gardener/schedule.md'), t('taskPlan'));
    setTaskReportLink($('logLink'), taskAPIPath(task.id, '/gardener/log.md'), t('workRecord'));
    $('stopTaskBtn').disabled = task.status === 'Finished';
    const resumeBtn = $('resumeTaskBtn');
    if (resumeBtn) {
      const resumable = task.status === 'Finished' && !task.awaitingUserInput;
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
  hideUsagePanel();
  if (!options.skipFileViewer && !document.hidden) renderFileViewer(task);
}

function shouldSkipFileViewer(previous, task) {
  if (!previous || !task || previous.id !== task.id) return false;
  const sameTaskShape = fileRefreshSignature(previous) === fileRefreshSignature(task);
  if (!sameTaskShape) return false;
  const last = state.lastFileRefreshAt[task.id] || 0;
  return Date.now() - last < (isCompactViewport() ? 15000 : 6000);
}

const MAX_FILE_REFRESH_SIGNATURE_PATH_CHARS = 400;
function fileRefreshSignaturePath(value) {
  const chars = Array.from(String(value || ''));
  return chars.length > MAX_FILE_REFRESH_SIGNATURE_PATH_CHARS ? chars.slice(0, MAX_FILE_REFRESH_SIGNATURE_PATH_CHARS).join('') : chars.join('');
}

function fileRefreshSignature(task) {
  const trees = (task.trees || []).map(tree => [tree.id, tree.forest || 1, tree.status || '', tree.fruitPath || '', !!tree.isValidation].join(':')).join('|');
  return [task.id, fileRefreshSignaturePath(task.workspacePath), task.status || '', trees].join('::');
}

async function renderFileViewer(task) {
  const treeFilter = $('fileTreeFilter');
  const fileSelect = $('fileSelect');
  const preview = $('filePreview');
  if (!task?.id || !treeFilter || !fileSelect || !preview) return;
  const token = ++state.fileViewerToken;
  fileSelect.disabled = true;
  fileSelect.innerHTML = `<option>${t('loadingFiles')}</option>`;
  if (!preview.textContent.trim()) setFilePreviewPlain(t('loadingFiles'));
  try {
    const data = await api(taskAPIPath(task.id, '/files'));
    if (token !== state.fileViewerToken) return;
    const allFiles = enrichFilesWithTreeIDs(task, Array.isArray(data.files) ? data.files : []);
    state.lastFileRefreshAt[task.id] = Date.now();
    state.fileListFingerprint[task.id] = allFiles.map(f => `${f.path}:${f.size}:${f.modTime || ''}:${(f.treeIds || []).join(',')}`).join('|');
    const fileForests = fileForestsForTask(task, allFiles);
    renderFileForestFilter(task, fileForests);
    const forestNo = state.selectedForests[task.id];
    if (!fileForests.length || !forestNo) {
      renderFileTreeFilter(task, forestNo, []);
      fileSelect.innerHTML = `<option>${t('noFiles')}</option>`;
      setFilePreviewEmpty(t('noFiles'), state.settings?.language === 'en' ? 'No stage has visible files yet.' : '当前还没有任何阶段产出可查看文件。');
      return;
    }
    const stageFiles = filesForForest(task, allFiles, forestNo);
    renderFileTreeFilter(task, forestNo, stageFiles);
    const treeId = state.selectedFileTree[task.id] || '';
    const files = treeId ? stageFiles.filter(file => (file.treeIds || []).includes(treeId)) : stageFiles;
    fileSelect.innerHTML = '';
    if (!files.length) {
      fileSelect.disabled = true;
      fileSelect.innerHTML = `<option>${t('noFiles')}</option>`;
      setFilePreviewEmpty(t('noFiles'), state.settings?.language === 'en' ? 'This selection has no visible files.' : '当前选择没有产出文件，因此无需查看。');
      return;
    }
    files.forEach(file => {
      const opt = document.createElement('option');
      opt.value = file.path;
      opt.textContent = file.path;
      fileSelect.appendChild(opt);
    });
    const previous = state.selectedFilePath[task.id];
    const selected = files.some(f => f.path === previous) ? previous : files[0].path;
    state.selectedFilePath[task.id] = selected;
    fileSelect.value = selected;
    fileSelect.disabled = false;
    fileSelect.onchange = () => {
      state.selectedFilePath[task.id] = fileSelect.value;
      state.selectedFileManual[task.id] = true;
      loadFilePreview(task, fileSelect.value);
    };
    await loadFilePreview(task, selected, token);
  } catch (err) {
    if (token !== state.fileViewerToken) return;
    fileSelect.disabled = true;
    fileSelect.innerHTML = `<option>${t('noFiles')}</option>`;
    setFilePreviewEmpty(t('noFiles'), `${t('openFailed')}${err.message}`);
  }
}

function isValidationLikeTree(tree) {
  return !!tree?.isValidation || /^\s*验证/.test(String(tree?.name || '')) || /validation/i.test(String(tree?.name || ''));
}

function treeCodeTokens(tree) {
  const text = [tree?.id, tree?.name, tree?.objective, tree?.prompt].join(' ');
  const out = new Set();
  for (const m of text.matchAll(/\b(\d{6})[._-]?(SH|SZ)\b/gi)) {
    const code = m[1];
    const market = m[2].toUpperCase();
    out.add(`${code}_${market}`);
    out.add(`${code}.${market}`);
    out.add(`${market.toLowerCase()}${code}`);
  }
  return out;
}

function enrichFilesWithTreeIDs(task, files) {
  const treeTokens = (task.trees || []).filter(tree => !isValidationLikeTree(tree)).map(tree => ({ tree, tokens: treeCodeTokens(tree) }));
  return files.map(file => {
    const ids = new Set(Array.isArray(file.treeIds) ? file.treeIds : []);
    const haystack = String(file.path || '').toLowerCase();
    treeTokens.forEach(({ tree, tokens }) => {
      for (const token of tokens) {
        if (haystack.includes(token.toLowerCase())) {
          ids.add(tree.id);
          break;
        }
      }
    });
    return { ...file, treeIds: [...ids] };
  });
}

function fileForestsForTask(task, files) {
  const treeForest = new Map((task.trees || []).filter(tree => !isValidationLikeTree(tree)).map(tree => [tree.id, Number(tree.forest || 1)]));
  const forestSet = new Set();
  files.forEach(file => (file.treeIds || []).forEach(id => {
    const no = treeForest.get(id);
    if (no) forestSet.add(no);
  }));
  return getForests(task.trees || []).filter(forest => forestSet.has(Number(forest.no)));
}

function filesForForest(task, files, forestNo) {
  const treeIDs = new Set((task.trees || []).filter(tree => !isValidationLikeTree(tree) && Number(tree.forest || 1) === Number(forestNo)).map(tree => tree.id));
  return files.filter(file => (file.treeIds || []).some(id => treeIDs.has(id)));
}

function renderFileForestFilter(task, fileForests) {
  const select = $('forestSelect');
  const prev = $('prevForestBtn');
  const next = $('nextForestBtn');
  if (!select) return;
  select.innerHTML = '';
  if (!fileForests.length) {
    const opt = document.createElement('option');
    opt.value = '';
    opt.textContent = t('noFiles');
    select.appendChild(opt);
    select.disabled = true;
    if (prev) prev.disabled = true;
    if (next) next.disabled = true;
    delete state.selectedForests[task.id];
    return;
  }
  const current = state.selectedForests[task.id];
  if (!fileForests.some(forest => Number(forest.no) === Number(current))) {
    state.selectedForests[task.id] = fileForests[fileForests.length - 1].no;
  }
  select.disabled = false;
  fileForests.forEach(forest => {
    const opt = document.createElement('option');
    opt.value = String(forest.no);
    opt.textContent = `${t('stage')} ${forest.no}`;
    select.appendChild(opt);
  });
  select.value = String(state.selectedForests[task.id]);
  const currentIndex = Math.max(0, fileForests.findIndex(o => String(o.no) === select.value));
  if (prev) {
    prev.disabled = currentIndex <= 0;
    prev.onclick = () => setSelectedForest(task, fileForests[Math.max(0, currentIndex - 1)].no);
  }
  if (next) {
    next.disabled = currentIndex >= fileForests.length - 1;
    next.onclick = () => setSelectedForest(task, fileForests[Math.min(fileForests.length - 1, currentIndex + 1)].no);
  }
  select.onchange = () => setSelectedForest(task, Number(select.value));
}

function setFilePreviewPlain(text) {
  const preview = $('filePreview');
  if (!preview) return;
  preview.classList.remove('file-preview-empty');
  preview.textContent = text || '';
}

function setFilePreviewEmpty(title, detail = '') {
  const preview = $('filePreview');
  if (!preview) return;
  preview.classList.add('file-preview-empty');
  preview.innerHTML = `
    <div class="file-empty-card" role="status">
      <div class="file-empty-icon" aria-hidden="true">⌁</div>
      <strong>${escapeHTML(title || t('noFiles'))}</strong>
      ${detail ? `<p>${escapeHTML(detail)}</p>` : ''}
    </div>
  `;
}

function renderFileTreeFilter(task, forestNo, stageFiles = []) {
  const treeFilter = $('fileTreeFilter');
  if (!treeFilter) return;
  const fileTreeIDs = new Set(stageFiles.flatMap(file => file.treeIds || []));
  const trees = (task.trees || []).filter(tree => !isValidationLikeTree(tree) && (!forestNo || Number(tree.forest || 1) === Number(forestNo)) && fileTreeIDs.has(tree.id));
  treeFilter.innerHTML = '';
  if (!trees.length) {
    const opt = document.createElement('option');
    opt.value = '';
    opt.textContent = t('noFiles');
    treeFilter.appendChild(opt);
    treeFilter.disabled = true;
    state.selectedFileTree[task.id] = '';
    return;
  }
  treeFilter.disabled = false;
  const all = document.createElement('option');
  all.value = '';
  all.textContent = t('allTreeFiles');
  treeFilter.appendChild(all);
  trees.forEach(tree => {
    const opt = document.createElement('option');
    opt.value = tree.id;
    opt.textContent = tree.name || tree.id;
    treeFilter.appendChild(opt);
  });
  const current = state.selectedFileTree[task.id] || '';
  treeFilter.value = trees.some(tree => tree.id === current) ? current : '';
  state.selectedFileTree[task.id] = treeFilter.value;
  treeFilter.onchange = () => {
    state.selectedFileTree[task.id] = treeFilter.value;
    delete state.selectedFilePath[task.id];
    renderFileViewer(task);
  };
}

async function loadFilePreview(task, relPath, parentToken = 0) {
  const preview = $('filePreview');
  if (!preview || !task?.id || !relPath) return;
  const token = parentToken || ++state.previewToken;
  setFilePreviewPlain(t('loadingFiles'));
  try {
    const text = await fetchText(taskAPIPath(task.id, `/files?path=${encodeURIComponent(relPath)}`));
    if (parentToken && token !== state.fileViewerToken) return;
    if (!parentToken && token !== state.previewToken) return;
    const chars = Array.from(text || '');
    const max = 120_000;
    setFilePreviewPlain(chars.length > max
      ? chars.slice(0, max).join('') + `\n\n${t('previewTruncated').replace('%d', String(max))}`
      : (text || t('emptyResult')));
  } catch (err) {
    if (parentToken && token !== state.fileViewerToken) return;
    if (!parentToken && token !== state.previewToken) return;
    setFilePreviewEmpty(t('fileTooLarge'), `${t('openFailed')}${err.message}`);
  }
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

function applyChatCollapsed() {
  const layout = $('forestView');
  const btn = $('toggleChatBtn');
  if (layout) layout.classList.toggle('chat-collapsed', !!state.chatCollapsed);
  if (btn) {
    btn.textContent = state.chatCollapsed ? t('expandChat') : t('collapseChat');
    btn.setAttribute('aria-pressed', state.chatCollapsed ? 'true' : 'false');
    btn.setAttribute('aria-expanded', state.chatCollapsed ? 'false' : 'true');
  }
}

function toggleChatCollapsed() {
  state.chatCollapsed = !state.chatCollapsed;
  saveChatCollapsed();
  applyChatCollapsed();
  const active = state.tasks.find(task => task.id === state.activeTaskId);
  if (active) {
    invalidateRenderCache(active.id);
    renderTask(active, { skipFileViewer: true });
  }
}

function renderTaskDashboard(task) {
  const panel = $('taskDashboardPanel');
  if (!panel || !task) return;
  const rt = task.runtime || {};
  const runningTrees = Number(rt.runningTrees ?? (task.trees || []).filter(tr => tr.status !== 'Finished').length);
  const severity = severityClass(rt.severity);
  const cue = String(rt.cue || '').trim();
  const phase = humanizePhase(rt.phase || (task.status === 'Finished' ? 'finished' : 'running'));
  const actionCue = taskActionCue(task, rt, cue, runningTrees);
  panel.className = `task-dashboard-panel ${severity}`;
  panel.innerHTML = `
    <div class="task-dashboard-head">
      <strong>${t('taskNow')}</strong>
      <span class="dashboard-cue-pill ${severity}">${escapeHTML(phase)}</span>
    </div>
    <div class="dashboard-cue ${severity}">
      <span>${t('taskNext')}</span>
      <p>${escapeHTML(actionCue)}</p>
    </div>
  `;
}

function taskActionCue(task, rt, cue, runningTrees) {
  const phase = String(rt?.phase || '');
  if (task?.awaitingUserInput) return t('awaitingUserInput');
  if (task?.status === 'Finished') return t('finishedTaskHint');
  if (String(rt?.severity || '') === 'blocked' && cue) return cue;
  if (phase === 'planning') return t('planningTask');
  if (phase === 'validating' || phase === 'deciding') return t('checkingResults');
  if (runningTrees > 0) return t('workingNormally');
  return cue || t('askProgressSafe');
}

function humanizePhase(phase) {
  const lang = state.settings?.language || 'zh-CN';
  const zh = {
    planning:'规划中', running_subtasks:'子任务执行中', validating:'验证中', deciding:'判断下一步', running:'运行中', awaiting_user:'等待补充需求', finished:'已完成', stopped:'已停止', unknown:'未知'
  };
  const en = {
    planning:'Planning', running_subtasks:'Running subtasks', validating:'Validating', deciding:'Deciding', running:'Running', awaiting_user:'Waiting for clarification', finished:'Finished', stopped:'Stopped', unknown:'Unknown'
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
  const progressLabel = task.status === 'Finished' ? statusText('Finished') : t('gardenerWorking');
  panel.innerHTML = `
    <span class="overview-mini-title">${t('overview')}</span>
    <span class="overview-mini-chip ${task.status || 'Running'}"><small>${t('gardenerProgress')}</small><b>${progressLabel}</b></span>
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

async function commitTitleEdit() {
  if (!state.editingTitle) return;
  const task = state.tasks.find(t => t.id === state.activeTaskId);
  if (!task) return cancelTitleEdit(false);
  const input = $('titleEditInput');
  const next = input.value.trim();
  if (!next || next === task.title) return cancelTitleEdit(true);
  $('renameTaskBtn').disabled = true;
  try {
    const data = await api(taskAPIPath(task.id), { method:'PATCH', body: JSON.stringify({ title: next }) });
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
  $('treeSummaryText').textContent = '';
  select.onchange = () => setSelectedForest(task, Number(select.value));
}

function setSelectedForest(task, forestNo) {
  state.selectedForests[task.id] = Number(forestNo);
  delete state.selectedFilePath[task.id];
  state.selectedFileTree[task.id] = '';
  renderForest(task);
  hideUsagePanel();
  renderTreeStatus(task);
  renderFileViewer(task);
}



function confirmDeleteTask(task) {
  closeDeleteConfirm(false);
  const overlay = $('deleteConfirmOverlay');
  const name = $('deleteConfirmTaskName');
  const confirmBtn = $('confirmDeleteTaskBtn');
  if (!overlay || !confirmBtn) return Promise.resolve(window.confirm(t('deleteConfirm')));
  pendingDeleteTaskId = task?.id || '';
  if (name) {
    name.textContent = task?.title || task?.id || t('genericTask');
    name.title = name.textContent;
  }
  overlay.classList.remove('hidden');
  overlay.setAttribute('aria-hidden', 'false');
  confirmBtn.disabled = false;
  setTimeout(() => confirmBtn.focus(), 0);
  return new Promise(resolve => { pendingDeleteResolve = resolve; });
}

function closeDeleteConfirm(result = false) {
  const overlay = $('deleteConfirmOverlay');
  if (overlay) {
    overlay.classList.add('hidden');
    overlay.setAttribute('aria-hidden', 'true');
  }
  pendingDeleteTaskId = '';
  if (pendingDeleteResolve) {
    const resolve = pendingDeleteResolve;
    pendingDeleteResolve = null;
    resolve(!!result);
  }
}

async function deleteTask(taskId) {
  const task = state.tasks.find(t => t.id === taskId) || { id: taskId, title: taskId };
  const confirmed = await confirmDeleteTask(task);
  if (!confirmed) return;
  try {
    await api(taskAPIPath(taskId), { method:'DELETE' });
    state.tasks = state.tasks.filter(task => task.id !== taskId);
    if (state.activeTaskId === taskId) backToList({ replaceRoute: true });
    renderTaskList();
    renderHomeGarden();
  } catch (err) {
    alert((t('deleteFailed') || 'Delete failed: ') + err.message);
  }
}

function hideUsagePanel() {
  const panel = $('usagePanel');
  if (panel) { panel.classList.add('hidden'); panel.innerHTML = ''; }
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
    const data = await api(taskAPIPath(task.id, '/usage'));
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

const MAX_RENDERED_USAGE_MODEL_CHARS = 120;
function renderedUsageModelName(model) {
  const chars = Array.from(String(model || 'unknown'));
  return chars.length > MAX_RENDERED_USAGE_MODEL_CHARS ? chars.slice(0, MAX_RENDERED_USAGE_MODEL_CHARS).join('') + '…' : chars.join('');
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
      <span class="usage-model-name">${escapeHTML(renderedUsageModelName(m.model))}</span>
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
  const status = running ? `<span class="gardener-progress-live"><span></span>${t('gardenerWorking')}</span>` : `<span class="status-pill Finished">${statusText('Finished')}</span>`;
  const items = rows.length
    ? rows.map(line => {
        const parsed = parseProgressDisplayLine(line);
        return `<li><time>${escapeHTML(parsed.time)}</time><span>${escapeHTML(humanizeText(parsed.text))}</span></li>`;
      }).join('')
    : `<li class="empty-progress"><span>${t('gardenerProgressEmpty')}</span></li>`;
  panel.innerHTML = `
    <div class="gardener-progress-head">
      <strong>${t('gardenerProgress')}</strong>
      ${status}
    </div>
    <ol class="gardener-progress-list">${items}</ol>
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
  panel.classList.add('hidden');
  panel.innerHTML = '';
}

function setTaskReportLink(anchor, url, title) {
  anchor.href = url;
  anchor.onclick = e => { e.preventDefault(); openReport(url, title); };
}

function setFruitLink(anchor, taskId, tree) {
  const url = taskAPIPath(taskId, `/trees/${tree.id}/fruit.md`);
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

const MAX_RENDERED_REPORT_ERROR_CHARS = 300;
function renderedReportErrorMessage(err) {
  const chars = Array.from(String(err?.message || err || ''));
  return chars.length > MAX_RENDERED_REPORT_ERROR_CHARS ? chars.slice(0, MAX_RENDERED_REPORT_ERROR_CHARS).join('') + '…' : chars.join('');
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
    $('reportBody').innerHTML = `<div class="report-loading">${t('openFailed')}${escapeHTML(renderedReportErrorMessage(err))}</div>`;
  }
}

function closeReport() {
  $('reportOverlay').classList.add('hidden');
  $('reportOverlay').setAttribute('aria-hidden', 'true');
}

const MAX_JSON_FORMAT_CHARS = 200000;
const MAX_JSONL_FORMAT_LINES = 1000;

function renderJSON(text) {
  const raw = String(text || '');
  if (!raw.trim()) return `<div class="report-loading">${t('emptyResult')}</div>`;
  let formatted = raw;
  try {
    if (raw.length <= MAX_JSON_FORMAT_CHARS) {
      if (/\.jsonl$/i.test(state.selectedFilePath[state.activeTaskId] || '')) {
        const lines = raw.split(/\r?\n/).filter(Boolean);
        if (lines.length <= MAX_JSONL_FORMAT_LINES) formatted = lines.map(line => JSON.stringify(JSON.parse(line), null, 2)).join('\n');
      } else {
        formatted = JSON.stringify(JSON.parse(raw), null, 2);
      }
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
  const rows = parseCSV(text, MAX_CSV_PREVIEW_ROWS + 1);
  if (!rows.length) return `<div class="report-loading">${t('emptyResult')}</div>`;
  const maxRows = MAX_CSV_PREVIEW_ROWS;
  const visibleRows = rows.slice(0, maxRows);
  const colCount = Math.max(...visibleRows.map(r => r.length));
  const normalized = visibleRows.map(r => Array.from({ length: colCount }, (_, i) => r[i] ?? ''));
  const [head, ...body] = normalized;
  const tableHead = `<thead><tr>${head.map(c => `<th>${escapeHTML(c)}</th>`).join('')}</tr></thead>`;
  const tableBody = `<tbody>${body.map(r => `<tr>${r.map(c => `<td>${escapeHTML(c)}</td>`).join('')}</tr>`).join('')}</tbody>`;
  const note = rows.length > maxRows ? `<div class="csv-note">Only showing first ${maxRows} rows.</div>` : '';
  return `${note}<div class="csv-table-wrap"><table>${tableHead}${tableBody}</table></div>`;
}

function parseCSV(text, maxRows = Infinity) {
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
    if (ch === '\n') { row.push(field); rows.push(row); if (rows.length >= maxRows) return rows.filter(r => r.some(c => String(c).trim() !== '')); row = []; field = ''; continue; }
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

function severityClass(severity) { return ['ok', 'info', 'warning', 'blocked'].includes(severity) ? severity : 'ok'; }
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

const MAX_RENDERED_CREATE_TASK_ERROR_CHARS = 300;
function renderedCreateTaskErrorMessage(err) {
  const chars = Array.from(String(err?.message || err || ''));
  return chars.length > MAX_RENDERED_CREATE_TASK_ERROR_CHARS ? chars.slice(0, MAX_RENDERED_CREATE_TASK_ERROR_CHARS).join('') + '…' : chars.join('');
}

let currentDirectoryPath = '';

const MAX_RENDERED_DIRECTORY_TEXT_CHARS = 500;
function renderedDirectoryText(value) {
  const chars = Array.from(String(value || ''));
  return chars.length > MAX_RENDERED_DIRECTORY_TEXT_CHARS ? chars.slice(0, MAX_RENDERED_DIRECTORY_TEXT_CHARS).join('') + '…' : chars.join('');
}

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
  $('directoryPath').textContent = renderedDirectoryText(currentDirectoryPath);
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
    btn.innerHTML = `<span>📁</span><strong>${escapeHTML(renderedDirectoryText(entry.name))}</strong>`;
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
    forceNextChatScrollToBottom();
    upsertTask(optimistic);
    renderTask(optimistic, { skipFileViewer: true });
  }
  try {
    const data = await api(taskAPIPath(taskId, '/resume'), { method:'POST', body:'{}' });
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
  catch (err) { alert(renderedCreateTaskErrorMessage(err)); } finally { $('createTaskBtn').disabled = false; }
};
if ($('toggleOverviewBtn')) $('toggleOverviewBtn').onclick = toggleOverviewCollapsed;
if ($('toggleChatBtn')) $('toggleChatBtn').onclick = toggleChatCollapsed;
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
      awaitingUserInput: false,
      messages: [...(originalTask.messages || []), { id: `local_${Date.now()}`, role: 'user', content, createdAt: new Date().toISOString() }]
    };
    forceNextChatScrollToBottom();
    upsertTask(optimistic);
    renderTask(optimistic, { skipFileViewer: true });
  }
  try {
    const data = await api(taskAPIPath(taskId, '/messages'), { method:'POST', body: JSON.stringify({ content }) });
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
$('stopTaskBtn').onclick = async () => { if (!state.activeTaskId) return; if (!confirm(t('stopConfirm'))) return; $('stopTaskBtn').disabled = true; try { await api(taskAPIPath(state.activeTaskId, '/stop'), { method:'POST', body:'{}' }); } catch(err){ alert(err.message); } };

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
if ($('messages')) $('messages').addEventListener('scroll', updateJumpToLatestButton, { passive: true });
if ($('jumpToLatestBtn')) $('jumpToLatestBtn').onclick = () => scrollMessagesToBottom({ smooth: true });
window.addEventListener('resize', updateJumpToLatestButton);
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
if ($('homeTaskSearchInput')) {
  $('homeTaskSearchInput').addEventListener('input', e => {
    state.homeTaskSearch = e.target.value || '';
    state.lastHomeSig = '';
    renderHomeGarden();
  });
  $('homeTaskSearchInput').addEventListener('search', e => {
    state.homeTaskSearch = e.target.value || '';
    state.lastHomeSig = '';
    renderHomeGarden();
  });
}
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
  state.settings.modelMode = normalizeModelModeValue($('modelModeSelect').value);
  state.settings.cliEngine = compatibleCLIEngineValue($('cliEngineSelect').value, state.settings.modelMode);
  setCurrentModelToken($('modelTokenInput').value.trim());
  await saveSettings();
  closeSettings();
};
$('modelModeSelect').onchange = () => {
  state.settings.modelMode = normalizeModelModeValue($('modelModeSelect').value);
  state.settings.cliEngine = compatibleCLIEngineValue($('cliEngineSelect').value, state.settings.modelMode);
  $('cliEngineSelect').value = state.settings.cliEngine;
  applyModelTokenField();
};
$('cliEngineSelect').onchange = () => {
  state.settings.cliEngine = compatibleCLIEngineValue($('cliEngineSelect').value, normalizeModelModeValue(state.settings.modelMode || 'default'));
  $('cliEngineSelect').value = state.settings.cliEngine;
};
$('modelTokenInput').oninput = () => setCurrentModelToken($('modelTokenInput').value.trim());
$('closeReportBtn').onclick = closeReport;
$('reportOverlay').onclick = e => { if (e.target === $('reportOverlay')) closeReport(); };
$('settingsOverlay').onclick = e => { if (e.target === $('settingsOverlay')) closeSettings(); };
if ($('deleteConfirmOverlay')) $('deleteConfirmOverlay').onclick = e => { if (e.target === $('deleteConfirmOverlay')) closeDeleteConfirm(false); };
if ($('cancelDeleteConfirmBtn')) $('cancelDeleteConfirmBtn').onclick = () => closeDeleteConfirm(false);
if ($('confirmDeleteTaskBtn')) $('confirmDeleteTaskBtn').onclick = () => closeDeleteConfirm(true);
$('copyReportBtn').onclick = async () => { if (!state.activeReportText) return; try { await navigator.clipboard.writeText(state.activeReportText); } catch {} };
document.addEventListener('keydown', e => { if (e.key === 'Escape') { closeDeleteConfirm(false); closeReport(); closeSettings(); closeDirectoryPicker(); } });
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
