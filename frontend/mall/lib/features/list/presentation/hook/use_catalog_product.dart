//frontend\sns\lib\features\home\presentation\hook\use_catalog_product.dart
import '../../../productBlueprint/infrastructure/product_blueprint_repository_http.dart';

/// ProductCard 用の “product blueprint 解決” 専用 hook
/// - catalog endpoint が product を返す場合はそれを優先
/// - 無い場合は productBlueprintId から fetch して補完（best-effort）
class UseCatalogProduct {
  UseCatalogProduct({ProductBlueprintRepositoryHttp? repo})
    : _pbRepo = repo ?? ProductBlueprintRepositoryHttp();

  final ProductBlueprintRepositoryHttp _pbRepo;

  void dispose() {
    _pbRepo.dispose();
  }

  Future<CatalogProductResult> load({
    required String productBlueprintId,

    /// すでに上位（catalog DTO 等）で取れている場合はこれを渡す
    required MallProductBlueprintResponse? initial,

    /// すでに上位（catalog DTO 等）で確定しているエラーがある場合はこれを渡す
    required String? initialError,
  }) async {
    final pbId = productBlueprintId.trim();
    final initErr = _asNonEmptyString(initialError);

    // 1) 既に product があるならそれを返す（追加 fetch しない）
    if (initial != null) {
      return CatalogProductResult(
        productBlueprint: initial,
        productBlueprintError: initErr,
      );
    }

    // 2) 既に上位が error を確定しているなら、それを尊重して fetch しない
    if (initErr != null) {
      return CatalogProductResult(
        productBlueprint: null,
        productBlueprintError: initErr,
      );
    }

    // 3) pbId が無いならここで終了
    if (pbId.isEmpty) {
      const err = 'productBlueprintId is unavailable (inventory not loaded)';
      return const CatalogProductResult(
        productBlueprint: null,
        productBlueprintError: err,
      );
    }

    // 4) best-effort fetch
    try {
      final pb = await _pbRepo.fetchProductBlueprintById(pbId);
      return CatalogProductResult(
        productBlueprint: pb,
        productBlueprintError: null,
      );
    } catch (e) {
      final err = e.toString();
      return CatalogProductResult(
        productBlueprint: null,
        productBlueprintError: err,
      );
    }
  }

  static String? _asNonEmptyString(String? v) {
    final s = (v ?? '').trim();
    return s.isEmpty ? null : s;
  }
}

class CatalogProductResult {
  const CatalogProductResult({
    required this.productBlueprint,
    required this.productBlueprintError,
  });

  final MallProductBlueprintResponse? productBlueprint;
  final String? productBlueprintError;
}
