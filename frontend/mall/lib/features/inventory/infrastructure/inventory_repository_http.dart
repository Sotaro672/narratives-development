// frontend\mall\lib\features\inventory\infrastructure\inventory_repository_http.dart
import 'dart:convert';

import 'package:flutter/foundation.dart';
import 'package:http/http.dart' as http;

// ✅ use shared resolution logic (single source of truth)
import '../../../app/config/api_base.dart';

// ============================================================
// Inventory DTOs
// ============================================================

@immutable
class MallInventoryModelStock {
  const MallInventoryModelStock({required this.products});

  /// productId -> true
  /// ※ backend が products を返さない（stockKeys only）場合は空になる
  final Map<String, bool> products;

  factory MallInventoryModelStock.fromJson(Map<String, dynamic> json) {
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
    return MallInventoryModelStock(products: products);
  }
}

@immutable
class MallInventoryResponse {
  const MallInventoryResponse({
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
  final Map<String, MallInventoryModelStock> stock;

  factory MallInventoryResponse.fromJson(Map<String, dynamic> json) {
    String s(dynamic v) => (v ?? '').toString().trim();

    final id = s(json['id']);

    // ✅ server-side field-name variance absorption
    final tb = s(json['tokenBlueprintId'] ?? json['tokenBlueprintID']);
    final pb = s(json['productBlueprintId'] ?? json['productBlueprintID']);

    // -------------------------
    // stock / stockKeys
    // -------------------------
    final Map<String, MallInventoryModelStock> stock = {};
    final List<String> stockKeys = [];

    final stockRaw = json['stock'];
    if (stockRaw is Map) {
      for (final e in stockRaw.entries) {
        final modelId = e.key.toString().trim();
        if (modelId.isEmpty) continue;

        stockKeys.add(modelId);

        final v = e.value;
        if (v is Map) {
          stock[modelId] = MallInventoryModelStock.fromJson(
            v.cast<String, dynamic>(),
          );
        } else {
          // stockKeys-only（value が bool/int など）でもキーだけ拾って空の products を入れる
          stock[modelId] = const MallInventoryModelStock(products: {});
        }
      }
    } else if (stockRaw is List) {
      // 念のため: stock が配列で返るケース
      for (final v in stockRaw) {
        final modelId = v.toString().trim();
        if (modelId.isEmpty) continue;
        stockKeys.add(modelId);
        stock[modelId] = const MallInventoryModelStock(products: {});
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

    return MallInventoryResponse(
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

  factory MallModelVariationDTO.fromJson(Map<String, dynamic> json) {
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

    // ✅ metadata は Map でも、すでに flatten 済みの shape を想定
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

    // id が空なら modelId で補完
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

  // ✅ base url is resolved from shared config
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

    // wrapper 吸収: {data:{...}} を許容
    final m = decoded.cast<String, dynamic>();
    final data = m['data'];
    if (data is Map) {
      return MallInventoryResponse.fromJson(data.cast<String, dynamic>());
    }
    return MallInventoryResponse.fromJson(m);
  }

  /// ✅ GET /mall/models?productBlueprintId=...   (buyer-facing)
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

    // wrapper 吸収: {data:{...}} を許容
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
