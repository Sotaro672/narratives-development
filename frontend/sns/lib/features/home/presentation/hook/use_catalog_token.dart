//frontend\sns\lib\features\home\presentation\hook\use_catalog_token.dart
import '../../../tokenBlueprint/infrastructure/token_blueprint_repository_http.dart';

/// TokenCard 用の “token blueprint patch 解決” 専用 hook
/// - resolvedTokenBlueprintId を入力として patch を best-effort 取得する
class UseCatalogToken {
  UseCatalogToken({TokenBlueprintRepositoryHTTP? repo})
    : _tbRepo = repo ?? TokenBlueprintRepositoryHTTP();

  // NOTE: TokenBlueprintRepositoryHTTP has no dispose() (same as ModelRepositoryHTTP).
  final TokenBlueprintRepositoryHTTP _tbRepo;

  void dispose() {
    // TokenBlueprintRepositoryHTTP: no dispose()
  }

  Future<CatalogTokenResult> load({
    required String resolvedTokenBlueprintId,
  }) async {
    final tbId = resolvedTokenBlueprintId.trim();
    if (tbId.isEmpty) {
      const err = 'tokenBlueprintId is empty';
      return const CatalogTokenResult(patch: null, error: err);
    }

    TokenBlueprintPatch? tbPatch;
    String? tbErr;

    try {
      tbPatch = await _tbRepo.fetchPatch(tbId);

      if (tbPatch == null) {
        tbErr = 'tokenBlueprint patch not found (404)';
      }
    } catch (e) {
      tbErr = e.toString();
    }

    return CatalogTokenResult(patch: tbPatch, error: _asNonEmptyString(tbErr));
  }

  static String? _asNonEmptyString(String? v) {
    final s = (v ?? '').trim();
    return s.isEmpty ? null : s;
  }
}

class CatalogTokenResult {
  const CatalogTokenResult({required this.patch, required this.error});

  final TokenBlueprintPatch? patch;
  final String? error;
}
