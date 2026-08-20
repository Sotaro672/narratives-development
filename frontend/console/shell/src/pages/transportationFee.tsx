// frontend/console/shell/src/pages/transportationFee.tsx

import { useCallback } from "react";
import { useNavigate } from "react-router-dom";

import PageStyle from "../layout/PageStyle/PageStyle";

export default function TransportationFee() {
  const navigate = useNavigate();

  const handleBack = useCallback(() => {
    navigate(-1);
  }, [navigate]);

  return (
    <PageStyle layout="single" title="料金設定" onBack={handleBack}>
      <div className="max-w-3xl">
        <div className="rounded-lg border border-slate-200 bg-white p-5">
          <h2 className="text-sm font-semibold text-slate-900">配送料金設定</h2>
          <p className="mt-2 text-sm text-slate-500">
            配送に関する料金設定を管理します。
          </p>
        </div>
      </div>
    </PageStyle>
  );
}