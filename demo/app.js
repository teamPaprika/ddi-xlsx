const $ = (selector) => document.querySelector(selector);
const formats = { ddi: 'DDI XML·JSON', 'ddi-xlsx': '일반 DDI 편집용 Excel', 'kosac-xlsx': '재단 코드북 Excel' };
const snippets = {
  check: 'return await api.checkCodebook(input, { format });',
  toXlsx: "const { xlsx, warnings } = await api.ddiToXlsx(input);\ndownload('codebook.xlsx', xlsx);\nreturn { bytes: xlsx.length, warnings };",
  fromXlsx: "const result = await api.xlsxToDdi(input);\ndownload('codebook.xml', result.xml, 'application/xml');\nreturn result.warnings;",
};
let worker, workerURL, ready = false, busy = false, generation = 0, requestID = 0, input, lastReport;
const pending = new Map(), downloadURLs = [];
function element(tag, text, className) {
  const node = document.createElement(tag);
  if (text !== undefined) node.textContent = String(text);
  if (className) node.className = className;
  return node;
}
function message(text, error = false) {
  $('#message').textContent = text;
  $('#message').classList.toggle('error', error);
}
function controls() {
  $('#run').disabled = !ready || busy || !input || !formats[$('#format').value];
  $('#format').disabled = busy;
  $('#input-file').disabled = busy;
  document.querySelectorAll('[data-example]').forEach((button) => { button.disabled = !ready || busy; });
  $('#runtime').textContent = !ready ? 'WASM 준비 중' : busy ? '실행 중' : 'WASM 준비됨';
  $('#runtime').dataset.ready = String(ready);
}
function clearDownloads() {
  downloadURLs.splice(0).forEach((url) => URL.revokeObjectURL(url));
  $('#downloads').replaceChildren();
}
function clearResult() {
  lastReport = undefined;
  $('#report').hidden = true;
  $('#timing').textContent = '';
  $('#output').textContent = '현재 입력으로 아직 실행하지 않았습니다.';
  clearDownloads();
}
function setInput(bytes, name, format) {
  input = bytes;
  if (format !== undefined) $('#format').value = format;
  $('#input-info').textContent = `${name} (${(bytes.length / 1024).toFixed(1)} KiB)`;
  $('#input-info').title = name;
  clearResult();
  formatHint();
  message(formats[$('#format').value]
    ? '입력을 준비했습니다. 실행할 코드를 확인하고 코드 실행을 누르세요.'
    : 'Excel 양식을 선택한 뒤 코드를 실행하세요.');
}
function formatHint() {
  $('#format-hint').textContent = !formats[$('#format').value]
    ? '재단 코드북과 일반 DDI 편집용 Excel 중에서 선택하세요.'
    : $('#format').value === 'kosac-xlsx'
    ? '재단 v4.0·v3.2의 S05와 코드북 규칙을 검사합니다. 일반 DDI 편집용 Excel과 다른 양식입니다.'
    : $('#format').value === 'ddi-xlsx'
      ? 'Variables·Categories·Groups·Document 시트의 일반 편집용 양식입니다. DDI로 되돌린 뒤 참조를 검사합니다.'
      : '문서 읽기와 지원하는 ID·변수·파일 참조를 검사합니다. XSD는 실행하지 않습니다.';
}
function send(type, data = {}) {
  return new Promise((resolve, reject) => {
    const id = ++requestID;
    pending.set(id, { resolve, reject });
    worker.postMessage({ id, type, ...data });
  });
}
async function request(type, data, done) {
  if (!ready || busy) return;
  const stamp = generation;
  busy = true; controls();
  message(type === 'run' ? '작업 스레드에서 코드를 실행합니다. 필요하면 중지·런타임 초기화를 누르세요.' : 'WASM으로 예시 입력을 준비합니다.');
  try {
    const result = await send(type, data);
    if (stamp === generation) done(result);
  } catch (error) {
    if (stamp === generation) {
      message(error.message, true);
      $('#output').textContent = [...(error.logs || []), error.message].join('\n\n');
      $('#console-panel').open = true;
    }
  } finally {
    if (stamp === generation) { busy = false; controls(); }
  }
}
function resetRuntime() {
  const stamp = ++generation;
  worker?.terminate();
  if (workerURL) URL.revokeObjectURL(workerURL);
  for (const promise of pending.values()) promise.reject(new Error('런타임을 초기화했습니다.'));
  pending.clear();
  ready = false; busy = false; controls(); clearResult();
  message('WASM을 브라우저 안에서 불러옵니다. 외부 서버에 파일을 보내지 않습니다.');
  const url = URL.createObjectURL(new Blob([workerSource], { type: 'text/javascript' }));
  workerURL = url;
  worker = new Worker(url, { type: 'module' });
  worker.onmessage = ({ data }) => {
    if (stamp !== generation) return;
    if (data.type === 'ready') {
      URL.revokeObjectURL(url); workerURL = undefined; ready = true; controls();
      if (!input) request('example', { example: 'valid' }, (example) => setInput(example.bytes, example.name, example.format));
      else message('새 WASM 런타임을 준비했습니다. 선택한 파일과 코드는 유지합니다.');
      return;
    }
    if (data.type === 'init-error') {
      URL.revokeObjectURL(url); message('WASM 준비 실패: ' + data.error, true);
      $('#runtime').textContent = 'WASM 준비 실패'; return;
    }
    const promise = pending.get(data.id);
    if (!promise) return;
    pending.delete(data.id);
    if (data.type === 'error') { const error = new Error(data.error); error.logs = data.logs || []; promise.reject(error); }
    else promise.resolve(data);
  };
  worker.onerror = (error) => {
    if (stamp !== generation) return;
    URL.revokeObjectURL(url); ready = false; busy = false; controls();
    const failure = new Error(error.message || '작업 스레드가 멈췄습니다. 런타임을 초기화하세요.');
    for (const promise of pending.values()) promise.reject(failure);
    pending.clear(); message(failure.message, true); $('#runtime').textContent = '런타임 오류';
  };
}
function findings() {
  const list = $('#findings'); list.replaceChildren();
  const query = $('#finding-search').value.trim().toLowerCase(), level = $('#severity').value;
  const issues = lastReport.issues.filter((issue) =>
    (level === 'all' || issue.Level === level) && Object.values(issue).join(' ').toLowerCase().includes(query));
  $('#finding-count').textContent = `발견 항목 ${issues.length} / ${lastReport.issues.length}`;
  if (!issues.length) {
    list.append(element('p', lastReport.issues.length ? '검색 조건에 맞는 항목이 없습니다.' : '실행한 검사 범위에서 발견 항목이 없습니다. 미실행 검사도 확인하세요.', 'empty-findings'));
    return;
  }
  for (const issue of issues) {
    const item = element('article', undefined, 'finding');
    const head = element('div', undefined, 'finding-head');
    head.append(element('span', issue.Level === 'R' ? '반려' : '경고', issue.Level === 'R' ? 'badge rejection' : 'badge warning'), element('b', issue.Rule));
    const location = issue.Sheet ? issue.Sheet + (issue.Cell ? '!' + issue.Cell : '') : issue.Path || '문서 전체';
    head.append(element('code', location)); item.append(head, element('p', issue.Message));
    const context = [issue.FileNo && `파일 ${issue.FileNo}`, issue.VarName && `변수 ${issue.VarName}`, issue.Code && `코드 ${issue.Code}`].filter(Boolean).join(' / ');
    if (context) item.append(element('p', context, 'hint'));
    const values = element('dl');
    values.append(element('dt', '읽은 값'), element('dd', issue.Actual || '(빈 값 또는 미제공)'), element('dt', '기대값'), element('dd', issue.Expected || '(엔진에서 제공하지 않음)'));
    item.append(values); list.append(item);
  }
}
function renderReport(report) {
  lastReport = report;
  $('#report').hidden = false;
  $('#report-summary').textContent = `반려 ${report.rejections}개 · 경고 ${report.warnings}개`;
  $('#report-format').textContent = formats[report.format] + (report.templateVersion ? ` v${report.templateVersion}` : '');
  const steps = $('#steps'); steps.replaceChildren();
  const names = { pass: '통과', fail: '오류', warning: '경고', not_run: '미실행' };
  for (const step of report.steps) {
    const row = element('div', undefined, 'step');
    const explanation = element('div'); explanation.append(element('b', step.name), element('small', step.detail));
    row.append(explanation, element('span', names[step.status], 'step-status ' + step.status)); steps.append(row);
  }
  $('#severity').value = 'all'; $('#finding-search').value = ''; findings();
  const skipped = report.steps.filter((step) => step.status === 'not_run').length;
  message(`실행한 검사에서 반려 ${report.rejections}개, 경고 ${report.warnings}개를 찾았습니다. ${skipped}개 검사는 미실행입니다.`, report.rejections > 0);
}
function save(name, value, mime) {
  const url = URL.createObjectURL(new Blob([value], { type: mime }));
  const link = element('a'); link.href = url; link.download = name; link.click();
  setTimeout(() => URL.revokeObjectURL(url), 1000);
}
function renderResult(result) {
  clearDownloads();
  $('#output').textContent = [...result.logs, result.text].join('\n\n');
  $('#timing').textContent = `${Math.round(result.elapsed)} ms`;
  if (result.result && Array.isArray(result.result.issues) && Array.isArray(result.result.steps) && typeof result.result.rejections === 'number') {
    renderReport(result.result);
  } else {
    $('#report').hidden = true; lastReport = undefined; $('#console-panel').open = true;
    message('코드 실행이 끝났습니다. 반환값과 출력, 생성한 파일을 확인하세요.');
  }
  for (const file of result.files) {
    const url = URL.createObjectURL(new Blob([file.value], { type: file.mime })); downloadURLs.push(url);
    const link = element('a', file.name + ' 받기'); link.href = url; link.download = file.name; $('#downloads').append(link);
  }
}
function runCode() {
  if (!input || !formats[$('#format').value]) return;
  clearResult();
  request('run', { code: $('#code').value, input, format: $('#format').value }, renderResult);
}
$('#run').onclick = runCode;
$('#reset').onclick = resetRuntime;
$('#restore-code').onclick = () => { $('#code').value = snippets.check; $('#code').focus(); };
$('#save-report').onclick = () => { if (lastReport) save('ddi-xlsx-check.json', JSON.stringify(lastReport, null, 2), 'application/json'); };
$('#severity').onchange = findings; $('#finding-search').oninput = findings;
$('#format').onchange = () => { formatHint(); clearResult(); controls(); message('입력 형식이 바뀌었습니다. 코드를 다시 실행하세요.'); };
$('#input-file').onchange = async () => {
  const file = $('#input-file').files[0]; if (!file) return;
  if (file.size > 20 * 1024 * 1024) { message('이 데모는 20 MiB 이하 파일을 받습니다.', true); return; }
  const stamp = generation; busy = true; controls();
  try {
    const bytes = new Uint8Array(await file.arrayBuffer());
    if (stamp === generation) setInput(bytes, file.name, /\.(xml|json)$/i.test(file.name) ? 'ddi' : '');
  } catch (error) { if (stamp === generation) message(error.message, true); }
  finally { if (stamp === generation) { busy = false; controls(); } }
};
document.addEventListener('click', (event) => {
  const example = event.target.closest('[data-example]');
  if (example) request('example', { example: example.dataset.example }, (data) => setInput(data.bytes, data.name, data.format));
  const snippet = event.target.closest('[data-snippet]');
  if (snippet) { $('#code').value = snippets[snippet.dataset.snippet]; $('#code').focus(); $('#workspace').scrollIntoView({ block: 'start' }); }
});
$('#code').addEventListener('keydown', (event) => {
  if ((event.metaKey || event.ctrlKey) && event.key === 'Enter') { event.preventDefault(); runCode(); }
  if (event.key === 'Tab') { event.preventDefault(); const editor = event.target; editor.setRangeText('  ', editor.selectionStart, editor.selectionEnd, 'end'); }
});
resetRuntime();
