// Page logic for the playground shortcode. The SDK runs in worker.js; this file
// only moves bytes to it and renders what comes back. Every text from a file or
// from the engine goes into the page as text, never as HTML.
const root = document.getElementById('pg');
const $ = (id) => document.getElementById(id);

const MAX_BYTES = 20 * 1024 * 1024;
const PREVIEW_CHARS = 200_000;
const KINDS = { ddi: 'DDI XML·JSON', 'ddi-xlsx': '편집용 Excel', 'kosac-xlsx': '재단 양식 Excel' };
// kinds: the input kinds an action takes; null means it needs no input.
const ACTIONS = {
  check: { kinds: ['ddi', 'ddi-xlsx', 'kosac-xlsx'], need: '먼저 입력을 고르세요.' },
  toXlsx: { kinds: ['ddi'], need: '이 작업은 DDI XML·JSON 입력에 씁니다.' },
  toDdi: { kinds: ['ddi-xlsx', 'kosac-xlsx'], need: '이 작업은 편집용 Excel이나 재단 양식 Excel 입력에 씁니다.' },
  template: { kinds: null },
  kosacTemplate: { kinds: null },
};

let engine = null; // runs the 검사·변환 buttons
let replEngine = null; // {engine, loaded}: runs the 코드 실행 tab, so user code cannot change the other engine
let ready = false;
let busy = false;
let running = null; // the op of the task in progress
let generation = 0;
let sequence = 0;
let input = null; // {bytes, name, kind}
const resultURLs = [];
const replURLs = [];

// el('div', {class: 'x'}, 'text', child) builds an element. Strings become text nodes.
function el(tag, attrs = {}, ...children) {
  const node = document.createElement(tag);
  for (const [key, value] of Object.entries(attrs)) {
    if (value === undefined || value === false) continue;
    if (key.startsWith('on')) node.addEventListener(key.slice(2), value);
    else node.setAttribute(key, value === true ? '' : value);
  }
  for (const child of children.flat()) {
    if (child !== undefined && child !== null && child !== false) node.append(child);
  }
  return node;
}

const size = (n) => (n < 1024 * 1024 ? `${(n / 1024).toFixed(1)} KiB` : `${(n / 1024 / 1024).toFixed(1)} MiB`);
const duration = (ms) => (ms < 1000 ? `${Math.round(ms)} ms` : `${(ms / 1000).toFixed(1)} s`);
const stem = () => (input?.name || 'codebook').replace(/\.[^.]+$/, '') || 'codebook';

function status(state, text) {
  root.dataset.state = state;
  $('pg-runtime').textContent = text;
}

function message(text, tone, elapsed) {
  const box = $('pg-msg');
  box.replaceChildren(text);
  if (elapsed !== undefined) box.append(el('span', { class: 'pg-time', 'aria-hidden': 'true' }, duration(elapsed)));
  if (tone) box.dataset.tone = tone;
  else delete box.dataset.tone;
}

// Every download is octet-stream, so a crafted file cannot render as a page on this origin.
function blobURL(data, list) {
  const url = URL.createObjectURL(new Blob([data], { type: 'application/octet-stream' }));
  list.push(url);
  return url;
}

// put appends to the result panel and skips the false that `cond && node` gives.
function put(...nodes) {
  $('pg-result').append(...nodes.flat().filter(Boolean));
}

function clearResult() {
  resultURLs.splice(0).forEach((url) => URL.revokeObjectURL(url));
  $('pg-result').replaceChildren();
}

function update() {
  const idle = ready && !busy;
  root.querySelectorAll('[data-example]').forEach((b) => { b.disabled = !idle; });
  for (const [name, action] of Object.entries(ACTIONS)) {
    const button = root.querySelector(`[data-action="${name}"]`);
    const fits = action.kinds === null || (input && action.kinds.includes(input.kind));
    button.disabled = !idle || !fits;
    button.title = fits ? '' : action.need;
  }
  $('pg-file').disabled = busy;
  root.querySelectorAll('[name="pg-kind"]').forEach((r) => {
    r.disabled = busy || !input;
    r.checked = input?.kind === r.value;
  });
  $('pg-run').disabled = !idle;
  $('pg-action-hint').textContent = !input
    ? '먼저 입력을 고르세요.'
    : !input.kind
      ? '입력 종류를 고르세요. 편집용 Excel과 재단 양식 Excel은 형식이 다릅니다.'
      : `${KINDS[input.kind]} 입력에 맞는 작업만 켜져 있습니다.`;
}

function setInput(bytes, name, kind) {
  input = { bytes, name, kind };
  $('pg-current').hidden = false;
  $('pg-name').textContent = name;
  $('pg-size').textContent = `${size(bytes.length)}${kind ? ` · ${KINDS[kind]}` : ''}`;
  clearResult();
  if (root.dataset.state === 'error') message('입력을 바꾸었습니다. 작업을 하려면, 먼저 엔진 다시 시작을 누르세요.', 'bad');
  else if (!kind) message('Excel 파일을 골랐습니다. 입력 종류에서 편집용 Excel이나 재단 양식 Excel을 고르세요.');
  else message(ready ? '입력을 준비했습니다. 작업을 고르세요.' : '입력을 준비했습니다. 엔진이 준비되면 작업을 고르세요.');
  update();
}

const LOAD_HINT = '로컬 미리보기에서는, 먼저 저장소 루트에서 scripts/build.sh를 실행하세요.';

// spawn starts a worker that runs the SDK. on.ready runs once the wasm has loaded.
// on.stop runs when the worker cannot load or stops later. Its pending calls fail with the
// error that on.stop returns, or with the original error.
function spawn(stamp, on) {
  const worker = new Worker(root.dataset.worker, { type: 'module' });
  const calls = new Map();
  let loaded = false;
  const stop = (error) => {
    const shown = on.stop(error, loaded) || error;
    for (const c of calls.values()) c.reject(shown);
    calls.clear();
  };
  worker.onmessage = ({ data }) => {
    if (stamp !== generation) return;
    if (data.type === 'ready') {
      loaded = true;
      on.ready(data.elapsed);
    } else if (data.type === 'init-error') {
      stop(new Error(`엔진을 불러오지 못했습니다: ${data.error}. ${LOAD_HINT}`));
    } else {
      const c = calls.get(data.id);
      if (!c) return;
      calls.delete(data.id);
      if (data.ok) c.resolve(data);
      else c.reject(Object.assign(new Error(data.error), { logs: data.logs, elapsed: data.elapsed }));
    }
  };
  worker.onerror = (event) => {
    if (stamp !== generation) return;
    event.preventDefault();
    const detail = event.message || '알 수 없는 오류';
    stop(Object.assign(new Error(loaded
      ? `엔진이 멈췄습니다: ${detail}. 엔진 다시 시작을 누르세요.`
      : `엔진을 불러오지 못했습니다: ${detail}. ${LOAD_HINT}`), { detail, loaded }));
  };
  return {
    worker,
    call: (op, payload) => new Promise((resolve, reject) => {
      const id = ++sequence;
      calls.set(id, { resolve, reject });
      worker.postMessage({ id, op, ...payload });
    }),
  };
}

function start(restart) {
  const stamp = ++generation;
  engine?.worker.terminate();
  replEngine?.engine.worker.terminate();
  replEngine = null;
  if (busy && running === 'repl') showReplOutput('엔진을 다시 시작해서 실행을 멈췄습니다.');
  ready = false;
  busy = false;
  running = null;
  status('loading', '엔진을 불러오는 중입니다.');
  update();
  engine = spawn(stamp, {
    ready: (elapsed) => {
      ready = true;
      status('ready', `엔진 준비됨 · ${duration(elapsed)}`);
      update();
      if (restart) message('엔진을 다시 시작했습니다. 입력은 그대로입니다.');
      else if (!input) useExample('ddi-xml', '예시 DDI XML을 입력으로 골랐습니다. 검사를 누르세요.');
      else if (input.kind) message('엔진이 준비되었습니다. 작업을 고르세요.');
    },
    stop: (error, loaded) => {
      ready = false;
      busy = false;
      status('error', loaded ? '엔진이 멈췄습니다.' : '엔진을 불러오지 못했습니다.');
      message(error.message, 'bad');
      update();
    },
  });
}

// The 코드 실행 engine starts on the first run and stays until a restart or a crash.
function repl() {
  if (!replEngine) {
    const stamp = generation;
    let resolve;
    let reject;
    const loaded = new Promise((res, rej) => { resolve = res; reject = rej; });
    const engine = spawn(stamp, {
      ready: () => resolve(engine),
      stop: (error) => {
        if (stamp === generation) replEngine = null;
        const shown = error.loaded
          ? new Error(`코드 실행 엔진이 멈췄습니다: ${error.detail}. 다시 실행하면, 이 엔진을 새로 만듭니다.`)
          : error;
        reject(shown);
        return shown;
      },
    });
    replEngine = { engine, loaded };
  }
  return replEngine.loaded;
}

// task runs op on the engine that target() gives, with the buttons off.
// Focus goes back to the control that started it.
async function task(op, payload, done, target = async () => engine) {
  if (!ready || busy) return;
  const stamp = generation;
  const focused = document.activeElement;
  busy = true;
  running = op;
  status('busy', '실행 중입니다.');
  update();
  try {
    const result = await (await target()).call(op, payload);
    if (stamp === generation) done(result.value, result.elapsed);
  } catch (error) {
    if (stamp !== generation) return;
    if (op === 'repl') showReplOutput([...(error.logs || []), error.message].join('\n\n'));
    else message(error.message, 'bad', error.elapsed);
  } finally {
    if (stamp === generation && ready) {
      busy = false;
      running = null;
      status('ready', '엔진 준비됨');
      update();
      if (focused?.isConnected && !focused.disabled && document.activeElement === document.body) focused.focus();
    }
  }
}

function useExample(name, note) {
  task('example', { name }, ([bytes, file, kind]) => {
    setInput(bytes, file, kind);
    if (note) message(note);
  });
}

// ---- result rendering ----

function section(title, ...children) {
  return el('div', { class: 'pg-section' }, el('span', { class: 'pg-label' }, title), ...children);
}

// files: [{name, data, mime, kind}]. A file with a kind can become the next input.
function fileCards(files) {
  return el('div', { class: 'pg-files' }, files.map((f) => el('div', { class: 'pg-file' },
    el('div', {}, el('b', {}, f.name), el('small', {}, size(typeof f.data === 'string' ? new Blob([f.data]).size : f.data.length))),
    el('a', { class: 'pg-btn', href: blobURL(f.data, resultURLs), download: f.name }, '받기'),
    f.kind && el('button', {
      type: 'button',
      class: 'pg-btn',
      onclick: () => {
        const bytes = typeof f.data === 'string' ? new TextEncoder().encode(f.data) : f.data;
        setInput(bytes, f.name, f.kind);
        message(`입력을 바꾸었습니다: ${f.name}. 다음 작업을 고르세요.`);
        $('pg-msg').focus();
      },
    }, '입력으로 쓰기'),
  )));
}

function viewer(entries) {
  const pre = el('pre', { tabindex: '0' });
  const tabs = entries.map((entry, i) => el('button', {
    type: 'button',
    class: 'pg-tab',
    'aria-pressed': String(i === 0),
    onclick: (event) => {
      tabs.forEach((t) => t.setAttribute('aria-pressed', String(t === event.currentTarget)));
      show(entry);
    },
  }, entry.label));
  const note = el('p', { class: 'pg-hint' });
  function show({ text }) {
    pre.textContent = text.length > PREVIEW_CHARS ? text.slice(0, PREVIEW_CHARS) : text;
    note.textContent = text.length > PREVIEW_CHARS
      ? `미리보기는 앞부분 ${PREVIEW_CHARS.toLocaleString()}자만 보여 줍니다. 전체 내용을 보려면, 파일을 받으세요.`
      : '';
  }
  show(entries[0]);
  return el('div', { class: 'pg-viewer' }, el('div', { class: 'pg-viewer-tabs', role: 'group', 'aria-label': '미리보기 형식' }, tabs), pre, note);
}

function warningList(warnings) {
  return warnings.length > 0 && section(`경고 ${warnings.length}개`, el('ul', { class: 'pg-list' }, warnings.map((w) => el('li', {}, w))));
}

function issueCard(issue) {
  const location = issue.Sheet ? issue.Sheet + (issue.Cell ? `!${issue.Cell}` : '') : issue.Path || '문서 전체';
  const context = [issue.FileNo && `파일 ${issue.FileNo}`, issue.VarName && `변수 ${issue.VarName}`, issue.Code && `코드 ${issue.Code}`]
    .filter(Boolean).join(' · ');
  return el('article', { class: 'pg-issue' },
    el('div', { class: 'pg-issue-head' },
      el('span', { class: 'pg-stat', 'data-tone': issue.Level === 'R' ? 'bad' : 'warn' }, issue.Level === 'R' ? '반려' : '경고'),
      el('b', {}, issue.Rule),
      el('code', {}, location)),
    el('p', {}, issue.Message),
    context && el('p', { class: 'pg-hint' }, context),
    (issue.Actual || issue.Expected) && el('dl', {},
      issue.Actual && [el('dt', {}, '읽은 값'), el('dd', {}, issue.Actual)],
      issue.Expected && [el('dt', {}, '기대값'), el('dd', {}, issue.Expected)]));
}

function issueList(issues) {
  const list = el('div', { class: 'pg-issues' });
  const level = el('select', { 'aria-label': '발견 항목 수준' },
    el('option', { value: 'all' }, '전체'), el('option', { value: 'R' }, '반려'), el('option', { value: 'W' }, '경고'));
  const search = el('input', { type: 'search', placeholder: '규칙, 시트, 변수, 메시지 검색', 'aria-label': '발견 항목 검색' });
  const count = el('span', { class: 'pg-label' });
  function draw() {
    const q = search.value.trim().toLowerCase();
    const shown = issues.filter((i) => (level.value === 'all' || i.Level === level.value)
      && Object.values(i).join(' ').toLowerCase().includes(q));
    count.textContent = `발견 항목 ${shown.length} / ${issues.length}`;
    list.replaceChildren(...(shown.length ? shown.map(issueCard) : [el('p', { class: 'pg-empty' }, '조건에 맞는 항목이 없습니다.')]));
  }
  level.addEventListener('change', draw);
  search.addEventListener('input', draw);
  draw();
  return el('div', { class: 'pg-section' }, count, el('div', { class: 'pg-filter' }, level, search), list);
}

const STEP = { pass: ['통과', 'ok'], fail: ['반려', 'bad'], warning: ['경고', 'warn'], not_run: ['실행 안 함', undefined] };

function renderCheck(report, elapsed) {
  const notRun = report.steps.filter((s) => s.status === 'not_run').length;
  message(`반려 ${report.rejections}개, 경고 ${report.warnings}개를 찾았습니다.`,
    report.rejections ? 'bad' : report.warnings ? 'warn' : 'ok', elapsed);
  put(
    el('div', { class: 'pg-summary' },
      el('span', { class: 'pg-stat', 'data-tone': report.rejections ? 'bad' : 'ok' }, `반려 ${report.rejections}`),
      el('span', { class: 'pg-stat', 'data-tone': report.warnings ? 'warn' : 'ok' }, `경고 ${report.warnings}`),
      el('span', { class: 'pg-stat' }, `실행 안 한 단계 ${notRun}`),
      el('span', { class: 'pg-stat' }, KINDS[report.format] + (report.templateVersion ? ` v${report.templateVersion}` : '')),
      el('a', {
        class: 'pg-btn pg-quiet',
        href: blobURL(JSON.stringify(report, null, 2), resultURLs),
        download: `${stem()}-check.json`,
      }, '결과 JSON 받기')),
    section('검사 단계', el('ul', { class: 'pg-steps' }, report.steps.map((s) => el('li', {},
      el('div', {}, el('b', {}, `${s.id} · ${s.name}`), el('small', {}, s.detail)),
      el('span', { class: 'pg-stat', 'data-tone': STEP[s.status][1] }, STEP[s.status][0]))))),
    report.issues.length > 0 && issueList(report.issues),
    notRun > 0 && el('p', { class: 'pg-hint' }, '"실행 안 함" 단계는 다른 도구나 시스템에서 검사합니다. 실행하는 곳은 각 단계의 설명에 있습니다.'),
  );
}

function renderToXlsx({ xlsx, warnings }, elapsed) {
  message(`편집용 Excel을 만들었습니다.${warnings.length ? ` 경고가 ${warnings.length}개 있습니다.` : ''}`, warnings.length ? 'warn' : 'ok', elapsed);
  put(
    section('만든 파일', fileCards([{ name: `${stem()}.xlsx`, data: xlsx, kind: 'ddi-xlsx' }])),
    warningList(warnings),
    el('p', { class: 'pg-hint' }, '고친 Excel 파일을 다시 입력으로 넣으면, DDI XML·JSON으로 변환합니다.'));
}

function renderDdi({ xml, json }, extra) {
  const base = stem();
  put(
    section('만든 파일', fileCards([
      { name: `${base}.xml`, data: xml, kind: 'ddi' },
      { name: `${base}.json`, data: json, kind: 'ddi' },
    ])),
    ...extra,
    section('미리보기', viewer([{ label: 'XML', text: xml }, { label: 'JSON', text: json }])));
}

function renderFromXlsx(out, elapsed) {
  message(`DDI XML·JSON을 만들었습니다.${out.warnings.length ? ` 경고가 ${out.warnings.length}개 있습니다.` : ''}`, out.warnings.length ? 'warn' : 'ok', elapsed);
  renderDdi(out, [warningList(out.warnings)]);
}

function renderKosacToDdi(out, elapsed) {
  const rejected = out.issues.filter((i) => i.Level === 'R').length;
  if (out.xml === null) {
    message('양식 구조 검사(S05)가 이 파일을 반려했습니다. 아래 항목을 고치세요.', 'bad', elapsed);
    put(issueList(out.issues));
    return;
  }
  message(rejected
    ? `DDI XML·JSON을 만들었습니다. 반려 항목이 ${rejected}개 있습니다.`
    : 'DDI XML·JSON을 만들었습니다.', rejected ? 'bad' : out.issues.length ? 'warn' : 'ok', elapsed);
  renderDdi(out, [
    out.issues.length > 0 && issueList(out.issues),
    el('p', { class: 'pg-hint' }, '내용 규칙(C)은 검사 작업에서 실행합니다. XSD 검사는 CLI나 xmllint로 실행합니다.'),
  ]);
}

function renderTemplate(name, kind) {
  return (xlsx, elapsed) => {
    message('빈 양식을 만들었습니다.', 'ok', elapsed);
    put(section('만든 파일', fileCards([{ name, data: xlsx, kind }])));
  };
}

// ---- equivalent code ----

function snippets(action, kind) {
  const js = (name, line) => `import { ${name} } from 'ddi-xlsx';\n\n${line}`;
  switch (action) {
    case 'check':
      return [
        kind === 'kosac-xlsx' && { label: 'CLI', code: 'ddi-xlsx kosac-check survey.xlsx' },
        { label: 'JavaScript', code: js('checkCodebook', `const report = await checkCodebook(input, { format: '${kind}' });`),
          repl: `return await api.checkCodebook(input, { format: '${kind}' });` },
        kind === 'kosac-xlsx' && { label: 'Go', code: 'cb, issues, err := template.Read(x, template.ReadOptions{})\nif err == nil && cb != nil {\n\tissues = append(issues, check.Check(cb, kosac.RefData{})...)\n}' },
      ];
    case 'toXlsx':
      return [
        { label: 'CLI', code: 'ddi-xlsx to-xlsx codebook.xml -o codebook.xlsx' },
        { label: 'JavaScript', code: js('ddiToXlsx', 'const { xlsx, warnings } = await ddiToXlsx(input);'),
          repl: "const { xlsx, warnings } = await api.ddiToXlsx(input);\ndownload('codebook.xlsx', xlsx);\nreturn warnings;" },
        { label: 'Go', code: 'x, warnings, err := workbook.ToXLSX(src)' },
      ];
    case 'toDdi':
      if (kind === 'kosac-xlsx') {
        return [
          { label: 'CLI', code: 'ddi-xlsx kosac-xml survey.xlsx -o codebook.xml --json codebook.json' },
          { label: 'JavaScript', code: js('kosacToDdi', 'const { xml, json, issues } = await kosacToDdi(input);'),
            repl: "const { xml, json, issues } = await api.kosacToDdi(input);\nif (xml) download('codebook.xml', xml, 'application/xml');\nreturn issues;" },
        ];
      }
      return [
        { label: 'CLI', code: 'ddi-xlsx from-xlsx codebook.xlsx --xml codebook.xml --json codebook.json' },
        { label: 'JavaScript', code: js('xlsxToDdi', 'const { xml, json, warnings } = await xlsxToDdi(input);'),
          repl: "const { xml, json, warnings } = await api.xlsxToDdi(input);\ndownload('codebook.xml', xml, 'application/xml');\nreturn warnings;" },
        { label: 'Go', code: 'xmlOut, jsonOut, warnings, err := workbook.FromXLSX(x)' },
      ];
    case 'template':
      return [
        { label: 'CLI', code: 'ddi-xlsx template -o template.xlsx' },
        { label: 'JavaScript', code: js('templateXlsx', 'const xlsx = await templateXlsx();') },
        { label: 'Go', code: 'blank, err := workbook.Template()' },
      ];
    case 'kosacTemplate':
      return [
        { label: 'CLI', code: 'ddi-xlsx kosac-template -o kosac-template.xlsx' },
        { label: 'JavaScript', code: js('kosacTemplateXlsx', 'const xlsx = await kosacTemplateXlsx();') },
        { label: 'Go', code: 'blank := template.Blank()' },
      ];
  }
  return [];
}

function codeSection(action, kind) {
  const items = snippets(action, kind).filter(Boolean);
  return el('details', { class: 'pg-section pg-snippets' },
    el('summary', { class: 'pg-label' }, '같은 작업을 코드로'),
    items.map((s) => el('div', { class: 'pg-snippet' },
      el('div', { class: 'pg-snippet-head' },
        el('span', { class: 'pg-label' }, s.label),
        s.repl && el('button', { type: 'button', class: 'pg-btn pg-quiet', onclick: () => openRepl(s.repl) }, '코드 실행 탭에서 열기')),
      el('pre', {}, s.code))));
}

// ---- actions ----

const RUN = {
  check: () => ['checkCodebook', [input.bytes, { format: input.kind }], renderCheck],
  toXlsx: () => ['ddiToXlsx', [input.bytes], renderToXlsx],
  toDdi: () => (input.kind === 'kosac-xlsx'
    ? ['kosacToDdi', [input.bytes], renderKosacToDdi]
    : ['xlsxToDdi', [input.bytes], renderFromXlsx]),
  template: () => ['templateXlsx', [], renderTemplate('template.xlsx', 'ddi-xlsx')],
  kosacTemplate: () => ['kosacTemplateXlsx', [], renderTemplate('kosac-template-v4.xlsx', 'kosac-xlsx')],
};

function runAction(action) {
  const [op, args, render] = RUN[action]();
  const kind = input?.kind;
  selectTab('result');
  clearResult();
  message('실행 중입니다.');
  task(op, { args }, (value, elapsed) => {
    render(value, elapsed);
    put(codeSection(action, kind));
  });
}

// ---- code (REPL) tab ----

function showReplOutput(text, files = []) {
  replURLs.splice(0).forEach((url) => URL.revokeObjectURL(url));
  $('pg-repl-out').textContent = text;
  $('pg-repl-files').replaceChildren(...files.map((f) => el('div', { class: 'pg-file' },
    el('div', {}, el('b', {}, f.name), el('small', {}, size(new Blob([f.data]).size))),
    el('a', { class: 'pg-btn', href: blobURL(f.data, replURLs), download: f.name }, '받기'))));
}

function runRepl() {
  if (!ready || busy) return;
  showReplOutput('실행 중입니다.');
  task('repl', { code: $('pg-code').value, input: input?.bytes ?? null, format: input?.kind ?? null }, (out, elapsed) => {
    showReplOutput([...out.logs, out.text, `(${duration(elapsed)})`].join('\n\n'), out.files);
  }, repl);
}

function openRepl(code) {
  $('pg-code').value = code;
  selectTab('repl');
  $('pg-code').focus();
}

function selectTab(name) {
  for (const t of ['result', 'repl']) {
    const on = t === name;
    $(`pg-tab-${t}`).setAttribute('aria-selected', String(on));
    $(`pg-tab-${t}`).tabIndex = on ? 0 : -1;
    $(`pg-panel-${t}`).hidden = !on;
  }
}

// ---- input ----

async function pickFile(file) {
  if (!file) return;
  if (!/\.(xml|json|xlsx)$/i.test(file.name)) {
    message('XML, JSON, Excel(.xlsx) 파일을 고르세요.', 'bad');
    return;
  }
  if (file.size > MAX_BYTES) {
    message(`파일이 너무 큽니다(${size(file.size)}). 최대 크기는 20 MiB입니다.`, 'bad');
    return;
  }
  // Keep the buttons off while the file is read, so no task runs on the old input.
  const stamp = generation;
  const wasBusy = busy;
  busy = true;
  update();
  try {
    const bytes = new Uint8Array(await file.arrayBuffer());
    if (stamp === generation) setInput(bytes, file.name, /\.(xml|json)$/i.test(file.name) ? 'ddi' : null);
  } finally {
    if (stamp === generation) {
      busy = wasBusy;
      update();
    }
  }
}

$('pg-file').addEventListener('change', (event) => {
  pickFile(event.target.files[0]).catch((error) => message(error.message, 'bad'));
  event.target.value = '';
});
const drop = $('pg-drop');
drop.addEventListener('dragover', (event) => { event.preventDefault(); drop.dataset.over = 'true'; });
drop.addEventListener('dragleave', () => { delete drop.dataset.over; });
drop.addEventListener('drop', (event) => {
  event.preventDefault();
  delete drop.dataset.over;
  if (busy) message('실행이 끝난 다음에 파일을 놓으세요.', 'warn');
  else pickFile(event.dataTransfer.files[0]).catch((error) => message(error.message, 'bad'));
});
root.querySelectorAll('[name="pg-kind"]').forEach((radio) => radio.addEventListener('change', () => {
  if (!input) return;
  setInput(input.bytes, input.name, radio.value);
}));
root.querySelectorAll('[data-example]').forEach((b) => b.addEventListener('click', () => useExample(b.dataset.example)));
root.querySelectorAll('[data-action]').forEach((b) => b.addEventListener('click', () => runAction(b.dataset.action)));
$('pg-reset').addEventListener('click', () => start(true));
$('pg-run').addEventListener('click', runRepl);
$('pg-tab-result').addEventListener('click', () => selectTab('result'));
$('pg-tab-repl').addEventListener('click', () => selectTab('repl'));
root.querySelector('.pg-tabs').addEventListener('keydown', (event) => {
  if (event.key !== 'ArrowLeft' && event.key !== 'ArrowRight') return;
  const next = $('pg-tab-result').getAttribute('aria-selected') === 'true' ? 'repl' : 'result';
  selectTab(next);
  $(`pg-tab-${next}`).focus();
});
$('pg-code').addEventListener('keydown', (event) => {
  if ((event.metaKey || event.ctrlKey) && event.key === 'Enter') {
    event.preventDefault();
    runRepl();
  }
});

update();
start(false);
