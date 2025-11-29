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

import "../styles/production.css";

export default function ProductionCreate() {
  const {
    onBack,
    onSave,
    hasSelectedProductBlueprint,
    selectedProductBlueprintForCard,
    assignee,
    creator,
    createdAt,
    setAssignee,
    selectedBrand,
    brandOptions,
    selectBrand,
    productRows,
    selectedProductId,
    selectProductById,
  } = useProductionCreate();

  return (
    <PageStyle
      layout="grid-2"
      title="生産計画の作成"
      onBack={onBack}
      onSave={onSave}
    >
      {/* ========== 左カラム ========== */}
      <div className="space-y-4">
        {hasSelectedProductBlueprint ? (
          <ProductBlueprintCard {...selectedProductBlueprintForCard} />
        ) : (
          <div className="flex h-full items-center justify-center text-gray-500">
            商品設計を選択してください
          </div>
        )}
      </div>

      {/* ========== 右カラム ========== */}
      <div className="space-y-4">
        <AdminCard
          title="管理情報"
          assigneeName={assignee}
          createdByName={creator}
          createdAt={createdAt}
          onEditAssignee={() => setAssignee("変更済み担当者")}
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

                  {/* ← 「すべてのブランド」削除済み */}

                  {brandOptions.map((b) => (
                    <button
                      key={b}
                      className={
                        "pb-select__row" +
                        (selectedBrand === b ? " is-active" : "")
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

        {/* 商品設計一覧 Table */}
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
                {productRows.map((p) => (
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
