// frontend/console/shell/src/pages/inquiryDetail.tsx

import PageStyle from "../../../shell/src/layout/PageStyle/PageStyle";
import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
} from "../../../shell/src/shared/ui/card";
import Empty from "../../../shell/src/shared/ui/empty";

import InquiryContentCard from "../features/inquiry/presentation/components/inquiryContentCard";
import InquiryInfoCard from "../features/inquiry/presentation/components/inquiryInfoCard";
import InquiryOrderInfoCard from "../features/inquiry/presentation/components/inquiryOrderInfoCard";
import InquiryReplyListCard from "../features/inquiry/presentation/components/inquiryReplyListCard";
import ReplyModal from "../features/inquiry/presentation/components/replyModal";
import { useInquiryDetailPage } from "../features/inquiry/presentation/hooks/useInquiryDetailPage";
import { useInquiryReply } from "../features/inquiry/presentation/hooks/useInquiryReply";
import { useOpenedReturnRefund } from "../features/inquiry/presentation/hooks/useOpenedReturnRefund";
import { textOrDash } from "../features/inquiry/presentation/utils/inquiryDetailView";
import {
  getInquiryStatusButtonVariant,
  getInquiryStatusLabel,
  isClosedStatus,
} from "../features/inquiry/presentation/utils/inquiryStatus";
import { getInquiryTypeLabel } from "../shared/types/inquiry";

import "../styles/inquiry-page.css";

export default function InquiryDetail() {
  const {
    inquiryId,
    memberId,
    detail,
    loading,
    statusUpdating,
    errorMessage,
    onBack,
    reloadDetail,
    clearErrorMessage,
    onToggleStatus,
  } = useInquiryDetailPage();

  const inquiry = detail?.inquiry ?? null;
  const orders = detail?.orders ?? [];

  const isUnopenedReturn =
    inquiry?.inquiryType === "return_unopened";
  const isOpenedReturn =
    inquiry?.inquiryType === "return_opened";
  const isReturnInquiry =
    isUnopenedReturn || isOpenedReturn;

  // Hook は常に同じ順序で呼び出す必要があるため、詳細取得前または
  // 商品問い合わせの場合は return_opened を仮値として渡す。
  // 実際に返品処理UIを表示・実行するのは return_unopened /
  // return_opened の Inquiry のみ。
  const returnInquiryType =
    inquiry?.inquiryType === "return_unopened"
      ? "return_unopened"
      : "return_opened";

  // Return Inquiry は Inquiry.OrderID + Inquiry.OrderItemIndex を正として
  // 対象 Order item を特定する。
  //
  // merchandiseRefundMaxAmount は backend が Order snapshot と配賦済みの
  // 消費税から算出した税込商品返金上限を正とする。
  const returnOrder =
    isReturnInquiry && inquiry?.orderId
      ? orders.find((order) => order.id === inquiry.orderId) ?? null
      : null;

  const returnOrderItem =
    returnOrder &&
    typeof inquiry?.orderItemIndex === "number"
      ? returnOrder.items.find(
          (item) => item.itemIndex === inquiry.orderItemIndex,
        ) ?? null
      : null;

  const merchandiseRefundMaxAmount =
    returnOrderItem?.merchandiseRefundMaxAmount ?? 0;

  const {
    replyModalOpen,
    replyContent,
    replyImages,
    replySubmitting,
    replyErrorMessage,
    onOpenReplyModal,
    onCloseReplyModal,
    onChangeReplyContent,
    onChangeReplyImages,
    onRemoveReplyImage,
    onSubmitReply,
  } = useInquiryReply({
    inquiryId,
    memberId,
    onReloadDetail: reloadDetail,
    onClearPageError: clearErrorMessage,
  });

  const {
    merchandiseRefundAmount,
    refundOutboundShipping,
    coverReturnShipping,
    submitting: returnRefundSubmitting,
    errorMessage: returnRefundErrorMessage,
    selectionLocked: returnRefundSelectionLocked,
    canSubmit: returnRefundCanSubmit,
    onChangeMerchandiseRefundAmount,
    onChangeRefundOutboundShipping,
    onChangeCoverReturnShipping,
    onSubmit: onSubmitReturnRefund,
  } = useOpenedReturnRefund({
    inquiryId,
    inquiryType: returnInquiryType,
    merchandiseRefundMaxAmount,
    onReloadDetail: reloadDetail,
    onClearPageError: clearErrorMessage,
  });

  const title =
    inquiry?.inquiryType === "product"
      ? textOrDash(inquiry.subject)
      : "";

  const status =
    getInquiryStatusLabel(inquiry?.status);

  const isResolved =
    inquiry?.status === "resolved";

  const isOpenOrInProgress =
    inquiry?.status === "open" ||
    inquiry?.status === "in_progress";

  const inquiryType = inquiry?.inquiryType
    ? getInquiryTypeLabel(inquiry.inquiryType)
    : "-";

  const pageTitle = (
    <div className="inq-detail__page-title">
      <span className="inq__chip">{inquiryType}</span>

      {title ? (
        <span className="inq-detail__page-title-text">
          {title}
        </span>
      ) : null}
    </div>
  );

  const statusTab = (
    <span
      className={[
        "inq-status-tab",
        inquiry?.status
          ? `inq-status-tab--${inquiry.status}`
          : "",
      ]
        .filter(Boolean)
        .join(" ")}
    >
      {status}
    </span>
  );

  const statusButtonVariant =
    getInquiryStatusButtonVariant(inquiry?.status);

  const hideStatusButton =
    isClosedStatus(inquiry?.status) ||
    (isReturnInquiry && isOpenOrInProgress);

  const statusButtonLabel = hideStatusButton
    ? undefined
    : isResolved
      ? "再対応する"
      : isOpenOrInProgress
        ? "対応済みにする"
        : undefined;

  const statusButtonBusyLabel = "更新中";

  const showReturnRefund =
    isReturnInquiry &&
    isOpenOrInProgress &&
    !isClosedStatus(inquiry?.status);

  if (loading) {
    return (
      <PageStyle
        layout="grid-2"
        title="問い合わせ詳細"
        onBack={onBack}
        onSave={undefined}
      >
        <Card>
          <CardHeader>
            <CardTitle>問い合わせ内容</CardTitle>
          </CardHeader>

          <CardContent>
            <Empty
              compact
              description="問い合わせ詳細を読み込み中です。"
            />
          </CardContent>
        </Card>

        <div>
          <Card>
            <CardHeader>
              <CardTitle>問い合わせ情報</CardTitle>
            </CardHeader>

            <CardContent>
              <Empty
                compact
                description="問い合わせ情報を読み込み中です。"
              />
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle>商品・注文情報</CardTitle>
            </CardHeader>

            <CardContent>
              <Empty
                compact
                description="商品・注文情報を読み込み中です。"
              />
            </CardContent>
          </Card>
        </div>
      </PageStyle>
    );
  }

  if (errorMessage && !detail) {
    return (
      <PageStyle
        layout="grid-2"
        title="問い合わせ詳細"
        onBack={onBack}
        onSave={undefined}
      >
        <Card>
          <CardHeader>
            <CardTitle>問い合わせ内容</CardTitle>
          </CardHeader>

          <CardContent>
            <Empty description={errorMessage} />
          </CardContent>
        </Card>

        <div>
          <Card>
            <CardHeader>
              <CardTitle>問い合わせ情報</CardTitle>
            </CardHeader>

            <CardContent>
              <Empty description="問い合わせ情報を表示できません。" />
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle>商品・注文情報</CardTitle>
            </CardHeader>

            <CardContent>
              <Empty description="商品・注文情報を表示できません。" />
            </CardContent>
          </Card>
        </div>
      </PageStyle>
    );
  }

  return (
    <>
      <PageStyle
        layout="grid-2"
        title={pageTitle}
        badge={statusTab}
        onBack={onBack}
        onSave={undefined}
        statusButtonLabel={statusButtonLabel}
        statusButtonBusyLabel={statusButtonBusyLabel}
        statusButtonVariant={statusButtonVariant}
        onStatusButtonClick={
          statusButtonLabel
            ? onToggleStatus
            : undefined
        }
        isStatusButtonLoading={statusUpdating}
        statusButtonDisabled={
          !detail ||
          isClosedStatus(inquiry?.status)
        }
      >
        <div>
          <InquiryContentCard
            content={inquiry?.content}
            images={inquiry?.images}
            errorMessage={errorMessage}
            showReturnRefund={showReturnRefund}
            merchandiseRefundAmount={merchandiseRefundAmount}
            merchandiseRefundMaxAmount={merchandiseRefundMaxAmount}
            refundOutboundShipping={refundOutboundShipping}
            coverReturnShipping={coverReturnShipping}
            returnRefundSubmitting={returnRefundSubmitting}
            returnRefundSelectionLocked={returnRefundSelectionLocked}
            returnRefundCanSubmit={returnRefundCanSubmit}
            returnRefundErrorMessage={returnRefundErrorMessage}
            onChangeMerchandiseRefundAmount={
              onChangeMerchandiseRefundAmount
            }
            onChangeRefundOutboundShipping={
              onChangeRefundOutboundShipping
            }
            onChangeCoverReturnShipping={
              onChangeCoverReturnShipping
            }
            onSubmitReturnRefund={onSubmitReturnRefund}
          />

          <InquiryReplyListCard
            replies={detail?.replies ?? []}
            memberId={memberId}
            brandName={detail?.brandName ?? ""}
            brandIcon={detail?.brandIcon ?? ""}
            userName={detail?.userName ?? ""}
            onOpenReplyModal={onOpenReplyModal}
          />
        </div>

        <div>
          <InquiryInfoCard
            userName={detail?.userName}
            createdAt={inquiry?.createdAt}
            updatedAt={inquiry?.updatedAt}
          />

          <InquiryOrderInfoCard
            productName={detail?.productName}
            brandName={detail?.brandName}
            orders={orders}
            isUnopenedReturn={isUnopenedReturn}
          />
        </div>
      </PageStyle>

      <ReplyModal
        open={replyModalOpen}
        content={replyContent}
        images={replyImages}
        submitting={replySubmitting}
        errorMessage={replyErrorMessage}
        onClose={onCloseReplyModal}
        onChangeContent={onChangeReplyContent}
        onChangeImages={onChangeReplyImages}
        onRemoveImage={onRemoveReplyImage}
        onSubmit={() => void onSubmitReply()}
      />
    </>
  );
}