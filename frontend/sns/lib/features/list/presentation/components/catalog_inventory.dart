// frontend/sns/lib/features/home/presentation/components/catalog_inventory.dart
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
  /// さらに best-effort で colorRGB/rgb も拾う
  /// price も best-effort で拾う
  final List<dynamic>? modelStockRows;

  // ------------------------------------------------------------
  // number helpers (best-effort)
  // ------------------------------------------------------------

  int? _toInt(dynamic v) {
    if (v == null) return null;
    if (v is int) return v;
    if (v is double) return v.toInt();
    if (v is num) return v.toInt();
    final s = v.toString().trim();
    if (s.isEmpty) return null;
    return int.tryParse(s);
  }

  // ------------------------------------------------------------
  // price helpers (best-effort)
  // ------------------------------------------------------------

  int? _pickPrice(dynamic r) {
    if (r == null) return null;

    // direct fields
    try {
      final x = _toInt(r.price);
      if (x != null) return x;
    } catch (_) {}
    try {
      final x = _toInt(r.priceYen);
      if (x != null) return x;
    } catch (_) {}
    try {
      final x = _toInt(r.amount);
      if (x != null) return x;
    } catch (_) {}
    try {
      final x = _toInt(r.value);
      if (x != null) return x;
    } catch (_) {}

    // nested: metadata.*
    try {
      final m = r.metadata;
      if (m != null) {
        try {
          final x = _toInt(m.price);
          if (x != null) return x;
        } catch (_) {}
        try {
          final x = _toInt(m.priceYen);
          if (x != null) return x;
        } catch (_) {}
        try {
          final x = _toInt(m.amount);
          if (x != null) return x;
        } catch (_) {}
      }
    } catch (_) {}

    // map access
    if (r is Map) {
      final m = r;

      final x1 = _toInt(
        m['price'] ?? m['priceYen'] ?? m['amount'] ?? m['value'],
      );
      if (x1 != null) return x1;

      final meta = m['metadata'];
      if (meta is Map) {
        final x2 = _toInt(
          meta['price'] ?? meta['priceYen'] ?? meta['amount'] ?? meta['value'],
        );
        if (x2 != null) return x2;
      }
    }

    return null;
  }

  String _formatYen(int price) => '¥$price';

  // ------------------------------------------------------------
  // color helpers (best-effort)
  // ------------------------------------------------------------

  int? _pickRgb(dynamic r) {
    if (r == null) return null;

    // direct fields
    try {
      final v = r.rgb;
      final x = _toInt(v);
      if (x != null) return x;
    } catch (_) {}
    try {
      final v = r.colorRgb;
      final x = _toInt(v);
      if (x != null) return x;
    } catch (_) {}
    try {
      final v = r.colorRGB;
      final x = _toInt(v);
      if (x != null) return x;
    } catch (_) {}

    // nested: color.rgb
    try {
      final c = r.color;
      if (c != null) {
        final x = _toInt(c.rgb);
        if (x != null) return x;
      }
    } catch (_) {}

    // nested: metadata.*
    try {
      final m = r.metadata;
      if (m != null) {
        try {
          final x = _toInt(m.colorRGB);
          if (x != null) return x;
        } catch (_) {}
        try {
          final x = _toInt(m.colorRgb);
          if (x != null) return x;
        } catch (_) {}
        try {
          final x = _toInt(m.rgb);
          if (x != null) return x;
        } catch (_) {}
        try {
          final c = m.color;
          if (c != null) {
            final x = _toInt(c.rgb);
            if (x != null) return x;
          }
        } catch (_) {}
      }
    } catch (_) {}

    // map access
    if (r is Map) {
      final m = r;
      final x1 = _toInt(m['colorRGB'] ?? m['colorRgb'] ?? m['rgb']);
      if (x1 != null) return x1;

      final meta = m['metadata'];
      if (meta is Map) {
        final x2 = _toInt(meta['colorRGB'] ?? meta['colorRgb'] ?? meta['rgb']);
        if (x2 != null) return x2;

        final c = meta['color'];
        if (c is Map) {
          final x3 = _toInt(c['rgb']);
          if (x3 != null) return x3;
        }
      }

      final c = m['color'];
      if (c is Map) {
        final x4 = _toInt(c['rgb']);
        if (x4 != null) return x4;
      }
    }

    return null;
  }

  Color? _rgbToColor(int? rgb) {
    if (rgb == null) return null;
    if (rgb <= 0) return null;

    if (rgb >= 0xFF000000) {
      return Color(rgb);
    }
    final v = (0xFF000000 | (rgb & 0x00FFFFFF));
    return Color(v);
  }

  Widget _colorSwatch(Color c) {
    return Container(
      width: 12,
      height: 12,
      decoration: BoxDecoration(
        color: c,
        shape: BoxShape.circle,
        border: Border.all(color: Colors.black12),
      ),
    );
  }

  // ------------------------------------------------------------
  // label helpers
  // ------------------------------------------------------------

  /// label が "modelNumber / size / color" の形なら先頭(modelNumber)だけ落とす
  /// ✅ データ自体は変えず、表示だけ変える
  String _stripModelNumberFromLabel(String label) {
    final s = label.trim();
    if (s.isEmpty) return s;

    // " / " 区切りを想定（UseCatalogInventory 側の join(' / ') と一致）
    final parts = s
        .split(' / ')
        .map((e) => e.trim())
        .where((e) => e.isNotEmpty)
        .toList();
    if (parts.length <= 1) return s;

    // 先頭を落として残りを表示
    return parts.sublist(1).join(' / ');
  }

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
                final rawLabel = (r.label ?? '').toString();

                // ✅ 表示だけ modelNumber を削除（データはそのまま）
                final label = _stripModelNumberFromLabel(rawLabel);

                // ✅ rgb を拾って色の丸だけ表示（文字列RGBは表示しない）
                final rgb = _pickRgb(r);
                final color = _rgbToColor(rgb);

                // ✅ price を拾う
                final price = _pickPrice(r);
                final priceText = (price != null) ? _formatYen(price) : '(未設定)';

                // ✅ RGB文字列は削除
                final line =
                    '${label.isNotEmpty ? label : '(名称なし)'}　/　在庫: $count　/　価格: $priceText';

                return Padding(
                  padding: const EdgeInsets.symmetric(vertical: 6),
                  child: Row(
                    crossAxisAlignment: CrossAxisAlignment.center,
                    children: [
                      if (color != null) ...[
                        _colorSwatch(color),
                        const SizedBox(width: 8),
                      ],
                      Expanded(
                        child: Text(
                          line,
                          style: Theme.of(context).textTheme.bodyMedium,
                        ),
                      ),
                    ],
                  ),
                );
              }),

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
