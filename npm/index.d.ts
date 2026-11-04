/**
 * Converts a DDI Codebook 2.5 document, as XML or JSON, to an editable xlsx workbook.
 * Rejects with an Error that carries the converter's message.
 */
export function ddiToXlsx(input: string | Uint8Array): Promise<{ xlsx: Uint8Array; warnings: string[] }>;

/**
 * Converts a workbook from `ddiToXlsx` or `templateXlsx`, edited or not, back to canonical DDI XML and JSON.
 * Rejects with an Error that carries the converter's message.
 */
export function xlsxToDdi(xlsx: Uint8Array): Promise<{ xml: string; json: string; warnings: string[] }>;

/**
 * Returns a blank workbook to fill in and convert with `xlsxToDdi`.
 */
export function templateXlsx(): Promise<Uint8Array>;

export type CheckFormat = 'ddi' | 'ddi-xlsx' | 'kosac-xlsx';

export interface CheckIssue {
  Rule: string;
  Level: 'R' | 'W';
  Sheet: string;
  Cell: string;
  Path: string;
  FileNo: string;
  VarName: string;
  Code: string;
  Actual: string;
  Expected: string;
  Message: string;
}

export interface CheckReport {
  format: CheckFormat;
  templateVersion?: string;
  rejections: number;
  warnings: number;
  steps: Array<{
    id: string;
    name: string;
    status: 'pass' | 'fail' | 'warning' | 'not_run';
    detail: string;
  }>;
  issues: CheckIssue[];
}

/** Runs the existing Go checks. XSD, response-data checks and external reference data are not supplied. */
export function checkCodebook(input: string | Uint8Array, options?: { format?: CheckFormat }): Promise<CheckReport>;

/** Returns the foundation's empty v4.0 workbook, not the generic DDI editing workbook. */
export function kosacTemplateXlsx(): Promise<Uint8Array>;

/**
 * Converts a foundation (KOSAC) v3.2 or v4.0 workbook to standard DDI Codebook 2.5 XML and JSON,
 * like `ddi-xlsx kosac-xml`. `xml` and `json` are null when the form structure check (S05)
 * rejects the workbook; `issues` says why. Check content rules (C) with `checkCodebook`; XSD runs in the CLI.
 */
export function kosacToDdi(xlsx: Uint8Array): Promise<{ xml: string | null; json: string | null; issues: CheckIssue[] }>;
