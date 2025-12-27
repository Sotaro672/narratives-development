//frontend\sns\lib\features\home\presentation\hook\use_catalog_inventory.dart
import '../../../inventory/infrastructure/inventory_repository_http.dart';

class CatalogModelStockRow {
  const CatalogModelStockRow({
    required this.modelId,
    required this.label,
    required this.stockCount,
  });

  final String modelId;
  final String label;
  final int stockCount;
}

class CatalogInventoryComputed {
  const CatalogInventoryComputed({
    required this.totalStock,
    required this.modelStockRows,
  });

  final int? totalStock;

  /// ✅ “model一覧 + stock(あれば)” を統合した表示行
  final List<CatalogModelStockRow> modelStockRows;
}

class UseCatalogModelsResult {
  const UseCatalogModelsResult({required this.models, required this.error});

  final List<SnsModelVariationDTO>? models;
  final String? error;
}

/// ✅ Inventory card 用の “モデル取得 + 計算” をまとめる hook
/// - pbId -> /sns/models で metadata を取得（UseCatalogModel を統合）
/// - modelId -> label 解決
/// - stockCount 解決（stock が無い/空なら 0）
class UseCatalogInventory {
  const UseCatalogInventory();

  void _log(String msg) {
    // ignore: avoid_print
    print('[UseCatalogInventory] $msg');
  }

  // ============================================================
  // Load models (replaces deleted UseCatalogModel)
  // ============================================================

  Future<UseCatalogModelsResult> loadModels({
    required InventoryRepositoryHttp invRepo,
    required String productBlueprintId,

    /// catalog endpoint などで既に modelVariations が渡っている場合はそれを優先
    List<SnsModelVariationDTO>? initial,
    String? initialError,
  }) async {
    final pbId = productBlueprintId.trim();

    // 1) initial があるならそれを返す
    if (initial != null) {
      final err = _asNonEmptyString(initialError);
      _log('use initial models count=${initial.length} err="${err ?? ''}"');
      return UseCatalogModelsResult(models: initial, error: err);
    }

    // 2) pbId が無いなら fetch 不可
    if (pbId.isEmpty) {
      final err = 'productBlueprintId is unavailable (skip model fetch)';
      _log('skip fetch: $err');
      return const UseCatalogModelsResult(models: null, error: null);
    }

    // 3) fetch
    try {
      _log('fetch models start pbId=$pbId');
      final models = await invRepo.fetchModelsByProductBlueprintId(pbId);
      _log('fetch models ok count=${models.length}');
      return UseCatalogModelsResult(models: models, error: null);
    } catch (e) {
      final err = e.toString();
      _log('fetch models error: $err');
      return UseCatalogModelsResult(
        models: null,
        error: _asNonEmptyString(err),
      );
    }
  }

  // ============================================================
  // Compute rows (models + stock)
  // ============================================================

  CatalogInventoryComputed compute({
    required SnsInventoryResponse? inventory,
    required List<SnsModelVariationDTO>? modelVariations,
  }) {
    final inv = inventory;

    // modelId -> meta
    final modelMap = <String, SnsModelVariationDTO>{};
    final modelIds = <String>[];

    if (modelVariations != null) {
      for (final v in modelVariations) {
        final id = v.id.trim();
        if (id.isEmpty) continue;
        modelMap[id] = v;
        modelIds.add(id);
      }
    }

    // fallback: models が無いなら inventory 側の keys を使う
    if (modelIds.isEmpty && inv != null) {
      modelIds.addAll(
        inv.modelIds.map((e) => e.trim()).where((e) => e.isNotEmpty),
      );
      if (modelIds.isEmpty) {
        modelIds.addAll(
          inv.stockKeys.map((e) => e.trim()).where((e) => e.isNotEmpty),
        );
      }
    }

    final uniqModelIds = _uniqPreserveOrder(modelIds);

    final rows = <CatalogModelStockRow>[];
    for (final modelId in uniqModelIds) {
      final meta = modelMap[modelId];
      final label = meta != null ? _modelLabel(meta) : modelId;

      // ✅ inventory が無い / stock が無い / 対象キーが無い -> 0
      final stock = inv?.stock[modelId];
      final count = stock != null ? _stockCount(stock) : 0;

      rows.add(
        CatalogModelStockRow(modelId: modelId, label: label, stockCount: count),
      );
    }

    rows.sort((a, b) => a.label.compareTo(b.label));

    // ✅ totalStock:
    // - inventory が無いなら null（まだ inventory が取れていない）
    // - inventory があるが stock が空なら 0
    final total = (inv == null) ? null : _totalStock(inv, rows);

    return CatalogInventoryComputed(totalStock: total, modelStockRows: rows);
  }

  // ============================================================
  // helpers
  // ============================================================

  static int _stockCount(SnsInventoryModelStock s) {
    if (s.products.isEmpty) return 0;
    var n = 0;
    for (final v in s.products.values) {
      if (v == true) n++;
    }
    return n;
  }

  static int _totalStock(
    SnsInventoryResponse inv,
    List<CatalogModelStockRow> rows,
  ) {
    // rows は “モデル一覧の正規形” なので rows 合計で良い（stock map 直走査だと model が消える）
    var sum = 0;
    for (final r in rows) {
      sum += r.stockCount;
    }
    return sum;
  }

  static String _modelLabel(SnsModelVariationDTO v) {
    final parts = <String>[];
    if (v.modelNumber.trim().isNotEmpty) parts.add(v.modelNumber.trim());
    if (v.size.trim().isNotEmpty) parts.add(v.size.trim());
    final color = v.colorName.trim();
    if (color.isNotEmpty) parts.add(color);
    if (parts.isEmpty) return '(empty)';
    return parts.join(' / ');
  }

  static List<String> _uniqPreserveOrder(List<String> xs) {
    final seen = <String>{};
    final out = <String>[];
    for (final x in xs) {
      final s = x.trim();
      if (s.isEmpty) continue;
      if (seen.add(s)) out.add(s);
    }
    return out;
  }

  static String? _asNonEmptyString(String? v) {
    final s = (v ?? '').trim();
    return s.isEmpty ? null : s;
  }
}
