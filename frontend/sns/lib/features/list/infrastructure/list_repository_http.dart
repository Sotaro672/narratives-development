// frontend/sns/lib/features/list/infrastructure/list_repository_http.dart
import 'dart:convert';

import 'package:flutter/foundation.dart';
import 'package:http/http.dart' as http;

/// SNS buyer-facing API base URL.
///
/// Priority:
/// 1) --dart-define=API_BASE_URL=https://...
/// 2) (fallback) Cloud Run default (edit as needed)
const String _fallbackBaseUrl =
    'https://narratives-backend-871263659099.asia-northeast1.run.app';

String _resolveApiBase() {
  const fromDefine = String.fromEnvironment('API_BASE_URL');
  final base = (fromDefine.isNotEmpty ? fromDefine : _fallbackBaseUrl).trim();
  return base.endsWith('/') ? base.substring(0, base.length - 1) : base;
}

/// Buyer-facing item (minimum fields needed by SNS).
@immutable
class SnsListItem {
  const SnsListItem({
    required this.id,
    required this.title,
    required this.description,
    required this.image,
    required this.prices,

    // ✅ optional linkage fields
    required this.inventoryId,
    required this.productBlueprintId,
    required this.tokenBlueprintId,
  });

  final String id;
  final String title;
  final String description;

  /// Image URL
  final String image;

  /// prices: [{modelId, price}, ...]
  final List<SnsListPriceRow> prices;

  /// Optional: inventory doc id (e.g. productBlueprintId__tokenBlueprintId)
  final String inventoryId;

  /// Optional: for fallback query
  final String productBlueprintId;
  final String tokenBlueprintId;

  factory SnsListItem.fromJson(Map<String, dynamic> json) {
    final pricesRaw = (json['prices'] as List?) ?? const [];
    final prices = pricesRaw
        .whereType<Map>()
        .map((m) => SnsListPriceRow.fromJson(m.cast<String, dynamic>()))
        .toList();

    String s(dynamic v) => (v ?? '').toString().trim();

    return SnsListItem(
      id: s(json['id']),
      title: s(json['title']),
      description: s(json['description']),
      image: s(json['image']),
      prices: prices,

      // ✅ backend が返していれば使う／無ければ空文字
      inventoryId: s(json['inventoryId']),
      productBlueprintId: s(json['productBlueprintId']),
      tokenBlueprintId: s(json['tokenBlueprintId']),
    );
  }
}

@immutable
class SnsListPriceRow {
  const SnsListPriceRow({required this.modelId, required this.price});

  final String modelId;
  final int price;

  factory SnsListPriceRow.fromJson(Map<String, dynamic> json) {
    final modelId = (json['modelId'] ?? '').toString().trim();

    final rawPrice = json['price'];
    int price = 0;
    if (rawPrice is int) {
      price = rawPrice;
    } else if (rawPrice is num) {
      price = rawPrice.toInt();
    } else if (rawPrice is String) {
      price = int.tryParse(rawPrice.trim()) ?? 0;
    }

    return SnsListPriceRow(modelId: modelId, price: price);
  }
}

/// Index response shape from backend SNS handler.
@immutable
class SnsListIndexResponse {
  const SnsListIndexResponse({
    required this.items,
    required this.totalCount,
    required this.totalPages,
    required this.page,
    required this.perPage,
  });

  final List<SnsListItem> items;
  final int totalCount;
  final int totalPages;
  final int page;
  final int perPage;

  factory SnsListIndexResponse.fromJson(Map<String, dynamic> json) {
    final itemsRaw = (json['items'] as List?) ?? const [];
    final items = itemsRaw
        .whereType<Map>()
        .map((m) => SnsListItem.fromJson(m.cast<String, dynamic>()))
        .toList();

    int asInt(dynamic v, {int def = 0}) {
      if (v is int) return v;
      if (v is num) return v.toInt();
      if (v is String) return int.tryParse(v.trim()) ?? def;
      return def;
    }

    return SnsListIndexResponse(
      items: items,
      totalCount: asInt(json['totalCount']),
      totalPages: asInt(json['totalPages']),
      page: asInt(json['page'], def: 1),
      perPage: asInt(json['perPage'], def: 20),
    );
  }
}

/// Simple HTTP repository for SNS list endpoints.
/// - GET /sns/lists?page=&perPage=
/// - GET /sns/lists/{id}
class ListRepositoryHttp {
  ListRepositoryHttp({http.Client? client}) : _client = client ?? http.Client();

  final http.Client _client;

  String get _base => _resolveApiBase();

  Uri _uri(String path, [Map<String, String>? query]) {
    final p = path.startsWith('/') ? path : '/$path';
    return Uri.parse('$_base$p').replace(queryParameters: query);
  }

  Future<SnsListIndexResponse> fetchLists({
    int page = 1,
    int perPage = 20,
  }) async {
    final uri = _uri('/sns/lists', {
      'page': page.toString(),
      'perPage': perPage.toString(),
    });

    final res = await _client.get(
      uri,
      headers: const {'Accept': 'application/json'},
    );

    final body = res.body;
    if (res.statusCode < 200 || res.statusCode >= 300) {
      throw HttpException(
        statusCode: res.statusCode,
        message: _extractError(body) ?? 'request failed',
        url: uri.toString(),
      );
    }

    final decoded = jsonDecode(body);
    if (decoded is! Map<String, dynamic>) {
      throw FormatException('Invalid JSON shape (expected object)');
    }
    return SnsListIndexResponse.fromJson(decoded);
  }

  Future<SnsListItem> fetchListById(String id) async {
    final listId = id.trim();
    if (listId.isEmpty) {
      throw ArgumentError('id is required');
    }

    final uri = _uri('/sns/lists/$listId');

    final res = await _client.get(
      uri,
      headers: const {'Accept': 'application/json'},
    );

    final body = res.body;
    if (res.statusCode < 200 || res.statusCode >= 300) {
      throw HttpException(
        statusCode: res.statusCode,
        message: _extractError(body) ?? 'request failed',
        url: uri.toString(),
      );
    }

    final decoded = jsonDecode(body);
    if (decoded is! Map<String, dynamic>) {
      throw FormatException('Invalid JSON shape (expected object)');
    }
    return SnsListItem.fromJson(decoded);
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
    } catch (_) {
      // ignore
    }
    return null;
  }
}

@immutable
class HttpException implements Exception {
  const HttpException({
    required this.statusCode,
    required this.message,
    required this.url,
  });

  final int statusCode;
  final String message;
  final String url;

  @override
  String toString() => 'HttpException($statusCode) $message ($url)';
}
