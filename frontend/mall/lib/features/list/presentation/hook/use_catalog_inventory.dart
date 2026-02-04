//frontend\mall\lib\features\list\presentation\hook\use_catalog_inventory.dart
import '../../../inventory/infrastructure/inventory_repository_http.dart';
import '../../infrastructure/list_repository_http.dart';

// ✅ NEW: modelRefs source (absolute schema)
import '../../../productBlueprint/infrastructure/product_blueprint_repository_http.dart';

class CatalogModelStockRow {
  const CatalogModelStockRow({
    required this.modelId,
    required this.label,
    required this.stockCount, // ✅ availableStock (= accumulation - reservedCount)
    required this.price,
    required this.rgb,
    required this.size,
    required this.colorName,

    // ✅ NEW: displayOrder from productBlueprint.modelRefs
    required this.displayOrder,
  });

  final String modelId;
  final String label;

  /// ✅ availableStock
  final int stockCount;

  final int? price;
  final int? rgb;
  final String? size;
  final String? colorName;

  /// ✅ displayOrder (1..N). 0 means "unknown/unset".
  final int displayOrder;
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
  const UseCatalogModelsResult({
    required this.models,
    required this.error,

    // ✅ NEW: modelId -> displayOrder
    required this.displayOrderByModelId,
  });

  final List<MallModelVariationDTO>? models;
  final String? error;

  /// ✅ modelId -> displayOrder (from productBlueprint.modelRefs)
  final Map<String, int> displayOrderByModelId;
}

// ✅ NEW: derived order maps for size/color based on model displayOrder
class DisplayOrderMaps {
  const DisplayOrderMaps({required this.sizeOrder, required this.colorOrder});

  final Map<String, int> sizeOrder; // size -> order (min displayOrder)
  final Map<String, int> colorOrder; // colorName -> order (min displayOrder)
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

    // ✅ NEW: productBlueprint.modelRefs
    List<MallProductBlueprintModelRef>? modelRefs,
  }) async {
    final pbId = productBlueprintId.trim();

    // ✅ build displayOrder map (absolute schema)
    final displayOrderByModelId = <String, int>{};
    if (modelRefs != null) {
      for (final r in modelRefs) {
        final mid = r.modelId.trim();
        if (mid.isEmpty) continue;
        final order = r.displayOrder;
        if (order > 0) {
          displayOrderByModelId[mid] = order;
        }
      }
    }

    if (initial != null) {
      final err = _asNonEmptyString(initialError);
      _log(
        'use initial models count=${initial.length} '
        'err="${err ?? ''}" '
        'modelRefs=${modelRefs?.length ?? 0}',
      );
      return UseCatalogModelsResult(
        models: initial,
        error: err,
        displayOrderByModelId: displayOrderByModelId,
      );
    }

    if (pbId.isEmpty) {
      final err = 'productBlueprintId is unavailable (skip model fetch)';
      _log('skip fetch: $err');
      return UseCatalogModelsResult(
        models: null,
        error: null,
        displayOrderByModelId: displayOrderByModelId,
      );
    }

    try {
      _log(
        'fetch models start pbId=$pbId '
        'modelRefs=${modelRefs?.length ?? 0}',
      );
      final models = await invRepo.fetchModelsByProductBlueprintId(pbId);
      _log('fetch models ok count=${models.length}');
      return UseCatalogModelsResult(
        models: models,
        error: null,
        displayOrderByModelId: displayOrderByModelId,
      );
    } catch (e) {
      final err = e.toString();
      _log('fetch models error: $err');
      return UseCatalogModelsResult(
        models: null,
        error: _asNonEmptyString(err),
        displayOrderByModelId: displayOrderByModelId,
      );
    }
  }

  CatalogInventoryComputed compute({
    required MallInventoryResponse? inventory,
    required List<MallModelVariationDTO>? modelVariations,

    /// ✅ list.prices を渡して modelId -> price を結合
    required List<MallListPriceRow> prices,

    // ✅ NEW: modelId -> displayOrder
    required Map<String, int> displayOrderByModelId,
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

    // models が無いなら inventory 側の modelIds を使う
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
      final count = stock != null ? _availableStock(stock) : 0;

      final price = priceMap[modelId];

      // ✅ FIX: ブラック(0x000000=0) を null 扱いしない
      final int? rgb = meta?.colorRGB;

      final s = (meta?.size ?? '').trim();
      final String? size = s.isNotEmpty ? s : null;

      final cn = (meta?.colorName ?? '').trim();
      final String? colorName = cn.isNotEmpty ? cn : null;

      final displayOrder = displayOrderByModelId[modelId] ?? 0;

      rows.add(
        CatalogModelStockRow(
          modelId: modelId,
          label: label,
          stockCount: count,
          price: price,
          rgb: rgb,
          size: size,
          colorName: colorName,
          displayOrder: displayOrder,
        ),
      );
    }

    // ✅ NEW: derive size/color order from model displayOrder (min order per group)
    final orderMaps = _buildSizeColorOrderMaps(rows);

    // ✅ sort: sizeOrder asc, then colorOrder asc, then model displayOrder asc, then label asc
    rows.sort(
      (a, b) => _cmpBySizeColorThenOrder(
        a,
        b,
        orderMaps.sizeOrder,
        orderMaps.colorOrder,
      ),
    );

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

  // ============================================================
  // NEW: size/color ordering derived from model displayOrder
  // ============================================================

  static DisplayOrderMaps _buildSizeColorOrderMaps(
    List<CatalogModelStockRow> rows,
  ) {
    const unknown = 1 << 30;

    final sizeOrder = <String, int>{};
    final colorOrder = <String, int>{};

    for (final r in rows) {
      final order = r.displayOrder > 0 ? r.displayOrder : unknown;

      final s = (r.size ?? '').trim();
      if (s.isNotEmpty) {
        final cur = sizeOrder[s] ?? unknown;
        if (order < cur) sizeOrder[s] = order;
      }

      final c = (r.colorName ?? '').trim();
      if (c.isNotEmpty) {
        final cur = colorOrder[c] ?? unknown;
        if (order < cur) colorOrder[c] = order;
      }
    }

    int norm(int v) => v == unknown ? 0 : v;

    return DisplayOrderMaps(
      sizeOrder: {for (final e in sizeOrder.entries) e.key: norm(e.value)},
      colorOrder: {for (final e in colorOrder.entries) e.key: norm(e.value)},
    );
  }

  static int _cmpBySizeColorThenOrder(
    CatalogModelStockRow a,
    CatalogModelStockRow b,
    Map<String, int> sizeOrder,
    Map<String, int> colorOrder,
  ) {
    const unknown = 1 << 30;

    int ord(Map<String, int> m, String? k) {
      final key = (k ?? '').trim();
      if (key.isEmpty) return unknown;
      final v = m[key] ?? 0;
      return v > 0 ? v : unknown;
    }

    final asz = ord(sizeOrder, a.size);
    final bsz = ord(sizeOrder, b.size);
    if (asz != bsz) return asz.compareTo(bsz);

    final acol = ord(colorOrder, a.colorName);
    final bcol = ord(colorOrder, b.colorName);
    if (acol != bcol) return acol.compareTo(bcol);

    final ao = a.displayOrder > 0 ? a.displayOrder : unknown;
    final bo = b.displayOrder > 0 ? b.displayOrder : unknown;
    if (ao != bo) return ao.compareTo(bo);

    return a.label.compareTo(b.label);
  }
}
