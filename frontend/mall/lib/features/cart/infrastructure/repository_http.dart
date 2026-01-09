//frontend\mall\lib\features\cart\infrastructure\repository_http.dart
import 'dart:convert';

import 'package:http/http.dart' as http;

import 'api.dart';

// ✅ 呼び出し側（use_payment.dart 等）が repository_http.dart だけ import しても
// CartHttpException を型として参照できるようにする
export 'api.dart' show CartHttpException;

/// Buyer-facing Cart repository (HTTP).
///
/// Backend endpoints (CartHandler):
/// - GET    /mall/me/cart?avatarId=...
/// - POST   /mall/me/cart/items           body: {avatarId, inventoryId, listId, modelId, qty}
/// - PUT    /mall/me/cart/items           body: {avatarId, inventoryId, listId, modelId, qty}
/// - DELETE /mall/me/cart/items           body: {avatarId, inventoryId, listId, modelId}
/// - DELETE /mall/me/cart?avatarId=...
///
/// NOTE:
/// - Web(CORS) ではカスタムヘッダ（例: x-avatar-id）を送ると preflight が走り、
///   backend 側で Access-Control-Allow-Headers に許可が無いとブロックされます。
/// - avatarId は **query/body のみ**で送ります（ヘッダには入れない）。
class CartRepositoryHttp {
  CartRepositoryHttp({http.Client? client, String? apiBase})
    : _api = CartApiClient(client: client, apiBase: apiBase);

  final CartApiClient _api;

  void dispose() {
    _api.dispose();
  }

  // ----------------------------
  // Public API (CartDTO only)
  // ----------------------------

  /// GET /mall/me/cart?avatarId=...
  Future<CartDTO> fetchCart({required String avatarId}) async {
    final aid = avatarId.trim();
    if (aid.isEmpty) throw ArgumentError('avatarId is required');

    final uri = _api.uri('/mall/me/cart', qp: {'avatarId': aid});
    final res = await _api.sendAuthed('GET', uri);

    return _decodeCart(res);
  }

  /// POST /mall/me/cart/items
  Future<CartDTO> addItem({
    required String avatarId,
    required String inventoryId,
    required String listId,
    required String modelId,
    int qty = 1,
  }) async {
    final aid = avatarId.trim();
    final invId = inventoryId.trim();
    final lid = listId.trim();
    final mid = modelId.trim();

    if (aid.isEmpty) throw ArgumentError('avatarId is required');
    if (invId.isEmpty) throw ArgumentError('inventoryId is required');
    if (lid.isEmpty) throw ArgumentError('listId is required');
    if (mid.isEmpty) throw ArgumentError('modelId is required');
    if (qty <= 0) throw ArgumentError('qty must be >= 1');

    // ✅ body only (no avatarId in query)
    final uri = _api.uri('/mall/me/cart/items');
    final bodyMap = <String, dynamic>{
      'avatarId': aid,
      'inventoryId': invId,
      'listId': lid,
      'modelId': mid,
      'qty': qty,
    };

    final res = await _api.sendAuthed('POST', uri, body: jsonEncode(bodyMap));
    return _decodeCart(res);
  }

  /// PUT /mall/me/cart/items
  /// - qty <= 0 is treated as remove by backend.
  Future<CartDTO> setItemQty({
    required String avatarId,
    required String inventoryId,
    required String listId,
    required String modelId,
    required int qty,
  }) async {
    final aid = avatarId.trim();
    final invId = inventoryId.trim();
    final lid = listId.trim();
    final mid = modelId.trim();

    if (aid.isEmpty) throw ArgumentError('avatarId is required');
    if (invId.isEmpty) throw ArgumentError('inventoryId is required');
    if (lid.isEmpty) throw ArgumentError('listId is required');
    if (mid.isEmpty) throw ArgumentError('modelId is required');

    final uri = _api.uri('/mall/me/cart/items');
    final bodyMap = <String, dynamic>{
      'avatarId': aid,
      'inventoryId': invId,
      'listId': lid,
      'modelId': mid,
      'qty': qty,
    };

    final res = await _api.sendAuthed('PUT', uri, body: jsonEncode(bodyMap));
    return _decodeCart(res);
  }

  /// DELETE /mall/me/cart/items
  Future<CartDTO> removeItem({
    required String avatarId,
    required String inventoryId,
    required String listId,
    required String modelId,
  }) async {
    final aid = avatarId.trim();
    final invId = inventoryId.trim();
    final lid = listId.trim();
    final mid = modelId.trim();

    if (aid.isEmpty) throw ArgumentError('avatarId is required');
    if (invId.isEmpty) throw ArgumentError('inventoryId is required');
    if (lid.isEmpty) throw ArgumentError('listId is required');
    if (mid.isEmpty) throw ArgumentError('modelId is required');

    final uri = _api.uri('/mall/me/cart/items');
    final bodyMap = <String, dynamic>{
      'avatarId': aid,
      'inventoryId': invId,
      'listId': lid,
      'modelId': mid,
    };

    final res = await _api.sendAuthed('DELETE', uri, body: jsonEncode(bodyMap));
    return _decodeCart(res);
  }

  /// DELETE /mall/me/cart?avatarId=...
  Future<void> clearCart({required String avatarId}) async {
    final aid = avatarId.trim();
    if (aid.isEmpty) throw ArgumentError('avatarId is required');

    final uri = _api.uri('/mall/me/cart', qp: {'avatarId': aid});
    final res = await _api.sendAuthed('DELETE', uri);

    if (res.statusCode == 204) return;

    // 200 + JSONで返す実装も吸収（壊さない）
    if (res.statusCode >= 200 && res.statusCode < 300) return;

    _api.throwHttpError(res);
  }

  // ----------------------------
  // DTO + parsing (CartDTO only)
  // ----------------------------

  CartDTO _decodeCart(http.Response res) {
    if (res.statusCode >= 200 && res.statusCode < 300) {
      final map = _api.decodeJsonMap(res.body);
      final data = _api.unwrapData(map);
      return CartDTO.fromJson(data);
    }

    _api.throwHttpError(res);
  }
}

// ----------------------------
// Models (CartDTO only)
// ----------------------------

class CartDTO {
  CartDTO({
    required this.avatarId,
    required this.items,
    required this.createdAt,
    required this.updatedAt,
    required this.expiresAt,
  });

  final String avatarId;

  /// itemKey -> item
  final Map<String, CartItemDTO> items;

  final DateTime? createdAt;
  final DateTime? updatedAt;
  final DateTime? expiresAt;

  int totalQty() {
    var sum = 0;
    for (final it in items.values) {
      final q = it.qty;
      if (q > 0) sum += q;
    }
    return sum;
  }

  /// ✅ “絶対に落ちない/捨てない”方針:
  factory CartDTO.fromJson(Map<String, dynamic> json) {
    final aid = (json['avatarId'] ?? json['id'] ?? '').toString().trim();

    final itemsRaw = json['items'];
    final Map<String, CartItemDTO> items = {};

    if (itemsRaw is Map) {
      for (final entry in itemsRaw.entries) {
        final key = entry.key.toString().trim();
        final v = entry.value;
        if (key.isEmpty) continue;

        // New-ish shape: itemKey -> { ... }
        if (v is Map) {
          final m = <String, dynamic>{};
          for (final e in v.entries) {
            m[e.key.toString()] = e.value;
          }
          items[key] = CartItemDTO.fromJson(m, fallbackKey: key);
          continue;
        }

        // Legacy-ish shape: itemKey(or modelId) -> qty
        final n = (v is int) ? v : int.tryParse(v.toString());
        items[key] = CartItemDTO(
          inventoryId: '',
          listId: '',
          modelId: key, // ✅ fallback
          qty: (n ?? 0),
          title: null,
          size: null,
          color: null,
          listImage: null,
          price: null,
          productName: null,
        );
      }
    }

    return CartDTO(
      avatarId: aid,
      items: items,
      createdAt: _tryParseTime(json['createdAt']),
      updatedAt: _tryParseTime(json['updatedAt']),
      expiresAt: _tryParseTime(json['expiresAt']),
    );
  }

  static DateTime? _tryParseTime(dynamic v) {
    if (v == null) return null;

    if (v is String) {
      final s = v.trim();
      if (s.isEmpty) return null;
      return DateTime.tryParse(s);
    }

    // Firestore Timestamp-like: {seconds, nanos}
    if (v is Map) {
      final sec = v['seconds'];
      final nanos = v['nanos'] ?? 0;

      final s = (sec is int) ? sec : int.tryParse(sec?.toString() ?? '');
      final n = (nanos is int)
          ? nanos
          : (int.tryParse(nanos?.toString() ?? '0') ?? 0);

      if (s == null) return null;

      return DateTime.fromMillisecondsSinceEpoch(
        s * 1000 + (n ~/ 1000000),
        isUtc: true,
      );
    }

    final s = v.toString().trim();
    if (s.isEmpty) return null;
    return DateTime.tryParse(s);
  }
}

class CartItemDTO {
  CartItemDTO({
    required this.inventoryId,
    required this.listId,
    required this.modelId,
    required this.qty,
    required this.title,
    required this.size,
    required this.color,
    required this.listImage,
    required this.price,
    required this.productName,
  });

  final String inventoryId;
  final String listId;
  final String modelId;
  final int qty;

  // ✅ backend が /mall/me/cart で返してくる（表示用）
  final String? title;
  final String? size;
  final String? color;

  // ✅ read-model fields
  final String? listImage; // backend: listImage
  final int? price; // backend: price (number)
  final String? productName; // backend: productName

  bool get isRoughlyValid => qty > 0;

  bool get hasCoreIds =>
      inventoryId.trim().isNotEmpty &&
      listId.trim().isNotEmpty &&
      modelId.trim().isNotEmpty;

  static int _toInt(dynamic v) {
    if (v == null) return 0;
    if (v is int) return v;
    if (v is double) return v.toInt();
    if (v is num) return v.toInt();
    final s = v.toString().trim();
    if (s.isEmpty) return 0;
    return int.tryParse(s) ?? 0;
  }

  static int? _toNullableInt(dynamic v) {
    if (v == null) return null;
    if (v is int) return v;
    if (v is double) return v.toInt();
    if (v is num) return v.toInt();
    final s = v.toString().trim();
    if (s.isEmpty) return null;
    return int.tryParse(s);
  }

  static String? _toNullableStr(dynamic v) {
    if (v == null) return null;
    final s = v.toString().trim();
    return s.isEmpty ? null : s;
  }

  factory CartItemDTO.fromJson(
    Map<String, dynamic> json, {
    String? fallbackKey,
  }) {
    final invId = (json['inventoryId'] ?? '').toString().trim();
    final lid = (json['listId'] ?? '').toString().trim();

    final mid0 = (json['modelId'] ?? '').toString().trim();
    final mid = mid0.isNotEmpty ? mid0 : (fallbackKey ?? '').trim();

    final qty = _toInt(json['qty']);

    final title = _toNullableStr(json['title']);
    final size = _toNullableStr(json['size']);
    final color = _toNullableStr(json['color']);

    final listImage = _toNullableStr(
      json['listImage'] ?? json['image'] ?? json['imageId'],
    );
    final price = _toNullableInt(json['price']);
    final productName = _toNullableStr(json['productName'] ?? json['name']);

    return CartItemDTO(
      inventoryId: invId,
      listId: lid,
      modelId: mid,
      qty: qty,
      title: title,
      size: size,
      color: color,
      listImage: listImage,
      price: price,
      productName: productName,
    );
  }
}
