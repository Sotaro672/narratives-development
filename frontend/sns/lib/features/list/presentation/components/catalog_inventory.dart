// frontend/sns/lib/features/home/presentation/components/catalog_inventory.dart
import 'package:flutter/material.dart';

class CatalogInventoryCard extends StatefulWidget {
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
  /// price / size / colorName も best-effort で拾う
  final List<dynamic>? modelStockRows;

  @override
  State<CatalogInventoryCard> createState() => _CatalogInventoryCardState();
}

class _CatalogInventoryCardState extends State<CatalogInventoryCard> {
  String? _selectedSize; // null = all
  int? _selectedRgb; // null = all

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

    try {
      final x = _toInt(r.rgb);
      if (x != null) return x;
    } catch (_) {}
    try {
      final x = _toInt(r.colorRgb);
      if (x != null) return x;
    } catch (_) {}
    try {
      final x = _toInt(r.colorRGB);
      if (x != null) return x;
    } catch (_) {}

    try {
      final c = r.color;
      if (c != null) {
        final x = _toInt(c.rgb);
        if (x != null) return x;
      }
    } catch (_) {}

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

    if (rgb >= 0xFF000000) return Color(rgb);
    return Color(0xFF000000 | (rgb & 0x00FFFFFF));
  }

  Widget _colorSwatch(Color c, {double size = 12}) {
    return Container(
      width: size,
      height: size,
      decoration: BoxDecoration(
        color: c,
        shape: BoxShape.circle,
        border: Border.all(color: Colors.black12),
      ),
    );
  }

  // ------------------------------------------------------------
  // size / colorName helpers (best-effort)
  // ------------------------------------------------------------

  String? _pickSize(dynamic r) {
    if (r == null) return null;

    try {
      final s = (r.size ?? '').toString().trim();
      if (s.isNotEmpty) return s;
    } catch (_) {}

    try {
      final m = r.metadata;
      if (m != null) {
        final s = (m.size ?? '').toString().trim();
        if (s.isNotEmpty) return s;
      }
    } catch (_) {}

    if (r is Map) {
      final s = (r['size'] ?? '').toString().trim();
      if (s.isNotEmpty) return s;

      final meta = r['metadata'];
      if (meta is Map) {
        final s2 = (meta['size'] ?? '').toString().trim();
        if (s2.isNotEmpty) return s2;
      }
    }

    // fallback: label parse "modelNumber / size / color"
    final label =
        (r is Map
                ? (r['label'] ?? '')
                : (() {
                    try {
                      return r.label ?? '';
                    } catch (_) {
                      return '';
                    }
                  })())
            .toString();

    final parts = label
        .split(' / ')
        .map((e) => e.trim())
        .where((e) => e.isNotEmpty)
        .toList();
    if (parts.length >= 2) {
      final s = parts[1].trim();
      if (s.isNotEmpty) return s;
    }
    return null;
  }

  String? _pickColorName(dynamic r) {
    if (r == null) return null;

    try {
      final s = (r.colorName ?? '').toString().trim();
      if (s.isNotEmpty) return s;
    } catch (_) {}

    try {
      final m = r.metadata;
      if (m != null) {
        final s = (m.colorName ?? '').toString().trim();
        if (s.isNotEmpty) return s;
      }
    } catch (_) {}

    if (r is Map) {
      final s = (r['colorName'] ?? '').toString().trim();
      if (s.isNotEmpty) return s;

      final meta = r['metadata'];
      if (meta is Map) {
        final s2 = (meta['colorName'] ?? '').toString().trim();
        if (s2.isNotEmpty) return s2;
      }
    }

    // fallback: label parse "modelNumber / size / color"
    final label =
        (r is Map
                ? (r['label'] ?? '')
                : (() {
                    try {
                      return r.label ?? '';
                    } catch (_) {
                      return '';
                    }
                  })())
            .toString();

    final parts = label
        .split(' / ')
        .map((e) => e.trim())
        .where((e) => e.isNotEmpty)
        .toList();
    if (parts.length >= 3) {
      final s = parts[2].trim();
      if (s.isNotEmpty) return s;
    }
    return null;
  }

  // ------------------------------------------------------------
  // label helpers
  // ------------------------------------------------------------

  /// label が "modelNumber / size / color" の形なら先頭(modelNumber)だけ落とす
  /// ✅ データ自体は変えず、表示だけ変える
  String _stripModelNumberFromLabel(String label) {
    final s = label.trim();
    if (s.isEmpty) return s;

    final parts = s
        .split(' / ')
        .map((e) => e.trim())
        .where((e) => e.isNotEmpty)
        .toList();
    if (parts.length <= 1) return s;

    return parts.sublist(1).join(' / ');
  }

  // ------------------------------------------------------------
  // filter helpers
  // ------------------------------------------------------------

  List<String> _collectSizes(List<dynamic> rows) {
    final set = <String>{};
    for (final r in rows) {
      final s = (_pickSize(r) ?? '').trim();
      if (s.isNotEmpty) set.add(s);
    }
    final out = set.toList()..sort();
    return out;
  }

  List<int> _collectRgbs(List<dynamic> rows) {
    final set = <int>{};
    for (final r in rows) {
      final rgb = _pickRgb(r);
      if (rgb != null && rgb > 0) set.add(rgb);
    }
    final out = set.toList()..sort();
    return out;
  }

  List<dynamic> _applyFilter(List<dynamic> rows) {
    return rows.where((r) {
      if (_selectedSize != null) {
        final s = (_pickSize(r) ?? '').trim();
        if (s != _selectedSize) return false;
      }
      if (_selectedRgb != null) {
        final rgb = _pickRgb(r);
        if (rgb == null || rgb <= 0) return false;
        if (rgb != _selectedRgb) return false;
      }
      return true;
    }).toList();
  }

  @override
  Widget build(BuildContext context) {
    final invErr = widget.inventoryError;
    final rows = widget.modelStockRows ?? const <dynamic>[];

    final sizes = _collectSizes(rows);
    final rgbs = _collectRgbs(rows);

    // 選択中の値が消えたらリセット
    if (_selectedSize != null && !sizes.contains(_selectedSize)) {
      _selectedSize = null;
    }
    if (_selectedRgb != null && !rgbs.contains(_selectedRgb)) {
      _selectedRgb = null;
    }

    final filtered = _applyFilter(rows);

    return Card(
      child: Padding(
        padding: const EdgeInsets.all(12),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            // ✅ 「在庫」タイトルは削除

            // ✅ Filters（「絞り込み」テキストと「すべて」ボタンは削除）
            if (sizes.isNotEmpty || rgbs.isNotEmpty) ...[
              if (sizes.isNotEmpty) ...[
                Text('サイズ', style: Theme.of(context).textTheme.labelMedium),
                const SizedBox(height: 6),
                Wrap(
                  spacing: 8,
                  runSpacing: 8,
                  children: [
                    ...sizes.map((s) {
                      return ChoiceChip(
                        label: Text(s),
                        selected: _selectedSize == s,
                        // ✅ 同じチップを押すと解除できる（すべてボタン無しでクリア可能）
                        onSelected: (_) => setState(() {
                          _selectedSize = (_selectedSize == s) ? null : s;
                        }),
                      );
                    }),
                  ],
                ),
                const SizedBox(height: 10),
              ],

              if (rgbs.isNotEmpty) ...[
                Text('色', style: Theme.of(context).textTheme.labelMedium),
                const SizedBox(height: 6),
                Wrap(
                  spacing: 8,
                  runSpacing: 8,
                  children: [
                    ...rgbs.map((rgb) {
                      final c = _rgbToColor(rgb);
                      final label = _colorNameForRgb(rows, rgb) ?? '';
                      return ChoiceChip(
                        selected: _selectedRgb == rgb,
                        // ✅ 同じチップを押すと解除できる
                        onSelected: (_) => setState(() {
                          _selectedRgb = (_selectedRgb == rgb) ? null : rgb;
                        }),
                        label: Row(
                          mainAxisSize: MainAxisSize.min,
                          children: [
                            if (c != null) ...[
                              _colorSwatch(c, size: 14),
                              const SizedBox(width: 6),
                            ],
                            Text(label.isNotEmpty ? label : 'color'),
                          ],
                        ),
                      );
                    }),
                  ],
                ),
                const SizedBox(height: 12),
              ],
            ],

            Text('モデル別', style: Theme.of(context).textTheme.titleSmall),
            const SizedBox(height: 6),

            if (filtered.isEmpty)
              Text(
                '該当するモデルがありません',
                style: Theme.of(context).textTheme.bodyMedium,
              )
            else
              ...filtered.map((r) {
                final count = (r.stockCount ?? 0).toString();
                final rawLabel = (r.label ?? '').toString();
                final label = _stripModelNumberFromLabel(rawLabel);

                final rgb = _pickRgb(r);
                final color = _rgbToColor(rgb);

                final price = _pickPrice(r);
                final priceText = (price != null) ? _formatYen(price) : '(未設定)';

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

            if (widget.inventory == null &&
                (invErr ?? '').trim().isNotEmpty) ...[
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

  /// 同じ rgb の行から、最初に見つかった colorName を拾う（表示用）
  String? _colorNameForRgb(List<dynamic> rows, int rgb) {
    for (final r in rows) {
      final rr = _pickRgb(r);
      if (rr == rgb) {
        final n = (_pickColorName(r) ?? '').trim();
        if (n.isNotEmpty) return n;
      }
    }
    return null;
  }
}
