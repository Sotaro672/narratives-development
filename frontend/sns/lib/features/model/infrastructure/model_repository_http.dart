// frontend/sns/lib/features/model/infrastructure/model_repository_http.dart
//
// Buyer-facing (SNS) Model repository over HTTP.
//
// Expected backend routes (SNS):
// - GET /sns/models?productBlueprintId={id}
//     -> returns either:
//        { "productBlueprintId": "...", "variations": [ ... ], "updatedAt": "..." }
//        OR { "items": [ ... ] }  (tolerated)
//
// - GET /sns/models/variations/{variationId}
//     -> returns a single variation metadata
//
// Fallbacks (for gradual rollout / route differences):
// - GET /sns/models/{variationId}
// - GET /sns/product-blueprints/{productBlueprintId} (to extract modelIds/modelVariationIds then fetch each variation)
//
// NOTE: SNS endpoints are public (no companyId boundary). This repo does NOT attach Firebase ID tokens.

import 'dart:convert';

import 'package:http/http.dart' as http;

/// Lightweight DTOs (self-contained).
class ModelColorDTO {
  final String name;
  final int rgb;

  const ModelColorDTO({required this.name, required this.rgb});

  factory ModelColorDTO.fromJson(Map<String, dynamic> json) {
    final name = (json['name'] ?? '').toString().trim();
    final rgbRaw = json['rgb'];
    final rgb = _asInt(rgbRaw) ?? 0;
    return ModelColorDTO(name: name, rgb: rgb);
  }

  Map<String, dynamic> toJson() => {'name': name, 'rgb': rgb};
}

class ModelVariationDTO {
  final String id;
  final String productBlueprintId;
  final String modelNumber;
  final String size;
  final ModelColorDTO color;
  final Map<String, int> measurements;

  final DateTime? createdAt;
  final String? createdBy;
  final DateTime? updatedAt;
  final String? updatedBy;

  const ModelVariationDTO({
    required this.id,
    required this.productBlueprintId,
    required this.modelNumber,
    required this.size,
    required this.color,
    required this.measurements,
    this.createdAt,
    this.createdBy,
    this.updatedAt,
    this.updatedBy,
  });

  factory ModelVariationDTO.fromJson(Map<String, dynamic> json) {
    final id = (json['id'] ?? json['ID'] ?? '').toString().trim();
    final pbId =
        (json['productBlueprintId'] ?? json['productBlueprintID'] ?? '')
            .toString()
            .trim();
    final modelNumber = (json['modelNumber'] ?? '').toString().trim();
    final size = (json['size'] ?? '').toString().trim();

    final colorJson = (json['color'] is Map)
        ? (json['color'] as Map).cast<String, dynamic>()
        : <String, dynamic>{};
    final color = ModelColorDTO.fromJson(colorJson);

    final measurements = _asMeasurements(json['measurements']);

    return ModelVariationDTO(
      id: id,
      productBlueprintId: pbId,
      modelNumber: modelNumber,
      size: size,
      color: color,
      measurements: measurements,
      createdAt: _asDateTime(json['createdAt']),
      createdBy: _asStringOrNull(json['createdBy']),
      updatedAt: _asDateTime(json['updatedAt']),
      updatedBy: _asStringOrNull(json['updatedBy']),
    );
  }

  Map<String, dynamic> toJson() => {
    'id': id,
    'productBlueprintId': productBlueprintId,
    'modelNumber': modelNumber,
    'size': size,
    'color': color.toJson(),
    'measurements': measurements,
    'createdAt': createdAt?.toIso8601String(),
    'createdBy': createdBy,
    'updatedAt': updatedAt?.toIso8601String(),
    'updatedBy': updatedBy,
  };
}

class ModelDataDTO {
  final String productBlueprintId;
  final List<ModelVariationDTO> variations;
  final DateTime? updatedAt;

  const ModelDataDTO({
    required this.productBlueprintId,
    required this.variations,
    this.updatedAt,
  });

  factory ModelDataDTO.fromJson(Map<String, dynamic> json) {
    final pbId =
        (json['productBlueprintId'] ?? json['productBlueprintID'] ?? '')
            .toString()
            .trim();

    final varsRaw = json['variations'];
    final itemsRaw = json['items'];

    final list = (varsRaw is List)
        ? varsRaw
        : (itemsRaw is List)
        ? itemsRaw
        : const <dynamic>[];

    final variations = list
        .whereType<Map>()
        .map((e) => ModelVariationDTO.fromJson(e.cast<String, dynamic>()))
        .toList();

    return ModelDataDTO(
      productBlueprintId: pbId,
      variations: variations,
      updatedAt: _asDateTime(json['updatedAt']),
    );
  }
}

/// HTTP error wrapper.
class ModelRepositoryHttpException implements Exception {
  final String message;
  final int? statusCode;
  final String? body;

  ModelRepositoryHttpException(this.message, {this.statusCode, this.body});

  @override
  String toString() {
    final sc = statusCode == null ? '' : ' status=$statusCode';
    return 'ModelRepositoryHttpException($message$sc)';
  }
}

class ModelRepositoryHTTP {
  final http.Client _client;
  final String _baseUrl;

  ModelRepositoryHTTP({http.Client? client, String? baseUrl})
    : _client = client ?? http.Client(),
      _baseUrl = _resolveBaseUrl(baseUrl);

  static String _resolveBaseUrl(String? provided) {
    final trimmedProvided = (provided ?? '').trim();
    if (trimmedProvided.isNotEmpty) {
      return _trimTrailingSlash(trimmedProvided);
    }

    // Prefer --dart-define=API_BASE_URL=...
    const envBase = String.fromEnvironment('API_BASE_URL', defaultValue: '');
    if (envBase.trim().isNotEmpty) {
      return _trimTrailingSlash(envBase.trim());
    }

    // Fallback (matches your current Cloud Run service).
    return 'https://narratives-backend-871263659099.asia-northeast1.run.app';
  }

  static String _trimTrailingSlash(String s) {
    if (s.endsWith('/')) return s.substring(0, s.length - 1);
    return s;
  }

  Uri _uri(String path, [Map<String, String>? query]) {
    final p = path.startsWith('/') ? path : '/$path';
    return Uri.parse('$_baseUrl$p').replace(queryParameters: query);
  }

  // ------------------------------------------------------------
  // Public APIs
  // ------------------------------------------------------------

  /// Fetch model variations (metadata list) for a product blueprint.
  ///
  /// Primary:
  /// - GET /sns/models?productBlueprintId=xxx
  ///
  /// Fallback:
  /// - GET /sns/product-blueprints/{id} -> modelIds/modelVariationIds -> GET variations
  Future<List<ModelVariationDTO>> fetchModelVariationsByProductBlueprintId(
    String productBlueprintId,
  ) async {
    final pbId = productBlueprintId.trim();
    if (pbId.isEmpty) {
      throw ModelRepositoryHttpException('productBlueprintId is empty');
    }

    // 1) Primary endpoint
    try {
      final res = await _client.get(
        _uri('/sns/models', {'productBlueprintId': pbId}),
      );
      if (res.statusCode == 200) {
        final jsonMap = _decodeObject(res.body);
        final data = ModelDataDTO.fromJson(jsonMap);
        // If backend omits productBlueprintId in response, tolerate it.
        return data.variations;
      }
      if (res.statusCode != 404) {
        throw ModelRepositoryHttpException(
          'Failed to fetch model variations by blueprintId',
          statusCode: res.statusCode,
          body: res.body,
        );
      }
      // If 404, fall through to fallback.
    } catch (_) {
      // network/parse error: still try fallback below
    }

    // 2) Fallback: product-blueprints -> modelIds -> per-id fetch
    final ids = await _fetchModelIdsFromProductBlueprint(pbId);
    final out = <ModelVariationDTO>[];
    for (final id in ids) {
      try {
        final v = await fetchModelVariationById(id);
        out.add(v);
      } catch (_) {
        // keep going; partial success is better for UX
      }
    }
    return out;
  }

  /// Fetch a single model variation metadata by variationId.
  ///
  /// Primary:
  /// - GET /sns/models/variations/{id}
  ///
  /// Fallback:
  /// - GET /sns/models/{id}
  Future<ModelVariationDTO> fetchModelVariationById(String variationId) async {
    final id = variationId.trim();
    if (id.isEmpty) {
      throw ModelRepositoryHttpException('variationId is empty');
    }

    // primary
    final primary = await _client.get(_uri('/sns/models/variations/$id'));
    if (primary.statusCode == 200) {
      final jsonMap = _decodeObject(primary.body);
      return ModelVariationDTO.fromJson(jsonMap);
    }
    if (primary.statusCode != 404) {
      throw ModelRepositoryHttpException(
        'Failed to fetch model variation',
        statusCode: primary.statusCode,
        body: primary.body,
      );
    }

    // fallback
    final fallback = await _client.get(_uri('/sns/models/$id'));
    if (fallback.statusCode == 200) {
      final jsonMap = _decodeObject(fallback.body);
      return ModelVariationDTO.fromJson(jsonMap);
    }

    throw ModelRepositoryHttpException(
      'Model variation not found',
      statusCode: fallback.statusCode,
      body: fallback.body,
    );
  }

  // ------------------------------------------------------------
  // Internal helpers
  // ------------------------------------------------------------

  Future<List<String>> _fetchModelIdsFromProductBlueprint(
    String productBlueprintId,
  ) async {
    final res = await _client.get(
      _uri('/sns/product-blueprints/$productBlueprintId'),
    );
    if (res.statusCode != 200) {
      throw ModelRepositoryHttpException(
        'Failed to fetch product blueprint for modelIds fallback',
        statusCode: res.statusCode,
        body: res.body,
      );
    }

    final jsonMap = _decodeObject(res.body);

    // Try common keys (you can standardize later)
    final candidates = <dynamic>[
      jsonMap['modelIds'],
      jsonMap['modelVariationIds'],
      jsonMap['variationIds'],
      // nested shapes
      (jsonMap['model'] is Map) ? (jsonMap['model'] as Map)['ids'] : null,
    ];

    for (final c in candidates) {
      if (c is List) {
        return c
            .map((e) => e.toString().trim())
            .where((s) => s.isNotEmpty)
            .toList();
      }
    }

    return <String>[];
  }
}

// ------------------------------------------------------------
// JSON / parsing helpers
// ------------------------------------------------------------

Map<String, dynamic> _decodeObject(String body) {
  final decoded = jsonDecode(body);
  if (decoded is Map<String, dynamic>) return decoded;
  if (decoded is Map) return decoded.cast<String, dynamic>();
  throw ModelRepositoryHttpException('Invalid JSON: expected object');
}

String? _asStringOrNull(dynamic v) {
  final s = (v ?? '').toString().trim();
  return s.isEmpty ? null : s;
}

int? _asInt(dynamic v) {
  if (v == null) return null;
  if (v is int) return v;
  if (v is num) return v.toInt();
  final s = v.toString().trim();
  if (s.isEmpty) return null;
  return int.tryParse(s);
}

DateTime? _asDateTime(dynamic v) {
  if (v == null) return null;
  if (v is DateTime) return v.toUtc();

  // Firestore might send ISO8601 strings via backend JSON
  final s = v.toString().trim();
  if (s.isEmpty) return null;
  final dt = DateTime.tryParse(s);
  return dt?.toUtc();
}

Map<String, int> _asMeasurements(dynamic v) {
  if (v == null) return const <String, int>{};
  if (v is Map) {
    final out = <String, int>{};
    v.forEach((key, value) {
      final k = key.toString().trim();
      if (k.isEmpty) return;
      final n = _asInt(value);
      if (n == null) return;
      out[k] = n;
    });
    return out;
  }
  return const <String, int>{};
}
