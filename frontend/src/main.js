import './style.css';
import {
  CheckAll,
  ExportGoodProxies,
  GetState,
  GetVersion,
  LoadFilePath,
  LoadURL,
  SaveSettings,
  SelectProxyFile,
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
      <p>Прокси проверяются через URL проверки, 3proxy получает только живой parent.</p>
    </div>
    <div class="actions">
      <button id="checkAll">Проверить все</button>
      <button id="startMonitor" class="secondary">Старт монитор</button>
      <button id="stopMonitor" class="danger">Стоп монитор</button>
    </div>
  </header>

  <main class="layout">
    <section class="settings">
      <h2>Загрузка</h2>
      <label>API URL</label>
      <input id="apiUrl" type="text" placeholder="https://proxy-example.comt/api/getproxy/?format=json..." />
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

      <h2>Проверка</h2>
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
      <label>Интервал монитора, сек</label>
      <input id="monitorEverySec" type="number" min="5" />
      <p class="hint">Это интервал смены IP: монитор обновляет parent.cfg и дергает reload.txt каждые N секунд.</p>
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
      <button id="saveSettings">Сохранить настройки</button>

      <h2>3proxy</h2>
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
            <p id="active" class="muted">Активный parent: нет</p>
          </div>
          <button id="exportGood" class="secondary">Сохранить good-proxies.txt</button>
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
            <tr><th>Прокси</th><th>Тип</th><th>Логин</th><th>Статус</th><th>Страна</th><th>Регион</th><th>Часовой пояс</th><th>HTTP / ошибка</th><th>Время</th></tr>
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
let isBusy = false;
let settingsDirty = false;
let applyingSettings = false;
let filtersBound = false;

const el = (id) => document.getElementById(id);

async function refresh() {
  state = await GetState();
  if (!settingsDirty) fillSettings(state.config);

  const resultByKey = new Map();
  let good = 0;
  for (const result of state.results || []) {
    const key = proxyKey(result.proxy);
    resultByKey.set(key, result);
    if (result.ok) good += 1;
  }

  el('total').textContent = state.proxies.length;
  el('good').textContent = good;
  el('monitor').textContent = state.monitorRunning ? 'on' : 'off';
  el('threeproxy').textContent = state.threeProxyRun ? 'on' : 'off';
  el('active').textContent = state.activeProxy
    ? `Активный parent: ${state.activeProxy.host}:${state.activeProxy.port}`
    : 'Активный parent: нет';

  const cfg = state.config.threeProxy;
  el('paths').textContent = [
    `3proxy.cfg: ${cfg.workDir}\\3proxy.cfg`,
    `parent.cfg: ${cfg.workDir}\\parent.cfg`,
    `reload.txt: ${cfg.workDir}\\reload.txt`,
    `embedded: ${cfg.workDir}\\embedded-3proxy`,
  ].join('\n');

  el('logs').textContent = (state.logs || []).slice().reverse().join('\n');
  syncCountryFilter(state.proxies, resultByKey);
  const filteredRows = state.proxies
    .map((proxy) => ({ proxy, result: resultByKey.get(proxyKey(proxy)) }))
    .filter(({ proxy, result }) => matchesFilters(proxy, result))
    .map(({ proxy, result }) => proxyRow(proxy, result));
  el('proxyRows').innerHTML = filteredRows.join('');
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
      <td>${status}</td>
      <td>${escapeHtml(country)}</td>
      <td>${escapeHtml(region)}</td>
      <td>${escapeHtml(timezone)}</td>
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
  if (isBusy) return;
  isBusy = true;
  document.body.classList.add('busy');
  try {
    const response = await action();
    if (response?.message) showToast(response.message, response.ok);
  } catch (error) {
    showToast(error.message || String(error), false);
  } finally {
    isBusy = false;
    document.body.classList.remove('busy');
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
el('loadFile').addEventListener('click', () => run(async () => {
  const file = await SelectProxyFile();
  if (!file) return { ok: true, message: 'file selection cancelled' };
  return LoadFilePath(file);
}));

function setValue(id, next) {
  const input = el(id);
  if (document.activeElement !== input) input.value = next ?? '';
}

function setAutoImportInterval(seconds, unit) {
  const selectedUnit = unit || bestIntervalUnit(seconds || 3600);
  setValue('autoImportUnit', selectedUnit);
  setValue('autoImportValue', Math.max(1, Math.round((seconds || 3600) / unitSeconds(selectedUnit))));
}

function value(id) {
  return el(id).value.trim();
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

function syncCountryFilter(proxies, resultByKey) {
  const select = el('countryFilter');
  const current = select.value || 'all';
  const countries = new Set();
  for (const proxy of proxies) {
    const result = resultByKey.get(proxyKey(proxy));
    const country = (result?.geo?.country || '').trim();
    if (country) countries.add(country);
  }
  select.innerHTML = ['<option value="all">Все страны</option>']
    .concat(Array.from(countries).sort().map((country) => `<option value="${escapeHtml(country)}">${escapeHtml(country)}</option>`))
    .join('');
  select.value = countries.has(current) || current === 'all' ? current : 'all';
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

function escapeHtml(value) {
  return value.replace(/[&<>"']/g, (char) => ({
    '&': '&amp;',
    '<': '&lt;',
    '>': '&gt;',
    '"': '&quot;',
    "'": '&#39;',
  }[char]));
}

function showToast(message, ok = true) {
  const toast = el('toast');
  toast.textContent = message;
  toast.className = `toast show ${ok ? 'success' : 'error'}`;
  window.clearTimeout(showToast.timer);
  showToast.timer = window.setTimeout(() => {
    toast.className = 'toast';
  }, 3200);
}

async function loadVersionInfo() {
  try {
    const version = await GetVersion();
    const info = [
      `${version.productName} v${version.version}`,
      version.author ? `by ${version.author}` : '',
    ].filter(Boolean).join(' • ');
    el('appInfo').textContent = info;
    el('appInfo').title = [
      `Product: ${version.productName}`,
      `Version: ${version.version}`,
      `Description: ${version.description || 'N/A'}`,
      `Author: ${version.author || 'N/A'}`,
      `Email: ${version.email || 'N/A'}`,
      `Website: ${version.website || 'N/A'}`,
      `Copyright: ${version.copyright || 'N/A'}`,
    ].join('\n');
    
    // Make website clickable
    if (version.website) {
      el('appInfo').style.cursor = 'pointer';
      el('appInfo').onclick = () => window.open(version.website, '_blank');
    }
  } catch (err) {
    el('appInfo').textContent = 'Proxy Checker';
    console.error('Failed to load version:', err);
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
  input.addEventListener('input', () => {
    if (!applyingSettings) settingsDirty = true;
  });
  input.addEventListener('change', () => {
    if (!applyingSettings) settingsDirty = true;
  });
});
window.setInterval(refresh, 2000);
