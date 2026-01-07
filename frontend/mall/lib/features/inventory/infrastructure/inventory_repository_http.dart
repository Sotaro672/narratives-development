// frontend/sns/lib/features/inventory/infrastructure/inventory_repository_http.dart
import 'dart:convert';

import 'package:flutter/foundation.dart';
import 'package:http/http.dart' as http;

/// Buyer-facing API base URL.
///
/// Priority:
/// 1) --dart-define=API_BASE_URL=https://...
/// 2) --dart-define=API_BASE=https://...      (backward compatible)
/// 3) fallback
const String _fallbackBaseUrl =
    'https://narratives-backend-871263659099.asia-northeast1.run.app';

/// ✅ Public so other repositories can reuse the exact same logic.
String resolveSnsApiBase() => _resolveApiBase();

String _resolveApiBase() {
  const fromDefineUrl = String.fromEnvironment('API_BASE_URL');
  const fromDefine = String.fromEnvironment('API_BASE');

  final raw =
      (fromDefineUrl.trim().isNotEmpty
              ? fromDefineUrl
              : (fromDefine.trim().isNotEmpty ? fromDefine : _fallbackBaseUrl))
          .trim();

  return raw.endsWith('/') ? raw.substring(0, raw.length - 1) : raw;
}

// ============================================================
// Inventory DTOs
// ============================================================

@immutable
class SnsInventoryModelStock {
  const SnsInventoryModelStock({required this.products});

  /// productId -> true
  /// ※ backend が products を返さない（stockKeys only）場合は空になる
  final Map<String, bool> products;

  factory SnsInventoryModelStock.fromJson(Map<String, dynamic> json) {
    final rawProducts = json['products'];
    final Map<String, bool> products = {};
    if (rawProducts is Map) {
      for (final e in rawProducts.entries) {
        final k = e.key.toString().trim();
        final v = e.value;
        if (k.isEmpty) continue;
        products[k] = v == true;
      }
    }
    return SnsInventoryModelStock(products: products);
  }
}

@immutable
class SnsInventoryResponse {
  const SnsInventoryResponse({
    required this.id,
    required this.tokenBlueprintId,
    required this.productBlueprintId,
    required this.modelIds,
    required this.stockKeys,
    required this.stock,
  });

  final String id;
  final String tokenBlueprintId;
  final String productBlueprintId;

  /// UI が “モデル一覧” 表示に使う想定
  /// - backend が modelIds を返さない場合は stockKeys から補完する
  final List<String> modelIds;

  /// ✅ stockKeys（modelId の集合）
  /// - backend が stockKeys-only を返す場合もここが埋まる
  final List<String> stockKeys;

  /// modelId -> stock detail（products）
  /// - backend が stockKeys-only の場合は、キーだけ拾って空 products を入れる
  final Map<String, SnsInventoryModelStock> stock;

  factory SnsInventoryResponse.fromJson(Map<String, dynamic> json) {
    String s(dynamic v) => (v ?? '').toString().trim();

    final id = s(json['id']);

    // ✅ server-side field-name variance absorption
    final tb = s(json['tokenBlueprintId'] ?? json['tokenBlueprintID']);
    final pb = s(json['productBlueprintId'] ?? json['productBlueprintID']);

    // -------------------------
    // stock / stockKeys
    // -------------------------
    final Map<String, SnsInventoryModelStock> stock = {};
    final List<String> stockKeys = [];

    final stockRaw = json['stock'];
    if (stockRaw is Map) {
      for (final e in stockRaw.entries) {
        final modelId = e.key.toString().trim();
        if (modelId.isEmpty) continue;

        stockKeys.add(modelId);

        final v = e.value;
        if (v is Map) {
          stock[modelId] = SnsInventoryModelStock.fromJson(
            v.cast<String, dynamic>(),
          );
        } else {
          // stockKeys-only（value が bool/int など）でもキーだけ拾って空の products を入れる
          stock[modelId] = const SnsInventoryModelStock(products: {});
        }
      }
    } else if (stockRaw is List) {
      // 念のため: stock が配列で返るケース
      for (final v in stockRaw) {
        final modelId = v.toString().trim();
        if (modelId.isEmpty) continue;
        stockKeys.add(modelId);
        stock[modelId] = const SnsInventoryModelStock(products: {});
      }
    }

    final normalizedStockKeys = _uniqPreserveOrder(stockKeys);

    // -------------------------
    // modelIds（なければ stockKeys から補完）
    // -------------------------
    final modelIdsRaw = json['modelIds'] ?? json['modelIDs'];
    final modelIds = <String>[];
    if (modelIdsRaw is List) {
      for (final v in modelIdsRaw) {
        final t = v.toString().trim();
        if (t.isNotEmpty) modelIds.add(t);
      }
    }
    final normalizedModelIds = modelIds.isNotEmpty
        ? _uniqPreserveOrder(modelIds)
        : normalizedStockKeys;

    return SnsInventoryResponse(
      id: id,
      tokenBlueprintId: tb,
      productBlueprintId: pb,
      modelIds: normalizedModelIds,
      stockKeys: normalizedStockKeys,
      stock: stock,
    );
  }

  static List<String> _uniqPreserveOrder(List<String> xs) {
    final seen = <String>{};
    final out = <String>[];
    for (final x in xs) {
      final t = x.trim();
      if (t.isEmpty) continue;
      if (seen.add(t)) out.add(t);
    }
    return out;
  }
}

// ============================================================
// Models DTOs (/mall/models)
// - backend log: dto.keys=[id productBlueprintId modelNumber size colorName colorRGB measurements]
// ============================================================

@immutable
class SnsModelVariationDTO {
  const SnsModelVariationDTO({
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

  /// ✅ flattened (catalogcolor を統合した形)
  final String colorName;

  /// ✅ backend: colorRGB (int)
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

  factory SnsModelVariationDTO.fromJson(Map<String, dynamic> json) {
    String s(dynamic v) => (v ?? '').toString().trim();

    final id = s(json['id'] ?? json['ID']);
    final pb = s(json['productBlueprintId'] ?? json['productBlueprintID']);
    final mn = s(json['modelNumber']);
    final sz = s(json['size']);

    final cn = s(json['colorName'] ?? json['ColorName']);

    // ✅ backend uses "colorRGB" (but tolerate variants)
    final rgb = _toInt(
      json['colorRGB'] ??
          json['colorRgb'] ??
          json['ColorRGB'] ??
          json['ColorRgb'] ??
          json['rgb'],
    );

    final Map<String, int> measurements = {};
    final m =
        json['measurements'] ?? json['Measurements'] ?? json['measurement'];
    if (m is Map) {
      for (final e in m.entries) {
        final k = e.key.toString().trim();
        if (k.isEmpty) continue;
        measurements[k] = _toInt(e.value);
      }
    }

    return SnsModelVariationDTO(
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
class _SnsModelItemDTO {
  const _SnsModelItemDTO({required this.modelId, required this.metadata});

  final String modelId;
  final SnsModelVariationDTO metadata;

  factory _SnsModelItemDTO.fromJson(Map<String, dynamic> json) {
    final modelId = (json['modelId'] ?? '').toString().trim();
    final metaRaw = json['metadata'];

    // ✅ metadata は Map でも、すでに flatten 済みの shape を想定
    final meta = (metaRaw is Map)
        ? SnsModelVariationDTO.fromJson(metaRaw.cast<String, dynamic>())
        : SnsModelVariationDTO(
            id: modelId,
            productBlueprintId: '',
            modelNumber: '',
            size: '',
            colorName: '',
            colorRGB: 0,
            measurements: const {},
          );

    // id が空なら modelId で補完
    final fixed = (meta.id.trim().isNotEmpty)
        ? meta
        : SnsModelVariationDTO(
            id: modelId,
            productBlueprintId: meta.productBlueprintId,
            modelNumber: meta.modelNumber,
            size: meta.size,
            colorName: meta.colorName,
            colorRGB: meta.colorRGB,
            measurements: meta.measurements,
          );

    return _SnsModelItemDTO(modelId: modelId, metadata: fixed);
  }
}

@immutable
class _SnsModelListResponseDTO {
  const _SnsModelListResponseDTO({
    required this.items,
    required this.totalCount,
    required this.totalPages,
    required this.page,
    required this.perPage,
  });

  final List<_SnsModelItemDTO> items;
  final int totalCount;
  final int totalPages;
  final int page;
  final int perPage;

  factory _SnsModelListResponseDTO.fromJson(Map<String, dynamic> json) {
    final itemsRaw = json['items'];
    final items = <_SnsModelItemDTO>[];
    if (itemsRaw is List) {
      for (final v in itemsRaw) {
        if (v is Map) {
          items.add(_SnsModelItemDTO.fromJson(v.cast<String, dynamic>()));
        }
      }
    }

    int asInt(dynamic v) =>
        (v is num) ? v.toInt() : int.tryParse(v.toString()) ?? 0;

    return _SnsModelListResponseDTO(
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

  String get _base => _resolveApiBase();

  Uri _uri(String path, [Map<String, String>? query]) {
    final p = path.startsWith('/') ? path : '/$path';
    return Uri.parse('$_base$p').replace(queryParameters: query);
  }

  /// GET /mall/inventories/{id}
  Future<SnsInventoryResponse> fetchInventoryById(String id) async {
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
      throw SnsHttpException(
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
      return SnsInventoryResponse.fromJson(data.cast<String, dynamic>());
    }
    return SnsInventoryResponse.fromJson(m);
  }

  /// GET /mall/inventories?productBlueprintId=...&tokenBlueprintId=...
  Future<SnsInventoryResponse> fetchInventoryByQuery({
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
      throw SnsHttpException(
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
      return SnsInventoryResponse.fromJson(data.cast<String, dynamic>());
    }
    return SnsInventoryResponse.fromJson(m);
  }

  /// ✅ GET /mall/models?productBlueprintId=...   (buyer-facing)
  Future<List<SnsModelVariationDTO>> fetchModelsByProductBlueprintId(
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
      throw SnsHttpException(
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
    final root = decoded.cast<String, dynamic>();
    final unwrapped = (root['data'] is Map)
        ? (root['data'] as Map).cast<String, dynamic>()
        : root;

    final dto = _SnsModelListResponseDTO.fromJson(unwrapped);
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
class SnsHttpException implements Exception {
  const SnsHttpException({
    required this.statusCode,
    required this.message,
    required this.url,
  });

  final int statusCode;
  final String message;
  final String url;

  @override
  String toString() => 'SnsHttpException($statusCode) $message ($url)';
}
