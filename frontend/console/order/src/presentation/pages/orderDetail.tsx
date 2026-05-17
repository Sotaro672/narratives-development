// frontend/console/order/src/presentation/pages/orderDetail.tsx
import PageStyle from "../../../../shell/src/layout/PageStyle/PageStyle";
import {
  Card,
  CardHeader,
  CardTitle,
  CardContent,
} from "../../../../shell/src/shared/ui/card";

import { safeDateLabelJa } from "../../../../shell/src/shared/util/dateJa";

// RGB utility
import {
  coerceRgbInt,
  rgbIntToHex,
} from "../../../../shell/src/shared/util/color";

import {
  formatJPY,
  useOrderDetail,
  OrderDetailItemDTO,
} from "../hooks/useOrderDetail";

function isAlcoholItem(it: OrderDetailItemDTO): boolean {
  return (
    String(it.kind ?? "").trim() === "alcohol" ||
    String(it.categoryKind ?? "").trim() === "alcohol" ||
    String(it.categoryCode ?? "").trim().startsWith("alcohol.")
  );
}

function getCategoryFieldValue(
  it: OrderDetailItemDTO,
  key: string,
): unknown {
  const fields = it.categoryFields;

  if (!fields || typeof fields !== "object") {
    return undefined;
  }

  return fields[key];
}

function hasDisplayValue(value: unknown): boolean {
  if (value === null || value === undefined) return false;
  if (typeof value === "string") return value.trim() !== "";
  return true;
}

function formatDisplayValue(value: unknown, unit?: string): string {
  if (!hasDisplayValue(value)) {
    return "-";
  }

  if (Array.isArray(value)) {
    const joined = value
      .map((v) => String(v ?? "").trim())
      .filter((v) => v !== "")
      .join(", ");

    return joined || "-";
  }

  if (typeof value === "boolean") {
    return value ? "あり" : "なし";
  }

  const text = String(value);

  if (unit && text.trim() !== "") {
    return `${text}${unit}`;
  }

  return text;
}

function formatVolume(it: OrderDetailItemDTO): string {
  if (it.volumeValue === null || it.volumeValue === undefined) {
    return "-";
  }

  const unit = String(it.volumeUnit ?? "").trim();

  return unit ? `${it.volumeValue}${unit}` : String(it.volumeValue);
}

export default function OrderDetail() {
  const {
    order,
    loading,
    error,

    items,
    quantity,
    totalPrice,
    anyTransferred,
    createdAt,
    shipping,
    userName,
    avatarName,
    listIds,
    pageTitle,

    onBack,
  } = useOrderDetail();

  const left = (
    <Card className="mt-4">
      <CardHeader>
        <CardTitle>注文情報</CardTitle>
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
            データがありません。
          </div>
        ) : (
          <div className="space-y-8 text-left">
            {/* =======================
                基本情報
                ======================= */}
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
                        <span className="order-badge is-paid">支払済</span>
                      ) : (
                        <span className="order-badge is-cancelled">未払い</span>
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
                      トークン
                    </th>
                    <td className="py-2 text-left">
                      {anyTransferred ? (
                        <span className="order-badge is-transferred">
                          移譲済み
                        </span>
                      ) : (
                        <span className="order-badge is-paid">未移譲</span>
                      )}
                    </td>
                  </tr>

                  <tr>
                    <th className="text-muted-foreground font-medium pr-4 py-2 align-top whitespace-nowrap text-left">
                      アイテム数
                    </th>
                    <td className="py-2 text-left">{items.length} 点</td>
                  </tr>

                  <tr>
                    <th className="text-muted-foreground font-medium pr-4 py-2 align-top whitespace-nowrap text-left">
                      数量合計
                    </th>
                    <td className="py-2 text-left">{quantity} 点</td>
                  </tr>

                  <tr>
                    <th className="text-muted-foreground font-medium pr-4 py-2 align-top whitespace-nowrap text-left">
                      合計金額
                    </th>
                    <td className="py-2 text-left">{formatJPY(totalPrice)}</td>
                  </tr>
                </tbody>
              </table>
            </div>

            {/* =======================
                配送先
                ======================= */}
            <div>
              <div className="text-sm font-semibold mb-2 text-left">配送先</div>
              <table className="w-full text-sm text-left">
                <tbody>
                  <tr>
                    <th className="text-muted-foreground font-medium pr-4 py-2 align-top whitespace-nowrap text-left">
                      郵便番号
                    </th>
                    <td className="py-2 text-left">{shipping?.zipCode ?? "-"}</td>
                  </tr>
                  <tr>
                    <th className="text-muted-foreground font-medium pr-4 py-2 align-top whitespace-nowrap text-left">
                      都道府県
                    </th>
                    <td className="py-2 text-left">{shipping?.state ?? "-"}</td>
                  </tr>
                  <tr>
                    <th className="text-muted-foreground font-medium pr-4 py-2 align-top whitespace-nowrap text-left">
                      市区町村
                    </th>
                    <td className="py-2 text-left">{shipping?.city ?? "-"}</td>
                  </tr>
                  <tr>
                    <th className="text-muted-foreground font-medium pr-4 py-2 align-top whitespace-nowrap text-left">
                      住所1
                    </th>
                    <td className="py-2 text-left">{shipping?.street ?? "-"}</td>
                  </tr>
                  <tr>
                    <th className="text-muted-foreground font-medium pr-4 py-2 align-top whitespace-nowrap text-left">
                      住所2
                    </th>
                    <td className="py-2 text-left">{shipping?.street2 ?? "-"}</td>
                  </tr>
                </tbody>
              </table>
            </div>

            {/* =======================
                items
                ======================= */}
            <div>
              <div className="text-sm font-semibold mb-2 text-left">アイテム</div>

              {items.length === 0 ? (
                <div className="text-sm text-muted-foreground text-left">
                  アイテムがありません。
                </div>
              ) : (
                <div className="space-y-4">
                  {items.map((it: OrderDetailItemDTO, idx: number) => {
                    const transferredAt = safeDateLabelJa(it.transferredAt, "-");

                    const qty = Number(it.qty ?? 0) || 0;
                    const price = Number(it.price ?? 0) || 0;

                    const tokenLabel = it.transferred ? "移譲済" : "未移譲";
                    const alcohol = isAlcoholItem(it);

                    const vintage = getCategoryFieldValue(it, "vintage");
                    const region = getCategoryFieldValue(it, "region");
                    const material = getCategoryFieldValue(it, "material");
                    const alcoholContent = getCategoryFieldValue(
                      it,
                      "alcoholContent",
                    );

                    return (
                      <Card key={idx}>
                        <CardHeader className="py-3">
                          <CardTitle className="text-base text-left">
                            アイテム {idx + 1}
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
                                      {formatVolume(it)}
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
                                      {formatDisplayValue(alcoholContent, "%")}
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
                                      {it.size ?? "-"}
                                    </td>
                                  </tr>

                                  <tr>
                                    <th className="text-muted-foreground font-medium pr-4 py-2 align-top whitespace-nowrap text-left">
                                      カラー
                                    </th>
                                    <td className="py-2 text-left">
                                      {(() => {
                                        const name = String(
                                          it.color ?? "",
                                        ).trim();
                                        const rgbInt = coerceRgbInt(it.rgb);
                                        const hex = rgbIntToHex(rgbInt);

                                        if (!name && !hex) return "-";

                                        return (
                                          <div className="flex items-center gap-2">
                                            {hex ? (
                                              <span
                                                className="inline-block h-4 w-4 rounded border"
                                                style={{ backgroundColor: hex }}
                                                aria-label={`color ${hex}`}
                                                title={hex}
                                              />
                                            ) : null}
                                            <span>{name || "-"}</span>
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
                                  {it.modelNumber ?? "-"}
                                </td>
                              </tr>

                              <tr>
                                <th className="text-muted-foreground font-medium pr-4 py-2 align-top whitespace-nowrap text-left">
                                  商品名
                                </th>
                                <td className="py-2 text-left">
                                  {it.productName ?? "-"}
                                </td>
                              </tr>

                              <tr>
                                <th className="text-muted-foreground font-medium pr-4 py-2 align-top whitespace-nowrap text-left">
                                  トークン名
                                </th>
                                <td className="py-2 text-left">
                                  {it.tokenName ?? "-"}
                                </td>
                              </tr>

                              <tr>
                                <th className="text-muted-foreground font-medium pr-4 py-2 align-top whitespace-nowrap text-left">
                                  数量
                                </th>
                                <td className="py-2 text-left">{qty}</td>
                              </tr>

                              <tr>
                                <th className="text-muted-foreground font-medium pr-4 py-2 align-top whitespace-nowrap text-left">
                                  金額
                                </th>
                                <td className="py-2 text-left">
                                  {formatJPY(price)}
                                </td>
                              </tr>

                              <tr>
                                <th className="text-muted-foreground font-medium pr-4 py-2 align-top whitespace-nowrap text-left">
                                  トークン
                                </th>
                                <td className="py-2 text-left">
                                  {it.transferred ? (
                                    <span className="order-badge is-transferred">
                                      {tokenLabel}
                                    </span>
                                  ) : (
                                    <span className="order-badge is-paid">
                                      {tokenLabel}
                                    </span>
                                  )}
                                </td>
                              </tr>

                              <tr>
                                <th className="text-muted-foreground font-medium pr-4 py-2 align-top whitespace-nowrap text-left">
                                  移譲日
                                </th>
                                <td className="py-2 text-left">{transferredAt}</td>
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
          <CardTitle className="text-left">購入者情報</CardTitle>
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
            <div className="text-sm text-muted-foreground text-left">-</div>
          ) : (
            <table className="w-full text-sm text-left">
              <tbody>
                <tr>
                  <th className="text-muted-foreground font-medium pr-4 py-2 align-top whitespace-nowrap text-left">
                    ユーザー名
                  </th>
                  <td className="py-2 text-left">{userName}</td>
                </tr>

                <tr>
                  <th className="text-muted-foreground font-medium pr-4 py-2 align-top whitespace-nowrap text-left">
                    アバター名
                  </th>
                  <td className="py-2 text-left">{avatarName}</td>
                </tr>
              </tbody>
            </table>
          )}
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle className="text-left">出品情報</CardTitle>
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
            <div className="text-sm text-muted-foreground text-left">-</div>
          ) : (
            <table className="w-full text-sm text-left">
              <tbody>
                <tr>
                  <th className="text-muted-foreground font-medium pr-4 py-2 align-top whitespace-nowrap text-left">
                    リストID
                  </th>
                  <td className="py-2 text-left">
                    {listIds.length > 0 ? listIds.join(", ") : "-"}
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
    <PageStyle layout="grid-2" title={pageTitle} onBack={onBack}>
      {[left, right]}
    </PageStyle>
  );
}