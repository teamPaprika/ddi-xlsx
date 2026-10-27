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
