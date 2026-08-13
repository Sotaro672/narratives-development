// frontend/console/shell/src/pages/productionCreate.tsx

import PageStyle from "../layout/PageStyle/PageStyle";
import AdminCard from "../features/admin/presentation/components/AdminCard";
import { Card, CardHeader, CardTitle, CardContent } from "../shared/ui/card";
import { Popover, PopoverTrigger, PopoverContent } from "../shared/ui/popover";
import ProductBlueprintCard from "../features/productBlueprint/presentation/cards/productBlueprintForm";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "../shared/ui/table";
import { useProductionCreate } from "../features/production/presentation/hook/useProductionCreate";
import ProductionQuantityCard from "../features/production/presentation/components/productionQuantityCard";

import "../styles/production.css";

export default function ProductionCreate() {
  const {
    onBack,
    onSave,
    selectedProductBlueprint,
    assignee,
    assigneeOptions,
    loadingMembers,
    onSelectAssignee,
    selectedBrand,
    brandOptions,
    selectBrand,
    productRows,
    selectedProductId,
    selectProductById,
    modelVariationsForCard,
    setQuantityRows,
  } = useProductionCreate();

  const productBlueprintCategoryCode =
    selectedProductBlueprint?.productBlueprintCategory.code ?? "";

  return (
    <PageStyle layout="grid-2" title="生産計画の作成" onBack={onBack} onSave={onSave}>
      <div className="space-y-4">
        {selectedProductBlueprint ? (
          <ProductBlueprintCard
            mode="view"
            productBlueprintPatch={selectedProductBlueprint}
          />
        ) : (
          <div className="flex h-full items-center justify-center text-gray-500">
            商品設計を選択してください
          </div>
        )}

        {selectedProductBlueprint && (
          <ProductionQuantityCard
            title="モデル別 生産数一覧"
            rows={modelVariationsForCard}
            productBlueprintCategory={productBlueprintCategoryCode}
            mode="edit"
            onChangeRows={setQuantityRows}
          />
        )}
      </div>

      <div className="space-y-4">
        <AdminCard
          mode="edit"
          title="管理情報"
          assigneeName={assignee}
          assigneeCandidates={assigneeOptions}
          loadingMembers={loadingMembers}
          onSelectAssignee={onSelectAssignee}
        />

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
                  {brandOptions.map((brand) => (
                    <button
                      key={brand}
                      className={`pb-select__row${selectedBrand === brand ? " is-active" : ""}`}
                      onClick={() => selectBrand(brand)}
                    >
                      {brand}
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
                {productRows.map((product) => (
                  <TableRow
                    key={product.id}
                    className={`cursor-pointer hover:bg-blue-50${
                      selectedProductId === product.id ? " bg-blue-100" : ""
                    }`}
                    onClick={() => selectProductById(product.id)}
                  >
                    <TableCell>{product.name}</TableCell>
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