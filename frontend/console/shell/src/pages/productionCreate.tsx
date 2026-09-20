// frontend/console/shell/src/pages/productionCreate.tsx

import PageStyle from "../layout/PageStyle/PageStyle";
import AdminCard from "../features/admin/presentation/components/AdminCard";
import { toProductBlueprintCategoryPathKey } from "../features/productBlueprint/domain/productBlueprintCategory";
import ProductBlueprintCard from "../features/productBlueprint/presentation/components/productBlueprintForm";
import ProductionQuantityCard from "../features/production/presentation/components/productionQuantityCard";
import { useProductionCreate } from "../features/production/presentation/hook/useProductionCreate";
import { Button } from "../shared/ui/button";
import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
} from "../shared/ui/card";
import { Empty } from "../shared/ui/empty";
import { ErrorMessage } from "../shared/ui/error";
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "../shared/ui/popover";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "../shared/ui/table";
import { Text } from "../shared/ui/text";

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
      <div className="page-column">
        {selectedProductBlueprint ? (
          <ProductBlueprintCard
            mode="view"
            productBlueprintPatch={selectedProductBlueprint}
          />
        ) : (
          <Empty
            compact
            description="商品設計を選択してください"
          />
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

      <div className="page-column">
        <AdminCard
          mode="edit"
          title="管理情報"
          assigneeName={assignee}
          assigneeCandidates={assigneeOptions}
          loadingMembers={loadingMembers}
          onSelectAssignee={onSelectAssignee}
        />

        <Card>
          <CardHeader>
            <CardTitle>ブランド選択</CardTitle>
          </CardHeader>

          <CardContent>
            <Popover>
              <PopoverTrigger>
                <Button
                  type="button"
                  variant="outline"
                  disabled={loadingBrands && brandOptions.length === 0}
                >
                  {selectedBrandName || "ブランドを選択してください"}
                </Button>
              </PopoverTrigger>

              <PopoverContent
                align="start"
                className="popover__content--compact popover__content--medium"
              >
                {brandOptions.length > 0 ? (
                  <div className="popover__list">
                    {brandOptions.map((brand) => {
                      const isSelected = selectedBrandId === brand.id;

                      return (
                        <button
                          key={brand.id}
                          type="button"
                          className={`popover__item${isSelected ? " is-active" : ""}`}
                          onClick={() => selectBrand(brand.id)}
                        >
                          {brand.name}
                        </button>
                      );
                    })}
                  </div>
                ) : loadingBrands ? (
                  <div className="popover__empty">
                    ブランドを読み込み中です。
                  </div>
                ) : (
                  <div className="popover__empty">
                    ブランドが登録されていません。
                  </div>
                )}

                {brandError && (
                  <ErrorMessage size="xs">
                    ブランド一覧の取得に失敗しました。
                  </ErrorMessage>
                )}
              </PopoverContent>
            </Popover>
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle>商品設計一覧</CardTitle>
          </CardHeader>

          <CardContent>
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>商品名</TableHead>
                </TableRow>
              </TableHeader>

              <TableBody>
                {productRows.map((product) => {
                  const isSelected = selectedProductId === product.id;

                  return (
                    <TableRow
                      key={product.id}
                      interactive
                      data-state={isSelected ? "selected" : undefined}
                      onClick={() => selectProductById(product.id)}
                    >
                      <TableCell>{product.name}</TableCell>
                    </TableRow>
                  );
                })}

                {productRows.length === 0 && (
                  <TableRow>
                    <TableCell align="center">
                      <Text as="span" size="sm" tone="muted">
                        対象の商品設計がありません
                      </Text>
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