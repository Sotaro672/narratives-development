// frontend/console/shell/src/pages/productionCreate.tsx

import PageStyle from "../layout/PageStyle/PageStyle";
import AdminCard from "../features/admin/presentation/components/AdminCard";
import ProductBlueprintCard from "../features/productBlueprint/presentation/cards/productBlueprintForm";
import { toProductBlueprintCategoryPathKey } from "../features/productBlueprint/domain/productBlueprintCategory";
import { useProductionCreate } from "../features/production/presentation/hook/useProductionCreate";
import ProductionQuantityCard from "../features/production/presentation/components/productionQuantityCard";
import { Card, CardContent, CardHeader, CardTitle } from "../shared/ui/card";
import { Popover, PopoverContent, PopoverTrigger } from "../shared/ui/popover";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "../shared/ui/table";

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
    selectedBrandId,
    selectedBrandName,
    brandOptions,
    loadingBrands,
    brandError,
    selectBrand,
    productRows,
    selectedProductId,
    selectProductById,
    quantityRows,
    setQuantityRows,
  } = useProductionCreate();

  const productBlueprintCategoryCode =
    selectedProductBlueprint?.productBlueprintCategoryPath
      ? toProductBlueprintCategoryPathKey(
          selectedProductBlueprint.productBlueprintCategoryPath,
        )
      : "";

  return (
    <PageStyle
      layout="grid-2"
      title="生産計画の作成"
      onBack={onBack}
      onSave={onSave}
    >
      <div className="production-create__column">
        {selectedProductBlueprint ? (
          <ProductBlueprintCard
            mode="view"
            productBlueprintPatch={selectedProductBlueprint}
          />
        ) : (
          <div className="production-create__product-empty">
            商品設計を選択してください
          </div>
        )}

        {selectedProductBlueprint && (
          <ProductionQuantityCard
            title="モデル別 生産数一覧"
            rows={quantityRows}
            productBlueprintCategory={productBlueprintCategoryCode}
            mode="edit"
            onChangeRows={setQuantityRows}
          />
        )}
      </div>

      <div className="production-create__column">
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
                  {selectedBrandName || "ブランドを選択してください"}
                </div>
              </PopoverTrigger>

              <PopoverContent>
                <div className="pb-select__list">
                  {brandOptions.map((brand) => (
                    <button
                      key={brand.id}
                      type="button"
                      className={`pb-select__row${
                        selectedBrandId === brand.id ? " is-active" : ""
                      }`}
                      onClick={() => selectBrand(brand.id)}
                    >
                      {brand.name}
                    </button>
                  ))}

                  {loadingBrands && (
                    <div className="pb-select__empty">
                      ブランドを読み込み中です。
                    </div>
                  )}

                  {!loadingBrands && brandOptions.length === 0 && (
                    <div className="pb-select__empty">
                      ブランドが登録されていません。
                    </div>
                  )}

                  {brandError && (
                    <div className="pb-select__empty pb-select__empty--error">
                      ブランド一覧の取得に失敗しました。
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
            <Table className="production-create__product-table">
              <TableHeader>
                <TableRow>
                  <TableHead>商品名</TableHead>
                </TableRow>
              </TableHeader>

              <TableBody>
                {productRows.map((product) => (
                  <TableRow
                    key={product.id}
                    className={`production-create__product-row${
                      selectedProductId === product.id ? " is-active" : ""
                    }`}
                    onClick={() => selectProductById(product.id)}
                  >
                    <TableCell>{product.name}</TableCell>
                  </TableRow>
                ))}

                {productRows.length === 0 && (
                  <TableRow>
                    <TableCell className="production-create__product-empty-cell">
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