//frontend\sns\lib\features\model\infrastructure\model_repository_http.dart
import 'dart:convert';

import 'package:http/http.dart' as http;

/// Buyer-facing model repository
/// - GET /mall/models?productBlueprintId=...
///
/// Backend response shape may be one of:
/// 1) [ {...}, {...} ]
/// 2) { "items": [ ... ] }
/// 3) { "modelVariations": [ ... ] }
/// 4) { "variations": [ ... ] }
/// 5) { "data": { "items": [ ... ] } } など
///
/// ✅ additionally supports backend shape:
/// { "items": [ { "modelId": "...", "metadata": { ...measurements... } }, ... ] }
class ModelRepositoryHTTP {
  ModelRepositoryHTTP({http.Client? client})
    : _client = client ?? http.Client();

  final http.Client _client;

  // NOTE: optional. (CatalogPage側で dispose しない方針でもOK)
  void dispose() {
    _client.close();
  }

  static const String _fallbackBaseUrl =
      'https://narratives-backend-871263659099.asia-northeast1.run.app';

  /// ✅ unify with other repos: --dart-define=API_BASE_URL=...
  static String _resolveApiBase() {
    const env = String.fromEnvironment('API_BASE_URL');
    final base = (env.trim().isNotEmpty ? env.trim() : _fallbackBaseUrl).trim();
    return base.endsWith('/') ? base.substring(0, base.length - 1) : base;
  }

  /// ✅ backend migrated sns -> mall (legacy removed)
  static const String _apiPrefix = '/mall';

  static Uri _buildUri(String path, [Map<String, String>? query]) {
    final base = _resolveApiBase();
    final p = path.startsWith('/') ? path : '/$path';
    final uri = Uri.parse('$base$p');
    return query == null ? uri : uri.replace(queryParameters: query);
  }

  Future<List<ModelVariationDTO>> fetchModelVariationsByProductBlueprintId(
    String productBlueprintId,
  ) async {
    final pbId = productBlueprintId.trim();
    if (pbId.isEmpty) {
      throw Exception('models: productBlueprintId is empty');
    }

    final uri = _buildUri('$_apiPrefix/models', {'productBlueprintId': pbId});
    final res = await _client.get(
      uri,
      headers: const {'accept': 'application/json'},
    );

    if (res.statusCode < 200 || res.statusCode >= 300) {
      throw Exception('models: http ${res.statusCode} body=${res.body}');
    }

    final decoded = jsonDecode(res.body);
    final list = _extractList(decoded);

    return list
        .whereType<Map>()
        .map((e) => ModelVariationDTO.fromJson(e.cast<String, dynamic>()))
        .toList();
  }

  /// Extract list from many possible response shapes.
  List<dynamic> _extractList(dynamic decoded) {
    if (decoded is List) return decoded;

    if (decoded is Map) {
      final m = decoded.cast<String, dynamic>();

      // common keys
      final directKeys = ['items', 'modelVariations', 'variations', 'models'];
      for (final k in directKeys) {
        final v = m[k];
        if (v is List) return v;
      }

      // nested "data"
      final data = m['data'];
      if (data is Map) {
        final dm = data.cast<String, dynamic>();
        for (final k in directKeys) {
          final v = dm[k];
          if (v is List) return v;
        }
      }
    }

    // fallback
    return <dynamic>[];
  }
}

// ============================================================
// DTOs (must match backend Catalog/SNSModelVariationDTO shape)
// ============================================================

class ModelVariationDTO {
  const ModelVariationDTO({
    required this.id,
    required this.productBlueprintId,
    required this.modelNumber,
    required this.size,
    required this.color,
    required this.measurements,
  });

  final String id;
  final String productBlueprintId;
  final String modelNumber;
  final String size;
  final ModelColorDTO color;
  final Map<String, int> measurements;

  static String _s(dynamic v) => (v ?? '').toString().trim();

  static int _toInt(dynamic v) {
    if (v is int) return v;
    if (v is double) return v.toInt();
    if (v is num) return v.toInt();
    return int.tryParse(v?.toString() ?? '') ?? 0;
  }

  static Map<String, int> _measurements(dynamic v) {
    if (v is! Map) return <String, int>{};
    final out = <String, int>{};
    v.forEach((k, val) {
      final key = _s(k);
      if (key.isEmpty) return;
      out[key] = _toInt(val);
    });
    return out;
  }

  /// ✅ backend が返す色の取り方を統一吸収
  /// - 旧: color: { name, rgb }
  /// - 新: colorName, colorRGB（フラット）
  static ModelColorDTO _color(Map<String, dynamic> json) {
    // 1) 旧/別実装: color object
    final colorRaw = json['color'] ?? json['Color'];
    if (colorRaw is Map) {
      return ModelColorDTO.fromJson(colorRaw.cast<String, dynamic>());
    }

    // 2) ✅ 現状backend: flat fields
    final name = _s(json['colorName'] ?? json['ColorName']);
    final rgb = _toInt(
      json['colorRGB'] ??
          json['colorRgb'] ??
          json['ColorRGB'] ??
          json['ColorRgb'],
    );

    if (name.isEmpty && rgb == 0) {
      return const ModelColorDTO(name: '', rgb: 0);
    }
    return ModelColorDTO(name: name, rgb: rgb);
  }

  factory ModelVariationDTO.fromJson(Map<String, dynamic> json) {
    // ✅ unwrap: backend shape { modelId, metadata: {...} } を吸収
    final metaRaw = json['metadata'];
    final Map<String, dynamic> src = (metaRaw is Map)
        ? metaRaw.cast<String, dynamic>()
        : json;

    // tolerate field-name variants (prefer src, fallback to outer json)
    final id = _s(
      src['id'] ??
          src['ID'] ??
          src['variationId'] ??
          json['modelId'] ??
          json['ModelID'],
    );

    final pbId = _s(
      src['productBlueprintId'] ??
          src['productBlueprintID'] ??
          src['product_blueprint_id'],
    );

    return ModelVariationDTO(
      id: id,
      productBlueprintId: pbId,
      modelNumber: _s(src['modelNumber']),
      size: _s(src['size']),
      color: _color(src),
      measurements: _measurements(
        src['measurements'] ??
            src['Measurements'] ??
            src['measurement'] ??
            src['Measurement'],
      ),
    );
  }
}

class ModelColorDTO {
  const ModelColorDTO({required this.name, required this.rgb});

  final String name;
  final int rgb;

  static String _s(dynamic v) => (v ?? '').toString().trim();

  static int _toInt(dynamic v) {
    if (v is int) return v;
    if (v is double) return v.toInt();
    if (v is num) return v.toInt();
    return int.tryParse(v?.toString() ?? '') ?? 0;
  }

  factory ModelColorDTO.fromJson(Map<String, dynamic> json) {
    // tolerate variants too
    return ModelColorDTO(
      name: _s(json['name'] ?? json['Name']),
      rgb: _toInt(json['rgb'] ?? json['RGB']),
    );
  }
}
