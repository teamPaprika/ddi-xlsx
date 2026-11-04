import { test } from 'node:test';
import assert from 'node:assert/strict';
import { readFile } from 'node:fs/promises';
import { checkCodebook, ddiToXlsx, kosacTemplateXlsx, kosacToDdi, xlsxToDdi } from '../index.js';

const xml = `<codeBook xmlns="ddi:codebook:2_5" version="2.5">
<stdyDscr><citation><titlStmt><titl>Synthetic browser example</titl></titlStmt></citation></stdyDscr>
<fileDscr ID="F1" URI="data.csv"/>
<dataDscr><var ID="V1" name="gender" files="F1"><labl>Gender</labl>
<catgry><catValu>01</catValu><labl>First code</labl></catgry><varFormat type="character"/>
</var></dataDscr></codeBook>`;

test('WASM checks DDI references and reports XSD as not run', async () => {
  const report = await checkCodebook(xml);
  assert.equal(report.rejections, 0);
  assert.deepEqual(report.issues, []);
  assert.equal(report.steps.find((s) => s.id === 'REF').status, 'pass');
  assert.equal(report.steps.find((s) => s.id === 'XSD').status, 'not_run');
  const broken = await checkCodebook(xml.replace('files="F1"', 'files="MISSING"'));
  assert.equal(broken.rejections, 1);
  assert.equal(broken.issues[0].Rule, 'REF');
  assert.equal(broken.issues[0].Actual, 'MISSING');
  assert.match(broken.issues[0].Path, /@files$/);
});

test('generic editing workbook and JSON use the existing conversion and reference checks', async () => {
  const { xlsx } = await ddiToXlsx(xml);
  const original = xlsx.slice();
  const report = await checkCodebook(xlsx, { format: 'ddi-xlsx' });
  assert.equal(report.rejections, 0);
  assert.deepEqual(xlsx, original);
  const out = await xlsxToDdi(xlsx);
  assert.match(out.xml, /<catValu>01<\/catValu>/);
  assert.equal((await checkCodebook(out.json)).rejections, 0);
});

test('legacy kosac extensions reject while XSD remains unexecuted', async () => {
  const legacy = xml.replace('version="2.5"', 'version="2.5" xmlns:kosac="urn:kosac:survey-meta:3.3"')
    .replace('</stdyDscr>', '<kosac:study/></stdyDscr>');
  const json = JSON.stringify({ codeBook: {
    '@xmlns': 'ddi:codebook:2_5', '@xmlns:kosac': 'urn:kosac:survey-meta:3.3',
    stdyDscr: { 'kosac:study': {} },
  } });
  for (const input of [legacy, json]) {
    const report = await checkCodebook(input);
    assert.equal(report.rejections, 1);
    assert(report.issues.some((issue) => issue.Rule === 'DDI-EXTENSION' && issue.Level === 'R'));
    assert.equal(report.steps.find((step) => step.id === 'REF').status, 'fail');
    assert.equal(report.steps.find((step) => step.id === 'XSD').status, 'not_run');
  }
});

test('foundation v4 blank runs S05 and C rules without claiming reference/CSV/XSD checks', async () => {
  const input = await kosacTemplateXlsx();
  const report = await checkCodebook(input, { format: 'kosac-xlsx' });
  assert.equal(report.templateVersion, '4.0');
  assert.equal(report.steps.find((s) => s.id === 'S05').status, 'pass');
  assert.equal(report.steps.find((s) => s.id === 'C').status, 'fail');
  assert(report.rejections > 0);
  assert(report.issues.some((issue) => issue.Rule.startsWith('C') && issue.Sheet));
  for (const id of ['REFERENCE', 'RESPONSE', 'XSD']) assert.equal(report.steps.find((s) => s.id === id).status, 'not_run');
});

test('a generic workbook is rejected as a foundation template before C rules run', async () => {
  const { xlsx } = await ddiToXlsx(xml);
  const report = await checkCodebook(xlsx, { format: 'kosac-xlsx' });
  assert(report.issues.some((issue) => issue.Rule === 'S05' && issue.Level === 'R'));
  assert.equal(report.steps.find((s) => s.id === 'C').status, 'not_run');
});

test('kosacToDdi writes standard DDI, or returns only S05 issues for another form', async () => {
  const { xlsx } = await ddiToXlsx(xml);
  const rejected = await kosacToDdi(xlsx);
  assert.equal(rejected.xml, null);
  assert.equal(rejected.json, null);
  assert(rejected.issues.some((issue) => issue.Rule === 'S05' && issue.Level === 'R'));
  await assert.rejects(kosacToDdi(await kosacTemplateXlsx()), /TITLE/);
  await assert.rejects(kosacToDdi(new Uint8Array([1, 2, 3])), /zip/);
  const input = new Uint8Array(await readFile(new URL('../../testdata/kosac/v40_sample_2024.xlsx', import.meta.url)));
  const out = await kosacToDdi(input);
  assert.match(out.xml, /^<\?xml[^>]*>\n<codeBook xmlns="ddi:codebook:2_5"/);
  assert.doesNotMatch(out.xml, /kosac:/);
  assert.equal(JSON.parse(out.json).codeBook['@xmlns'], 'ddi:codebook:2_5');
  assert(!out.issues.some((issue) => issue.Level === 'R'));
});

test('legacy foundation v3.2 reports its own version and unexecuted recoding', async () => {
  const input = new Uint8Array(await readFile(new URL('../../testdata/kosac/v32_sample_2024.xlsx', import.meta.url)));
  const report = await checkCodebook(input, { format: 'kosac-xlsx' });
  assert.equal(report.templateVersion, '3.2');
  assert.equal(report.steps.find((step) => step.id === 'S05').status, 'pass');
  assert.equal(report.steps.find((step) => step.id === 'C').status, 'fail');
  assert.equal(report.steps.find((step) => step.id === 'RECODING').status, 'not_run');
});

test('malformed files fail their report alone; unsupported formats and arguments reject', async () => {
  assert((await checkCodebook('<broken')).rejections > 0);
  assert((await checkCodebook(new Uint8Array([1, 2, 3]), { format: 'kosac-xlsx' })).rejections > 0);
  const brokenWorkbook = await checkCodebook(new Uint8Array([1, 2, 3]), { format: 'ddi-xlsx' });
  assert(brokenWorkbook.rejections > 0);
  assert.equal(brokenWorkbook.steps.find((step) => step.id === 'READ').status, 'fail');
  assert.equal(brokenWorkbook.steps.find((step) => step.id === 'REF').status, 'not_run');
  for (const format of ['unknown', '__proto__', 'toString']) {
    await assert.rejects(checkCodebook(xml, { format }), /format must be/);
  }
  await assert.rejects(checkCodebook('not bytes', { format: 'ddi-xlsx' }), /Uint8Array/);
  assert.equal((await checkCodebook(xml)).rejections, 0);
});

test('supplied v4 codebooks keep unresolved errors while corrected rules stop rejecting valid metadata', async () => {
  for (const year of ['2024', '2025']) {
    const input = new Uint8Array(await readFile(new URL(`../../testdata/kosac/v40_sample_${year}.xlsx`, import.meta.url)));
    const original = input.slice();
    const report = await checkCodebook(input, { format: 'kosac-xlsx' });
    assert.deepEqual(input, original);
    assert(!report.issues.some((i) => i.Rule === 'C09' && i.Level === 'R'));
    assert(!report.issues.some((i) => i.Rule === 'C02' && i.Cell === 'E27'));
    assert(!report.issues.some((i) => ['C12', 'C17'].includes(i.Rule)));
    assert.equal(report.issues.filter((i) => i.Rule === 'C22').length, year === '2024' ? 0 : 1);
    assert(report.issues.some((i) => i.Rule === 'C10' && i.Level === 'R'));
    if (year === '2024') {
      assert.equal(report.issues.filter((i) => i.Rule === 'C25').length, 2);
      assert(report.issues.some((i) => i.Rule === 'C18' && i.Level === 'W' && i.Expected === 'REPORT:p331-'));
    } else {
      assert(report.issues.some((i) => i.Rule === 'C18' && i.Level === 'R' && i.Cell === 'E15'));
      assert.equal(report.issues.filter((i) => i.Rule === 'C15').length, 2);
    }
    assert(!report.issues.some((i) => i.Rule === 'C25' && i.Level === 'R'));
    for (const id of ['RECODING', 'REFERENCE', 'RESPONSE', 'XSD']) assert.equal(report.steps.find((s) => s.id === id).status, 'not_run');
  }
});
