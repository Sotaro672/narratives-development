// frontend/console/tokenBlueprint/src/domain/entity/tokenBlueprint.ts

/**
 * console/tokenBlueprint の domain 型は、shell の shared 型を唯一のソースとして参照する。
 * これにより module 間で TokenBlueprint 型がズレない（minted も boolean に統一される）。
 */
import type {
  ContentFileType as SharedContentFileType,
  ContentFile as SharedContentFile,
  TokenBlueprint as SharedTokenBlueprint,
} from "../../../../shell/src/shared/types/tokenBlueprint";

// そのまま re-export（必要ならこのファイルから import して使う）
export type ContentFileType = SharedContentFileType;
export type ContentFile = SharedContentFile;

/**
 * TokenBlueprint
 * - shell/shared を正として使う
 * - 過去に console 側で companyId を要求していた場合に備え、互換のため optional で拡張だけ許す
 *   （不要なら削除してOK）
 */
export type TokenBlueprint = SharedTokenBlueprint & {
  companyId?: string;
};
