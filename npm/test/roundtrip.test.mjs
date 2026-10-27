import { test } from 'node:test';
import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import { ddiToXlsx, templateXlsx, xlsxToDdi } from '../index.js';

const fixture = (name) =>
  readFileSync(new URL(`../../testdata/kofac_2025_edu/${name}`, import.meta.url));

test('KOFAC 2025 round-trips to the canonical XML and JSON', async () => {
  const toX = await ddiToXlsx(fixture('survey.xml'));
  assert.ok(toX.xlsx instanceof Uint8Array);
  assert.deepEqual(toX.warnings, []);

  const { xml, json, warnings } = await xlsxToDdi(toX.xlsx);
  assert.equal(xml, fixture('canonical.xml').toString('utf8'));
  assert.equal(json, fixture('canonical.json').toString('utf8'));
  assert.deepEqual(warnings, []);
});

test('the blank template converts to DDI', async () => {
  const xlsx = await templateXlsx();
  assert.ok(xlsx instanceof Uint8Array);
  const { xml, warnings } = await xlsxToDdi(xlsx);
  assert.match(xml, /<stdyDscr>/);
  assert.deepEqual(warnings, []);
});

test('a non-DDI document rejects with the Go error message', async () => {
  await assert.rejects(ddiToXlsx('<x/>'), (err) => {
    assert.ok(err instanceof Error);
    assert.match(err.message, /not a DDI Codebook 2\.5 document/);
    return true;
  });
});

test('a bad argument fails only its own call', async () => {
  await assert.rejects(xlsxToDdi('not bytes'), /syscall\/js/);
  // A Go error here, not "Go program has already exited", shows the module survived.
  await assert.rejects(xlsxToDdi(new Uint8Array([1, 2, 3])), /zip/);
});

test('a call that stops the Go program fails alone; the next call loads a fresh one', async () => {
  // Copying this "array" in asks the Go side for 4 GiB, which ends the Go program.
  // Its crash report and exit code go to the console, so they are kept out of
  // the test output.
  const { log, warn } = console;
  console.log = console.warn = () => {};
  try {
    await assert.rejects(xlsxToDdi({ length: 2 ** 32 - 1 }), { message: 'ddi-xlsx converter stopped; try again' });
  } finally {
    Object.assign(console, { log, warn });
  }
  const { xml } = await xlsxToDdi((await ddiToXlsx(fixture('survey.xml'))).xlsx);
  assert.equal(xml, fixture('canonical.xml').toString('utf8'));
});
