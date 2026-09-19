// frontend/console/shell/src/pages/orderDetail.tsx

import { formatJPY, useOrderDetail } from "../features/order/presentation/hooks/useOrderDetail";
import type { OrderDetailItemDTO } from "../features/order/presentation/hooks/useOrderDetail";
import PageStyle from "../layout/PageStyle/PageStyle";
import { Card, CardContent, CardHeader, CardTitle } from "../shared/ui/card";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableRow,
} from "../shared/ui/table";
import Text from "../shared/ui/text";
import { getOrderStatusLabel } from "../shared/types/order";
import { coerceRgbInt, rgbIntToHex } from "../shared/util/color";
import { safeDateTimeLabelJa } from "../shared/util/dateJa";

import "../styles/orderDetail.css";

function isAlcoholItem(item: OrderDetailItemDTO): boolean {
  return item.productBlueprintCategoryPath[0] === "alcohol";
}

function getCategoryFieldValue(item: OrderDetailItemDTO, key: string): unknown {
  return item.categoryFields?.[key];
}

function hasDisplayValue(value: unknown): boolean {
  if (value === null || value === undefined) return false;
  if (typeof value === "string") return value.trim() !== "";
  return true;
}

function formatDisplayValue(value: unknown, unit?: string): string {
  if (!hasDisplayValue(value)) return "-";

  if (Array.isArray(value)) {
    const joined = value
      .map((item) => String(item ?? "").trim())
      .filter(Boolean)
      .join(", ");
    return joined || "-";
  }

  if (typeof value === "boolean") {
    return value ? "あり" : "なし";
  }

  const text = String(value);
  return unit ? `${text}${unit}` : text;
}

function formatVolume(item: OrderDetailItemDTO): string {
  if (item.volumeValue === undefined) return "-";

  const unit = item.volumeUnit?.trim();
  return unit ? `${item.volumeValue}${unit}` : String(item.volumeValue);
}

export default function OrderDetail() {
  const {
    order,
    loading,
    error,
    dispatching,
    dispatchError,
    canDispatch,
    hasReturnInProgress,
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
    onBack,
    goListDetail,
    goReturnInquiryDetail,
    onDispatch,
  } = useOrderDetail();

  const isCancelled =
    items.length > 0 &&
    items.every((item) => item.isCancelled);

  const left = (
    <Card className="order-detail__main-card">
      <CardHeader>
        <CardTitle>注文情報</CardTitle>
      </CardHeader>

      <CardContent>
        {dispatchError ? (
          <Text
            as="div"
            tone="destructive"
            wrap="pre-wrap"
            className="order-detail__dispatch-error"
            role="alert"
          >
            発送処理に失敗しました: {dispatchError}
          </Text>
        ) : null}

        {loading ? (
          <Text as="div" tone="muted" className="order-detail__message">
            読み込み中...
          </Text>
        ) : error ? (
          <Text
            as="div"
            tone="destructive"
            wrap="pre-wrap"
            className="order-detail__message"
            role="alert"
          >
            {error}
          </Text>
        ) : !order ? (
          <Text as="div" tone="muted" className="order-detail__message">
            データがありません。
          </Text>
        ) : (
          <div className="order-detail__sections">
            <div>
              <Text
                as="div"
                weight="semibold"
                className="order-detail__section-title"
              >
                基本情報
              </Text>

              <Table className="order-detail__table">
                <TableBody>
                  <TableRow>
                    <TableHead scope="row" className="order-detail__label-cell">
                      注文日
                    </TableHead>
                    <TableCell className="order-detail__value-cell">
                      {createdAt}
                    </TableCell>
                  </TableRow>

                  <TableRow>
                    <TableHead scope="row" className="order-detail__label-cell">
                      リストID
                    </TableHead>
                    <TableCell className="order-detail__value-cell">
                      {lists.length > 0 ? (
                        <div className="order-detail__list-links">
                          {lists.map((list) => (
                            <button
                              key={list.id}
                              type="button"
                              className="order-detail__list-link"
                              onClick={() => goListDetail(list.id)}
                            >
                              {list.readableId}
                            </button>
                          ))}
                        </div>
                      ) : (
                        "-"
                      )}
                    </TableCell>
                  </TableRow>

                  <TableRow>
                    <TableHead scope="row" className="order-detail__label-cell">
                      アイテム数
                    </TableHead>
                    <TableCell className="order-detail__value-cell">
                      {items.length} 点
                    </TableCell>
                  </TableRow>

                  <TableRow>
                    <TableHead scope="row" className="order-detail__label-cell">
                      数量合計
                    </TableHead>
                    <TableCell className="order-detail__value-cell">
                      {quantity} 点
                    </TableCell>
                  </TableRow>

                  <TableRow>
                    <TableHead scope="row" className="order-detail__label-cell">
                      商品小計
                    </TableHead>
                    <TableCell className="order-detail__value-cell">
                      {formatJPY(subtotal)}
                    </TableCell>
                  </TableRow>

                  <TableRow>
                    <TableHead scope="row" className="order-detail__label-cell">
                      配送料
                    </TableHead>
                    <TableCell className="order-detail__value-cell">
                      {formatJPY(shippingAmount)}
                    </TableCell>
                  </TableRow>

                  <TableRow>
                    <TableHead scope="row" className="order-detail__label-cell">
                      消費税
                    </TableHead>
                    <TableCell className="order-detail__value-cell">
                      {formatJPY(consumptionTax)}
                    </TableCell>
                  </TableRow>

                  <TableRow>
                    <TableHead scope="row" className="order-detail__label-cell">
                      合計金額
                    </TableHead>
                    <TableCell className="order-detail__value-cell">
                      {formatJPY(totalPrice)}
                    </TableCell>
                  </TableRow>
                </TableBody>
              </Table>
            </div>

            <div>
              <Text
                as="div"
                weight="semibold"
                className="order-detail__section-title"
              >
                配送先
              </Text>

              <Table className="order-detail__table">
                <TableBody>
                  <TableRow>
                    <TableHead scope="row" className="order-detail__label-cell">
                      郵便番号
                    </TableHead>
                    <TableCell className="order-detail__value-cell order-detail__value-cell--spaced">
                      {shipping?.zipCode ?? "-"}
                    </TableCell>

                    <TableHead scope="row" className="order-detail__label-cell">
                      都道府県
                    </TableHead>
                    <TableCell className="order-detail__value-cell order-detail__value-cell--spaced">
                      {shipping?.state ?? "-"}
                    </TableCell>

                    <TableHead scope="row" className="order-detail__label-cell">
                      市町村
                    </TableHead>
                    <TableCell className="order-detail__value-cell">
                      {shipping?.city ?? "-"}
                    </TableCell>
                  </TableRow>

                  <TableRow>
                    <TableHead scope="row" className="order-detail__label-cell">
                      住所1
                    </TableHead>
                    <TableCell className="order-detail__value-cell" colSpan={5}>
                      {shipping?.street ?? "-"}
                    </TableCell>
                  </TableRow>

                  <TableRow>
                    <TableHead scope="row" className="order-detail__label-cell">
                      住所2
                    </TableHead>
                    <TableCell className="order-detail__value-cell" colSpan={5}>
                      {shipping?.street2 ?? "-"}
                    </TableCell>
                  </TableRow>
                </TableBody>
              </Table>
            </div>

            <div>
              <Text
                as="div"
                weight="semibold"
                className="order-detail__section-title"
              >
                アイテム
              </Text>

              {items.length === 0 ? (
                <Text as="div" tone="muted" className="order-detail__message">
                  アイテムがありません。
                </Text>
              ) : (
                <div className="order-detail__items">
                  {items.map((item, index) => {
                    const transferredAt = safeDateTimeLabelJa(
                      item.transferredAt,
                      "-",
                    );
                    const alcohol = isAlcoholItem(item);
                    const vintage = getCategoryFieldValue(item, "vintage");
                    const region = getCategoryFieldValue(item, "region");
                    const material = getCategoryFieldValue(item, "material");
                    const alcoholContent = getCategoryFieldValue(
                      item,
                      "alcoholContent",
                    );

                    return (
                      <Card key={index}>
                        <CardHeader className="order-detail__item-header">
                          <CardTitle className="order-detail__item-title">
                            アイテム {index + 1}
                          </CardTitle>
                        </CardHeader>

                        <CardContent className="order-detail__item-content">
                          <Table className="order-detail__table">
                            <TableBody>
                              {alcohol ? (
                                <>
                                  <TableRow>
                                    <TableHead
                                      scope="row"
                                      className="order-detail__label-cell"
                                    >
                                      容量
                                    </TableHead>
                                    <TableCell className="order-detail__value-cell">
                                      {formatVolume(item)}
                                    </TableCell>
                                  </TableRow>

                                  <TableRow>
                                    <TableHead
                                      scope="row"
                                      className="order-detail__label-cell"
                                    >
                                      ヴィンテージ
                                    </TableHead>
                                    <TableCell className="order-detail__value-cell">
                                      {formatDisplayValue(vintage)}
                                    </TableCell>
                                  </TableRow>

                                  <TableRow>
                                    <TableHead
                                      scope="row"
                                      className="order-detail__label-cell"
                                    >
                                      地域・産地
                                    </TableHead>
                                    <TableCell className="order-detail__value-cell">
                                      {formatDisplayValue(region)}
                                    </TableCell>
                                  </TableRow>

                                  <TableRow>
                                    <TableHead
                                      scope="row"
                                      className="order-detail__label-cell"
                                    >
                                      素材
                                    </TableHead>
                                    <TableCell className="order-detail__value-cell">
                                      {formatDisplayValue(material)}
                                    </TableCell>
                                  </TableRow>

                                  <TableRow>
                                    <TableHead
                                      scope="row"
                                      className="order-detail__label-cell"
                                    >
                                      アルコール度数
                                    </TableHead>
                                    <TableCell className="order-detail__value-cell">
                                      {formatDisplayValue(alcoholContent, "%")}
                                    </TableCell>
                                  </TableRow>
                                </>
                              ) : (
                                <>
                                  <TableRow>
                                    <TableHead
                                      scope="row"
                                      className="order-detail__label-cell"
                                    >
                                      サイズ
                                    </TableHead>
                                    <TableCell className="order-detail__value-cell">
                                      {item.size ?? "-"}
                                    </TableCell>
                                  </TableRow>

                                  <TableRow>
                                    <TableHead
                                      scope="row"
                                      className="order-detail__label-cell"
                                    >
                                      カラー
                                    </TableHead>
                                    <TableCell className="order-detail__value-cell">
                                      {(() => {
                                        const name = item.color?.trim() ?? "";
                                        const rgbInt = coerceRgbInt(item.rgb);
                                        const hex = rgbIntToHex(rgbInt);

                                        if (!name && !hex) {
                                          return "-";
                                        }

                                        return (
                                          <div className="order-detail__color">
                                            {hex ? (
                                              <span
                                                className="order-detail__color-swatch"
                                                style={{ backgroundColor: hex }}
                                                aria-label={`color ${hex}`}
                                                title={hex}
                                              />
                                            ) : null}
                                            <span>{name || "-"}</span>
                                          </div>
                                        );
                                      })()}
                                    </TableCell>
                                  </TableRow>
                                </>
                              )}

                              <TableRow>
                                <TableHead
                                  scope="row"
                                  className="order-detail__label-cell"
                                >
                                  型番
                                </TableHead>
                                <TableCell className="order-detail__value-cell">
                                  {item.modelNumber ?? "-"}
                                </TableCell>
                              </TableRow>

                              <TableRow>
                                <TableHead
                                  scope="row"
                                  className="order-detail__label-cell"
                                >
                                  商品名
                                </TableHead>
                                <TableCell className="order-detail__value-cell">
                                  {item.productName ?? "-"}
                                </TableCell>
                              </TableRow>

                              <TableRow>
                                <TableHead
                                  scope="row"
                                  className="order-detail__label-cell"
                                >
                                  トークン名
                                </TableHead>
                                <TableCell className="order-detail__value-cell">
                                  {item.tokenName ?? "-"}
                                </TableCell>
                              </TableRow>

                              <TableRow>
                                <TableHead
                                  scope="row"
                                  className="order-detail__label-cell"
                                >
                                  数量
                                </TableHead>
                                <TableCell className="order-detail__value-cell">
                                  {item.qty}
                                </TableCell>
                              </TableRow>

                              <TableRow>
                                <TableHead
                                  scope="row"
                                  className="order-detail__label-cell"
                                >
                                  金額
                                </TableHead>
                                <TableCell className="order-detail__value-cell">
                                  {formatJPY(item.price)}
                                </TableCell>
                              </TableRow>

                              <TableRow>
                                <TableHead
                                  scope="row"
                                  className="order-detail__label-cell"
                                >
                                  移譲日
                                </TableHead>
                                <TableCell className="order-detail__value-cell">
                                  {transferredAt}
                                </TableCell>
                              </TableRow>
                            </TableBody>
                          </Table>
                        </CardContent>
                      </Card>
                    );
                  })}
                </div>
              )}
            </div>
          </div>
        )}
      </CardContent>
    </Card>
  );

  const right = (
    <div className="page-column page-column--offset-top sticky-aside">
      <Card>
        <CardHeader>
          <CardTitle className="order-detail__card-title">
            購入者情報
          </CardTitle>
        </CardHeader>

        <CardContent>
          {loading ? (
            <Text as="div" tone="muted" className="order-detail__message">
              読み込み中...
            </Text>
          ) : error ? (
            <Text
              as="div"
              tone="destructive"
              wrap="pre-wrap"
              className="order-detail__message"
              role="alert"
            >
              {error}
            </Text>
          ) : !order ? (
            <Text as="div" tone="muted" className="order-detail__message">
              -
            </Text>
          ) : (
            <Table className="order-detail__table">
              <TableBody>
                <TableRow>
                  <TableHead scope="row" className="order-detail__label-cell">
                    ユーザー名
                  </TableHead>
                  <TableCell className="order-detail__value-cell">
                    {userName}
                  </TableCell>
                </TableRow>

                <TableRow>
                  <TableHead scope="row" className="order-detail__label-cell">
                    メールアドレス
                  </TableHead>
                  <TableCell className="order-detail__value-cell">
                    {email}
                  </TableCell>
                </TableRow>
              </TableBody>
            </Table>
          )}
        </CardContent>
      </Card>
    </div>
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
      statusButtonLabel={
        isCancelled
          ? undefined
          : hasReturnInProgress
            ? "返品対応"
            : canDispatch
              ? "発送"
              : getOrderStatusLabel(order?.paid ?? true)
      }
      statusButtonBusyLabel="発送中..."
      onStatusButtonClick={
        isCancelled
          ? undefined
          : hasReturnInProgress
            ? goReturnInquiryDetail
            : onDispatch
      }
      isStatusButtonLoading={!hasReturnInProgress && dispatching}
      statusButtonDisabled={
        loading ||
        (
          !hasReturnInProgress &&
          (
            dispatching ||
            !canDispatch
          )
        )
      }
    >
      {[left, right]}
    </PageStyle>
  );
}