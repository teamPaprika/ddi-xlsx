const sdk = { ddiToXlsx, xlsxToDdi, templateXlsx, checkCodebook, kosacTemplateXlsx };
const examples = {
  valid: `<?xml version="1.0" encoding="UTF-8"?>
<codeBook xmlns="ddi:codebook:2_5" version="2.5">
  <stdyDscr><citation><titlStmt><titl>가상 참여자 조사</titl></titlStmt></citation></stdyDscr>
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
</codeBook>`,
};
examples.broken = examples.valid.replace('files="F1"', 'files="F404"');
const replState = {};
const AsyncFunction = Object.getPrototypeOf(async function () {}).constructor;
function printable(value) {
  if (value === undefined) return 'undefined';
  try {
    return typeof value === 'string' ? value : JSON.stringify(value, (_, v) =>
      v instanceof Uint8Array ? { type: 'Uint8Array', bytes: v.length, firstBytes: [...v.slice(0, 16)] } : v, 2);
  } catch { return String(value); }
}

self.onmessage = async ({ data: message }) => {
  const { id, type } = message;
  const files = [], logs = [];
  try {
    if (type === 'example') {
      let bytes, format, name;
      if (message.example === 'foundation') {
        bytes = await sdk.kosacTemplateXlsx(); format = 'kosac-xlsx'; name = 'foundation-empty-v4.xlsx';
      } else if (message.example === 'workbook' || message.example === 'wrong-template') {
        ({ xlsx: bytes } = await sdk.ddiToXlsx(examples.valid));
        format = message.example === 'wrong-template' ? 'kosac-xlsx' : 'ddi-xlsx'; name = 'ddi-editing.xlsx';
      } else {
        const broken = message.example === 'broken';
        bytes = new TextEncoder().encode(broken ? examples.broken : examples.valid);
        format = 'ddi'; name = broken ? 'missing-file-reference.xml' : 'synthetic-codebook.xml';
      }
      self.postMessage({ id, type: 'example', bytes, format, name });
      return;
    }
    if (type !== 'run') throw new Error('지원하지 않는 실행 요청입니다.');
    const output = (...values) => logs.push(values.map(printable).join(' '));
    const download = (name, value, mime = 'application/octet-stream') => {
      if (typeof name !== 'string' || !(typeof value === 'string' || value instanceof Uint8Array)) {
        throw new TypeError('download(name, data): name은 문자열, data는 문자열 또는 Uint8Array여야 합니다.');
      }
      files.push({ name, value, mime });
    };
    const started = performance.now();
    const run = new AsyncFunction('api', 'input', 'format', 'examples', 'state', 'console', 'download', message.code);
    const value = await run(sdk, message.input, message.format, examples, replState,
      { log: output, info: output, warn: output, error: output, table: output }, download);
    // A value that cannot be cloned still appears in the REPL as text.
    let result;
    try { result = structuredClone(value); } catch { result = printable(value); }
    self.postMessage({ id, type: 'result', result, text: printable(value), logs, files, elapsed: performance.now() - started });
  } catch (error) {
    self.postMessage({ id, type: 'error', error: error?.message || String(error), logs });
  }
};

// Loading the actual WASM before "ready" keeps initialization out of run timings.
try {
  await sdk.checkCodebook(examples.valid);
  self.postMessage({ type: 'ready' });
} catch (error) {
  self.postMessage({ type: 'init-error', error: error?.message || String(error) });
}
