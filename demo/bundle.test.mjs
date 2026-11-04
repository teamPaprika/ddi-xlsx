import { test } from 'node:test';
import assert from 'node:assert/strict';
import { readFile } from 'node:fs/promises';
import { Worker } from 'node:worker_threads';
import { gunzipSync } from 'node:zlib';

test('the single HTML runs its embedded browser worker, WASM and REPL without external assets', { timeout: 30000 }, async (t) => {
  const html = await readFile(new URL('./dist/index.html', import.meta.url), 'utf8');
  assert(!/<(?:script|link)\b[^>]*(?:src|href)=/i.test(html));
  assert(Buffer.byteLength(html) < 8 * 1024 * 1024, 'embedded HTML grew beyond the browser-safe package size');
  const marker = '<script>const workerSource=';
  const start = html.indexOf(marker) + marker.length;
  assert(start >= marker.length);
  const source = JSON.parse(html.slice(start, html.indexOf(';\n', start)));
  const compressed = source.match(/data:application\/gzip;base64,([A-Za-z0-9+/=]+)/);
  assert(compressed, 'embedded compressed WASM missing');
  assert.deepEqual(gunzipSync(Buffer.from(compressed[1], 'base64')), await readFile(new URL('../npm/ddi-xlsx.wasm', import.meta.url)));
  assert(!source.includes("import './wasm_exec.js'"));
  const worker = new Worker(`
    const { parentPort, workerData } = require('node:worker_threads');
    globalThis.crypto ??= require('node:crypto').webcrypto;
    const localFetch = globalThis.fetch;
    globalThis.fetch = (url, options) => {
      if (!String(url).startsWith('data:application/gzip;base64,')) throw new Error('Unexpected external fetch');
      return localFetch(url, options);
    };
    globalThis.process = undefined;
    globalThis.self = globalThis;
    self.postMessage = (data) => parentPort.postMessage(data);
    parentPort.on('message', (data) => self.onmessage({ data }));
    import('data:text/javascript;base64,' + Buffer.from(workerData).toString('base64'))
      .catch((error) => parentPort.postMessage({ type: 'init-error', error: error.message }));
  `, { eval: true, workerData: source });
  t.after(() => worker.terminate());
  const messages = [], waiting = [];
  worker.on('message', (data) => {
    const index = waiting.findIndex((wait) => wait.match(data));
    if (index < 0) messages.push(data);
    else { const [wait] = waiting.splice(index, 1); clearTimeout(wait.timer); wait.resolve(data); }
  });
  const receive = (match) => {
    const index = messages.findIndex(match);
    if (index >= 0) return Promise.resolve(messages.splice(index, 1)[0]);
    return new Promise((resolve, reject) => {
      const wait = { match, resolve, timer: setTimeout(() => reject(new Error('Worker response timed out')), 10000) };
      waiting.push(wait);
    });
  };
  let id = 0;
  const send = (message) => {
    const current = ++id;
    const response = receive((data) => data.id === current);
    worker.postMessage({ id: current, ...message });
    return response;
  };
  const ready = await receive((data) => data.type === 'ready' || data.type === 'init-error');
  assert.equal(ready.type, 'ready', ready.error);
  const valid = await send({ type: 'example', example: 'valid' });
  const result = await send({ type: 'run', input: valid.bytes, format: valid.format, code: 'return await api.checkCodebook(input, { format });' });
  assert.equal(result.result.rejections, 0);
  assert(result.result.steps.some((step) => step.id === 'XSD' && step.status === 'not_run'));
  const broken = await send({ type: 'example', example: 'broken' });
  const invalid = await send({ type: 'run', input: broken.bytes, format: broken.format, code: 'return await api.checkCodebook(input, { format });' });
  assert.equal(invalid.result.rejections, 1);
  assert.equal(invalid.result.issues[0].Actual, 'F404');
  const args = { type: 'run', input: valid.bytes, format: valid.format };
  assert.equal((await send({ ...args, code: 'state.count = (state.count || 0) + 1; return state.count;' })).result, 1);
  assert.equal((await send({ ...args, code: 'state.count = (state.count || 0) + 1; return state.count;' })).result, 2);
  const conversion = await send({ ...args, code: "const out = await api.ddiToXlsx(input); download('example.xlsx', out.xlsx); console.log('converted'); return out.warnings;" });
  assert.equal(conversion.files[0].name, 'example.xlsx');
  assert(conversion.files[0].value instanceof Uint8Array);
  assert(conversion.files[0].value.length > 1000);
  assert.deepEqual(conversion.logs, ['converted']);
  const failed = await send({ ...args, code: "console.log('before error'); throw new Error('intentional error');" });
  assert.equal(failed.type, 'error'); assert.equal(failed.error, 'intentional error');
  assert.deepEqual(failed.logs, ['before error']);
  assert.equal((await send({ ...args, code: 'return 6 * 7;' })).result, 42);
  const foundation = await send({ type: 'example', example: 'foundation' });
  const checked = await send({ type: 'run', input: foundation.bytes, format: foundation.format, code: 'return await api.checkCodebook(input, { format });' });
  assert.equal(checked.result.templateVersion, '4.0');
  assert(checked.result.rejections > 0);
  assert.equal(checked.result.steps.find((step) => step.id === 'RECODING').status, 'not_run');
  for (const [year, rejections, warnings] of [['2024', 1, 155], ['2025', 13, 672]]) {
    const input = new Uint8Array(await readFile(new URL(`../testdata/kosac/v40_sample_${year}.xlsx`, import.meta.url)));
    const checked = await send({ type: 'run', input, format: 'kosac-xlsx', code: 'return await api.checkCodebook(input, { format });' });
    assert.equal(checked.type, 'result', checked.error);
    assert.equal(checked.result.rejections, rejections);
    assert.equal(checked.result.warnings, warnings);
    assert(!checked.result.issues.some((issue) => issue.Rule === 'C25' && issue.Level === 'R'));
    for (const id of ['RECODING', 'REFERENCE', 'RESPONSE', 'XSD']) assert.equal(checked.result.steps.find((step) => step.id === id).status, 'not_run');
  }
  worker.postMessage({ id: ++id, type: 'run', code: 'while (true) {}', input: valid.bytes, format: valid.format });
  await new Promise((resolve) => setTimeout(resolve, 50));
  await worker.terminate();
});
