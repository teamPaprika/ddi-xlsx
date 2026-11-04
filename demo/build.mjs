import { readFile, writeFile, copyFile, mkdir } from 'node:fs/promises';
import { execFileSync } from 'node:child_process';
import { fileURLToPath } from 'node:url';
import path from 'node:path';
import { gzipSync } from 'node:zlib';

const root = fileURLToPath(new URL('../', import.meta.url));
const run = (args, options = {}) => execFileSync('go', args, { cwd: root, stdio: 'inherit', ...options });
run(['build', '-trimpath', '-ldflags=-s -w', '-o', 'npm/ddi-xlsx.wasm', './wasm'], {
  env: { ...process.env, GOOS: 'js', GOARCH: 'wasm' },
});
const goroot = execFileSync('go', ['env', 'GOROOT'], { cwd: root, encoding: 'utf8' }).trim();
await copyFile(path.join(goroot, 'lib/wasm/wasm_exec.js'), path.join(root, 'npm/wasm_exec.js'));
const read = (name) => readFile(path.join(root, name), 'utf8');
const wasm = await readFile(path.join(root, 'npm/ddi-xlsx.wasm'));
const compressedWasm = gzipSync(wasm, { level: 9 });
let sdk = await read('npm/index.js');
const asset = "new URL('./ddi-xlsx.wasm', import.meta.url)";
const streaming = 'result = await WebAssembly.instantiateStreaming(response, go.importObject);';
if (sdk.split(asset).length !== 2 || sdk.split(streaming).length !== 2 || !sdk.startsWith("import './wasm_exec.js';")) {
  throw new Error('SDK loader changed; update the single-file packaging step.');
}
// Compress the embedded asset so large inline scripts fit browser document limits.
sdk = sdk.replace("import './wasm_exec.js';", '')
  .replace(asset, `new URL('data:application/gzip;base64,${compressedWasm.toString('base64')}')`)
  .replace(streaming, `if (typeof DecompressionStream === 'undefined') throw new Error('이 데모는 gzip 압축 해제를 지원하는 최신 브라우저가 필요합니다.');
    const unpacked = new Response(response.body.pipeThrough(new DecompressionStream('gzip')), {
      headers: { 'Content-Type': 'application/wasm' },
    });
    result = await WebAssembly.instantiateStreaming(unpacked, go.importObject);`)
  .replace(/^export (?=async function )/gm, '');
const worker = `${await read('npm/wasm_exec.js')}\n${sdk}\n${await read('demo/worker.js')}`;
const jsString = (value) => JSON.stringify(value).replaceAll('<', '\\u003c');
const app = (await read('demo/app.js')).replaceAll('</script', '<\\/script');
const html = (await read('demo/index.html'))
  .replace('<!--STYLE-->', `<style>${await read('demo/style.css')}</style>`)
  .replace('<!--SCRIPT-->', `<script>const workerSource=${jsString(worker)};\n${app}</script>`);
await mkdir(path.join(root, 'demo/dist'), { recursive: true });
const output = path.join(root, 'demo/dist/index.html');
await writeFile(output, html);
console.log(`Built ${output} (${(Buffer.byteLength(html) / 1024 / 1024).toFixed(1)} MiB). No backend or external assets required.`);
