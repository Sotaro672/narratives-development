class ProductBlueprintPatchDTO {
  const ProductBlueprintPatchDTO({required this.items});

  /// flatten 済みの key-value
  final List<ProductBlueprintPatchItemDTO> items;

  factory ProductBlueprintPatchDTO.fromJson(dynamic raw) {
    final items = <ProductBlueprintPatchItemDTO>[];

    void add(String key, dynamic value) {
      final k = key.trim();
      if (k.isEmpty) return;
      items.add(ProductBlueprintPatchItemDTO(key: k, value: _stringify(value)));
    }

    void walk(dynamic v, {String prefix = ''}) {
      if (v == null) {
        add(prefix, null);
        return;
      }

      if (v is Map) {
        final m = Map<String, dynamic>.from(v);
        final keys = m.keys.toList()..sort();
        for (final k in keys) {
          final next = prefix.isEmpty ? k : '$prefix.$k';
          walk(m[k], prefix: next);
        }
        return;
      }

      if (v is List) {
        for (var i = 0; i < v.length; i++) {
          final next = '$prefix[$i]';
          walk(v[i], prefix: next);
        }
        return;
      }

      add(prefix, v);
    }

    // raw が Map なら flatten、Map 以外なら "value" として 1 行化
    if (raw is Map || raw is List) {
      walk(raw, prefix: '');
    } else {
      add('value', raw);
    }

    // key 空の要素を除外
    final normalized = items.where((e) => e.key.trim().isNotEmpty).toList();
    return ProductBlueprintPatchDTO(items: normalized);
  }

  static String _stringify(dynamic v) {
    if (v == null) return '-';
    if (v is String) {
      final s = v.trim();
      return s.isEmpty ? '-' : s;
    }
    if (v is num || v is bool) return v.toString();
    return v.toString();
  }
}

class ProductBlueprintPatchItemDTO {
  const ProductBlueprintPatchItemDTO({required this.key, required this.value});

  final String key;
  final String value;
}
