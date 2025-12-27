import 'package:flutter/material.dart';

class CatalogInventoryCard extends StatelessWidget {
  const CatalogInventoryCard({
    super.key,
    required this.productBlueprintId,
    required this.tokenBlueprintId,
    required this.totalStock,
    required this.inventory,
    required this.inventoryError,

    // ✅ NEW: catalog_query で統合された modelVariations を受け取る
    required this.modelVariations,
    required this.modelVariationsError,
  });

  final String productBlueprintId;
  final String tokenBlueprintId;

  /// 画面上部の「total stock」表示に使う（VMで計算済みでOK）
  final int? totalStock;

  /// vm.inventory（型に依存しないため Object?）
  final Object? inventory;

  final String? inventoryError;

  /// vm.modelVariations（elements must have: id/modelId, modelNumber, size, colorName(or color.name), measurements, stockCount/products）
  final List<dynamic>? modelVariations;

  final String? modelVariationsError;

  @override
  Widget build(BuildContext context) {
    final pbId = productBlueprintId.trim();
    final tbId = tokenBlueprintId.trim();

    final inv = inventory;
    final invErr = (inventoryError ?? '').trim();

    final models = modelVariations;
    final modelErr = (modelVariationsError ?? '').trim();

    return Card(
      child: Padding(
        padding: const EdgeInsets.all(12),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Text('Inventory', style: Theme.of(context).textTheme.titleMedium),
            const SizedBox(height: 8),
            _KeyValueRow(
              label: 'productBlueprintId',
              value: pbId.isNotEmpty ? pbId : '(unknown)',
            ),
            const SizedBox(height: 6),
            _KeyValueRow(
              label: 'tokenBlueprintId',
              value: tbId.isNotEmpty ? tbId : '(unknown)',
            ),
            const SizedBox(height: 6),
            _KeyValueRow(
              label: 'total stock',
              value: totalStock != null
                  ? totalStock.toString()
                  : '(not loaded)',
            ),

            // -----------------------
            // Inventory error (optional)
            // -----------------------
            if (inv == null && invErr.isNotEmpty) ...[
              const SizedBox(height: 10),
              Text(
                'inventory error: $invErr',
                style: Theme.of(context).textTheme.labelSmall,
              ),
            ],

            const SizedBox(height: 12),
            Text('Models', style: Theme.of(context).textTheme.titleSmall),
            const SizedBox(height: 6),

            // -----------------------
            // Models section
            // -----------------------
            if (models == null) ...[
              if (modelErr.isNotEmpty)
                Text(
                  'model error: $modelErr',
                  style: Theme.of(context).textTheme.labelSmall,
                )
              else
                Text(
                  'models are not loaded',
                  style: Theme.of(context).textTheme.labelSmall,
                ),
            ] else if (models.isEmpty) ...[
              Text('(empty)', style: Theme.of(context).textTheme.bodyMedium),
            ] else ...[
              ...models.map((m) => _ModelRow(model: m)),
            ],
          ],
        ),
      ),
    );
  }
}

class _ModelRow extends StatelessWidget {
  const _ModelRow({required this.model});

  final dynamic model;

  @override
  Widget build(BuildContext context) {
    final modelId = _readModelId(model);
    final modelNumber = _readString(model, candidates: ['modelNumber']);
    final size = _readString(model, candidates: ['size']);

    // ✅ Color integrated: colorName/colorRGB (fallback: color.name/color.rgb)
    final colorName = _readColorName(model);
    final colorRgb = _readColorRgb(model);

    // ✅ stock: if missing => 0
    final stock = _readStockCount(model);

    final titleParts = <String>[
      modelNumber,
      size,
      colorName,
    ].where((s) => s.trim().isNotEmpty).toList();

    final title = titleParts.isNotEmpty ? titleParts.join(' / ') : '(empty)';

    final measurements = _readMeasurements(model);

    return Padding(
      padding: const EdgeInsets.symmetric(vertical: 8),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Text(title, style: Theme.of(context).textTheme.bodyLarge),
          const SizedBox(height: 4),
          Text(
            'modelId: ${modelId.isNotEmpty ? modelId : '(empty)'}   stock: $stock'
            '${colorRgb != null ? '   rgb: $colorRgb' : ''}',
            style: Theme.of(context).textTheme.labelSmall,
          ),
          if (measurements.isNotEmpty) ...[
            const SizedBox(height: 6),
            Wrap(
              spacing: 8,
              runSpacing: 8,
              children: measurements.entries.map((e) {
                return Chip(
                  label: Text('${e.key}: ${e.value}'),
                  visualDensity: VisualDensity.compact,
                );
              }).toList(),
            ),
          ],
        ],
      ),
    );
  }
}

class _KeyValueRow extends StatelessWidget {
  const _KeyValueRow({required this.label, required this.value});

  final String label;
  final String value;

  @override
  Widget build(BuildContext context) {
    return Row(
      children: [
        SizedBox(
          width: 160,
          child: Text(label, style: Theme.of(context).textTheme.labelMedium),
        ),
        Expanded(child: Text(value)),
      ],
    );
  }
}

// ============================================================
// Robust readers (Map or typed DTO)
// ============================================================

String _s(dynamic v) => (v ?? '').toString().trim();

String _readString(dynamic obj, {required List<String> candidates}) {
  for (final k in candidates) {
    // Map access
    if (obj is Map) {
      final v = obj[k];
      final out = _s(v);
      if (out.isNotEmpty) return out;
    }
  }

  // typed DTO access (best-effort, per known keys)
  for (final k in candidates) {
    try {
      switch (k) {
        case 'modelNumber':
          final out = _s((obj as dynamic).modelNumber);
          if (out.isNotEmpty) return out;
          break;
        case 'size':
          final out = _s((obj as dynamic).size);
          if (out.isNotEmpty) return out;
          break;
        case 'colorName':
          final out = _s((obj as dynamic).colorName);
          if (out.isNotEmpty) return out;
          break;
      }
    } catch (_) {
      // ignore
    }
  }

  return '';
}

String _readModelId(dynamic obj) {
  // Map: id or modelId
  if (obj is Map) {
    final a = _s(obj['id']);
    if (a.isNotEmpty) return a;
    final b = _s(obj['modelId']);
    if (b.isNotEmpty) return b;
  }

  // typed
  try {
    final a = _s((obj as dynamic).id);
    if (a.isNotEmpty) return a;
  } catch (_) {}
  try {
    final b = _s((obj as dynamic).modelId);
    if (b.isNotEmpty) return b;
  } catch (_) {}

  return '';
}

String _readColorName(dynamic obj) {
  // Map: colorName
  if (obj is Map) {
    final a = _s(obj['colorName']);
    if (a.isNotEmpty) return a;

    // Map: color.name
    final c = obj['color'];
    if (c is Map) {
      final b = _s(c['name']);
      if (b.isNotEmpty) return b;
    }
  }

  // typed: colorName
  try {
    final a = _s((obj as dynamic).colorName);
    if (a.isNotEmpty) return a;
  } catch (_) {}

  // typed: color.name
  try {
    final c = (obj as dynamic).color;
    final b = _s((c as dynamic).name);
    if (b.isNotEmpty) return b;
  } catch (_) {}

  return '';
}

int? _readColorRgb(dynamic obj) {
  // Map: colorRGB
  if (obj is Map) {
    final v = obj['colorRGB'];
    if (v is int) return v;
    if (v is num) return v.toInt();

    // Map: color.rgb
    final c = obj['color'];
    if (c is Map) {
      final r = c['rgb'];
      if (r is int) return r;
      if (r is num) return r.toInt();
    }
  }

  // typed: colorRGB
  try {
    final v = (obj as dynamic).colorRGB;
    if (v is int) return v;
    if (v is num) return v.toInt();
  } catch (_) {}

  // typed: color.rgb
  try {
    final c = (obj as dynamic).color;
    final r = (c as dynamic).rgb;
    if (r is int) return r;
    if (r is num) return r.toInt();
  } catch (_) {}

  return null;
}

Map<String, int> _readMeasurements(dynamic obj) {
  dynamic raw;

  if (obj is Map) {
    raw = obj['measurements'];
  } else {
    try {
      raw = (obj as dynamic).measurements;
    } catch (_) {
      raw = null;
    }
  }

  if (raw is Map) {
    final out = <String, int>{};
    raw.forEach((k, v) {
      final key = _s(k);
      if (key.isEmpty) return;
      if (v is int) {
        out[key] = v;
      } else if (v is num) {
        out[key] = v.toInt();
      } else {
        final parsed = int.tryParse(v.toString());
        if (parsed != null) out[key] = parsed;
      }
    });
    return out;
  }

  return <String, int>{};
}

int _readStockCount(dynamic obj) {
  // 1) explicit stockCount
  if (obj is Map) {
    final v = obj['stockCount'];
    if (v is int) return v;
    if (v is num) return v.toInt();
  } else {
    try {
      final v = (obj as dynamic).stockCount;
      if (v is int) return v;
      if (v is num) return v.toInt();
    } catch (_) {}
  }

  // 2) products map/list (len) -> stock count
  dynamic products;
  if (obj is Map) {
    products = obj['products'];
  } else {
    try {
      products = (obj as dynamic).products;
    } catch (_) {
      products = null;
    }
  }

  if (products is Map) return products.length;
  if (products is List) return products.length;

  // 3) fallback: no stock info => 0
  return 0;
}
