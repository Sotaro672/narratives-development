// frontend/production/src/pages/productionCreate.tsx
import * as React from "react";
import { useNavigate } from "react-router-dom";
import PageStyle from "../../../shell/src/layout/PageStyle/PageStyle";
import AdminCard from "../../../admin/src/presentation/components/AdminCard";
import {
  Card,
  CardHeader,
  CardTitle,
  CardContent,
} from "../../../shared/ui/card";
import { Input } from "../../../shared/ui/input";
import { Search, Package2 } from "lucide-react";
import {
  Popover,
  PopoverTrigger,
  PopoverContent,
} from "../../../shared/ui/popover";

import ColorVariationCard from "../../../model/src/pages/ColorVariationCard";
import SizeVariationCard, {
  type SizeRow,
} from "../../../model/src/pages/SizeVariationCard";
import ModelNumberCard, {
  type ModelNumber,
} from "../../../model/src/pages/ModelNumberCard";
import ProductionQuantityCard, {
  type QuantityCell,
} from "./productionQuantityCard";

import "./productionCreate.css";

/**
 * ProductionCreate
 * - 生産計画の新規作成ページ
 * - 左ペイン：
 *    - 商品設計選択カード（Popoverで選択）
 *    - ColorVariationCard（閲覧）
 *    - SizeVariationCard（閲覧）
 *    - ModelNumberCard（閲覧）
 *    - ProductionQuantityCard（編集）
 * - 右ペイン：管理情報(AdminCard)
 */

type ProductBlueprint = {
  id: string;
  name: string;
  brand: string;
  description?: string;
};

const MOCK_PRODUCT_BLUEPRINTS: ProductBlueprint[] = [
  {
    id: "PB-2025-001",
    name: "シルクブラウス プレミアムライン",
    brand: "LUMINA Fashion",
    description: "上質シルクを使用したVIP向け定番ブラウス。",
  },
  {
    id: "PB-2025-002",
    name: "カシミヤニット 限定コレクション",
    brand: "LUMINA Fashion",
    description: "冬季限定のカシミヤ100%ニットシリーズ。",
  },
  {
    id: "PB-2025-003",
    name: "リラックスフィット テーパードパンツ",
    brand: "LUMINA Casual",
    description: "日常使いに最適なストレッチ素材パンツ。",
  },
];

export default function ProductionCreate() {
  const navigate = useNavigate();

  // 選択中の商品設計ID
  const [selectedId, setSelectedId] = React.useState<string | null>(
    MOCK_PRODUCT_BLUEPRINTS[0]?.id ?? null
  );
  // Popover 内検索用キーワード
  const [keyword, setKeyword] = React.useState("");

  // カラー（閲覧用）
  const [colors] = React.useState<string[]>([
    "ホワイト",
    "ブラック",
    "ネイビー",
  ]);

  // サイズ（閲覧用）
  const [sizes] = React.useState<SizeRow[]>([
    { id: "1", sizeLabel: "S", chest: 48, waist: 58, length: 60, shoulder: 38 },
    { id: "2", sizeLabel: "M", chest: 50, waist: 60, length: 62, shoulder: 40 },
    { id: "3", sizeLabel: "L", chest: 52, waist: 62, length: 64, shoulder: 42 },
  ]);

  // モデルナンバー（閲覧用）
  const [modelNumbers] = React.useState<ModelNumber[]>([
    { size: "S", color: "ホワイト", code: "LM-SB-S-WHT" },
    { size: "M", color: "ホワイト", code: "LM-SB-M-WHT" },
    { size: "L", color: "ホワイト", code: "LM-SB-L-WHT" },
    { size: "M", color: "ブラック", code: "LM-SB-M-BLK" },
    { size: "L", color: "ブラック", code: "LM-SB-L-BLK" },
    { size: "M", color: "ネイビー", code: "LM-SB-M-NVY" },
    { size: "L", color: "ネイビー", code: "LM-SB-L-NVY" },
  ]);

  // 生産数（編集モード）
  const sizeLabels = React.useMemo(
    () => sizes.map((s) => s.sizeLabel),
    [sizes]
  );
  const [quantities, setQuantities] = React.useState<QuantityCell[]>([
    { size: "S", color: "ホワイト", qty: 10 },
    { size: "M", color: "ホワイト", qty: 20 },
    { size: "L", color: "ホワイト", qty: 30 },
    { size: "M", color: "ブラック", qty: 15 },
    { size: "L", color: "ブラック", qty: 12 },
    { size: "M", color: "ネイビー", qty: 18 },
    { size: "L", color: "ネイビー", qty: 10 },
  ]);

  const handleChangeQty = React.useCallback(
    (size: string, color: string, nextQty: number) => {
      setQuantities((prev) => {
        const idx = prev.findIndex(
          (q) => q.size === size && q.color === color
        );
        if (idx === -1) {
          return [...prev, { size, color, qty: nextQty }];
        }
        const next = prev.slice();
        next[idx] = { ...next[idx], qty: nextQty };
        return next;
      });
    },
    []
  );

  // 管理情報
  const [assignee, setAssignee] = React.useState("未設定");
  const [creator] = React.useState("現在のユーザー");
  const [createdAt] = React.useState(() =>
    new Date().toLocaleDateString("ja-JP")
  );

  const onBack = React.useCallback(() => {
    navigate(-1);
  }, [navigate]);

  // 選択中の商品設計
  const selected = React.useMemo(
    () => MOCK_PRODUCT_BLUEPRINTS.find((p) => p.id === selectedId) ?? null,
    [selectedId]
  );

  // 作成（保存）時のダミー処理
  const onCreate = React.useCallback(() => {
    if (!selectedId) {
      alert("商品設計を選択してください。");
      return;
    }

    const picked = MOCK_PRODUCT_BLUEPRINTS.find((p) => p.id === selectedId);
    console.log("生産計画作成:", {
      selectedProductBlueprintId: selectedId,
      selectedProductName: picked?.name,
      quantities,
    });

    alert(
      `商品設計「${picked?.name ?? selectedId}」を元に生産計画を作成しました（ダミー）`
    );
    navigate("/production");
  }, [navigate, selectedId, quantities]);

  // Popover内の一覧フィルタ
  const filtered = React.useMemo(() => {
    const k = keyword.trim().toLowerCase();
    if (!k) return MOCK_PRODUCT_BLUEPRINTS;
    return MOCK_PRODUCT_BLUEPRINTS.filter(
      (p) =>
        p.id.toLowerCase().includes(k) ||
        p.name.toLowerCase().includes(k) ||
        p.brand.toLowerCase().includes(k)
    );
  }, [keyword]);

  // 閲覧モード用 no-op
  const noop = () => {};
  const noopStr = (_: string) => {};

  return (
    <PageStyle
      layout="grid-2"
      title="生産計画の作成"
      onBack={onBack}
      onSave={onCreate}
    >
      {/* --- 左カラム --- */}
      <div className="space-y-4">
        {/* 商品設計選択カード（Popover利用） */}
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
              {/* トリガー：選択中の商品設計表示 */}
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

              {/* コンテンツ：検索 + 候補一覧（商品名のみ表示） */}
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
                        onClick={() => {
                          setSelectedId(p.id);
                        }}
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

        {/* カラーバリエーション（閲覧モード） */}
        <ColorVariationCard
          mode="view"
          colors={colors}
          colorInput=""
          onChangeColorInput={noopStr}
          onAddColor={noop}
          onRemoveColor={noopStr}
        />

        {/* サイズバリエーション（閲覧モード） */}
        <SizeVariationCard
          mode="view"
          sizes={sizes}
          onRemove={noop}
        />

        {/* モデルナンバー（閲覧モード） */}
        <ModelNumberCard
          mode="view"
          sizes={sizes}
          colors={colors}
          modelNumbers={modelNumbers}
        />

        {/* 生産数（編集モード） */}
        <ProductionQuantityCard
          mode="edit"
          sizes={sizeLabels}
          colors={colors}
          quantities={quantities}
          onChangeQty={handleChangeQty}
        />
      </div>

      {/* --- 右カラム：管理情報 --- */}
      <AdminCard
        title="管理情報"
        assigneeName={assignee}
        createdByName={creator}
        createdAt={createdAt}
        onEditAssignee={() => setAssignee("変更済み担当者")}
        onClickAssignee={() => console.log("Assignee clicked:", assignee)}
        onClickCreatedBy={() => console.log("CreatedBy clicked:", creator)}
      />
    </PageStyle>
  );
}
