//frontend\console\production\src\infrastructure\model\modelVariationGateway.ts
// model module 依存を infrastructure で局所化するための gateway
export {
  listModelVariationsByProductBlueprintId,
} from "../../../../model/src/infrastructure/repository/modelRepositoryHTTP";
