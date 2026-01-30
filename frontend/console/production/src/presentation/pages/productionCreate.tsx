// frontend/console/production/src/presentation/pages/productionCreate.tsx

import PageStyle from "../../../../shell/src/layout/PageStyle/PageStyle";
import AdminCard from "../../../../admin/src/presentation/components/AdminCard";

import {
  Card,
  CardHeader,
  CardTitle,
  CardContent,
} from "../../../../shell/src/shared/ui/card";

import {
  Popover,
  PopoverTrigger,
  PopoverContent,
} from "../../../../shell/src/shared/ui/popover";

import ProductBlueprintCard from "../../../../productBlueprint/src/presentation/components/productBlueprintCard";

// Table UI
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "../../../../shell/src/shared/ui/table";

import { useProductionCreate } from "../hook/useProductionCreate";

// ★ ProductionQuantityCard（InventoryCard互換デザイン）
import ProductionQuantityCard from "../components/productionQuantityCard";

import "../styles/production.css";

// ✅ dto/detail.go を正: ProductionQuantityCard は detail の ProductionQuantityRow を受け取る
import type { ProductionQuantityRow as DetailQuantityRow } from "../../application/detail/types";

// ✅ create 側の row を detail row にアダプトするために型を参照
import type { ProductionQuantityRow as CreateQuantityRow } from "../create/types";

type ProductRow = {
  id: string;
  name: string;
};

export default function ProductionCreate() {
  const {
    onBack,
    onSave,

    hasSelectedProductBlueprint,
    selectedProductBlueprintForCard,

    // 管理カード用
    assignee,
    assigneeOptions,
    loadingMembers,
    onSelectAssignee,

    // ブランド選択用
    selectedBrand,
    brandOptions,
    selectBrand,

    // 商品設計テーブル用
    productRows,
    selectedProductId,
    selectProductById,

    // ProductionQuantityCard 用（create 側の row 形状）
    modelVariationsForCard,
    setQuantityRows,
  } = useProductionCreate();

  // ✅ create row -> detail row 変換（ProductionQuantityCard の props に合わせる）
  // 重要: displayOrder を絶対に潰さない（ここが今回の根本原因）
  const rowsForCard: DetailQuantityRow[] = (Array.isArray(modelVariationsForCard)
    ? modelVariationsForCard
    : []
  ).map((r: CreateQuantityRow, index: number) => {
    // modelId と modelVariationId が同一IDである前提（ログでも一致）
    const id =
      (r as any).modelId?.trim?.() ||
      (r.modelVariationId ?? "").trim() ||
      String(index);

    return {
      modelId: id,
      modelNumber: r.modelNumber ?? "",
      size: r.size ?? "",
      color: r.color ?? "",
      rgb: (r as any).rgb ?? null,
      quantity: r.quantity ?? 0,
      // ✅ create 側に入っている displayOrder をそのまま渡す（潰さない）
      displayOrder: (r as any).displayOrder,
    };
  });

  // ✅ デバッグ: Card に渡す直前の displayOrder
  console.log(
    "[debug] rows just before card",
    rowsForCard.map((r) => r.displayOrder),
  );

  // ✅ onChangeRows: detail row -> create row に戻して state 更新
  const handleChangeRows = (rows: DetailQuantityRow[]) => {
    const next: CreateQuantityRow[] = (Array.isArray(rows) ? rows : []).map(
      (r, index) => {
        const id = (r.modelId ?? "").trim() || String(index);

        return {
          // ✅ TS2322 対策: create row が modelId 必須なら必ず入れる
          ...(typeof ({} as any as CreateQuantityRow).modelId === "string"
            ? ({ modelId: id } as any)
            : ({} as any)),
          modelVariationId: id,
          modelNumber: r.modelNumber,
          size: r.size,
          color: r.color,
          rgb: r.rgb ?? null,
          quantity: r.quantity ?? 0,
          // ✅ displayOrder を保持したいなら create row 側にも持たせる（型にあれば入る）
          ...(typeof ({} as any as CreateQuantityRow).displayOrder === "number"
            ? ({ displayOrder: r.displayOrder } as any)
            : ({} as any)),
        } as CreateQuantityRow;
      },
    );

    setQuantityRows(next);
  };

  return (
    <PageStyle
      layout="grid-2"
      title="生産計画の作成"
      onBack={onBack}
      onSave={onSave}
    >
      {/* ========== 左カラム ========== */}
      <div className="space-y-4">
        {/* 商品設計カード */}
        {hasSelectedProductBlueprint ? (
          <ProductBlueprintCard mode="view" {...selectedProductBlueprintForCard} />
        ) : (
          <div className="flex h-full items-center justify-center text-gray-500">
            商品設計を選択してください
          </div>
        )}

        {/* ★★★ ProductionQuantityCard（編集モード） ★★★ */}
        {hasSelectedProductBlueprint && (
          <ProductionQuantityCard
            title="モデル別 生産数一覧"
            rows={rowsForCard}
            mode="edit"
            onChangeRows={handleChangeRows}
          />
        )}
      </div>

      {/* ========== 右カラム ========== */}
      <div className="space-y-4">
        {/* 管理情報カード */}
        <AdminCard
          mode="edit"
          title="管理情報"
          assigneeName={assignee}
          assigneeCandidates={assigneeOptions}
          loadingMembers={loadingMembers}
          onSelectAssignee={onSelectAssignee}
        />

        {/* ブランド選択 */}
        <Card className="pb-select">
          <CardHeader>
            <CardTitle>ブランド選択</CardTitle>
          </CardHeader>
          <CardContent>
            <Popover>
              <PopoverTrigger>
                <div className="pb-select__trigger">
                  {selectedBrand || "ブランドを選択してください"}
                </div>
              </PopoverTrigger>

              <PopoverContent>
                <div className="pb-select__list">
                  {brandOptions.map((b: string) => (
                    <button
                      key={b}
                      className={
                        "pb-select__row" + (selectedBrand === b ? " is-active" : "")
                      }
                      onClick={() => selectBrand(b)}
                    >
                      {b}
                    </button>
                  ))}

                  {brandOptions.length === 0 && (
                    <div className="pb-select__empty">
                      ブランドが登録されていません。
                    </div>
                  )}
                </div>
              </PopoverContent>
            </Popover>
          </CardContent>
        </Card>

        {/* 商品設計一覧テーブル */}
        <Card>
          <CardHeader>
            <CardTitle>商品設計一覧</CardTitle>
          </CardHeader>
          <CardContent>
            <Table className="border rounded">
              <TableHeader>
                <TableRow>
                  <TableHead>商品名</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {productRows.map((p: ProductRow) => (
                  <TableRow
                    key={p.id}
                    className={
                      "cursor-pointer hover:bg-blue-50" +
                      (selectedProductId === p.id ? " bg-blue-100" : "")
                    }
                    onClick={() => selectProductById(p.id)}
                  >
                    <TableCell>{p.name}</TableCell>
                  </TableRow>
                ))}

                {productRows.length === 0 && (
                  <TableRow>
                    <TableCell className="text-center text-gray-500">
                      対象の商品設計がありません
                    </TableCell>
                  </TableRow>
                )}
              </TableBody>
            </Table>
          </CardContent>
        </Card>
      </div>
    </PageStyle>
  );
}
