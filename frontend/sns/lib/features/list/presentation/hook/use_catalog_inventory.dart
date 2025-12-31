// frontend/sns/lib/features/home/presentation/hook/use_catalog_inventory.dart
import '../../../inventory/infrastructure/inventory_repository_http.dart';
import '../../infrastructure/list_repository_http.dart'; // ✅ 追加

class CatalogModelStockRow {
  const CatalogModelStockRow({
    required this.modelId,
    required this.label,
    required this.stockCount,
    required this.price, // ✅ NEW
    required this.rgb, // ✅ NEW: colorRGB（24bit想定）
    required this.size, // ✅ NEW
    required this.colorName, // ✅ NEW
  });

  final String modelId;
  final String label;
  final int stockCount;

  /// ✅ modelId に紐づく価格（無ければ null）
  final int? price;

  /// ✅ modelId に紐づく色（24bit RGB: 0xRRGGBB / 無ければ null）
  final int? rgb;

  /// ✅ size（無ければ null）
  final String? size;

  /// ✅ colorName（無ければ null）
  final String? colorName;
}

class CatalogInventoryComputed {
  const CatalogInventoryComputed({
    required this.totalStock,
    required this.modelStockRows,
  });

  final int? totalStock;
  final List<CatalogModelStockRow> modelStockRows;
}

class UseCatalogModelsResult {
  const UseCatalogModelsResult({required this.models, required this.error});

  final List<SnsModelVariationDTO>? models;
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
    List<SnsModelVariationDTO>? initial,
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
    required SnsInventoryResponse? inventory,
    required List<SnsModelVariationDTO>? modelVariations,

    /// ✅ NEW: list.prices を渡して modelId -> price を結合
    required List<SnsListPriceRow> prices,
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

    // fallback: models が無いなら inventory 側の keys
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

    // ✅ priceMap
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
      final count = stock != null ? _stockCount(stock) : 0;

      // ✅ attach price (may be null if not found)
      final price = priceMap[modelId];

      // ✅ attach rgb (meta.colorRGB). 0/負数は「無し」とみなす
      final int? rgb = (meta != null && meta.colorRGB > 0)
          ? meta.colorRGB
          : null;

      // ✅ attach size / colorName
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

    final total = (inv == null) ? null : _totalStock(inv, rows);
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

  static int _totalStock(
    SnsInventoryResponse inv,
    List<CatalogModelStockRow> rows,
  ) {
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
