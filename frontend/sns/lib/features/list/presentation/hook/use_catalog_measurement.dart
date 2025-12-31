// frontend/sns/lib/features/list/presentation/hook/use_catalog_measurement.dart

/// 1行（サイズ×採寸値）
class CatalogMeasurementRowVM {
  const CatalogMeasurementRowVM({
    required this.size,
    required this.colorName,
    required this.measurements,
  });

  final String size;
  final String colorName;
  final Map<String, int> measurements;
}

/// テーブル全体
class CatalogMeasurementVM {
  const CatalogMeasurementVM({
    required this.title,
    required this.keys,
    required this.rows,
    required this.showColor,
  });

  final String title;
  final List<String> keys;
  final List<CatalogMeasurementRowVM> rows;
  final bool showColor;

  bool get hasRows => rows.isNotEmpty;
  bool get hasKeys => keys.isNotEmpty;
}

/// ✅ UseCatalog 側で使う型名（VMと同一でOK）
typedef CatalogMeasurementTable = CatalogMeasurementVM;

/// 採寸テーブルの組み立て（ロジック）
class UseCatalogMeasurement {
  const UseCatalogMeasurement();

  /// ✅ 互換のため（持つリソースがないのでno-op）
  void dispose() {}

  /// ✅ UseCatalog 側が呼ぶ想定のAPI
  CatalogMeasurementTable compute({
    required List<dynamic>? models,
    String title = '採寸（サイズ別）',
  }) {
    return build(models: models ?? const [], title: title);
  }

  /// ✅ CatalogMeasurementCard（style側）から呼ぶVMビルダー
  CatalogMeasurementVM build({
    required List<dynamic> models,
    required String title,
  }) {
    final rows = <CatalogMeasurementRowVM>[];
    final keySet = <String>{};

    for (final raw in models) {
      final meta = _unwrapMetadata(raw);

      final size = _pickSize(meta);
      final colorName = _pickColorName(meta);
      final meas = _pickMeasurements(meta);

      for (final k in meas.keys) {
        final s = k.trim();
        if (s.isNotEmpty) keySet.add(s);
      }

      rows.add(
        CatalogMeasurementRowVM(
          size: size.isNotEmpty ? size : '(未設定)',
          colorName: colorName,
          measurements: meas,
        ),
      );
    }

    final keys = keySet.toList()..sort();
    final showColor = rows.any((r) => r.colorName.trim().isNotEmpty);

    // size -> colorName の順に安定ソート
    rows.sort((a, b) {
      final s = a.size.compareTo(b.size);
      if (s != 0) return s;
      return a.colorName.compareTo(b.colorName);
    });

    final t = title.trim();
    return CatalogMeasurementVM(
      title: t.isNotEmpty ? t : '採寸（サイズ別）',
      keys: keys,
      rows: rows,
      showColor: showColor,
    );
  }

  // ------------------------------------------------------------
  // helpers
  // ------------------------------------------------------------

  static String _s(dynamic v) => (v ?? '').toString().trim();

  /// /sns/models の item 形式: { modelId, metadata: {...} } を剥がす
  static dynamic _unwrapMetadata(dynamic raw) {
    if (raw == null) return null;

    if (raw is Map) {
      final meta = raw['metadata'] ?? raw['Metadata'];
      return meta ?? raw;
    }

    // DTOオブジェクト側に metadata があるケースも吸収（無ければそのまま）
    try {
      final meta = (raw as dynamic).metadata;
      if (meta != null) return meta;
    } catch (_) {}

    return raw;
  }

  static String _pickSize(dynamic meta) {
    if (meta == null) return '';

    if (meta is Map) {
      return _s(meta['size'] ?? meta['Size']);
    }

    try {
      return _s((meta as dynamic).size);
    } catch (_) {}

    return '';
  }

  static String _pickColorName(dynamic meta) {
    if (meta == null) return '';

    // flat: colorName
    if (meta is Map) {
      final flat = _s(meta['colorName'] ?? meta['ColorName']);
      if (flat.isNotEmpty) return flat;

      // nested: color: { name, rgb }
      final c = meta['color'] ?? meta['Color'];
      if (c is Map) {
        final n = _s(c['name'] ?? c['Name']);
        if (n.isNotEmpty) return n;
      }
      return '';
    }

    try {
      final flat = _s((meta as dynamic).colorName);
      if (flat.isNotEmpty) return flat;
    } catch (_) {}

    try {
      final c = (meta as dynamic).color;
      if (c != null) {
        try {
          final n = _s((c as dynamic).name);
          if (n.isNotEmpty) return n;
        } catch (_) {}
      }
    } catch (_) {}

    return '';
  }

  static int _toInt(dynamic v) {
    if (v == null) return 0;
    if (v is int) return v;
    if (v is double) return v.toInt();
    if (v is num) return v.toInt();
    return int.tryParse(v.toString()) ?? 0;
  }

  static Map<String, int> _pickMeasurements(dynamic meta) {
    if (meta == null) return <String, int>{};

    dynamic m;

    if (meta is Map) {
      m = meta['measurements'] ?? meta['Measurements'];
    } else {
      try {
        m = (meta as dynamic).measurements;
      } catch (_) {
        m = null;
      }
    }

    if (m is! Map) return <String, int>{};

    final out = <String, int>{};
    m.forEach((k, val) {
      final key = _s(k);
      if (key.isEmpty) return;
      out[key] = _toInt(val);
    });
    return out;
  }
}
