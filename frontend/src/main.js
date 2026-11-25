import './style.css';
import {
  CheckAll,
  StopCheckAll,
  ClearProxies,
  ExportGoodProxies,
  GetState,
  GetVersion,
  LoadFilePath,
  LoadURL,
  RemoveDeadProxies,
  SaveSettings,
  SelectProxyFile,
  SetManualParent,
  SetManualParentFromList,
  SetParentMode,
  StartMonitor,
  StartThreeProxy,
  StopMonitor,
  StopThreeProxy,
} from '../wailsjs/go/main/DesktopApp';

const app = document.querySelector('#app');

app.innerHTML = `
  <header class="topbar">
    <div>
      <h1>Proxy Checker</h1>
      <p>Загружай прокси -> Проверить все -> Запусти 3proxy -> Настрой parent -> Старт монитор</p>
    </div>
    <div class="actions">
      <button id="checkAll">Проверить все</button>
      <button id="stopCheck" class="secondary" style="display:none">Стоп проверку</button>
      <button id="startMonitor" class="secondary">Старт монитор</button>
      <button id="stopMonitor" class="danger">Стоп монитор</button>
    </div>
  </header>

  <main class="layout">
    <section class="settings">

      <!-- БЛОК: ЗАГРУЗКА -->
      <div class="collapsible-block">
        <button class="collapse-btn" data-target="block-load">▼ Загрузка прокси</button>
        <div id="block-load" class="collapse-body">
          <label>API URL</label>
          <input id="apiUrl" type="text" placeholder="https://proxy-example.com/api/..." />
          <label>Тип прокси для строк без протокола</label>
          <select id="proxyTypeMode">
            <option value="auto">AUTO: проверить HTTP и SOCKS5</option>
            <option value="connect">HTTP</option>
            <option value="socks5">SOCKS5</option>
          </select>
          <label class="check">
            <input id="autoImport" type="checkbox" />
            Авто-импорт прокси из API
          </label>
          <label>Интервал авто-импорта</label>
          <div class="two">
            <input id="autoImportValue" type="number" min="1" />
            <select id="autoImportUnit">
              <option value="second">секунды</option>
              <option value="minute">минуты</option>
              <option value="day">дни</option>
              <option value="week">недели</option>
              <option value="month">месяцы</option>
            </select>
          </div>
          <div class="button-row">
            <button id="loadApi">Загрузить API</button>
            <button id="loadFile" class="secondary">Выбрать файл</button>
          </div>
          <button id="clearProxies" class="danger" style="width:100%;margin-top:6px">Очистить список прокси</button>
        </div>
      </div>

      <!-- БЛОК: ПРОВЕРКА -->
      <div class="collapsible-block">
        <button class="collapse-btn" data-target="block-check">▼ Проверка прокси</button>
        <div id="block-check" class="collapse-body">
          <label>URL проверки</label>
          <input id="testUrl" type="text" />
          <div class="two">
            <div>
              <label>Timeout, сек</label>
              <input id="checkTimeoutSec" type="number" min="1" />
            </div>
            <div>
              <label>Потоков</label>
              <input id="workers" type="number" min="1" />
            </div>
          </div>
          <label class="check">
            <input id="geoLookup" type="checkbox" />
            Определять страну, регион и часовой пояс
          </label>
          <label>Geo provider</label>
          <select id="geoProvider">
            <option value="auto">AUTO: много источников подряд</option>
            <option value="2ip">2ip</option>
            <option value="2ip_json">2ip JSON</option>
            <option value="ipinfo">ipinfo</option>
            <option value="ipapi">ip-api.com</option>
            <option value="ifconfig">ifconfig.co</option>
            <option value="ipwhois">ipwho.is</option>
            <option value="ipapi_co">ipapi.co</option>
            <option value="ip_sb">api.ip.sb</option>
          </select>
          <label>URL geo API</label>
          <input id="geoLookupUrl" type="text" placeholder="Провайдер подставит URL автоматически" />
          <label class="check">
            <input id="allowInsecure" type="checkbox" />
            TLS без строгой проверки, как curl --insecure
          </label>
          <label>Авто-перепроверка всех прокси (сек, 0 = выключено)</label>
          <input id="recheckEverySec" type="number" min="0" placeholder="0" />
          <p class="hint">Монитор будет автоматически перезапускать проверку всех прокси с указанным интервалом. Требует запущенного монитора.</p>
          <button id="saveSettings">Сохранить настройки</button>
        </div>
      </div>

      <!-- БЛОК: 3PROXY -->
      <div class="collapsible-block">
        <button class="collapse-btn" data-target="block-3proxy">▼ 3proxy</button>
        <div id="block-3proxy" class="collapse-body">
          <label>Путь к 3proxy.exe, если не встроен</label>
          <input id="exePath" type="text" placeholder="Можно оставить пустым для embedded/3proxy" />
          <label>Рабочая папка</label>
          <input id="workDir" type="text" />
          <div class="two">
            <div>
              <label>Proxy port</label>
              <input id="proxyPort" type="text" />
            </div>
            <div>
              <label>Admin port</label>
              <input id="adminPort" type="text" />
            </div>
          </div>
          <label>Allowed IP</label>
          <input id="allowedIp" type="text" />
          <div class="button-row">
            <button id="start3proxy" class="secondary">Запустить 3proxy</button>
            <button id="stop3proxy" class="danger">Остановить</button>
          </div>
        </div>
      </div>

      <!-- БЛОК: УПРАВЛЕНИЕ PARENT -->
      <div class="collapsible-block">
        <button class="collapse-btn" data-target="block-parent">▼ Управление parent</button>
        <div id="block-parent" class="collapse-body">

          <div class="mode-tabs">
            <button class="mode-tab active" id="tabAuto" data-mode="auto">Авто</button>
            <button class="mode-tab" id="tabManual" data-mode="manual">Ручной</button>
          </div>

          <!-- АВТО: монитор переключает из проверенных -->
          <div id="autoParentBlock" class="mode-panel">
            <p class="hint">Монитор каждые N секунд берёт следующий рабочий прокси из таблицы и применяет как parent 3proxy. Сначала запусти <b>Проверить все</b>.</p>
            <label>Интервал смены parent (сек)</label>
            <input id="monitorEverySec" type="number" min="5" placeholder="120" />
            <label>Страна (фильтр пула, пусто = любая)</label>
            <select id="monitorCountrySelect">
              <option value="">Любая страна</option>
            </select>
            <p class="hint" id="autoParentStatus"></p>
            <button id="saveMonitorSettings" class="secondary" style="width:100%;margin-top:8px">Сохранить и применить</button>
          </div>

          <!-- РУЧНОЙ: фиксированный прокси -->
          <div id="manualParentBlock" class="mode-panel hidden">
            <p class="hint">Укажите конкретный прокси. Монитор будет применять только его, без автоматической смены.</p>

            <div class="sub-section">
              <p class="sub-label">Выбрать из проверенных</p>
              <label>Фильтр по стране</label>
              <select id="parentCountryFilter">
                <option value="all">Все страны</option>
              </select>
              <div class="proxy-select-opts">
                <label class="cb-label"><input type="checkbox" id="showLabel" checked> Метка</label>
                <label class="cb-label"><input type="checkbox" id="showIP"> IP</label>
              </div>
              <div class="proxy-search-wrap">
                <input type="text" id="parentProxySearch" placeholder="Поиск по метке или IP..." autocomplete="off">
                <div id="parentProxyList" class="proxy-dropdown hidden"></div>
                <input type="hidden" id="parentProxySelect">
              </div>
              <button id="applyFromList" class="secondary" style="width:100%;margin-top:8px">Применить выбранный</button>
            </div>

            <div class="separator"></div>

            <div class="sub-section">
              <p class="sub-label">Ввести вручную</p>
              <div class="two">
                <div>
                  <label>IP / хост</label>
                  <input id="manualHost" type="text" placeholder="1.2.3.4" />
                </div>
                <div>
                  <label>Порт</label>
                  <input id="manualPort" type="text" placeholder="1080" />
                </div>
              </div>
              <label>Тип</label>
              <select id="manualType">
                <option value="socks5">SOCKS5</option>
                <option value="connect">HTTP</option>
              </select>
              <div class="two" style="margin-top:6px">
                <div>
                  <label>Логин</label>
                  <input id="manualLogin" type="text" placeholder="(необязательно)" />
                </div>
                <div>
                  <label>Пароль</label>
                  <input id="manualPassword" type="text" placeholder="(необязательно)" />
                </div>
              </div>
              <div class="button-row" style="margin-top:8px">
                <button id="applyManualParent" class="secondary">Применить</button>
                <button id="clearManualParent" class="danger">Сбросить</button>
              </div>
            </div>

            <div id="currentManualParent" class="active-parent-badge active-parent-none" style="display:none"></div>
          </div>

        </div>
      </div>

    </section>

    <section class="workspace">
      <div class="stats">
        <div class="metric"><b id="total">0</b><span>Всего</span></div>
        <div class="metric"><b id="good">0</b><span>Рабочих</span></div>
        <div class="metric"><b id="monitor">off</b><span>Монитор</span></div>
        <div class="metric"><b id="threeproxy">off</b><span>3proxy</span></div>
      </div>

      <section class="table-wrap">
        <div class="section-head">
          <div>
            <h2>Прокси</h2>
            <div id="active" class="active-parent-badge active-parent-none">Активный parent: нет</div>
          </div>
          <div class="actions">
            <button id="removeDead" class="danger">Удалить FAIL</button>
            <button id="exportGood" class="secondary">Сохранить good-proxies.txt</button>
          </div>
        </div>
        <div class="filters">
          <div>
            <label>Статус</label>
            <select id="statusFilter">
              <option value="all">Все</option>
              <option value="ok">Только рабочие</option>
              <option value="fail">Только с ошибкой</option>
              <option value="unchecked">Не проверенные</option>
            </select>
          </div>
          <div>
            <label>Страна</label>
            <select id="countryFilter">
              <option value="all">Все страны</option>
            </select>
          </div>
        </div>
        <table>
          <thead>
            <tr><th>Прокси</th><th>Тип</th><th>Логин</th><th>Метка</th><th>Статус</th><th>Страна</th><th>Регион</th><th>HTTP / ошибка</th><th>Время</th></tr>
          </thead>
          <tbody id="proxyRows"></tbody>
        </table>
      </section>

      <section class="bottom">
        <div>
          <h2>Лог</h2>
          <pre id="logs"></pre>
        </div>
        <div>
          <h2>Файлы 3proxy</h2>
          <pre id="paths"></pre>
        </div>
      </section>
    </section>
  </main>

  <footer class="footer">
    <div id="appInfo" class="app-info">Loading version...</div>
  </footer>

  <div id="toast" class="toast"></div>
`;

let state = null;
let settingsDirty = false;
let applyingSettings = false;
let filtersBound = false;

const el = (id) => document.getElementById(id);

// --- Сворачиваемые блоки ---
document.querySelectorAll('.collapse-btn').forEach((btn) => {
  btn.addEventListener('click', () => {
    const targetId = btn.dataset.target;
    const body = document.getElementById(targetId);
    const collapsed = body.classList.toggle('collapsed');
    btn.textContent = btn.textContent.replace(collapsed ? '▼' : '▶', collapsed ? '▶' : '▼');
  });
});

// --- Табы режима parent ---
function setParentMode(mode) {
  el('tabAuto').classList.toggle('active', mode === 'auto');
  el('tabManual').classList.toggle('active', mode === 'manual');
  el('autoParentBlock').classList.toggle('hidden', mode !== 'auto');
  el('manualParentBlock').classList.toggle('hidden', mode !== 'manual');
}

document.querySelectorAll('.mode-tab').forEach((tab) => {
  tab.addEventListener('click', () => {
    const mode = tab.dataset.mode;
    setParentMode(mode);
    run(() => SetParentMode(mode));
  });
});

el('saveMonitorSettings').addEventListener('click', () => run(saveSettings));

el('applyFromList').addEventListener('click', () => {
  const key = el('parentProxySelect').value;
  if (!key) { showToast('Выберите прокси из списка', false); return; }
  run(() => SetManualParentFromList(key));
});

el('applyManualParent').addEventListener('click', () => {
  const host = el('manualHost').value.trim();
  const port = el('manualPort').value.trim();
  if (!host || !port) { showToast('Введите хост и порт', false); return; }
  run(() => SetManualParent(host, port, el('manualType').value, el('manualLogin').value.trim(), el('manualPassword').value.trim()));
});

el('clearManualParent').addEventListener('click', () => {
  run(() => SetManualParent('', '', '', '', ''));
});

el('clearProxies').addEventListener('click', () => {
  run(() => ClearProxies());
});

el('parentCountryFilter').addEventListener('change', () => {
  if (state) syncParentProxySelect(state.proxies || [], buildResultMap(state.results || []));
});

el('showLabel').addEventListener('change', () => {
  if (state) syncParentProxySelect(state.proxies || [], buildResultMap(state.results || []));
});
el('showIP').addEventListener('change', () => {
  if (state) syncParentProxySelect(state.proxies || [], buildResultMap(state.results || []));
});

el('parentProxySearch').addEventListener('input', () => {
  renderProxyDropdown(el('parentProxySearch').value);
  el('parentProxySelect').value = '';
});
el('parentProxySearch').addEventListener('focus', () => {
  renderProxyDropdown(el('parentProxySearch').value);
});
el('parentProxySearch').addEventListener('blur', () => {
  setTimeout(() => el('parentProxyList').classList.add('hidden'), 150);
});

async function refresh() {
  try {
    const s = await GetState();
    if (!s || !s.proxies) return;
    state = s;
  } catch (e) {
    return;
  }
  if (!settingsDirty) fillSettings(state.config);

  const resultByKey = buildResultMap(state.results || []);
  let good = 0;
  for (const result of state.results || []) {
    if (result.ok) good += 1;
  }

  el('total').textContent = state.proxies.length;
  el('good').textContent = good;
  el('monitor').textContent = state.monitorRunning ? 'on' : 'off';
  el('threeproxy').textContent = state.threeProxyRun ? 'on' : 'off';
  // Кнопки проверки
  const checkBtn = el('checkAll');
  const stopCheckBtn = el('stopCheck');
  if (state.checkRunning) {
    checkBtn.textContent = 'Проверка...';
    checkBtn.disabled = true;
    stopCheckBtn.style.display = '';
  } else {
    checkBtn.textContent = 'Проверить все';
    checkBtn.disabled = false;
    stopCheckBtn.style.display = 'none';
  }
  const activeEl = el('active');
  if (state.activeProxy) {
    const ap = state.activeProxy;
    const labelPart = ap.label ? `<span class="active-parent-label">${escapeHtml(ap.label)}</span>` : '';
    activeEl.innerHTML = `Активный parent: <span class="active-parent-ip">${escapeHtml(ap.host)}</span><span class="active-parent-port">:${escapeHtml(ap.port)}</span>${labelPart}`;
    activeEl.className = 'active-parent-badge active-parent-on';
  } else {
    activeEl.textContent = 'Активный parent: нет';
    activeEl.className = 'active-parent-badge active-parent-none';
  }

  // Синхронизируем режим parent (без сброса если пользователь уже нажал таб)
  const mode = state.parentMode || 'auto';
  setParentMode(mode);

  // Статус авто-parent
  const autoStatus = el('autoParentStatus');
  if (state.activeProxy && mode === 'auto') {
    autoStatus.textContent = `Активный: ${state.activeProxy.host}:${state.activeProxy.port}`;
    autoStatus.style.color = '#27ae60';
  } else if (mode === 'auto') {
    autoStatus.textContent = good > 0 ? `Готово ${good} прокси. Запусти монитор.` : 'Нет проверенных прокси. Нажми "Проверить все".';
    autoStatus.style.color = '#888';
  }

  const manualBadge = el('currentManualParent');
  if (state.manualParent) {
    const mp = state.manualParent;
    const labelPart = mp.label ? `<span class="active-parent-label">${escapeHtml(mp.label)}</span>` : '';
    manualBadge.innerHTML = `Применён: <span class="active-parent-ip">${escapeHtml(mp.host)}</span><span class="active-parent-port">:${escapeHtml(mp.port)}</span>${labelPart}`;
    manualBadge.className = 'active-parent-badge active-parent-on';
    manualBadge.style.display = '';
  } else if (mode === 'manual') {
    manualBadge.textContent = 'Не установлен — трафик заблокирован';
    manualBadge.className = 'active-parent-badge active-parent-none';
    manualBadge.style.display = '';
  } else {
    manualBadge.style.display = 'none';
  }

  const cfg = state.config.threeProxy;
  el('paths').textContent = [
    `3proxy.cfg: ${cfg.workDir}\\3proxy.cfg`,
    `reload.txt: ${cfg.workDir}\\reload.txt`,
    `embedded: ${cfg.workDir}\\embedded-3proxy`,
  ].join('\n');

  el('logs').textContent = (state.logs || []).slice().reverse().join('\n');
  syncCountryFilter(state.proxies, resultByKey);
  syncMonitorCountrySelect(state.proxies, resultByKey, state.config.monitorCountry || '');
  syncParentProxySelect(state.proxies, resultByKey);

  const filteredRows = state.proxies
    .map((proxy) => ({ proxy, result: resultByKey.get(proxyKey(proxy)) }))
    .filter(({ proxy, result }) => matchesFilters(proxy, result))
    .map(({ proxy, result }) => proxyRow(proxy, result));
  el('proxyRows').innerHTML = filteredRows.join('');
}

function buildResultMap(results) {
  const map = new Map();
  for (const result of results) {
    map.set(proxyKey(result.proxy), result);
  }
  return map;
}

function proxyRow(proxy, result) {
  let status = '<span class="muted">не проверен</span>';
  let code = '';
  let duration = '';
  let effectiveType = proxy.type;
  let country = '';
  let region = '';
  let timezone = '';
  if (result) {
    status = result.ok ? '<span class="ok">OK</span>' : '<span class="fail">FAIL</span>';
    code = result.statusCode || result.error || '';
    duration = result.duration ? `${Math.round(result.duration / 1000000)} ms` : '';
    effectiveType = result.proxy?.type || effectiveType;
    country = result.geo?.country || '';
    region = result.geo?.region || '';
    timezone = result.geo?.timezone || '';
  }
  return `
    <tr>
      <td>${escapeHtml(proxy.host)}:${escapeHtml(proxy.port)}</td>
      <td>${escapeHtml(proxyTypeLabel(effectiveType))}</td>
      <td>${escapeHtml(proxy.login || '')}</td>
      <td>${proxy.label ? `<span class="proxy-label">${escapeHtml(proxy.label)}</span>` : ''}</td>
      <td>${status}</td>
      <td>${escapeHtml(country)}</td>
      <td>${escapeHtml(region)}</td>
      <td>${escapeHtml(String(code))}</td>
      <td>${escapeHtml(duration)}</td>
    </tr>`;
}

function fillSettings(config) {
  if (!config) return;
  applyingSettings = true;
  setValue('apiUrl', config.proxyApiUrl);
  setValue('proxyTypeMode', config.proxyTypeMode || 'auto');
  el('autoImport').checked = Boolean(config.autoImport);
  setAutoImportInterval(config.autoImportSec, config.autoImportUnit);
  setValue('testUrl', config.testUrl);
  setValue('checkTimeoutSec', config.checkTimeoutSec);
  setValue('workers', config.workers);
  setValue('monitorEverySec', config.monitorEverySec);
  setValue('recheckEverySec', config.recheckEverySec || 0);
  el('geoLookup').checked = Boolean(config.geoLookup);
  setValue('geoProvider', config.geoProvider || 'auto');
  setValue('geoLookupUrl', config.geoLookupUrl || 'https://ipinfo.io/json');
  syncGeoLookupUrl();
  el('allowInsecure').checked = Boolean(config.allowInsecure);
  const tp = config.threeProxy;
  setValue('exePath', tp.exePath);
  setValue('workDir', tp.workDir);
  setValue('proxyPort', tp.proxyPort);
  setValue('adminPort', tp.adminPort);
  setValue('allowedIp', tp.allowedIp);
  applyingSettings = false;
}

function readSettings() {
  const config = structuredClone(state.config);
  config.proxyApiUrl = value('apiUrl');
  config.proxyTypeMode = value('proxyTypeMode');
  config.autoImport = el('autoImport').checked;
  config.autoImportUnit = value('autoImportUnit');
  config.autoImportSec = intervalToSeconds(numberValue('autoImportValue'), config.autoImportUnit);
  config.testUrl = value('testUrl');
  config.checkTimeoutSec = numberValue('checkTimeoutSec');
  config.workers = numberValue('workers');
  config.monitorEverySec = numberValue('monitorEverySec');
  config.recheckEverySec = numberValue('recheckEverySec');
  config.monitorCountry = el('monitorCountrySelect').value || '';
  config.geoLookup = el('geoLookup').checked;
  config.geoProvider = value('geoProvider') || 'auto';
  config.geoLookupUrl = value('geoLookupUrl');
  config.allowInsecure = el('allowInsecure').checked;
  config.threeProxy.exePath = value('exePath');
  config.threeProxy.workDir = value('workDir');
  config.threeProxy.proxyPort = value('proxyPort');
  config.threeProxy.adminPort = value('adminPort');
  config.threeProxy.allowedIp = value('allowedIp');
  return config;
}

async function run(action) {
  try {
    const response = await action();
    if (response?.message) showToast(response.message, response.ok);
  } catch (error) {
    showToast(error.message || String(error), false);
  } finally {
    await refresh();
  }
}

async function saveSettings() {
  const response = await SaveSettings(readSettings());
  if (response?.ok) settingsDirty = false;
  return response;
}

el('saveSettings').addEventListener('click', () => run(saveSettings));
el('loadApi').addEventListener('click', () => run(async () => LoadURL(value('apiUrl'))));
el('checkAll').addEventListener('click', () => run(async () => {
  await saveSettings();
  return CheckAll();
}));
el('stopCheck').addEventListener('click', () => run(() => StopCheckAll()));
el('startMonitor').addEventListener('click', () => run(async () => {
  await saveSettings();
  return StartMonitor();
}));
el('stopMonitor').addEventListener('click', () => run(StopMonitor));
el('start3proxy').addEventListener('click', () => run(async () => {
  await saveSettings();
  return StartThreeProxy();
}));
el('stop3proxy').addEventListener('click', () => run(StopThreeProxy));
el('exportGood').addEventListener('click', () => run(ExportGoodProxies));
el('removeDead').addEventListener('click', () => run(RemoveDeadProxies));
el('loadFile').addEventListener('click', () => run(async () => {
  const file = await SelectProxyFile();
  if (!file) return { ok: true, message: 'file selection cancelled' };
  return LoadFilePath(file);
}));

function setValue(id, next) {
  const input = el(id);
  if (!input) return;
  if (document.activeElement !== input) input.value = next ?? '';
}

function setAutoImportInterval(seconds, unit) {
  const selectedUnit = unit || bestIntervalUnit(seconds || 3600);
  setValue('autoImportUnit', selectedUnit);
  setValue('autoImportValue', Math.max(1, Math.round((seconds || 3600) / unitSeconds(selectedUnit))));
}

function value(id) {
  const input = el(id);
  return input ? input.value.trim() : '';
}

function numberValue(id) {
  return Number.parseInt(value(id), 10) || 0;
}

function intervalToSeconds(amount, unit) {
  return Math.max(1, amount || 1) * unitSeconds(unit);
}

function unitSeconds(unit) {
  switch (unit) {
    case 'second': return 1;
    case 'minute': return 60;
    case 'day': return 86400;
    case 'week': return 604800;
    case 'month': return 2592000;
    default: return 3600;
  }
}

function bestIntervalUnit(seconds) {
  if (seconds % 2592000 === 0) return 'month';
  if (seconds % 604800 === 0) return 'week';
  if (seconds % 86400 === 0) return 'day';
  if (seconds % 60 === 0) return 'minute';
  return 'second';
}

function proxyKey(proxy) {
  return [proxy.host, proxy.port, proxy.login || '', proxy.password || ''].join('|').toLowerCase();
}

function proxyTypeLabel(type) {
  switch ((type || 'auto').toLowerCase()) {
    case 'connect': return 'HTTP';
    case 'socks5': return 'SOCKS5';
    default: return 'AUTO';
  }
}

function syncGeoLookupUrl() {
  const provider = value('geoProvider') || 'auto';
  const input = el('geoLookupUrl');
  if (!input) return;
  const defaults = {
    auto: 'AUTO: 2ip.ua/json -> 2ip.ua -> ipwho.is -> ifconfig.co -> ipapi.co -> ip-api.com -> ipinfo.io -> api.ip.sb',
    '2ip': 'https://2ip.ua',
    '2ip_json': 'https://2ip.ua/json',
    ipinfo: 'https://ipinfo.io/json',
    ipapi: 'http://ip-api.com/json/',
    ifconfig: 'https://ifconfig.co/json',
    ipwhois: 'https://ipwho.is/',
    ipapi_co: 'https://ipapi.co/json/',
    ip_sb: 'https://api.ip.sb/geoip',
  };
  const next = defaults[provider] || defaults.auto;
  if (document.activeElement !== input) input.value = next;
  input.disabled = true;
}

function collectCountries(proxies, resultByKey, okOnly) {
  const countries = new Set();
  for (const proxy of proxies) {
    const result = resultByKey.get(proxyKey(proxy));
    if (okOnly && !result?.ok) continue;
    const country = (result?.geo?.country || '').trim();
    if (country) countries.add(country);
  }
  return Array.from(countries).sort();
}

function syncCountryFilter(proxies, resultByKey) {
  const select = el('countryFilter');
  const current = select.value || 'all';
  const countries = collectCountries(proxies, resultByKey, false);
  select.innerHTML = ['<option value="all">Все страны</option>']
    .concat(countries.map((c) => `<option value="${escapeHtml(c)}">${escapeHtml(c)}</option>`))
    .join('');
  select.value = countries.includes(current) || current === 'all' ? current : 'all';
}

function syncMonitorCountrySelect(proxies, resultByKey, currentValue) {
  const select = el('monitorCountrySelect');
  const countries = collectCountries(proxies, resultByKey, true);
  const prev = select.value || currentValue || '';
  select.innerHTML = ['<option value="">Любая страна</option>']
    .concat(countries.map((c) => `<option value="${escapeHtml(c)}">${escapeHtml(c)}</option>`))
    .join('');
  if (prev && countries.includes(prev)) select.value = prev;
  else select.value = '';
}

// Глобальный кеш прокси для поиска
let _parentProxyItems = [];

function buildProxyDisplayLabel(proxy, country) {
  const showLabel = el('showLabel').checked;
  const showIP = el('showIP').checked;
  const hasLabel = !!proxy.label;
  const parts = [];
  if (showIP || !hasLabel) parts.push(`${proxy.host}:${proxy.port}${country ? ' [' + country + ']' : ''}`);
  if (showLabel && hasLabel) parts.push(proxy.label);
  return parts.join(' | ') || `${proxy.host}:${proxy.port}`;
}

function renderProxyDropdown(filter) {
  const list = el('parentProxyList');
  const q = (filter || '').toLowerCase().trim();
  const filtered = q
    ? _parentProxyItems.filter((item) => item.searchText.includes(q))
    : _parentProxyItems;
  if (filtered.length === 0) {
    list.innerHTML = '<div class="proxy-dropdown-empty">Ничего не найдено</div>';
  } else {
    list.innerHTML = filtered
      .map((item) => `<div class="proxy-dropdown-item" data-key="${escapeHtml(item.key)}">${escapeHtml(item.label)}</div>`)
      .join('');
    list.querySelectorAll('.proxy-dropdown-item').forEach((row) => {
      row.addEventListener('mousedown', (e) => {
        e.preventDefault();
        selectParentProxy(row.dataset.key, row.textContent);
      });
    });
  }
  list.classList.remove('hidden');
}

function selectParentProxy(key, label) {
  el('parentProxySelect').value = key;
  el('parentProxySearch').value = label;
  el('parentProxyList').classList.add('hidden');
}

function syncParentProxySelect(proxies, resultByKey) {
  const countryFilter = el('parentCountryFilter').value || 'all';

  const countrySel = el('parentCountryFilter');
  const currentCountry = countrySel.value || 'all';
  const countries = collectCountries(proxies, resultByKey, true);
  countrySel.innerHTML = ['<option value="all">Все страны</option>']
    .concat(countries.map((c) => `<option value="${escapeHtml(c)}">${escapeHtml(c)}</option>`))
    .join('');
  countrySel.value = countries.includes(currentCountry) || currentCountry === 'all' ? currentCountry : 'all';

  const currentKey = el('parentProxySelect').value;
  _parentProxyItems = [];
  for (const proxy of proxies) {
    const result = resultByKey.get(proxyKey(proxy));
    if (!result?.ok) continue;
    const country = result?.geo?.country || '';
    if (countryFilter !== 'all' && country !== countryFilter) continue;
    const key = proxyKey(proxy);
    const label = buildProxyDisplayLabel(proxy, country);
    _parentProxyItems.push({ key, label, searchText: (label + ' ' + (proxy.label || '') + ' ' + proxy.host).toLowerCase() });
  }
  // Восстанавливаем выбранный элемент если он ещё есть
  if (currentKey) {
    const found = _parentProxyItems.find((i) => i.key === currentKey);
    if (found) el('parentProxySearch').value = found.label;
  }
}

function matchesFilters(proxy, result) {
  const status = el('statusFilter').value || 'all';
  const country = el('countryFilter').value || 'all';
  if (status === 'ok' && !result?.ok) return false;
  if (status === 'fail' && (!result || result.ok)) return false;
  if (status === 'unchecked' && result) return false;
  if (country !== 'all' && (result?.geo?.country || '') !== country) return false;
  return true;
}

function escapeHtml(str) {
  return String(str).replace(/[&<>"']/g, (c) => ({'&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;',"'":'&#39;'}[c]));
}

function showToast(message, ok = true) {
  const toast = el('toast');
  toast.textContent = message;
  toast.className = `toast show ${ok ? 'success' : 'error'}`;
  window.clearTimeout(showToast.timer);
  showToast.timer = window.setTimeout(() => { toast.className = 'toast'; }, 3200);
}

async function loadVersionInfo() {
  try {
    const version = await GetVersion();
    const info = [`${version.productName} v${version.version}`, version.author ? `by ${version.author}` : ''].filter(Boolean).join(' - ');
    el('appInfo').textContent = info;
    if (version.website) {
      el('appInfo').style.cursor = 'pointer';
      el('appInfo').onclick = () => window.open(version.website, '_blank');
    }
  } catch (err) {
    el('appInfo').textContent = 'Proxy Checker';
  }
}

refresh();
loadVersionInfo();
if (!filtersBound) {
  filtersBound = true;
  ['statusFilter', 'countryFilter'].forEach((id) => {
    el(id).addEventListener('change', () => refresh());
  });
  el('geoProvider').addEventListener('change', () => {
    syncGeoLookupUrl();
    if (!applyingSettings) settingsDirty = true;
  });
}
document.querySelectorAll('.settings input, .settings select').forEach((input) => {
  input.addEventListener('input', () => { if (!applyingSettings) settingsDirty = true; });
  input.addEventListener('change', () => { if (!applyingSettings) settingsDirty = true; });
});
window.setInterval(refresh, 2000);