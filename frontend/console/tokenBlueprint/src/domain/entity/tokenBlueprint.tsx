// frontend/console/tokenBlueprint/src/domain/entity/tokenBlueprint.ts

/**
 * console/tokenBlueprint の domain 型は、shell の shared 型を唯一のソースとして参照する。
 * これにより module 間で TokenBlueprint 型がズレない。
 */
import type {
  ContentFileType as SharedContentFileType,
  ContentFile as SharedContentFile,
  TokenBlueprint as SharedTokenBlueprint,
} from "../../../../shell/src/shared/types/tokenBlueprint";

// そのまま re-export（必要ならこのファイルから import して使う）
export type ContentFileType = SharedContentFileType;
export type ContentFile = SharedContentFile;

// Shared を完全に正とする（companyId なども含めて一致させる）
export type TokenBlueprint = SharedTokenBlueprint;
