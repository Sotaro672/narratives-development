// frontend/console/shell/src/pages/orderDetail.tsx

import OrderBasicInfo from "../features/order/presentation/components/orderBasicInfo";
import OrderBuyerCard from "../features/order/presentation/components/orderBuyerCard";
import OrderItemList from "../features/order/presentation/components/orderItemList";
import OrderShippingAddress from "../features/order/presentation/components/orderShippingAddress";
import { useOrderDetail } from "../features/order/presentation/hooks/useOrderDetail";

import PageStyle from "../layout/PageStyle/PageStyle";

import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
} from "../shared/ui/card";
import Empty from "../shared/ui/empty";
import { ErrorMessage } from "../shared/ui/error";
import Text from "../shared/ui/text";

import "../styles/orderDetail.css";

export default function OrderDetail() {
  const {
    order,
    loading,
    error,
    dispatchError,
    isCancelled,
    items,
    quantity,
    subtotal,
    shippingAmount,
    consumptionTax,
    totalPrice,
    createdAt,
    shipping,
    userName,
    email,
    lists,
    pageTitle,
    statusButtonLabel,
    statusButtonBusyLabel,
    onStatusButtonClick,
    isStatusButtonLoading,
    statusButtonDisabled,
    onBack,
    goListDetail,
  } = useOrderDetail();

  const left = (
    <Card className="order-detail__main-card">
      <CardHeader>
        <CardTitle>注文情報</CardTitle>
      </CardHeader>

      <CardContent>
        {dispatchError ? (
          <ErrorMessage className="order-detail__dispatch-error">
            発送処理に失敗しました: {dispatchError}
          </ErrorMessage>
        ) : null}

        {loading ? (
          <Text as="div" tone="muted" className="order-detail__message">
            読み込み中...
          </Text>
        ) : error ? (
          <ErrorMessage className="order-detail__message">
            {error}
          </ErrorMessage>
        ) : !order ? (
          <Empty description="データがありません。" />
        ) : (
          <div className="order-detail__sections">
            <OrderBasicInfo
              createdAt={createdAt}
              lists={lists}
              itemCount={items.length}
              quantity={quantity}
              subtotal={subtotal}
              shippingAmount={shippingAmount}
              consumptionTax={consumptionTax}
              totalPrice={totalPrice}
              onListClick={goListDetail}
            />

            <OrderShippingAddress shipping={shipping} />

            <OrderItemList items={items} />
          </div>
        )}
      </CardContent>
    </Card>
  );

  const right = (
    <OrderBuyerCard
      loading={loading}
      error={error}
      orderExists={Boolean(order)}
      userName={userName}
      email={email}
    />
  );

  const cancelledStatusTab = isCancelled ? (
    <span className="page-header__btn page-header__btn--ghost">
      キャンセル済
    </span>
  ) : null;

  return (
    <PageStyle
      layout="grid-2"
      title={pageTitle}
      onBack={onBack}
      actions={cancelledStatusTab}
      statusButtonLabel={statusButtonLabel}
      statusButtonBusyLabel={statusButtonBusyLabel}
      onStatusButtonClick={onStatusButtonClick}
      isStatusButtonLoading={isStatusButtonLoading}
      statusButtonDisabled={statusButtonDisabled}
    >
      {[left, right]}
    </PageStyle>
  );
}