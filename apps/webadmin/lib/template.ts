/**
 * Gera e baixa um modelo de planilha em CSV (UTF-8 com BOM) para o usuario
 * preencher e depois importar. O Excel pt-BR abre CSV com delimitador `;`.
 */
export function downloadCSVTemplate(filename: string, rows: string[][], delimiter = ";") {
  const escape = (value: string) => {
    const s = value ?? "";
    return /["\n\r]/.test(s) || s.includes(delimiter) ? `"${s.replace(/"/g, '""')}"` : s;
  };
  const body = rows.map((row) => row.map(escape).join(delimiter)).join("\r\n");
  // BOM para o Excel reconhecer os acentos em UTF-8.
  const blob = new Blob(["\ufeff" + body], { type: "text/csv;charset=utf-8" });
  const url = URL.createObjectURL(blob);
  const a = document.createElement("a");
  a.href = url;
  a.download = filename;
  document.body.appendChild(a);
  a.click();
  a.remove();
  URL.revokeObjectURL(url);
}
