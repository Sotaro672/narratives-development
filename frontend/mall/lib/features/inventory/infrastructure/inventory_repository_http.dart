// frontend/mall/lib/features/inventory/infrastructure/inventory_repository_http.dart
import 'dart:convert';

import 'package:flutter/foundation.dart';
import 'package:http/http.dart' as http;

// ✅ use shared resolution logic (single source of truth)
import '../../../app/config/api_base.dart';

// ============================================================
// Inventory DTOs (MATCHES backend mall inventory handler)
// ============================================================

@immutable
class MallInventoryModelStock {
  const MallInventoryModelStock({
    required this.accumulation,
    required this.reservedCount,
  });

  /// ✅ backend: stock.{modelId}.accumulation
  final int accumulation;

  /// ✅ backend: stock.{modelId}.reservedCount
  final int reservedCount;

  static int _toInt(dynamic v) {
    if (v == null) return 0;
    if (v is int) return v;
    if (v is double) return v.toInt();
    if (v is num) return v.toInt();
    final s = v.toString().trim();
    if (s.isEmpty) return 0;
    return int.tryParse(s) ?? 0;
  }

  factory MallInventoryModelStock.fromJson(Map<String, dynamic> json) {
    return MallInventoryModelStock(
      accumulation: _toInt(json['accumulation']),
      reservedCount: _toInt(json['reservedCount']),
    );
  }

  /// ✅ UIで使う: availableStock = accumulation - reservedCount
  int get availableStock {
    final v = accumulation - reservedCount;
    return v < 0 ? 0 : v;
  }
}

@immutable
class MallInventoryResponse {
  const MallInventoryResponse({
    required this.id,
    required this.tokenBlueprintId,
    required this.productBlueprintId,
    required this.modelIds,
    required this.stock,
  });

  final String id;
  final String tokenBlueprintId;
  final String productBlueprintId;

  /// ✅ backend: modelIds
  final List<String> modelIds;

  /// ✅ backend: stock (modelId -> {accumulation,reservedCount,...})
  final Map<String, MallInventoryModelStock> stock;

  factory MallInventoryResponse.fromJson(Map<String, dynamic> json) {
    String s(dynamic v) => (v ?? '').toString().trim();

    final id = s(json['id']);
    final tb = s(json['tokenBlueprintId']);
    final pb = s(json['productBlueprintId']);

    // modelIds
    final modelIds = <String>[];
    final modelIdsRaw = json['modelIds'];
    if (modelIdsRaw is List) {
      for (final v in modelIdsRaw) {
        final t = v.toString().trim();
        if (t.isNotEmpty) modelIds.add(t);
      }
    }

    // stock
    final Map<String, MallInventoryModelStock> stock = {};
    final stockRaw = json['stock'];
    if (stockRaw is Map) {
      for (final e in stockRaw.entries) {
        final modelId = e.key.toString().trim();
        if (modelId.isEmpty) continue;

        final v = e.value;
        if (v is Map) {
          stock[modelId] = MallInventoryModelStock.fromJson(
            v.cast<String, dynamic>(),
          );
        } else {
          // handler と一致しない形は「旧式互換」扱いなので、ここではゼロ固定
          stock[modelId] = const MallInventoryModelStock(
            accumulation: 0,
            reservedCount: 0,
          );
        }
      }
    }

    return MallInventoryResponse(
      id: id,
      tokenBlueprintId: tb,
      productBlueprintId: pb,
      modelIds: modelIds,
      stock: stock,
    );
  }
}

// ============================================================
// Models DTOs (/mall/models)
// ============================================================

@immutable
class MallModelVariationDTO {
  const MallModelVariationDTO({
    required this.id,
    required this.productBlueprintId,
    required this.modelNumber,
    required this.size,
    required this.colorName,
    required this.colorRGB,
    required this.measurements,
  });

  final String id;
  final String productBlueprintId;
  final String modelNumber;
  final String size;

  final String colorName;
  final int colorRGB;

  final Map<String, int> measurements;

  static int _toInt(dynamic v) {
    if (v == null) return 0;
    if (v is int) return v;
    if (v is double) return v.toInt();
    if (v is num) return v.toInt();
    final s = v.toString().trim();
    if (s.isEmpty) return 0;
    return int.tryParse(s) ?? 0;
  }

  factory MallModelVariationDTO.fromJson(Map<String, dynamic> json) {
    String s(dynamic v) => (v ?? '').toString().trim();

    final id = s(json['id']);
    final pb = s(json['productBlueprintId']);
    final mn = s(json['modelNumber']);
    final sz = s(json['size']);

    final cn = s(json['colorName']);
    final rgb = _toInt(json['colorRGB']);

    final Map<String, int> measurements = {};
    final m = json['measurements'];
    if (m is Map) {
      for (final e in m.entries) {
        final k = e.key.toString().trim();
        if (k.isEmpty) continue;
        measurements[k] = _toInt(e.value);
      }
    }

    return MallModelVariationDTO(
      id: id,
      productBlueprintId: pb,
      modelNumber: mn,
      size: sz,
      colorName: cn,
      colorRGB: rgb,
      measurements: measurements,
    );
  }
}

@immutable
class _MallModelItemDTO {
  const _MallModelItemDTO({required this.modelId, required this.metadata});

  final String modelId;
  final MallModelVariationDTO metadata;

  factory _MallModelItemDTO.fromJson(Map<String, dynamic> json) {
    final modelId = (json['modelId'] ?? '').toString().trim();
    final metaRaw = json['metadata'];

    final meta = (metaRaw is Map)
        ? MallModelVariationDTO.fromJson(metaRaw.cast<String, dynamic>())
        : MallModelVariationDTO(
            id: modelId,
            productBlueprintId: '',
            modelNumber: '',
            size: '',
            colorName: '',
            colorRGB: 0,
            measurements: const {},
          );

    final fixed = (meta.id.trim().isNotEmpty)
        ? meta
        : MallModelVariationDTO(
            id: modelId,
            productBlueprintId: meta.productBlueprintId,
            modelNumber: meta.modelNumber,
            size: meta.size,
            colorName: meta.colorName,
            colorRGB: meta.colorRGB,
            measurements: meta.measurements,
          );

    return _MallModelItemDTO(modelId: modelId, metadata: fixed);
  }
}

@immutable
class _MallModelListResponseDTO {
  const _MallModelListResponseDTO({
    required this.items,
    required this.totalCount,
    required this.totalPages,
    required this.page,
    required this.perPage,
  });

  final List<_MallModelItemDTO> items;
  final int totalCount;
  final int totalPages;
  final int page;
  final int perPage;

  factory _MallModelListResponseDTO.fromJson(Map<String, dynamic> json) {
    final itemsRaw = json['items'];
    final items = <_MallModelItemDTO>[];
    if (itemsRaw is List) {
      for (final v in itemsRaw) {
        if (v is Map) {
          items.add(_MallModelItemDTO.fromJson(v.cast<String, dynamic>()));
        }
      }
    }

    int asInt(dynamic v) =>
        (v is num) ? v.toInt() : int.tryParse(v.toString()) ?? 0;

    return _MallModelListResponseDTO(
      items: items,
      totalCount: asInt(json['totalCount']),
      totalPages: asInt(json['totalPages']),
      page: asInt(json['page']),
      perPage: asInt(json['perPage']),
    );
  }
}

// ============================================================
// Repository
// ============================================================

class InventoryRepositoryHttp {
  InventoryRepositoryHttp({http.Client? client})
    : _client = client ?? http.Client();

  final http.Client _client;

  String get _base => resolveApiBase();

  Uri _uri(String path, [Map<String, String>? query]) {
    final p = path.startsWith('/') ? path : '/$path';
    return Uri.parse('$_base$p').replace(queryParameters: query);
  }

  /// GET /mall/inventories/{id}
  Future<MallInventoryResponse> fetchInventoryById(String id) async {
    final invId = id.trim();
    if (invId.isEmpty) {
      throw ArgumentError('id is required');
    }

    final uri = _uri('/mall/inventories/$invId');
    final res = await _client.get(
      uri,
      headers: const {'Accept': 'application/json'},
    );

    final body = res.body;
    if (res.statusCode < 200 || res.statusCode >= 300) {
      throw MallHttpException(
        statusCode: res.statusCode,
        message: _extractError(body) ?? 'request failed',
        url: uri.toString(),
      );
    }

    final decoded = jsonDecode(body);
    if (decoded is! Map) {
      throw const FormatException('Invalid JSON shape (expected object)');
    }

    // wrapper 吸収: {data:{...}} を許容
    final m = decoded.cast<String, dynamic>();
    final data = m['data'];
    if (data is Map) {
      return MallInventoryResponse.fromJson(data.cast<String, dynamic>());
    }
    return MallInventoryResponse.fromJson(m);
  }

  /// GET /mall/inventories?productBlueprintId=...&tokenBlueprintId=...
  Future<MallInventoryResponse> fetchInventoryByQuery({
    required String productBlueprintId,
    required String tokenBlueprintId,
  }) async {
    final pb = productBlueprintId.trim();
    final tb = tokenBlueprintId.trim();
    if (pb.isEmpty || tb.isEmpty) {
      throw ArgumentError(
        'productBlueprintId and tokenBlueprintId are required',
      );
    }

    final uri = _uri('/mall/inventories', {
      'productBlueprintId': pb,
      'tokenBlueprintId': tb,
    });

    final res = await _client.get(
      uri,
      headers: const {'Accept': 'application/json'},
    );

    final body = res.body;
    if (res.statusCode < 200 || res.statusCode >= 300) {
      throw MallHttpException(
        statusCode: res.statusCode,
        message: _extractError(body) ?? 'request failed',
        url: uri.toString(),
      );
    }

    final decoded = jsonDecode(body);
    if (decoded is! Map) {
      throw const FormatException('Invalid JSON shape (expected object)');
    }

    final m = decoded.cast<String, dynamic>();
    final data = m['data'];
    if (data is Map) {
      return MallInventoryResponse.fromJson(data.cast<String, dynamic>());
    }
    return MallInventoryResponse.fromJson(m);
  }

  /// GET /mall/models?productBlueprintId=...
  Future<List<MallModelVariationDTO>> fetchModelsByProductBlueprintId(
    String productBlueprintId, {
    int page = 1,
    int perPage = 200,
  }) async {
    final pb = productBlueprintId.trim();
    if (pb.isEmpty) {
      throw ArgumentError('productBlueprintId is required');
    }

    final p = page <= 0 ? 1 : page;
    final pp = (perPage <= 0) ? 200 : (perPage > 200 ? 200 : perPage);

    final uri = _uri('/mall/models', {
      'productBlueprintId': pb,
      'page': '$p',
      'perPage': '$pp',
    });

    final res = await _client.get(
      uri,
      headers: const {'Accept': 'application/json'},
    );

    final body = res.body;
    if (res.statusCode < 200 || res.statusCode >= 300) {
      throw MallHttpException(
        statusCode: res.statusCode,
        message: _extractError(body) ?? 'request failed',
        url: uri.toString(),
      );
    }

    final decoded = jsonDecode(body);
    if (decoded is! Map) {
      throw const FormatException('Invalid JSON shape (expected object)');
    }

    final root = decoded.cast<String, dynamic>();
    final unwrapped = (root['data'] is Map)
        ? (root['data'] as Map).cast<String, dynamic>()
        : root;

    final dto = _MallModelListResponseDTO.fromJson(unwrapped);
    return dto.items.map((it) => it.metadata).toList();
  }

  void dispose() {
    _client.close();
  }

  String? _extractError(String body) {
    try {
      final decoded = jsonDecode(body);
      if (decoded is Map && decoded['error'] != null) {
        return decoded['error'].toString();
      }
      if (decoded is Map && decoded['message'] != null) {
        return decoded['message'].toString();
      }
    } catch (_) {
      // ignore
    }
    return null;
  }
}

@immutable
class MallHttpException implements Exception {
  const MallHttpException({
    required this.statusCode,
    required this.message,
    required this.url,
  });

  final int statusCode;
  final String message;
  final String url;

  @override
  String toString() => 'MallHttpException($statusCode) $message ($url)';
}
