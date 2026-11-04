// Runs the npm SDK, and so the Go wasm, off the page's main thread.
// Messages in:  {id, op, args} for an SDK function, {id, op: 'example', name},
//               or {id, op: 'repl', code, input, format}.
// Messages out: {type: 'ready' | 'init-error'} once, then {id, ok, value | error, logs, files}.
import * as sdk from './sdk/index.js';

const xml = `<?xml version="1.0" encoding="UTF-8"?>
<codeBook xmlns="ddi:codebook:2_5" version="2.5">
  <stdyDscr><citation><titlStmt><titl>합성 참여자 조사</titl></titlStmt></citation></stdyDscr>
  <fileDscr ID="F1" URI="responses.csv"><fileTxt><fileName>responses.csv</fileName></fileTxt></fileDscr>
  <dataDscr>
    <var ID="V_GENDER" name="gender" files="F1"><labl>성별</labl>
      <catgry><catValu>01</catValu><labl>남자</labl></catgry>
      <catgry><catValu>02</catValu><labl>여자</labl></catgry>
      <varFormat type="character"/>
    </var>
    <var ID="V_AGE" name="age" files="F1"><labl>나이</labl>
      <valrng><range min="0" max="120"/></valrng><varFormat type="numeric"/>
    </var>
  </dataDscr>
</codeBook>
`;
const examples = { valid: xml, broken: xml.replace('files="F1"', 'files="F404"') };
const utf8 = (s) => new TextEncoder().encode(s);

// Each example gives the bytes, a file name and the input kind.
const makers = {
  'ddi-xml': async () => [utf8(examples.valid), 'synthetic-codebook.xml', 'ddi'],
  'ddi-broken': async () => [utf8(examples.broken), 'missing-file-reference.xml', 'ddi'],
  'ddi-json': async () => [utf8((await sdk.xlsxToDdi((await sdk.ddiToXlsx(examples.valid)).xlsx)).json), 'synthetic-codebook.json', 'ddi'],
  'ddi-xlsx': async () => [(await sdk.ddiToXlsx(examples.valid)).xlsx, 'synthetic-codebook.xlsx', 'ddi-xlsx'],
  'ddi-template': async () => [await sdk.templateXlsx(), 'template.xlsx', 'ddi-xlsx'],
  'kosac-template': async () => [await sdk.kosacTemplateXlsx(), 'kosac-template-v4.xlsx', 'kosac-xlsx'],
  'kosac-wrong': async () => [(await sdk.ddiToXlsx(examples.valid)).xlsx, 'not-a-foundation-form.xlsx', 'kosac-xlsx'],
};

const api = Object.freeze({ ...sdk });
const state = {};
const AsyncFunction = (async () => {}).constructor;

function printable(value) {
  if (value === undefined) return 'undefined';
  if (typeof value === 'string') return value;
  try {
    return JSON.stringify(value, (_, v) => (v instanceof Uint8Array ? `Uint8Array(${v.length})` : v), 2);
  } catch {
    return String(value);
  }
}

async function repl({ code, input, format }) {
  const logs = [];
  const files = [];
  const out = (...values) => logs.push(values.map(printable).join(' '));
  const download = (name, data, mime = 'application/octet-stream') => {
    if (typeof name !== 'string' || !(typeof data === 'string' || data instanceof Uint8Array)) {
      throw new TypeError('download(name, data)에 맞지 않는 값을 넣었습니다. name은 문자열이고, data는 문자열이나 Uint8Array입니다.');
    }
    files.push({ name, data, mime });
  };
  const console = { log: out, info: out, warn: out, error: out, table: out };
  try {
    const run = new AsyncFunction('api', 'input', 'format', 'examples', 'state', 'console', 'download', code);
    const value = await run(api, input, format, examples, state, console, download);
    return { text: printable(value), logs, files };
  } catch (error) {
    throw Object.assign(error instanceof Error ? error : new Error(String(error)), { logs });
  }
}

self.onmessage = async ({ data }) => {
  const { id, op } = data;
  const started = performance.now();
  try {
    let value;
    if (op === 'repl') value = await repl(data);
    else if (op === 'example' && Object.hasOwn(makers, data.name)) value = await makers[data.name]();
    else if (op !== 'example' && Object.hasOwn(sdk, op)) value = await sdk[op](...(data.args || []));
    else throw new Error(`지원하지 않는 작업입니다: ${op}`);
    self.postMessage({ id, ok: true, value, elapsed: performance.now() - started });
  } catch (error) {
    self.postMessage({ id, ok: false, error: error?.message || String(error), logs: error?.logs, elapsed: performance.now() - started });
  }
};

// Load the wasm now, so the first user action does not pay for it.
try {
  const started = performance.now();
  await sdk.templateXlsx();
  self.postMessage({ type: 'ready', elapsed: performance.now() - started });
} catch (error) {
  self.postMessage({ type: 'init-error', error: error?.message || String(error) });
}
