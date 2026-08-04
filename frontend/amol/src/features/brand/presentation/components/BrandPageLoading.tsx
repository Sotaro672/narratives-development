// frontend/amol/src/features/brand/presentation/components/BrandPageLoading.tsx

export default function BrandPageLoading() {
  return (
    <div
      className="brand-page brand-page-centered"
      aria-live="polite"
      aria-busy="true"
    >
      <div
        className="brand-page-loading"
        role="status"
      >
        ブランド情報を読み込み中...
      </div>
    </div>
  );
}