// frontend/console/model/src/application/modelUpdateService.tsx

// アプリケーション層からは API 層の型・関数をそのまま再エクスポートするだけにする。
// これにより、呼び出し側は従来どおり application 層を参照しつつ、
// 実際の HTTP や Firebase Auth などの詳細は infrastructure/api 側に隠蔽される。

export type {
  ModelVariationUpdateRequest,
  ModelVariationResponse,
} from "../infrastructure/api/modelUpdateApi";

export {
  updateModelVariation,
  deleteModelVariation,
} from "../infrastructure/api/modelUpdateApi";
