//frontend\sns\lib\features\home\presentation\hook\use_catalog_inventory.dart
import '../../../inventory/infrastructure/inventory_repository_http.dart';
import '../../../model/infrastructure/model_repository_http.dart';

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
  final List<CatalogModelStockRow> modelStockRows;
}

/// ✅ Inventory card 用の hook
/// - modelVariations のロード（旧 UseCatalogModel を統合）
/// - totalStock の算出
/// - modelId -> 表示ラベルの解決
/// - modelStockRows の生成（model が主 / stock が無ければ 0）
class UseCatalogInventory {
  const UseCatalogInventory();

  void _log(String msg) {
    // ignore: avoid_print
    print('[UseCatalogInventory] $msg');
  }

  // ============================================================
  // ✅ NEW: 旧 UseCatalogModel.load を統合
  // ============================================================

  Future<UseCatalogInventoryModelResult> loadModels({
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
      return UseCatalogInventoryModelResult(models: initial, error: err);
    }

    // 2) pbId が無いなら fetch 不可
    if (pbId.isEmpty) {
      const err = 'productBlueprintId is unavailable (skip model fetch)';
      _log('skip fetch: $err');
      return const UseCatalogInventoryModelResult(models: null, error: err);
    }

    // 3) fetch
    try {
      _log('fetch start pbId=$pbId');
      final models = await modelRepo.fetchModelVariationsByProductBlueprintId(
        pbId,
      );
      _log('fetch ok count=${models.length}');
      return UseCatalogInventoryModelResult(models: models, error: null);
    } catch (e) {
      final err = e.toString();
      _log('fetch error: $err');
      return UseCatalogInventoryModelResult(
        models: null,
        error: _asNonEmptyString(err),
      );
    }
  }

  static String? _asNonEmptyString(String? v) {
    final s = (v ?? '').trim();
    return s.isEmpty ? null : s;
  }

  // ============================================================
  // compute (rows + totals)
  // - modelVariations を主として rows を作る
  // - inv.stock が無い / 空 / 該当modelIdが無い => stockCount=0
  // ============================================================

  CatalogInventoryComputed compute({
    required SnsInventoryResponse? inventory,
    required List<ModelVariationDTO>? modelVariations,
  }) {
    final inv = inventory;

    // modelId -> variation
    final modelMap = <String, ModelVariationDTO>{};
    if (modelVariations != null) {
      for (final v in modelVariations) {
        final id = v.id.trim();
        if (id.isNotEmpty) modelMap[id] = v;
      }
    }

    final rows = <CatalogModelStockRow>[];

    // ✅ 1) まず modelVariations を主として rows を生成
    if (modelVariations != null) {
      for (final v in modelVariations) {
        final modelId = v.id.trim();
        if (modelId.isEmpty) continue;

        final label = _modelLabel(v);

        // ✅ inv が無い / stock が無い / 해당キーが無い => 0
        final stockObj = (inv != null) ? inv.stock[modelId] : null;
        final count = (stockObj != null) ? _stockCount(stockObj) : 0;

        rows.add(
          CatalogModelStockRow(
            modelId: modelId,
            label: label,
            stockCount: count,
          ),
        );
      }
    }

    // ✅ 2) 追加：inventory にあるが modelVariations に無い modelId も拾う（欠損耐性）
    if (inv != null) {
      for (final e in inv.stock.entries) {
        final modelId = e.key.trim();
        if (modelId.isEmpty) continue;
        if (modelMap.containsKey(modelId)) continue;

        final label = modelId; // metadata 欠損なので id を出す
        final count = _stockCount(e.value);
        rows.add(
          CatalogModelStockRow(
            modelId: modelId,
            label: label,
            stockCount: count,
          ),
        );
      }
    }

    // 表示安定のため label でソート（必要なければ外してOK）
    rows.sort((a, b) => a.label.compareTo(b.label));

    // totalStock:
    // - inventory が無いなら null（未ロード）
    // - inventory があるなら stock が空でも 0
    final total = (inv != null) ? _totalStock(inv) : null;

    return CatalogInventoryComputed(totalStock: total, modelStockRows: rows);
  }

  static int _stockCount(SnsInventoryModelStock s) {
    if (s.products.isEmpty) return 0;
    var n = 0;
    for (final v in s.products.values) {
      if (v == true) n++;
    }
    return n;
  }

  static int _totalStock(SnsInventoryResponse inv) {
    var sum = 0;
    for (final v in inv.stock.values) {
      sum += _stockCount(v);
    }
    return sum;
  }

  static String _modelLabel(ModelVariationDTO v) {
    final parts = <String>[];
    if (v.modelNumber.trim().isNotEmpty) parts.add(v.modelNumber.trim());
    if (v.size.trim().isNotEmpty) parts.add(v.size.trim());
    final color = v.color.name.trim();
    if (color.isNotEmpty) parts.add(color);
    if (parts.isEmpty) return '(empty)';
    return parts.join(' / ');
  }
}

class UseCatalogInventoryModelResult {
  const UseCatalogInventoryModelResult({
    required this.models,
    required this.error,
  });

  final List<ModelVariationDTO>? models;
  final String? error;
}
