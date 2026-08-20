// frontend\console\shell\src\pages\transportationFeeCreate.tsx

import { CircleAlert, X } from "lucide-react";

import FeeEditCard from "../features/transportation/presentation/component/feeEditCard";
import IslandFeeEditCard from "../features/transportation/presentation/component/islandFeeEditCard";
import PlanNameCard from "../features/transportation/presentation/component/planNameCard";
import SinglePrefectureFeeCard from "../features/transportation/presentation/component/singlePrefectureFeeCard";
import { useTransportationFeeCreate } from "../features/transportation/presentation/hook/useTransportationFeeCreate";
import PageStyle from "../layout/PageStyle/PageStyle";

export default function TransportationFee() {
  const { vm, handlers } = useTransportationFeeCreate();
  const disabled = vm.loading || vm.saving;

  return (
    <>
      <PageStyle
        layout="single"
        title="料金設定"
        onBack={handlers.onBack}
        actions={
          <>
            <button type="button" disabled={disabled || !vm.isDirty} onClick={handlers.onReset} className="page-header__btn page-header__btn--ghost">
              リセット
            </button>

            <button
              type="button"
              disabled={disabled || !vm.transportation}
              onClick={() => void handlers.onSave()}
              className="page-header__btn"
              aria-busy={vm.saving}
            >
              {vm.saving ? "保存中" : "登録"}
            </button>
          </>
        }
      >
        <div className="mx-auto w-full max-w-5xl">
          {vm.successMessage && (
            <div role="status" className="mb-6 rounded-lg border border-emerald-200 bg-emerald-50 px-4 py-3 text-sm text-emerald-700">
              {vm.successMessage}
            </div>
          )}

          {vm.loading ? (
            <div className="rounded-lg border border-slate-200 bg-white px-5 py-12 text-center text-sm text-slate-500">
              配送料金設定を読み込んでいます...
            </div>
          ) : vm.transportation ? (
            <div className="space-y-6">
              <PlanNameCard name={vm.transportation.name} disabled={disabled} onChangeName={handlers.onChangeName} />

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

              <IslandFeeEditCard islands={vm.islandRates} disabled={disabled} onChangeAmount={handlers.onChangeIslandRateAmount} />

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

      {vm.error && (
        <div
          className="fixed inset-0 z-[100] flex items-center justify-center bg-black/40 px-4 py-6"
          role="presentation"
          onMouseDown={(event) => {
            if (event.target === event.currentTarget) {
              handlers.onDismissError();
            }
          }}
        >
          <div
            role="dialog"
            aria-modal="true"
            aria-labelledby="transportation-error-title"
            aria-describedby="transportation-error-description"
            className="w-full max-w-md overflow-hidden rounded-xl border border-[hsl(var(--border))] bg-[hsl(var(--card))] shadow-2xl"
          >
            <div className="flex items-center justify-between border-b border-[hsl(var(--border))] px-5 py-4">
              <div className="flex items-center gap-2">
                <CircleAlert size={20} className="shrink-0 text-red-500" aria-hidden="true" />
                <h2 id="transportation-error-title" className="text-sm font-semibold text-[hsl(var(--foreground))]">
                  入力内容を確認してください
                </h2>
              </div>

              <button
                type="button"
                onClick={handlers.onDismissError}
                className="inline-flex h-8 w-8 shrink-0 items-center justify-center rounded-md text-[hsl(var(--muted-foreground))] transition hover:bg-[hsl(var(--muted))] hover:text-[hsl(var(--foreground))]"
                aria-label="閉じる"
              >
                <X size={18} aria-hidden="true" />
              </button>
            </div>

            <div className="px-5 py-6">
              <p id="transportation-error-description" className="text-sm leading-6 text-[hsl(var(--foreground))]">
                {vm.error}
              </p>
            </div>

            <div className="flex items-center justify-end border-t border-[hsl(var(--border))] px-5 py-4">
              <button
                type="button"
                onClick={handlers.onDismissError}
                className="inline-flex h-9 items-center justify-center rounded-[10px] bg-[hsl(var(--primary))] px-5 text-sm font-medium text-[hsl(var(--primary-foreground))] transition hover:opacity-90"
              >
                閉じる
              </button>
            </div>
          </div>
        </div>
      )}
    </>
  );
}