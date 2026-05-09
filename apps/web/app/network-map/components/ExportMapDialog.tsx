'use client';

import { useCallback, useState } from 'react';

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

type ExportFormat = 'png' | 'pdf';
type PaperSize = 'a3' | 'a4';

interface ExportMapDialogProps {
  open: boolean;
  onClose: () => void;
  mapContainerRef?: React.RefObject<HTMLElement | null>;
}

// ---------------------------------------------------------------------------
// Komponen
// ---------------------------------------------------------------------------

export default function ExportMapDialog({
  open,
  onClose,
  mapContainerRef,
}: ExportMapDialogProps) {
  const [format, setFormat] = useState<ExportFormat>('png');
  const [paperSize, setPaperSize] = useState<PaperSize>('a4');
  const [includeNodeList, setIncludeNodeList] = useState(false);
  const [exporting, setExporting] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const handleExport = useCallback(async () => {
    setExporting(true);
    setError(null);

    try {
      if (format === 'png') {
        await exportPNG(mapContainerRef?.current);
      } else {
        await exportPDF(mapContainerRef?.current, paperSize, includeNodeList);
      }
      onClose();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Gagal export');
    } finally {
      setExporting(false);
    }
  }, [format, paperSize, includeNodeList, mapContainerRef, onClose]);

  if (!open) return null;

  return (
    <div className="fixed inset-0 z-[2000] flex items-center justify-center bg-black/50">
      <div className="mx-4 w-full max-w-sm rounded-lg bg-white p-6 shadow-xl">
        <h2 className="mb-4 text-lg font-semibold text-gray-900">
          Export Peta
        </h2>

        {/* Format selection*/}
        <div className="mb-4">
          <label className="mb-1 block text-sm font-medium text-gray-700">
            Format
          </label>
          <div className="flex gap-2">
            <FormatButton
              label="PNG"
              description="Screenshot peta"
              active={format === 'png'}
              onClick={() => setFormat('png')}
            />
            <FormatButton
              label="PDF"
              description="Dokumen cetak"
              active={format === 'pdf'}
              onClick={() => setFormat('pdf')}
            />
          </div>
        </div>

        {/* PDF options*/}
        {format === 'pdf' && (
          <div className="mb-4 space-y-3">
            <div>
              <label className="mb-1 block text-sm font-medium text-gray-700">
                Ukuran Kertas
              </label>
              <select
                value={paperSize}
                onChange={(e) => setPaperSize(e.target.value as PaperSize)}
                className="w-full rounded-md border border-gray-300 px-3 py-2 text-sm focus:border-blue-500 focus:outline-none focus:ring-1 focus:ring-blue-500"
              >
                <option value="a4">A4</option>
                <option value="a3">A3</option>
              </select>
            </div>

            <label className="flex items-center gap-2">
              <input
                type="checkbox"
                checked={includeNodeList}
                onChange={(e) => setIncludeNodeList(e.target.checked)}
                className="h-4 w-4 rounded border-gray-300 text-blue-600 focus:ring-blue-500"
              />
              <span className="text-sm text-gray-700">
                Sertakan daftar node
              </span>
            </label>
          </div>
        )}

        {error && <p className="mb-3 text-sm text-red-500">{error}</p>}

        {/* Actions*/}
        <div className="flex gap-2">
          <button
            onClick={handleExport}
            disabled={exporting}
            className="flex-1 rounded-md bg-blue-600 px-4 py-2 text-sm font-medium text-white hover:bg-blue-700 disabled:opacity-50"
          >
            {exporting ? 'Mengexport…' : 'Export'}
          </button>
          <button
            onClick={onClose}
            className="rounded-md border border-gray-300 px-4 py-2 text-sm text-gray-700 hover:bg-gray-50"
          >
            Batal
          </button>
        </div>
      </div>
    </div>
  );
}

// ---------------------------------------------------------------------------
// Format button
// ---------------------------------------------------------------------------

function FormatButton({
  label,
  description,
  active,
  onClick,
}: {
  label: string;
  description: string;
  active: boolean;
  onClick: () => void;
}) {
  return (
    <button
      onClick={onClick}
      className={`flex-1 rounded-md border-2 px-3 py-2 text-left transition-colors ${
        active
          ? 'border-blue-600 bg-blue-50'
          : 'border-gray-200 hover:border-gray-300'
      }`}
    >
      <p className="text-sm font-medium text-gray-900">{label}</p>
      <p className="text-xs text-gray-500">{description}</p>
    </button>
  );
}

// ---------------------------------------------------------------------------
// Export implementations
// ---------------------------------------------------------------------------

async function exportPNG(
  container: HTMLElement | null | undefined,
): Promise<void> {
  if (!container) {
    throw new Error('Map container tidak ditemukan');
  }

  // Dynamically import html2canvas
  const html2canvas = (await import('html2canvas')).default;

  const canvas = await html2canvas(container, {
    useCORS: true,
    allowTaint: true,
    scale: 2, // Higher resolution
  });

  // Unduh gambar
  const link = document.createElement('a');
  link.download = `peta-jaringan-${new Date().toISOString().slice(0, 10)}.png`;
  link.href = canvas.toDataURL('image/png');
  link.click();
}

async function exportPDF(
  container: HTMLElement | null | undefined,
  paperSize: PaperSize,
  _includeNodeList: boolean,
): Promise<void> {
  if (!container) {
    throw new Error('Map container tidak ditemukan');
  }

  // Dynamically import dependencies
  const [html2canvasModule, jsPDFModule] = await Promise.all([
    import('html2canvas'),
    import('jspdf'),
  ]);
  const html2canvas = html2canvasModule.default;
  const { jsPDF } = jsPDFModule;

  // Tangkap peta sebagai gambar
  const canvas = await html2canvas(container, {
    useCORS: true,
    allowTaint: true,
    scale: 2,
  });

  // Buat PDF
  const orientation = 'landscape';
  const pdf = new jsPDF({
    orientation,
    unit: 'mm',
    format: paperSize,
  });

  const pageWidth = pdf.internal.pageSize.getWidth();
  const pageHeight = pdf.internal.pageSize.getHeight();

  // Title
  pdf.setFontSize(14);
  pdf.text('Peta Jaringan FTTH', 10, 15);

  // Date
  pdf.setFontSize(8);
  pdf.text(
    `Diekspor: ${new Date().toLocaleDateString('id-ID', {
      day: 'numeric',
      month: 'long',
      year: 'numeric',
      hour: '2-digit',
      minute: '2-digit',
    })}`,
    10,
    22,
  );

  // Gambar peta
  const imgData = canvas.toDataURL('image/png');
  const mapMargin = 10;
  const mapTop = 28;
  const mapWidth = pageWidth - mapMargin * 2;
  const mapHeight = pageHeight - mapTop - 30; // Leave room untuk legend

  pdf.addImage(imgData, 'PNG', mapMargin, mapTop, mapWidth, mapHeight);

  // Legend
  const legendY = pageHeight - 18;
  pdf.setFontSize(7);
  pdf.setTextColor(100);

  const legendItems = [
    { color: '#1e3a5f', label: 'OLT' },
    { color: '#3b82f6', label: 'ODP' },
    { color: '#22c55e', label: 'ONT Online' },
    { color: '#ef4444', label: 'ONT Offline' },
  ];

  let legendX = mapMargin;
  for (const item of legendItems) {
    pdf.setFillColor(item.color);
    pdf.rect(legendX, legendY, 3, 3, 'F');
    pdf.text(item.label, legendX + 5, legendY + 2.5);
    legendX += 30;
  }

  // Unduh
  pdf.save(`peta-jaringan-${new Date().toISOString().slice(0, 10)}.pdf`);
}
