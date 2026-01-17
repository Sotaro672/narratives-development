// frontend/mall/lib/features/list/presentation/hook/use_catalog_inventory.dart
import '../../../inventory/infrastructure/inventory_repository_http.dart';
import '../../infrastructure/list_repository_http.dart';

class CatalogModelStockRow {
  const CatalogModelStockRow({
    required this.modelId,
    required this.label,
    required this.stockCount, // ✅ availableStock (= accumulation - reservedCount)
    required this.price,
    required this.rgb,
    required this.size,
    required this.colorName,
  });

  final String modelId;
  final String label;

  /// ✅ availableStock
  final int stockCount;

  final int? price;
  final int? rgb;
  final String? size;
  final String? colorName;
}

class CatalogInventoryComputed {
  const CatalogInventoryComputed({
    required this.totalStock, // ✅ total availableStock
    required this.modelStockRows,
  });

  final int? totalStock;
  final List<CatalogModelStockRow> modelStockRows;
}

class UseCatalogModelsResult {
  const UseCatalogModelsResult({required this.models, required this.error});

  final List<MallModelVariationDTO>? models;
  final String? error;
}

class UseCatalogInventory {
  const UseCatalogInventory();

  void _log(String msg) {
    // ignore: avoid_print
    print('[UseCatalogInventory] $msg');
  }

  Future<UseCatalogModelsResult> loadModels({
    required InventoryRepositoryHttp invRepo,
    required String productBlueprintId,
    List<MallModelVariationDTO>? initial,
    String? initialError,
  }) async {
    final pbId = productBlueprintId.trim();

    if (initial != null) {
      final err = _asNonEmptyString(initialError);
      _log('use initial models count=${initial.length} err="${err ?? ''}"');
      return UseCatalogModelsResult(models: initial, error: err);
    }

    if (pbId.isEmpty) {
      final err = 'productBlueprintId is unavailable (skip model fetch)';
      _log('skip fetch: $err');
      return const UseCatalogModelsResult(models: null, error: null);
    }

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

  CatalogInventoryComputed compute({
    required MallInventoryResponse? inventory,
    required List<MallModelVariationDTO>? modelVariations,

    /// ✅ list.prices を渡して modelId -> price を結合
    required List<MallListPriceRow> prices,
  }) {
    final inv = inventory;

    // modelId -> meta
    final modelMap = <String, MallModelVariationDTO>{};
    final modelIds = <String>[];

    if (modelVariations != null) {
      for (final v in modelVariations) {
        final id = v.id.trim();
        if (id.isEmpty) continue;
        modelMap[id] = v;
        modelIds.add(id);
      }
    }

    // models が無いなら inventory 側の modelIds を使う（旧式互換は不要なので stockKeys fallback は不要）
    if (modelIds.isEmpty && inv != null) {
      modelIds.addAll(
        inv.modelIds.map((e) => e.trim()).where((e) => e.isNotEmpty),
      );
    }

    final uniqModelIds = _uniqPreserveOrder(modelIds);

    // priceMap: modelId -> price
    final priceMap = <String, int>{};
    for (final row in prices) {
      final mid = row.modelId.trim();
      if (mid.isEmpty) continue;
      priceMap[mid] = row.price;
    }

    final rows = <CatalogModelStockRow>[];
    for (final modelId in uniqModelIds) {
      final meta = modelMap[modelId];
      final label = meta != null ? _modelLabel(meta) : modelId;

      final stock = inv?.stock[modelId];
      final count = stock != null ? _availableStock(stock) : 0; // ✅ HERE

      final price = priceMap[modelId];

      final int? rgb = (meta != null && meta.colorRGB > 0)
          ? meta.colorRGB
          : null;

      final s = (meta?.size ?? '').trim();
      final String? size = s.isNotEmpty ? s : null;

      final cn = (meta?.colorName ?? '').trim();
      final String? colorName = cn.isNotEmpty ? cn : null;

      rows.add(
        CatalogModelStockRow(
          modelId: modelId,
          label: label,
          stockCount: count,
          price: price,
          rgb: rgb,
          size: size,
          colorName: colorName,
        ),
      );
    }

    rows.sort((a, b) => a.label.compareTo(b.label));

    final total = (inv == null) ? null : _totalAvailableStock(rows);
    return CatalogInventoryComputed(totalStock: total, modelStockRows: rows);
  }

  /// ✅ availableStock = accumulation - reservedCount
  static int _availableStock(MallInventoryModelStock s) {
    final a = s.accumulation;
    final r = s.reservedCount;
    final v = a - r;
    return v < 0 ? 0 : v;
  }

  static int _totalAvailableStock(List<CatalogModelStockRow> rows) {
    var sum = 0;
    for (final r in rows) {
      sum += r.stockCount;
    }
    return sum;
  }

  static String _modelLabel(MallModelVariationDTO v) {
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
