//frontend\sns\lib\features\home\presentation\components\catalog_inventory.dart
import 'package:flutter/material.dart';

class CatalogInventoryCard extends StatelessWidget {
  const CatalogInventoryCard({
    super.key,
    required this.productBlueprintId,
    required this.tokenBlueprintId,
    required this.totalStock,
    required this.inventory,
    required this.inventoryError,
    required this.modelStockRows,
  });

  final String productBlueprintId;
  final String tokenBlueprintId;

  final int? totalStock;

  /// vm.inventory（型に依存しないため Object?）
  final Object? inventory;

  final String? inventoryError;

  /// vm.modelStockRows（elements must have: label, modelId, stockCount）
  final List<dynamic>? modelStockRows;

  @override
  Widget build(BuildContext context) {
    final inv = inventory;
    final invErr = inventoryError;
    final rows = modelStockRows ?? const [];

    return Card(
      child: Padding(
        padding: const EdgeInsets.all(12),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Text('在庫', style: Theme.of(context).textTheme.titleMedium),
            const SizedBox(height: 12),

            Text('モデル別', style: Theme.of(context).textTheme.titleSmall),
            const SizedBox(height: 6),

            if (rows.isEmpty)
              Text('(空)', style: Theme.of(context).textTheme.bodyMedium)
            else
              ...rows.map((r) {
                final count = (r.stockCount ?? 0).toString();
                final label = (r.label ?? '').toString();

                // ✅ modelId の表示は削除
                // ✅ model metadata (label) と stock を 1 行で表示
                final line =
                    '${label.isNotEmpty ? label : '(名称なし)'}　/　在庫: $count';

                return Padding(
                  padding: const EdgeInsets.symmetric(vertical: 6),
                  child: Text(
                    line,
                    style: Theme.of(context).textTheme.bodyMedium,
                  ),
                );
              }),

            // ✅ totalStock / productBlueprintId / tokenBlueprintId の表示行は削除
            if (inv == null && (invErr ?? '').trim().isNotEmpty) ...[
              const SizedBox(height: 10),
              Text(
                '在庫エラー: ${invErr!.trim()}',
                style: Theme.of(context).textTheme.labelSmall,
              ),
            ],
          ],
        ),
      ),
    );
  }
}
