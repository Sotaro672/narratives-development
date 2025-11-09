// frontend/list/src/pages/listDetail.tsx
import * as React from "react";
import { useNavigate, useParams } from "react-router-dom";
import PageStyle from "../../../../shell/src/layout/PageStyle/PageStyle";
import AdminCard from "../../../../admin/src/presentation/components/AdminCard";
import {
  Card,
  CardHeader,
  CardTitle,
  CardContent,
} from "../../../../shared/ui/card";
import ProductBlueprintCard from "../../../../productBlueprint/src/presentation/components/productBlueprintCard";
import InventoryCard, {
  type InventoryRow,
} from "../../../../inventory/src/presentation/components/inventoryCard";
import TokenBlueprintCard from "../../../../tokenBlueprint/src/presentation/components/tokenBlueprintCard";
import {
  Popover,
  PopoverTrigger,
  PopoverContent,
} from "../../../../shared/ui/popover";
import "../styles/list.css";

type InventoryItem = {
  id: string;
  sku: string;
  productName: string;
  color: string;
  size: string;
  stock: number;
};

export default function ListDetail() {
  const navigate = useNavigate();
  const { listId } = useParams<{ listId: string }>();

  // モック在庫データ（後でAPI連携に差し替え）
  const [items] = React.useState<InventoryItem[]>([
    {
      id: "inv-001",
      sku: "LUM-SS25-001-BLK-S",
      productName: "シルクブラウス プレミアムライン",
      color: "ブラック",
      size: "S",
      stock: 30,
    },
    {
      id: "inv-002",
      sku: "LUM-SS25-001-BLK-M",
      productName: "シルクブラウス プレミアムライン",
      color: "ブラック",
      size: "M",
      stock: 42,
    },
    {
      id: "inv-003",
      sku: "LUM-SS25-002-IVR-F",
      productName: "リネンシャツ リラックスフィット",
      color: "アイボリー",
      size: "F",
      stock: 18,
    },
  ]);

  // 選択状態（在庫選択カード用）
  const [selected, setSelected] = React.useState<Record<string, boolean>>({});
  const [quantities, setQuantities] = React.useState<Record<string, number>>({});

  const handleToggle = (id: string) => {
    setSelected((prev) => ({
      ...prev,
      [id]: !prev[id],
    }));
    setQuantities((prev) =>
      prev[id] != null
        ? prev
        : {
            ...prev,
            [id]: 1,
          }
    );
  };

  const handleQuantityChange = (id: string, value: string) => {
    const num = Number(value);
    if (Number.isNaN(num) || num < 0) return;
    setQuantities((prev) => ({
      ...prev,
      [id]: num,
    }));
  };

  const totalSelected = React.useMemo(
    () =>
      items.reduce((sum, item) => {
        if (!selected[item.id]) return sum;
        const q = quantities[item.id] ?? 0;
        return sum + q;
      }, 0),
    [items, selected, quantities]
  );

  // InventoryCard 用 rows: InventoryRow 定義に準拠（閲覧専用表示）
  const inventoryRows: InventoryRow[] = React.useMemo(
    () =>
      items.map((item) => ({
        modelCode: item.sku, // SKUを型番として扱う
        size: item.size,
        colorName: item.color,
        stock: item.stock,
        // colorCode は任意。必要になったらここで割り当て
      })),
    [items]
  );

  const onBack = React.useCallback(() => navigate(-1), [navigate]);

  return (
    <PageStyle
      layout="grid-2"
      title={`リスト詳細：${listId ?? "不明ID"}`}
      onBack={onBack}
      onSave={undefined}
    >
      {/* 左カラム：在庫選択カード + 下部カード群 */}
      <div>
        {/* 在庫選択カード（Popover導入） */}
        <Card className="inventory-select-card">
          <CardHeader className="inventory-select-header">
            <div className="inventory-select-header-left">
              <CardTitle className="inventory-select-title">
                在庫選択
              </CardTitle>

              {/* 仕様ヘルプ／ガイド用ポップオーバー */}
              <Popover>
                <PopoverTrigger>
                  <button
                    type="button"
                    className="inventory-popover-trigger"
                    aria-label="在庫選択ルールの説明を開く"
                  >
                    ?
                  </button>
                </PopoverTrigger>
                <PopoverContent
                  align="start"
                  className="inventory-popover"
                >
                  <div className="inventory-popover-title">
                    在庫選択について
                  </div>
                  <ul className="inventory-popover-list">
                    <li>チェックした行のみ「選択在庫」として確定されます。</li>
                    <li>選択数は在庫数以内で入力してください。</li>
                    <li>
                      下部の「設計 / 在庫 / トークン設計」カードは参照専用です。
                    </li>
                  </ul>
                </PopoverContent>
              </Popover>
            </div>

            <div className="inventory-select-summary">
              <span className="summary-label">選択合計数</span>
              <span className="summary-value">{totalSelected}</span>
            </div>
          </CardHeader>

          <CardContent className="inventory-select-body">
            <div className="inventory-select-table-wrapper">
              <table className="inventory-select-table">
                <thead>
                  <tr>
                    <th className="col-select"></th>
                    <th>商品名 / SKU</th>
                    <th>カラー</th>
                    <th>サイズ</th>
                    <th>在庫数</th>
                    <th className="col-qty">選択数</th>
                  </tr>
                </thead>
                <tbody>
                  {items.map((item) => {
                    const isChecked = !!selected[item.id];
                    const qty = quantities[item.id] ?? (isChecked ? 1 : 0);
                    return (
                      <tr
                        key={item.id}
                        className={isChecked ? "is-selected" : ""}
                      >
                        <td className="col-select">
                          <input
                            type="checkbox"
                            checked={isChecked}
                            onChange={() => handleToggle(item.id)}
                          />
                        </td>
                        <td>
                          <div className="cell-main-text">
                            {item.productName}
                          </div>
                          <div className="cell-sub-text">{item.sku}</div>
                        </td>
                        <td>{item.color}</td>
                        <td>{item.size}</td>
                        <td>{item.stock}</td>
                        <td className="col-qty">
                          <input
                            type="number"
                            min={0}
                            max={item.stock}
                            disabled={!isChecked}
                            value={qty}
                            onChange={(e) =>
                              handleQuantityChange(item.id, e.target.value)
                            }
                            className="qty-input"
                          />
                        </td>
                      </tr>
                    );
                  })}
                </tbody>
              </table>
            </div>

            <div className="inventory-select-actions">
              <button
                className="primary-button"
                onClick={() => {
                  console.log("選択在庫:", {
                    items,
                    selected,
                    quantities,
                    totalSelected,
                  });
                }}
              >
                選択在庫を確定
              </button>
            </div>
          </CardContent>
        </Card>

        {/* 下部カード群：設計〜在庫〜トークン設計（閲覧モード想定） */}
        <div className="list-detail-linked-cards">
          {/* ProductBlueprintCard / TokenBlueprintCard は閲覧専用想定 */}
          <ProductBlueprintCard
            {...({ mode: "view" } as any)}
          />
          <InventoryCard
            title="モデル別在庫一覧（参照）"
            rows={inventoryRows}
          />
          <TokenBlueprintCard
            {...({ mode: "view" } as any)}
          />
        </div>
      </div>

      {/* 右カラム：管理情報 */}
      <AdminCard
        title="管理情報"
        assigneeName={"佐藤 美咲"}
        createdByName={"山田 太郎"}
        createdAt={"2025/10/25 14:30"}
        onEditAssignee={() => console.log("edit assignee")}
        onClickAssignee={() => console.log("assignee clicked")}
        onClickCreatedBy={() => console.log("createdBy clicked")}
      />
    </PageStyle>
  );
}
