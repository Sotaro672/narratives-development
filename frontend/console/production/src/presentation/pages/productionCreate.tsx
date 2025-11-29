// frontend/console/production/src/presentation/pages/productionCreate.tsx
import * as React from "react";
import { useNavigate } from "react-router-dom";
import PageStyle from "../../../../shell/src/layout/PageStyle/PageStyle";
import AdminCard from "../../../../admin/src/presentation/components/AdminCard";

import {
  Card,
  CardHeader,
  CardTitle,
  CardContent,
} from "../../../../shell/src/shared/ui/card";

import { Input } from "../../../../shell/src/shared/ui/input";
import { Search, Package2 } from "lucide-react";
import {
  Popover,
  PopoverTrigger,
  PopoverContent,
} from "../../../../shell/src/shared/ui/popover";

import ProductionQuantityCard, {
  type QuantityCell,
} from "../components/productionQuantityCard";

import type { SizeRow } from "../../../../model/src/domain/entity/catalog";

import ProductBlueprintCard from "../../../../productBlueprint/src/presentation/components/productBlueprintCard";

import "../styles/production.css";

export default function ProductionCreate() {
  const navigate = useNavigate();

  // ==========================
  // 商品設計選択
  // ==========================
  const [productBlueprints] = React.useState<
    { id: string; name: string; brand?: string; description?: string }[]
  >([]);

  const [selectedId, setSelectedId] = React.useState<string | null>(null);
  const [keyword, setKeyword] = React.useState("");

  // ==========================
  // サイズ・カラー・数量（API連携予定）
  // ==========================
  const [colors] = React.useState<string[]>([]);
  const [sizes] = React.useState<SizeRow[]>([]);
  const [quantities, setQuantities] = React.useState<QuantityCell[]>([]);

  const handleChangeQty = React.useCallback(
    (size: string, color: string, nextQty: number) => {
      setQuantities((prev) => {
        const idx = prev.findIndex((q) => q.size === size && q.color === color);
        if (idx === -1) return [...prev, { size, color, qty: nextQty }];
        const next = [...prev];
        next[idx] = { ...next[idx], qty: nextQty };
        return next;
      });
    },
    []
  );

  // ==========================
  // 管理情報
  // ==========================
  const [assignee, setAssignee] = React.useState("未設定");
  const [creator] = React.useState("現在のユーザー");
  const [createdAt] = React.useState(() =>
    new Date().toLocaleDateString("ja-JP")
  );

  const onBack = React.useCallback(() => {
    navigate("/production");
  }, [navigate]);

  // ==========================
  // 選択中の商品設計
  // ==========================
  const selected = React.useMemo(
    () => productBlueprints.find((p) => p.id === selectedId) ?? null,
    [selectedId, productBlueprints]
  );

  const selectedForCard: any =
    selected ??
    ({
      id: "",
      name: "",
      brand: "",
      description: "",
    } as any);

  // ==========================
  // 保存
  // ==========================
  const onCreate = React.useCallback(() => {
    if (!selectedId) {
      alert("商品設計を選択してください。");
      return;
    }

    console.log("生産計画作成:", {
      productBlueprintId: selectedId,
      colors,
      sizes,
      quantities,
    });

    alert("生産計画を作成しました（ダミー）");
    navigate("/production");
  }, [navigate, selectedId, colors, sizes, quantities]);

  // ==========================
  // 商品設計検索フィルタ
  // ==========================
  const filtered = React.useMemo(() => {
    const k = keyword.trim().toLowerCase();
    if (!k) return productBlueprints;

    return productBlueprints.filter(
      (p) =>
        p.id.toLowerCase().includes(k) ||
        p.name.toLowerCase().includes(k) ||
        p.brand?.toLowerCase().includes(k)
    );
  }, [keyword, productBlueprints]);

  const sizeLabels = React.useMemo(
    () => sizes.map((s) => s.sizeLabel),
    [sizes]
  );

  return (
    <PageStyle
      layout="grid-2"
      title="生産計画の作成"
      onBack={onBack}
      onSave={onCreate}
    >
      {/* --- 左カラム：ProductBlueprintCardのみ --- */}
      <div className="space-y-4">
        <ProductBlueprintCard {...selectedForCard} />
      </div>

      {/* --- 右カラム --- */}
      <div className="space-y-4">
        {/* 管理情報 */}
        <AdminCard
          title="管理情報"
          assigneeName={assignee}
          createdByName={creator}
          createdAt={createdAt}
          onEditAssignee={() => setAssignee("変更済み担当者")}
          onClickAssignee={() => console.log("Assignee clicked:", assignee)}
        />

        {/* 商品設計選択カード（右カラムへ移動済み） */}
        <Card className="pb-select">
          <CardHeader className="pb-select__header">
            <div className="pb-select__header-left">
              <div className="pb-select__icon-wrap">
                <Package2 className="pb-select__icon" size={16} />
              </div>
              <div className="pb-select__titles">
                <CardTitle className="pb-select__title">
                  商品設計を選択
                </CardTitle>
              </div>
            </div>
          </CardHeader>

          <CardContent className="pb-select__body">
            <Popover>
              <PopoverTrigger>
                <div className="pb-select__trigger">
                  {selected ? (
                    <div className="pb-select__trigger-title">
                      {selected.name}
                    </div>
                  ) : (
                    <div className="pb-select__trigger-placeholder">
                      商品設計を選択してください
                    </div>
                  )}
                </div>
              </PopoverTrigger>

              <PopoverContent align="start" className="pb-select__popover">
                <div className="pb-select__search">
                  <Search className="pb-select__search-icon" size={14} />
                  <Input
                    value={keyword}
                    onChange={(e) => setKeyword(e.target.value)}
                    placeholder="型番 / 商品名 / ブランドで検索"
                    className="pb-select__search-input"
                  />
                </div>

                <div className="pb-select__list">
                  {filtered.map((p) => {
                    const isActive = p.id === selectedId;
                    return (
                      <button
                        key={p.id}
                        type="button"
                        className={
                          "pb-select__row" + (isActive ? " is-active" : "")
                        }
                        onClick={() => setSelectedId(p.id)}
                      >
                        <div className="pb-select__row-title">{p.name}</div>
                      </button>
                    );
                  })}

                  {filtered.length === 0 && (
                    <div className="pb-select__empty">
                      条件に一致する商品設計がありません。
                    </div>
                  )}
                </div>
              </PopoverContent>
            </Popover>
          </CardContent>
        </Card>

        {/* 生産数（編集モード） → 右カラムへ移動済み */}
        <ProductionQuantityCard
          mode="edit"
          sizes={sizeLabels}
          colors={colors}
          quantities={quantities}
          onChangeQty={handleChangeQty}
        />
      </div>
    </PageStyle>
  );
}
