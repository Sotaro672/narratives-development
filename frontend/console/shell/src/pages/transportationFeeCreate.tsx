// frontend/console/shell/src/pages/transportationFeeCreate.tsx

import { CircleAlert } from "lucide-react";

import FeeEditCard from "../features/transportation/presentation/component/feeEditCard";
import IslandFeeEditCard from "../features/transportation/presentation/component/islandFeeEditCard";
import PlanNameCard from "../features/transportation/presentation/component/planNameCard";
import SinglePrefectureFeeCard from "../features/transportation/presentation/component/singlePrefectureFeeCard";
import { useTransportationFeeCreate } from "../features/transportation/presentation/hook/useTransportationFeeCreate";
import PageStyle from "../layout/PageStyle/PageStyle";
import { Button } from "../shared/ui/button";
import { Card } from "../shared/ui/card";
import Empty from "../shared/ui/empty";
import { Modal, ModalButton } from "../shared/ui/modal";

import "../styles/transportation.css";

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
            <Button
              type="button"
              variant="ghost"
              size="sm"
              disabled={disabled || !vm.isDirty}
              onClick={handlers.onReset}
            >
              リセット
            </Button>

            <Button
              type="button"
              size="sm"
              disabled={disabled || !vm.transportation}
              onClick={() => void handlers.onSave()}
              aria-busy={vm.saving}
            >
              {vm.saving ? "保存中" : "登録"}
            </Button>
          </>
        }
      >
        <div className="transportation-page__container">
          {vm.successMessage && (
            <div role="status" className="transportation-page__success">
              {vm.successMessage}
            </div>
          )}

          {vm.loading ? (
            <Card>
              <Empty
                compact
                description="配送料金設定を読み込んでいます..."
              />
            </Card>
          ) : vm.transportation ? (
            <div className="page-column">
              <PlanNameCard
                name={vm.transportation.name}
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
                <Card>
                  <Empty
                    compact
                    description="配送料金データを取得できませんでした。"
                  />
                </Card>
              )}
            </div>
          ) : (
            <Card>
              <Empty
                compact
                description="配送料金設定を表示できませんでした。"
              />
            </Card>
          )}
        </div>
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
            {" 入力内容を確認してください"}
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