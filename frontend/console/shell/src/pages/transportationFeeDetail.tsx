// frontend/console/shell/src/pages/transportationFeeDetail.tsx

import { useCallback, useState } from "react";
import { CircleAlert } from "lucide-react";
import { useParams } from "react-router-dom";

import { AdminCard } from "../features/admin/presentation/components/AdminCard";
import FeeEditCard from "../features/transportation/presentation/component/feeEditCard";
import IslandFeeEditCard from "../features/transportation/presentation/component/islandFeeEditCard";
import PlanNameCard from "../features/transportation/presentation/component/planNameCard";
import SinglePrefectureFeeCard from "../features/transportation/presentation/component/singlePrefectureFeeCard";
import { useTransportationFeeDetail } from "../features/transportation/presentation/hook/useTransportationFeeDetail";
import PageStyle from "../layout/PageStyle/PageStyle";
import { Modal, ModalButton } from "../shared/ui/modal";
import { safeDateTimeLabelJa } from "../shared/util/dateJa";

import "../styles/transportation.css";

export default function TransportationFeeDetail() {
  const { transportationId } = useParams<{
    transportationId: string;
  }>();

  const { vm, handlers } = useTransportationFeeDetail(transportationId);
  const [isEditing, setIsEditing] = useState(false);
  const transportation = vm.transportation;

  const createdAt = transportation
    ? safeDateTimeLabelJa(transportation.createdAt, "")
    : "";

  const updatedAt = transportation
    ? safeDateTimeLabelJa(transportation.updatedAt, "")
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

  const disabled = !isEditing || vm.saving || vm.deleting;

  const left = (
    <div className="transportation-page__stack">
      {vm.loading ? (
        <div className="transportation-page__state">
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
            <div className="transportation-page__state">
              配送料金データを取得できませんでした。
            </div>
          )}
        </>
      ) : (
        <div className="transportation-page__state">
          配送料金設定を表示できませんでした。
        </div>
      )}
    </div>
  );

  const right = (
    <div className="page-column">
      {vm.loading ? (
        <div className="transportation-page__state">
          管理情報を読み込んでいます...
        </div>
      ) : transportation ? (
        <AdminCard
          title="管理情報"
          mode="view"
          showAssignee={false}
          createdByName={transportation.createdByName || null}
          createdAt={createdAt || null}
          updatedByName={transportation.updatedByName || null}
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
        onEdit={!isEditing ? handleEdit : undefined}
        onDelete={isEditing ? handleDelete : undefined}
        onCancel={isEditing ? handleCancel : undefined}
        onSave={isEditing ? handleSave : undefined}
        isSaving={vm.saving}
      >
        {[left, right]}
      </PageStyle>

      <Modal
        open={Boolean(vm.error)}
        title={
          <>
            <CircleAlert
              size={20}
              className="transportation-error-modal__icon"
              aria-hidden="true"
            />
            {" 配送料金設定を取得できませんでした"}
          </>
        }
        description={vm.error || undefined}
        onClose={handlers.onDismissError}
        closeLabel="閉じる"
        footer={
          <ModalButton
            variant="primary"
            onClick={handlers.onDismissError}
          >
            閉じる
          </ModalButton>
        }
      />
    </>
  );
}