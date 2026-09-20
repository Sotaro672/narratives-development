// frontend/mall/src/features/brand/presentation/components/BrandPageLoading.tsx

import TextState from "../../../../components/ui/TextState";

export default function BrandPageLoading() {
  return (
    <div
      className="brand-page brand-page-centered"
      aria-live="polite"
      aria-busy="true"
    >
      <TextState variant="loading" className="brand-page-loading">
        ブランド情報を読み込み中...
      </TextState>
    </div>
  );
}