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

  void _log(String msg) {
    // ignore: avoid_print
    print('[UseCatalogToken] $msg');
  }

  Future<CatalogTokenResult> load({
    required String resolvedTokenBlueprintId,
  }) async {
    final tbId = resolvedTokenBlueprintId.trim();
    if (tbId.isEmpty) {
      const err = 'tokenBlueprintId is empty';
      _log('skip fetchPatch: $err');
      return const CatalogTokenResult(patch: null, error: err);
    }

    TokenBlueprintPatch? tbPatch;
    String? tbErr;

    try {
      _log('fetchPatch start tokenBlueprintId=$tbId');
      tbPatch = await _tbRepo.fetchPatch(tbId);

      if (tbPatch == null) {
        tbErr = 'tokenBlueprint patch not found (404)';
        _log('fetchPatch result: null (404)');
      } else {
        _log(
          'fetchPatch ok '
          'name="${(tbPatch.name ?? '').trim()}" '
          'symbol="${(tbPatch.symbol ?? '').trim()}" '
          'brandId="${(tbPatch.brandId ?? '').trim()}" '
          'minted=${tbPatch.minted} '
          'hasIconUrl=${(tbPatch.iconUrl ?? '').trim().isNotEmpty}',
        );
      }
    } catch (e) {
      tbErr = e.toString();
      _log('fetchPatch error: $tbErr');
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
