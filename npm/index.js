import './wasm_exec.js';

let api;

function load() {
  api ??= instantiate().catch((err) => {
    // A failed load must not stick: let the next call try again.
    api = undefined;
    throw err;
  });
  return api;
}

async function instantiate() {
  const go = new Go();
  const url = new URL('./ddi-xlsx.wasm', import.meta.url);
  let result;
  // wasm_exec.js defines a stub globalThis.process in browsers, so test for
  // Node itself. This import never runs in a browser; the comments keep
  // bundlers from trying to resolve it.
  if (typeof process !== 'undefined' && process.versions?.node) {
    const { readFile } = await import(/* webpackIgnore: true */ /* @vite-ignore */ 'node:fs/promises');
    result = await WebAssembly.instantiate(await readFile(url), go.importObject);
  } else {
    const response = await fetch(url);
    if (!response.ok) throw new Error(`ddi-xlsx: could not load ${url}: HTTP ${response.status}`);
    result = await WebAssembly.instantiateStreaming(response, go.importObject);
  }
  // Not awaited: it settles only when the Go program exits. Its synchronous part
  // runs main up to select{}, so __ddiXlsx is already set. A fatal Go error,
  // such as running out of memory, exits the program; the next call then
  // loads a fresh one.
  go.run(result.instance).finally(() => {
    api = undefined;
    // A Go timer still pending at the exit would fire into the dead program and
    // throw an uncaught error, which ends a Node process. The field is internal
    // to wasm_exec.js, so a release without it only skips the clean-up.
    go._scheduledTimeouts?.forEach((timeout) => clearTimeout(timeout));
  });
  return globalThis.__ddiXlsx;
}

async function call(name, ...args) {
  const res = (await load())[name](...args);
  // The call during which the Go program exits returns nothing.
  if (res === undefined) throw new Error('ddi-xlsx converter stopped; try again');
  if (!res.ok) throw new Error(res.error);
  return res;
}

export async function ddiToXlsx(input) {
  const bytes = typeof input === 'string' ? new TextEncoder().encode(input) : input;
  const { xlsx, warnings } = await call('toXlsx', bytes);
  return { xlsx, warnings };
}

export async function xlsxToDdi(xlsx) {
  const { xml, json, warnings } = await call('fromXlsx', xlsx);
  const utf8 = new TextDecoder();
  return { xml: utf8.decode(xml), json: utf8.decode(json), warnings };
}

export async function templateXlsx() {
  return (await call('template')).xlsx;
}

export async function checkCodebook(input, { format = 'ddi' } = {}) {
  const methods = { ddi: 'checkDDI', 'ddi-xlsx': 'checkWorkbook', 'kosac-xlsx': 'checkKosac' };
  if (!Object.hasOwn(methods, format)) throw new Error('format must be ddi, ddi-xlsx, or kosac-xlsx');
  const bytes = typeof input === 'string' && format === 'ddi' ? new TextEncoder().encode(input) : input;
  if (!(bytes instanceof Uint8Array)) throw new TypeError('input must be a Uint8Array, or a string for ddi');
  const { report } = await call(methods[format], bytes);
  return JSON.parse(new TextDecoder().decode(report));
}

export async function kosacTemplateXlsx() {
  return (await call('kosacTemplate')).xlsx;
}
