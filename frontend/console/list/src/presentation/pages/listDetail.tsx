// frontend/list/src/pages/listDetail.tsx
import * as React from "react";
import PageStyle from "../../../../shell/src/layout/PageStyle/PageStyle";
import AdminCard from "../../../../admin/src/presentation/components/AdminCard";
import {
  Card,
  CardHeader,
  CardTitle,
  CardContent,
} from "../../../../shell/src/shared/ui/card";
import ProductBlueprintCard from "../../../../productBlueprint/src/presentation/components/productBlueprintCard";
import InventoryCard, {
  type InventoryRow,
} from "../../../../inventory/src/presentation/components/inventoryCard";
import TokenBlueprintCard from "../../../../tokenBlueprint/src/presentation/components/tokenBlueprintCard";
import {
  Popover,
  PopoverTrigger,
  PopoverContent,
} from "../../../../shell/src/shared/ui/popover";
import "../styles/list.css";

import { useListDetail } from "../../../../list/src/presentation/hook/useListDetail";

export default function ListDetail() {
  const vm = useListDetail();

  return (
    <PageStyle
      layout="grid-2"
      title={vm.pageTitle}
      onBack={vm.onBack}
      onSave={undefined}
    >
      {/* 左カラム：在庫選択カード + 下部カード群 */}
      <div>
        {/* 在庫選択カード（Popover導入） */}
        <Card className="inventory-select-card">
          <CardHeader className="inventory-select-header">
            <div className="inventory-select-header-left">
              <CardTitle className="inventory-select-title">在庫選択</CardTitle>

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
                <PopoverContent align="start" className="inventory-popover">
                  <div className="inventory-popover-title">在庫選択について</div>
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
              <span className="summary-value">{vm.totalSelected}</span>
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
                  {vm.items.map((item) => {
                    const isChecked = !!vm.selected[item.id];
                    const qty =
                      vm.quantities[item.id] ?? (isChecked ? 1 : 0);

                    return (
                      <tr
                        key={item.id}
                        className={isChecked ? "is-selected" : ""}
                      >
                        <td className="col-select">
                          <input
                            type="checkbox"
                            checked={isChecked}
                            onChange={() => vm.handleToggle(item.id)}
                          />
                        </td>
                        <td>
                          <div className="cell-main-text">{item.productName}</div>
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
                              vm.handleQuantityChange(item.id, e.target.value)
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
              <button className="primary-button" onClick={vm.onConfirmSelected}>
                選択在庫を確定
              </button>
            </div>
          </CardContent>
        </Card>

        {/* 下部カード群：設計〜在庫〜トークン設計（閲覧モード想定） */}
        <div className="list-detail-linked-cards">
          <ProductBlueprintCard {...({ mode: "view" } as any)} />
          <InventoryCard title="モデル別在庫一覧（参照）" rows={vm.inventoryRows as InventoryRow[]} />
          <TokenBlueprintCard {...({ mode: "view" } as any)} />
        </div>
      </div>

      {/* 右カラム：管理情報 */}
      <AdminCard
        title="管理情報"
        assigneeName={vm.admin.assigneeName}
        createdByName={vm.admin.createdByName}
        createdAt={vm.admin.createdAt}
        onEditAssignee={vm.admin.onEditAssignee}
        onClickAssignee={vm.admin.onClickAssignee}
      />
    </PageStyle>
  );
}
