// frontend/mall/src/features/howToUse/presentation/components/HowToUsePdfButton.tsx

export default function HowToUsePdfButton() {
  const handlePrint = () => {
    window.print();
  };

  return (
    <button
      type="button"
      className="how-to-use-detail-page__pdf-button"
      onClick={handlePrint}
      aria-label="PDFで保存"
      title="PDFで保存"
    >
      PDFで保存
    </button>
  );
}