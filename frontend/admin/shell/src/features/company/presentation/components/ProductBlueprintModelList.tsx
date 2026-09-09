// frontend/admin/shell/src/features/company/presentation/components/ProductBlueprintModelList.tsx

import type { ContractProductBlueprintModelRef } from "../../../../shared/type/contractProductBlueprintDetail";
import { formatModelMeta } from "../../../../shared/util/modelMetaFormat";

type ProductBlueprintModelListProps = {
  modelRefs: ContractProductBlueprintModelRef[];
};

export default function ProductBlueprintModelList({
  modelRefs,
}: ProductBlueprintModelListProps) {
  return (
    <section className="ui-detail-section">
      <h2 className="ui-detail-section__title">モデル</h2>
      {modelRefs.length > 0 ? (
        <dl className="ui-detail-definition-list">
          {modelRefs
            .slice()
            .sort((a, b) => a.displayOrder - b.displayOrder)
            .map((modelRef) => (
              <div key={modelRef.modelId}>
                <dd>{formatModelMeta(modelRef)}</dd>
              </div>
            ))}
        </dl>
      ) : (
        <p>モデル情報はありません。</p>
      )}
    </section>
  );
}