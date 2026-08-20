// frontend/console/shell/src/pages/transportationFee.tsx

import FeeEditCard from "../features/transportation/presentation/component/feeEditCard";
import IslandFeeEditCard from "../features/transportation/presentation/component/islandFeeEditCard";
import SinglePrefectureFeeCard from "../features/transportation/presentation/component/singlePrefectureFeeCard";
import { useTransportationFee } from "../features/transportation/presentation/hook/useTransportationFee";
import PageStyle from "../layout/PageStyle/PageStyle";

export default function TransportationFee() {
  const { vm, handlers } = useTransportationFee();
  const disabled = vm.loading || vm.saving;

  return (
    <PageStyle
      layout="single"
      title="料金設定"
      onBack={handlers.onBack}
      actions={
        <>
          <button
            type="button"
            disabled={disabled || !vm.isDirty}
            onClick={handlers.onReset}
            className="page-header__btn page-header__btn--ghost"
          >
            リセット
          </button>

          <button
            type="button"
            disabled={disabled || !vm.transportation || !vm.isDirty}
            onClick={() => void handlers.onSave()}
            className="page-header__btn"
            aria-busy={vm.saving}
          >
            {vm.saving ? "保存中" : vm.exists ? "更新" : "登録"}
          </button>
        </>
      }
    >
      <div className="mx-auto w-full max-w-5xl">
        {vm.error && (
          <div
            role="alert"
            className="mb-6 rounded-lg border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-700"
          >
            {vm.error}
          </div>
        )}

        {vm.successMessage && (
          <div
            role="status"
            className="mb-6 rounded-lg border border-emerald-200 bg-emerald-50 px-4 py-3 text-sm text-emerald-700"
          >
            {vm.successMessage}
          </div>
        )}

        {vm.loading ? (
          <div className="rounded-lg border border-slate-200 bg-white px-5 py-12 text-center text-sm text-slate-500">
            配送料金設定を読み込んでいます...
          </div>
        ) : vm.transportation ? (
          <div className="space-y-6">
            {vm.regions.map((region) => {
              if (region.region === "hokkaido" || region.region === "okinawa") {
                return (
                  <SinglePrefectureFeeCard
                    key={region.region}
                    region={region}
                    disabled={disabled}
                    onChangePrefectureAmount={handlers.onChangePrefectureAmount}
                  />
                );
              }

              return (
                <FeeEditCard
                  key={region.region}
                  region={region}
                  disabled={disabled}
                  onChangeRegionAmount={handlers.onChangeRegionAmount}
                  onChangePrefectureAmount={handlers.onChangePrefectureAmount}
                />
              );
            })}

            <IslandFeeEditCard
              islands={vm.islandRates}
              disabled={disabled}
              onChangeAmount={handlers.onChangeIslandRateAmount}
            />

            {vm.regions.length === 0 && vm.islandRates.length === 0 && (
              <div className="rounded-lg border border-slate-200 bg-white px-5 py-12 text-center text-sm text-slate-500">
                配送料金データを取得できませんでした。
              </div>
            )}
          </div>
        ) : (
          <div className="rounded-lg border border-slate-200 bg-white px-5 py-12 text-center text-sm text-slate-500">
            配送料金設定を表示できませんでした。
          </div>
        )}
      </div>
    </PageStyle>
  );
}