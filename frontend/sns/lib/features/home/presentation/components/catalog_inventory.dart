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
  /// さらに best-effort で colorRGB/rgb も拾う
  final List<dynamic>? modelStockRows;

  // ------------------------------------------------------------
  // color helpers (best-effort)
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

  /// modelStockRows の要素から rgb を可能な限り拾う
  /// 想定:
  /// - r.rgb
  /// - r.colorRgb / r.colorRGB
  /// - r.color?.rgb
  /// - r.metadata?.colorRGB / r.metadata?.colorRgb / r.metadata?.color?.rgb
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

    // map access (in case rows are Map)
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

  /// backend の rgb(int) を Flutter の Color に変換
  /// - 24bit (0xRRGGBB) 想定 → alpha を FF 付与
  /// - すでに ARGB (>= 0xFF000000) っぽい場合はそのまま
  Color? _rgbToColor(int? rgb) {
    if (rgb == null) return null;
    if (rgb <= 0) return null;

    // already ARGB?
    if (rgb >= 0xFF000000) {
      return Color(rgb);
    }

    // 24-bit RGB -> 0xFFRRGGBB
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

                // ✅ rgb を拾って色として表示
                final rgb = _pickRgb(r);
                final color = _rgbToColor(rgb);

                // ✅ model metadata (label) と stock を 1 行で表示
                final line =
                    '${label.isNotEmpty ? label : '(名称なし)'}　/　在庫: $count';

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
