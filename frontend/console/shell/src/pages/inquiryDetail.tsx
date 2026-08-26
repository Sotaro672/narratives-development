// frontend/console/shell/src/pages/inquiryDetail.tsx

import PageStyle from "../../../shell/src/layout/PageStyle/PageStyle";
import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
} from "../../../shell/src/shared/ui/card";

import InquiryContentCard from "../features/inquiry/presentation/components/inquiryContentCard";
import InquiryInfoCard from "../features/inquiry/presentation/components/inquiryInfoCard";
import InquiryOrderInfoCard from "../features/inquiry/presentation/components/inquiryOrderInfoCard";
import InquiryReplyListCard from "../features/inquiry/presentation/components/inquiryReplyListCard";
import ReplyModal from "../features/inquiry/presentation/components/replyModal";
import { useInquiryDetailPage } from "../features/inquiry/presentation/hooks/useInquiryDetailPage";
import { useInquiryReply } from "../features/inquiry/presentation/hooks/useInquiryReply";
import {
  textOrDash,
} from "../features/inquiry/presentation/utils/inquiryDetailView";
import {
  getInquiryStatusButtonVariant,
  getInquiryStatusLabel,
  isClosedStatus,
} from "../features/inquiry/presentation/utils/inquiryStatus";
import {
  getInquiryTypeLabel,
} from "../shared/types/inquiry";

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

  const inquiry = detail?.inquiry ?? null;
  const orders = detail?.orders ?? [];

  const title =
    inquiry?.inquiryType === "product"
      ? textOrDash(inquiry.subject)
      : "";

  const status =
    getInquiryStatusLabel(inquiry?.status);

  const isUnopenedReturn =
    inquiry?.inquiryType === "return_unopened";

  const isOpenedReturn =
    inquiry?.inquiryType === "return_opened";

  const inquiryType = inquiry?.inquiryType
    ? getInquiryTypeLabel(inquiry.inquiryType)
    : "-";

  const pageTitle = (
    <div className="inq-detail__page-title">
      <span className="inq__chip">
        {inquiryType}
      </span>

      {title ? (
        <span className="inq-detail__page-title-text">
          {title}
        </span>
      ) : null}
    </div>
  );

  const statusButtonVariant =
    getInquiryStatusButtonVariant(inquiry?.status);

  const statusButtonLabel =
    isUnopenedReturn &&
    (
      inquiry?.status === "open" ||
      inquiry?.status === "in_progress"
    )
      ? "返品受領"
      : status;

  const statusButtonBusyLabel =
    isUnopenedReturn
      ? "返品処理中"
      : "更新中";

  const statusButtonDisabled =
    !detail ||
    isClosedStatus(inquiry?.status);

  const hideStatusButton =
    isOpenedReturn &&
    inquiry?.status === "open";

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
            <CardTitle>
              問い合わせ内容
            </CardTitle>
          </CardHeader>

          <CardContent>
            <div className="inq__empty">
              問い合わせ詳細を読み込み中です。
            </div>
          </CardContent>
        </Card>

        <div>
          <Card>
            <CardHeader>
              <CardTitle>
                問い合わせ情報
              </CardTitle>
            </CardHeader>

            <CardContent>
              <div className="inq__empty">
                問い合わせ情報を読み込み中です。
              </div>
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle>
                商品・注文情報
              </CardTitle>
            </CardHeader>

            <CardContent>
              <div className="inq__empty">
                商品・注文情報を読み込み中です。
              </div>
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
            <CardTitle>
              問い合わせ内容
            </CardTitle>
          </CardHeader>

          <CardContent>
            <div className="inq__empty">
              {errorMessage}
            </div>
          </CardContent>
        </Card>

        <div>
          <Card>
            <CardHeader>
              <CardTitle>
                問い合わせ情報
              </CardTitle>
            </CardHeader>

            <CardContent>
              <div className="inq__empty">
                問い合わせ情報を表示できません。
              </div>
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle>
                商品・注文情報
              </CardTitle>
            </CardHeader>

            <CardContent>
              <div className="inq__empty">
                商品・注文情報を表示できません。
              </div>
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
        onBack={onBack}
        onSave={undefined}
        statusButtonLabel={
          hideStatusButton
            ? undefined
            : statusButtonLabel
        }
        statusButtonBusyLabel={statusButtonBusyLabel}
        statusButtonVariant={statusButtonVariant}
        onStatusButtonClick={
          hideStatusButton
            ? undefined
            : onToggleStatus
        }
        isStatusButtonLoading={statusUpdating}
        statusButtonDisabled={statusButtonDisabled}
      >
        <div>
          <InquiryContentCard
            content={inquiry?.content}
            images={inquiry?.images}
            errorMessage={errorMessage}
          />

          <InquiryReplyListCard
            replies={detail?.replies ?? []}
            memberId={memberId}
            onOpenReplyModal={onOpenReplyModal}
          />
        </div>

        <div>
          <InquiryInfoCard
            userFullName={detail?.userFullName}
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