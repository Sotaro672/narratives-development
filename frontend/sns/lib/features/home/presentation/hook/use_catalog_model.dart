// frontend\sns\lib\features\home\presentation\hook\use_catalog_model.dart
import '../../../model/infrastructure/model_repository_http.dart';

class UseCatalogModel {
  const UseCatalogModel();

  void _log(String msg) {
    // ignore: avoid_print
    print('[UseCatalogModel] $msg');
  }

  Future<UseCatalogModelResult> load({
    required ModelRepositoryHTTP modelRepo,
    required String productBlueprintId,

    /// catalog endpoint などで既に modelVariations が渡っている場合はそれを優先する
    List<ModelVariationDTO>? initial,
    String? initialError,
  }) async {
    final pbId = productBlueprintId.trim();

    // 1) initial があるならそれをそのまま返す（error も持ち回り）
    if (initial != null) {
      final err = _asNonEmptyString(initialError);
      _log(
        'use initial modelVariations count=${initial.length} err="${err ?? ''}"',
      );
      return UseCatalogModelResult(models: initial, error: err);
    }

    // 2) pbId が無いなら fetch 不可
    if (pbId.isEmpty) {
      final err = 'productBlueprintId is unavailable (skip model fetch)';
      _log('skip fetch: $err');
      return const UseCatalogModelResult(
        models: null,
        error: 'productBlueprintId is unavailable (skip model fetch)',
      );
    }

    // 3) fetch
    try {
      _log('fetch start pbId=$pbId');
      final models = await modelRepo.fetchModelVariationsByProductBlueprintId(
        pbId,
      );
      _log('fetch ok count=${models.length}');
      return UseCatalogModelResult(models: models, error: null);
    } catch (e) {
      final err = e.toString();
      _log('fetch error: $err');
      return UseCatalogModelResult(models: null, error: _asNonEmptyString(err));
    }
  }

  static String? _asNonEmptyString(String? v) {
    final s = (v ?? '').trim();
    return s.isEmpty ? null : s;
  }
}

class UseCatalogModelResult {
  const UseCatalogModelResult({required this.models, required this.error});

  final List<ModelVariationDTO>? models;
  final String? error;
}
