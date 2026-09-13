// frontend/mall/src/features/howToUse/presentation/components/HowToUsePdfButton.tsx

export function printHowToUsePdf() {
  window.print();
}

export default function HowToUsePdfButton() {
  return (
    <button
      type="button"
      className="how-to-use-detail-page__pdf-button"
      onClick={printHowToUsePdf}
      aria-label="PDFで保存"
      title="PDFで保存"
    >
      PDF
    </button>
  );
}