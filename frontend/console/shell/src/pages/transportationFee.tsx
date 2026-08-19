// frontend/console/shell/src/pages/transportationFee.tsx

import PageStyle from "../layout/PageStyle/PageStyle";

export default function TransportationFee() {
  return (
    <PageStyle layout="single" title="料金設定">
      <div className="max-w-3xl">
        <div className="rounded-lg border border-slate-200 bg-white p-5">
          <h2 className="text-sm font-semibold text-slate-900">
            配送料金設定
          </h2>

          <p className="mt-2 text-sm text-slate-500">
            配送に関する料金設定を管理します。
          </p>
        </div>
      </div>
    </PageStyle>
  );
}