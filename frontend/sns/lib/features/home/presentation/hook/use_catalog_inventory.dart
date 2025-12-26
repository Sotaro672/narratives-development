// frontend\sns\lib\features\home\presentation\hook\use_catalog_inventory.dart
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

/// ✅ Inventory card 用の “計算だけ” をまとめる hook
/// - totalStock の算出
/// - modelId -> 表示ラベルの解決（ModelVariationDTOがあればそれを優先）
/// - modelStockRows の生成
class UseCatalogInventory {
  const UseCatalogInventory();

  CatalogInventoryComputed compute({
    required SnsInventoryResponse? inventory,
    required List<ModelVariationDTO>? modelVariations,
  }) {
    final inv = inventory;
    if (inv == null) {
      return const CatalogInventoryComputed(
        totalStock: null,
        modelStockRows: [],
      );
    }

    // modelId -> variation
    final modelMap = <String, ModelVariationDTO>{};
    if (modelVariations != null) {
      for (final v in modelVariations) {
        final id = v.id.trim();
        if (id.isNotEmpty) modelMap[id] = v;
      }
    }

    final rows = <CatalogModelStockRow>[];
    for (final e in inv.stock.entries) {
      final modelId = e.key.trim();
      final stock = e.value;
      final meta = modelMap[modelId];

      final label = meta != null
          ? _modelLabel(meta)
          : (modelId.isNotEmpty ? modelId : '(no model)');

      final count = _stockCount(stock);

      rows.add(
        CatalogModelStockRow(modelId: modelId, label: label, stockCount: count),
      );
    }

    // 表示安定のため label でソート（必要なければ外してOK）
    rows.sort((a, b) => a.label.compareTo(b.label));

    final total = _totalStock(inv);

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
