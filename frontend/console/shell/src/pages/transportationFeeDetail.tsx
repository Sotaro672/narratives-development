// frontend/console/shell/src/pages/transportationFeeDetail.tsx

import { useCallback, useState } from "react";
import { CircleAlert, X } from "lucide-react";
import { useParams } from "react-router-dom";

import { AdminCard } from "../features/admin/presentation/components/AdminCard";
import FeeEditCard from "../features/transportation/presentation/component/feeEditCard";
import IslandFeeEditCard from "../features/transportation/presentation/component/islandFeeEditCard";
import PlanNameCard from "../features/transportation/presentation/component/planNameCard";
import SinglePrefectureFeeCard from "../features/transportation/presentation/component/singlePrefectureFeeCard";
import { useTransportationFeeDetail } from "../features/transportation/presentation/hook/useTransportationFeeDetail";
import PageStyle from "../layout/PageStyle/PageStyle";
import { safeDateTimeLabelJa } from "../shared/util/dateJa";

export default function TransportationFeeDetail() {
  const { transportationId } = useParams<{
    transportationId: string;
  }>();

  const {
    vm,
    handlers,
  } = useTransportationFeeDetail(transportationId);

  const [isEditing, setIsEditing] = useState(false);

  const transportation = vm.transportation;

  const createdAt = transportation
    ? safeDateTimeLabelJa(
        transportation.createdAt,
        "",
      )
    : "";

  const updatedAt = transportation
    ? safeDateTimeLabelJa(
        transportation.updatedAt,
        "",
      )
    : "";

  const handleEdit = useCallback(() => {
    setIsEditing(true);
  }, []);

  const handleCancel = useCallback(() => {
    handlers.onReset();
    setIsEditing(false);
  }, [handlers]);

  const handleSave = useCallback(async () => {
    await handlers.onSave();
    setIsEditing(false);
  }, [handlers]);

  const handleDelete = useCallback(async () => {
    await handlers.onDelete();
  }, [handlers]);

  const disabled =
    !isEditing ||
    vm.saving ||
    vm.deleting;

  const left = (
    <div className="space-y-6">
      {vm.loading ? (
        <div className="rounded-lg border border-slate-200 bg-white px-5 py-12 text-center text-sm text-slate-500">
          配送料金設定を読み込んでいます...
        </div>
      ) : transportation ? (
        <>
          <PlanNameCard
            name={transportation.name}
            disabled={disabled}
            onChangeName={handlers.onChangeName}
          />

          {vm.regions.map((region) => {
            if (
              region.region === "hokkaido" ||
              region.region === "okinawa"
            ) {
              return (
                <SinglePrefectureFeeCard
                  key={region.region}
                  region={region}
                  disabled={disabled}
                  onChangePrefectureAmount={
                    handlers.onChangePrefectureAmount
                  }
                />
              );
            }

            return (
              <FeeEditCard
                key={region.region}
                region={region}
                disabled={disabled}
                onChangeRegionAmount={
                  handlers.onChangeRegionAmount
                }
                onChangePrefectureAmount={
                  handlers.onChangePrefectureAmount
                }
              />
            );
          })}

          <IslandFeeEditCard
            islands={vm.islandRates}
            disabled={disabled}
            onChangeAmount={
              handlers.onChangeIslandRateAmount
            }
          />

          {vm.regions.length === 0 &&
            vm.islandRates.length === 0 && (
              <div className="rounded-lg border border-slate-200 bg-white px-5 py-12 text-center text-sm text-slate-500">
                配送料金データを取得できませんでした。
              </div>
            )}
        </>
      ) : (
        <div className="rounded-lg border border-slate-200 bg-white px-5 py-12 text-center text-sm text-slate-500">
          配送料金設定を表示できませんでした。
        </div>
      )}
    </div>
  );

  const right = (
    <div className="space-y-4">
      {vm.loading ? (
        <div className="rounded-lg border border-slate-200 bg-white px-5 py-12 text-center text-sm text-slate-500">
          管理情報を読み込んでいます...
        </div>
      ) : transportation ? (
        <AdminCard
          title="管理情報"
          mode="view"
          showAssignee={false}
          createdByName={
            transportation.createdByName || null
          }
          createdAt={createdAt || null}
          updatedByName={
            transportation.updatedByName || null
          }
          updatedAt={updatedAt || null}
        />
      ) : null}
    </div>
  );

  return (
    <>
      <PageStyle
        layout="grid-2"
        title="配送料金詳細"
        onBack={handlers.onBack}
        onEdit={
          !isEditing
            ? handleEdit
            : undefined
        }
        onDelete={
          isEditing
            ? handleDelete
            : undefined
        }
        onCancel={
          isEditing
            ? handleCancel
            : undefined
        }
        onSave={
          isEditing
            ? handleSave
            : undefined
        }
        isSaving={vm.saving}
      >
        {[left, right]}
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
            aria-labelledby="transportation-detail-error-title"
            aria-describedby="transportation-detail-error-description"
            className="w-full max-w-md overflow-hidden rounded-xl border border-[hsl(var(--border))] bg-[hsl(var(--card))] shadow-2xl"
          >
            <div className="flex items-center justify-between border-b border-[hsl(var(--border))] px-5 py-4">
              <div className="flex items-center gap-2">
                <CircleAlert
                  size={20}
                  className="shrink-0 text-red-500"
                  aria-hidden="true"
                />

                <h2
                  id="transportation-detail-error-title"
                  className="text-sm font-semibold text-[hsl(var(--foreground))]"
                >
                  配送料金設定を取得できませんでした
                </h2>
              </div>

              <button
                type="button"
                onClick={handlers.onDismissError}
                className="inline-flex h-8 w-8 shrink-0 items-center justify-center rounded-md text-[hsl(var(--muted-foreground))] transition hover:bg-[hsl(var(--muted))] hover:text-[hsl(var(--foreground))]"
                aria-label="閉じる"
              >
                <X
                  size={18}
                  aria-hidden="true"
                />
              </button>
            </div>

            <div className="px-5 py-6">
              <p
                id="transportation-detail-error-description"
                className="text-sm leading-6 text-[hsl(var(--foreground))]"
              >
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