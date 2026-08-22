// frontend/console/shell/src/pages/orderDetail.tsx

import PageStyle from "../layout/PageStyle/PageStyle";
import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
} from "../shared/ui/card";
import {
  coerceRgbInt,
  rgbIntToHex,
} from "../shared/util/color";
import { safeDateTimeLabelJa } from "../shared/util/dateJa";
import {
  formatJPY,
  useOrderDetail,
} from "../features/order/presentation/hooks/useOrderDetail";
import type { OrderDetailItemDTO } from "../features/order/presentation/hooks/useOrderDetail";

function isAlcoholItem(item: OrderDetailItemDTO): boolean {
  return (
    item.kind === "alcohol" ||
    item.categoryKind === "alcohol" ||
    item.categoryCode?.startsWith("alcohol.") === true
  );
}

function getCategoryFieldValue(
  item: OrderDetailItemDTO,
  key: string,
): unknown {
  return item.categoryFields?.[key];
}

function hasDisplayValue(value: unknown): boolean {
  if (value === null || value === undefined) return false;
  if (typeof value === "string") return value.trim() !== "";
  return true;
}

function formatDisplayValue(
  value: unknown,
  unit?: string,
): string {
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
  return unit
    ? `${item.volumeValue}${unit}`
    : String(item.volumeValue);
}

export default function OrderDetail() {
  const {
    order,
    loading,
    error,
    dispatching,
    dispatchError,
    canDispatch,
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
    onDispatch,
  } = useOrderDetail();

  const left = (
    <Card className="mt-4">
      <CardHeader>
        <CardTitle>注文情報</CardTitle>
      </CardHeader>

      <CardContent>
        {dispatchError ? (
          <div className="mb-4 text-sm text-red-600 whitespace-pre-wrap text-left">
            発送処理に失敗しました: {dispatchError}
          </div>
        ) : null}

        {loading ? (
          <div className="text-sm text-muted-foreground text-left">
            読み込み中...
          </div>
        ) : error ? (
          <div className="text-sm text-red-600 whitespace-pre-wrap text-left">
            {error}
          </div>
        ) : !order ? (
          <div className="text-sm text-muted-foreground text-left">
            データがありません。
          </div>
        ) : (
          <div className="space-y-8 text-left">
            <div>
              <div className="text-sm font-semibold mb-2 text-left">
                基本情報
              </div>

              <table className="w-full text-sm text-left">
                <tbody>
                  <tr>
                    <th className="text-muted-foreground font-medium pr-4 py-2 align-top whitespace-nowrap text-left">
                      支払
                    </th>
                    <td className="py-2 text-left">
                      {order.paid ? (
                        <span className="order-badge is-paid">
                          支払済
                        </span>
                      ) : (
                        <span className="order-badge is-cancelled">
                          未払い
                        </span>
                      )}
                    </td>
                  </tr>

                  <tr>
                    <th className="text-muted-foreground font-medium pr-4 py-2 align-top whitespace-nowrap text-left">
                      注文日
                    </th>
                    <td className="py-2 text-left">{createdAt}</td>
                  </tr>

                  <tr>
                    <th className="text-muted-foreground font-medium pr-4 py-2 align-top whitespace-nowrap text-left">
                      リストID
                    </th>
                    <td className="py-2 text-left">
                      {lists.length > 0 ? (
                        <div className="flex flex-wrap gap-x-2 gap-y-1">
                          {lists.map((list) => (
                            <button
                              key={list.id}
                              type="button"
                              className="text-blue-600 hover:underline"
                              onClick={() => goListDetail(list.id)}
                            >
                              {list.readableId}
                            </button>
                          ))}
                        </div>
                      ) : (
                        "-"
                      )}
                    </td>
                  </tr>

                  <tr>
                    <th className="text-muted-foreground font-medium pr-4 py-2 align-top whitespace-nowrap text-left">
                      アイテム数
                    </th>
                    <td className="py-2 text-left">
                      {items.length} 点
                    </td>
                  </tr>

                  <tr>
                    <th className="text-muted-foreground font-medium pr-4 py-2 align-top whitespace-nowrap text-left">
                      数量合計
                    </th>
                    <td className="py-2 text-left">
                      {quantity} 点
                    </td>
                  </tr>

                  <tr>
                    <th className="text-muted-foreground font-medium pr-4 py-2 align-top whitespace-nowrap text-left">
                      商品小計
                    </th>
                    <td className="py-2 text-left">
                      {formatJPY(subtotal)}
                    </td>
                  </tr>

                  <tr>
                    <th className="text-muted-foreground font-medium pr-4 py-2 align-top whitespace-nowrap text-left">
                      配送料
                    </th>
                    <td className="py-2 text-left">
                      {formatJPY(shippingAmount)}
                    </td>
                  </tr>

                  <tr>
                    <th className="text-muted-foreground font-medium pr-4 py-2 align-top whitespace-nowrap text-left">
                      消費税
                    </th>
                    <td className="py-2 text-left">
                      {formatJPY(consumptionTax)}
                    </td>
                  </tr>

                  <tr>
                    <th className="text-muted-foreground font-medium pr-4 py-2 align-top whitespace-nowrap text-left">
                      合計金額
                    </th>
                    <td className="py-2 text-left">
                      {formatJPY(totalPrice)}
                    </td>
                  </tr>
                </tbody>
              </table>
            </div>

            <div>
              <div className="text-sm font-semibold mb-2 text-left">
                配送先
              </div>

              <table className="w-full text-sm text-left">
                <tbody>
                  <tr>
                    <th className="text-muted-foreground font-medium pr-4 py-2 align-top whitespace-nowrap text-left">
                      郵便番号
                    </th>
                    <td className="py-2 pr-6 text-left">
                      {shipping?.zipCode ?? "-"}
                    </td>

                    <th className="text-muted-foreground font-medium pr-4 py-2 align-top whitespace-nowrap text-left">
                      都道府県
                    </th>
                    <td className="py-2 pr-6 text-left">
                      {shipping?.state ?? "-"}
                    </td>

                    <th className="text-muted-foreground font-medium pr-4 py-2 align-top whitespace-nowrap text-left">
                      市町村
                    </th>
                    <td className="py-2 text-left">
                      {shipping?.city ?? "-"}
                    </td>
                  </tr>

                  <tr>
                    <th className="text-muted-foreground font-medium pr-4 py-2 align-top whitespace-nowrap text-left">
                      住所1
                    </th>
                    <td
                      className="py-2 text-left"
                      colSpan={5}
                    >
                      {shipping?.street ?? "-"}
                    </td>
                  </tr>

                  <tr>
                    <th className="text-muted-foreground font-medium pr-4 py-2 align-top whitespace-nowrap text-left">
                      住所2
                    </th>
                    <td
                      className="py-2 text-left"
                      colSpan={5}
                    >
                      {shipping?.street2 ?? "-"}
                    </td>
                  </tr>
                </tbody>
              </table>
            </div>

            <div>
              <div className="text-sm font-semibold mb-2 text-left">
                アイテム
              </div>

              {items.length === 0 ? (
                <div className="text-sm text-muted-foreground text-left">
                  アイテムがありません。
                </div>
              ) : (
                <div className="space-y-4">
                  {items.map((item, index) => {
                    const transferredAt = safeDateTimeLabelJa(
                      item.transferredAt,
                      "-",
                    );
                    const alcohol = isAlcoholItem(item);
                    const vintage = getCategoryFieldValue(
                      item,
                      "vintage",
                    );
                    const region = getCategoryFieldValue(
                      item,
                      "region",
                    );
                    const material = getCategoryFieldValue(
                      item,
                      "material",
                    );
                    const alcoholContent = getCategoryFieldValue(
                      item,
                      "alcoholContent",
                    );

                    return (
                      <Card key={index}>
                        <CardHeader className="py-3">
                          <CardTitle className="text-base text-left">
                            アイテム {index + 1}
                          </CardTitle>
                        </CardHeader>

                        <CardContent className="pt-0">
                          <table className="w-full text-sm text-left">
                            <tbody>
                              {alcohol ? (
                                <>
                                  <tr>
                                    <th className="text-muted-foreground font-medium pr-4 py-2 align-top whitespace-nowrap text-left">
                                      容量
                                    </th>
                                    <td className="py-2 text-left">
                                      {formatVolume(item)}
                                    </td>
                                  </tr>

                                  <tr>
                                    <th className="text-muted-foreground font-medium pr-4 py-2 align-top whitespace-nowrap text-left">
                                      ヴィンテージ
                                    </th>
                                    <td className="py-2 text-left">
                                      {formatDisplayValue(vintage)}
                                    </td>
                                  </tr>

                                  <tr>
                                    <th className="text-muted-foreground font-medium pr-4 py-2 align-top whitespace-nowrap text-left">
                                      地域・産地
                                    </th>
                                    <td className="py-2 text-left">
                                      {formatDisplayValue(region)}
                                    </td>
                                  </tr>

                                  <tr>
                                    <th className="text-muted-foreground font-medium pr-4 py-2 align-top whitespace-nowrap text-left">
                                      素材
                                    </th>
                                    <td className="py-2 text-left">
                                      {formatDisplayValue(material)}
                                    </td>
                                  </tr>

                                  <tr>
                                    <th className="text-muted-foreground font-medium pr-4 py-2 align-top whitespace-nowrap text-left">
                                      アルコール度数
                                    </th>
                                    <td className="py-2 text-left">
                                      {formatDisplayValue(
                                        alcoholContent,
                                        "%",
                                      )}
                                    </td>
                                  </tr>
                                </>
                              ) : (
                                <>
                                  <tr>
                                    <th className="text-muted-foreground font-medium pr-4 py-2 align-top whitespace-nowrap text-left">
                                      サイズ
                                    </th>
                                    <td className="py-2 text-left">
                                      {item.size ?? "-"}
                                    </td>
                                  </tr>

                                  <tr>
                                    <th className="text-muted-foreground font-medium pr-4 py-2 align-top whitespace-nowrap text-left">
                                      カラー
                                    </th>
                                    <td className="py-2 text-left">
                                      {(() => {
                                        const name =
                                          item.color?.trim() ?? "";
                                        const rgbInt = coerceRgbInt(
                                          item.rgb,
                                        );
                                        const hex =
                                          rgbIntToHex(rgbInt);

                                        if (!name && !hex) {
                                          return "-";
                                        }

                                        return (
                                          <div className="flex items-center gap-2">
                                            {hex ? (
                                              <span
                                                className="inline-block h-4 w-4 rounded border"
                                                style={{
                                                  backgroundColor:
                                                    hex,
                                                }}
                                                aria-label={`color ${hex}`}
                                                title={hex}
                                              />
                                            ) : null}

                                            <span>
                                              {name || "-"}
                                            </span>
                                          </div>
                                        );
                                      })()}
                                    </td>
                                  </tr>
                                </>
                              )}

                              <tr>
                                <th className="text-muted-foreground font-medium pr-4 py-2 align-top whitespace-nowrap text-left">
                                  型番
                                </th>
                                <td className="py-2 text-left">
                                  {item.modelNumber ?? "-"}
                                </td>
                              </tr>

                              <tr>
                                <th className="text-muted-foreground font-medium pr-4 py-2 align-top whitespace-nowrap text-left">
                                  商品名
                                </th>
                                <td className="py-2 text-left">
                                  {item.productName ?? "-"}
                                </td>
                              </tr>

                              <tr>
                                <th className="text-muted-foreground font-medium pr-4 py-2 align-top whitespace-nowrap text-left">
                                  トークン名
                                </th>
                                <td className="py-2 text-left">
                                  {item.tokenName ?? "-"}
                                </td>
                              </tr>

                              <tr>
                                <th className="text-muted-foreground font-medium pr-4 py-2 align-top whitespace-nowrap text-left">
                                  数量
                                </th>
                                <td className="py-2 text-left">
                                  {item.qty}
                                </td>
                              </tr>

                              <tr>
                                <th className="text-muted-foreground font-medium pr-4 py-2 align-top whitespace-nowrap text-left">
                                  金額
                                </th>
                                <td className="py-2 text-left">
                                  {formatJPY(item.price)}
                                </td>
                              </tr>

                              <tr>
                                <th className="text-muted-foreground font-medium pr-4 py-2 align-top whitespace-nowrap text-left">
                                  移譲日
                                </th>
                                <td className="py-2 text-left">
                                  {transferredAt}
                                </td>
                              </tr>
                            </tbody>
                          </table>
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
    <div className="mt-4 space-y-4 text-left">
      <Card>
        <CardHeader>
          <CardTitle className="text-left">
            購入者情報
          </CardTitle>
        </CardHeader>

        <CardContent>
          {loading ? (
            <div className="text-sm text-muted-foreground text-left">
              読み込み中...
            </div>
          ) : error ? (
            <div className="text-sm text-red-600 whitespace-pre-wrap text-left">
              {error}
            </div>
          ) : !order ? (
            <div className="text-sm text-muted-foreground text-left">
              -
            </div>
          ) : (
            <table className="w-full text-sm text-left">
              <tbody>
                <tr>
                  <th className="text-muted-foreground font-medium pr-4 py-2 align-top whitespace-nowrap text-left">
                    ユーザー名
                  </th>
                  <td className="py-2 text-left">
                    {userName}
                  </td>
                </tr>

                <tr>
                  <th className="text-muted-foreground font-medium pr-4 py-2 align-top whitespace-nowrap text-left">
                    メールアドレス
                  </th>
                  <td className="py-2 text-left">
                    {email}
                  </td>
                </tr>
              </tbody>
            </table>
          )}
        </CardContent>
      </Card>
    </div>
  );

  return (
    <PageStyle
      layout="grid-2"
      title={pageTitle}
      onBack={onBack}
      statusButtonLabel={
        canDispatch ? "発送" : "発送済"
      }
      statusButtonBusyLabel="発送中..."
      onStatusButtonClick={onDispatch}
      isStatusButtonLoading={dispatching}
      statusButtonDisabled={
        loading ||
        dispatching ||
        !canDispatch
      }
    >
      {[left, right]}
    </PageStyle>
  );
}