// frontend/amol/src/features/scan-result/utils/format.ts

import type {
  TokenBlueprintPatchVM,
} from "../../shared/types/scanResult";

export function safeUrl(
  raw: string,
): string {
  const value = raw.trim();

  if (!value) {
    return "";
  }

  try {
    const url = new URL(value);

    if (
      url.protocol &&
      url.host
    ) {
      return url.toString();
    }
  } catch {
    // URLとして解釈できない場合はencodeURIした値を返す
  }

  return encodeURI(value);
}

export function tokenBlueprintPatchHasAnyField(
  vm: TokenBlueprintPatchVM | null,
): boolean {
  if (!vm) {
    return false;
  }

  return Boolean(
    vm.id.trim() ||
      vm.tokenName.trim() ||
      vm.symbol.trim() ||
      vm.brandName.trim() ||
      vm.companyName.trim() ||
      vm.description.trim() ||
      vm.tokenIcon.trim(),
  );
}