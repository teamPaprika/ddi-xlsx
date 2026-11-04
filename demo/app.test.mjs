import { test } from 'node:test';
import assert from 'node:assert/strict';
import { readFile } from 'node:fs/promises';
import vm from 'node:vm';

const html = await readFile(new URL('./dist/index.html', import.meta.url), 'utf8');
const start = html.indexOf('<script>const workerSource=');
const app = html.slice(html.indexOf(';\n', start) + 2, html.lastIndexOf('</script>'));

function mountedApp() {
  const node = () => ({
    children: [], value: '', textContent: '', hidden: false, disabled: false, dataset: {}, files: [], listeners: {},
    append(...children) { this.children.push(...children); },
    replaceChildren(...children) { this.children = children; },
    classList: { toggle() {} }, addEventListener(type, listener) { this.listeners[type] = listener; }, focus() {},
  });
  const elements = new Map([...html.matchAll(/\bid="([^"]+)"/g)].map((match) => ['#' + match[1], node()]));
  const get = (id) => {
    assert(elements.has(id), `missing element ${id}`);
    return elements.get(id);
  };
  get('#format').value = 'ddi';
  get('#severity').value = 'all';
  get('#code').value = 'return await api.checkCodebook(input, { format });';
  const workers = [];
  const context = vm.createContext({
    document: { querySelector: get, querySelectorAll: () => [], createElement: node, addEventListener() {} },
    Blob, Uint8Array, workerSource: '// Worker is tested separately in bundle.test.mjs',
    URL: { createObjectURL: () => 'blob:unit-test', revokeObjectURL() {} },
    Worker: class {
      constructor() { this.sent = []; workers.push(this); }
      postMessage(message) { this.sent.push(message); }
      terminate() {}
    },
  });
  vm.runInContext(app, context);
  vm.runInContext('ready = true; controls();', context);
  return { context, get, worker: workers[0] };
}

test('a new Excel requires choosing its format before click or keyboard can run it', async () => {
  const { get, worker } = mountedApp();
  get('#input-file').files = [{ name: 'foundation.xlsx', arrayBuffer: async () => new Uint8Array([1, 2, 3]).buffer }];
  await get('#input-file').onchange();
  assert.equal(get('#format').value, '');
  assert.equal(get('#run').disabled, true);
  assert.match(get('#format-hint').textContent, /재단 코드북.*일반 DDI/);
  get('#run').onclick();
  get('#code').listeners.keydown({ key: 'Enter', ctrlKey: true, preventDefault() {} });
  assert.equal(worker.sent.length, 0);
  get('#format').value = 'kosac-xlsx';
  get('#format').onchange();
  assert.equal(get('#run').disabled, false);
  get('#run').onclick();
  assert.equal(worker.sent.length, 1);
  assert.equal(worker.sent[0].type, 'run');
  assert.equal(worker.sent[0].format, 'kosac-xlsx');
  assert(html.includes('<option value="" disabled>Excel 양식을 선택하세요</option>'), 'Excel format prompt missing from the built HTML');
});

test('XML selects DDI while another Excel clears a previous editing-workbook format', async () => {
  const { get } = mountedApp();
  get('#format').value = 'ddi-xlsx';
  get('#input-file').files = [{ name: 'next.xlsx', arrayBuffer: async () => new Uint8Array([1]).buffer }];
  await get('#input-file').onchange();
  assert.equal(get('#format').value, '');
  assert.equal(get('#run').disabled, true);
  get('#input-file').files = [{ name: 'codebook.xml', arrayBuffer: async () => new TextEncoder().encode('<codeBook/>').buffer }];
  await get('#input-file').onchange();
  assert.equal(get('#format').value, 'ddi');
  assert.equal(get('#run').disabled, false);
});

test('accepted history warnings and unexecuted recoding remain visible without a submission-success claim', () => {
  const { context, get } = mountedApp();
  context.report = {
    format: 'kosac-xlsx', templateVersion: '4.0', rejections: 0, warnings: 1,
    steps: [
      { id: 'C', name: '코드북 규칙', detail: '입력 형식·참조', status: 'warning' },
      { id: 'RECODING', name: '재코딩 실행·재현', detail: '변환 실행·영향 건수는 검사하지 않습니다.', status: 'not_run' },
    ],
    issues: [{ Rule: 'C25', Level: 'W', Sheet: '05_재코딩이력', Cell: 'E3', Actual: '(코드북 04)', Expected: '수정 결과 별도 확인', Message: '코드북 수정 이력으로 수용했습니다.' }],
  };
  vm.runInContext('renderReport(report)', context);
  assert.equal(get('#report').hidden, false);
  assert.equal(get('#report-summary').textContent, '반려 0개 · 경고 1개');
  assert.match(get('#message').textContent, /1개 검사는 미실행/);
  assert.equal(get('#steps').children[1].children[1].textContent, '미실행');
  assert.equal(get('#steps').children[1].children[1].className, 'step-status not_run');
  assert.equal(get('#findings').children[0].children[0].children[0].textContent, '경고');
  assert(html.includes('<code>TRANSFORM_TYPE</code>'), 'v4 input guidance missing from the built HTML');
  assert(html.includes('미착수·확인 필요·규칙 확정·처리 완료·검증 완료·적용 제외'), 'history status guidance is stale');
  get('#severity').value = 'R';
  get('#severity').onchange();
  assert.equal(get('#finding-count').textContent, '발견 항목 0 / 1');
});
