//frontend\sns\lib\features\cart\infrastructure\cart_repository_http.dart
import 'dart:convert';

import 'package:firebase_auth/firebase_auth.dart';
import 'package:http/http.dart' as http;

// ✅ 共通 resolver を使う（fallback/環境変数名のブレを防ぐ）
import '../../../app/config/api_base.dart';

/// Buyer-facing Cart repository (HTTP).
///
/// ✅ This repository ONLY handles:
/// - CartDTO
/// - CartItemDTO
///
/// Backend endpoints (CartHandler):
/// - GET    /sns/cart?avatarId=...
/// - POST   /sns/cart/items           body: {avatarId, inventoryId, listId, modelId, qty}
/// - PUT    /sns/cart/items           body: {avatarId, inventoryId, listId, modelId, qty}
/// - DELETE /sns/cart/items           body: {avatarId, inventoryId, listId, modelId}
/// - DELETE /sns/cart?avatarId=...
///
/// NOTE:
/// - Web(CORS) ではカスタムヘッダ（例: x-avatar-id）を送ると preflight が走り、
///   backend 側で Access-Control-Allow-Headers に許可が無いとブロックされます。
/// - avatarId は **query/body のみ**で送ります（ヘッダには入れない）。
class CartRepositoryHttp {
  CartRepositoryHttp({http.Client? client, String? apiBase})
    : _client = client ?? http.Client(),
      _apiBase = (apiBase ?? '').trim();

  final http.Client _client;

  /// Optional override. If empty, resolveSnsApiBase() will be used.
  final String _apiBase;

  void dispose() {
    _client.close();
  }

  // ----------------------------
  // Public API (CartDTO only)
  // ----------------------------

  /// GET /sns/cart?avatarId=...
  Future<CartDTO> fetchCart({required String avatarId}) async {
    final aid = avatarId.trim();
    if (aid.isEmpty) throw ArgumentError('avatarId is required');

    final uri = _uri('/sns/cart', qp: {'avatarId': aid});
    final res = await _sendAuthed('GET', uri);

    return _decodeCart(res);
  }

  /// POST /sns/cart/items
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
    final uri = _uri('/sns/cart/items');
    final bodyMap = <String, dynamic>{
      'avatarId': aid,
      'inventoryId': invId,
      'listId': lid,
      'modelId': mid,
      'qty': qty,
    };
    final body = jsonEncode(bodyMap);

    final res = await _sendAuthed('POST', uri, body: body);
    return _decodeCart(res);
  }

  /// PUT /sns/cart/items
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

    final uri = _uri('/sns/cart/items');
    final bodyMap = <String, dynamic>{
      'avatarId': aid,
      'inventoryId': invId,
      'listId': lid,
      'modelId': mid,
      'qty': qty,
    };
    final body = jsonEncode(bodyMap);

    final res = await _sendAuthed('PUT', uri, body: body);
    return _decodeCart(res);
  }

  /// DELETE /sns/cart/items
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

    final uri = _uri('/sns/cart/items');
    final bodyMap = <String, dynamic>{
      'avatarId': aid,
      'inventoryId': invId,
      'listId': lid,
      'modelId': mid,
    };
    final body = jsonEncode(bodyMap);

    final res = await _sendAuthed('DELETE', uri, body: body);
    return _decodeCart(res);
  }

  /// DELETE /sns/cart?avatarId=...
  Future<void> clearCart({required String avatarId}) async {
    final aid = avatarId.trim();
    if (aid.isEmpty) throw ArgumentError('avatarId is required');

    final uri = _uri('/sns/cart', qp: {'avatarId': aid});
    final res = await _sendAuthed('DELETE', uri);

    if (res.statusCode == 204) return;
    _throwHttpError(res);
  }

  // ----------------------------
  // Auth / sending
  // ----------------------------

  Future<Map<String, String>> _headersJsonAuthed({
    bool forceRefreshToken = false,
  }) async {
    final u = FirebaseAuth.instance.currentUser;
    if (u == null) {
      throw CartHttpException(statusCode: 401, message: 'not_signed_in');
    }

    final idToken = await u.getIdToken(forceRefreshToken);
    final tok = (idToken ?? '').trim(); // ✅ null-safe
    if (tok.isEmpty) {
      throw CartHttpException(statusCode: 401, message: 'empty_id_token');
    }

    // ✅ token は絶対にログに出さない
    return <String, String>{'Authorization': 'Bearer $tok', ..._headersJson()};
  }

  /// Sends request with Firebase Authorization header.
  /// - If 401, retry once with forceRefreshToken=true.
  Future<http.Response> _sendAuthed(
    String method,
    Uri uri, {
    String? body,
  }) async {
    http.Response res;

    final h1 = await _headersJsonAuthed(forceRefreshToken: false);
    res = await _sendRaw(method, uri, headers: h1, body: body);

    if (res.statusCode != 401) return res;

    final h2 = await _headersJsonAuthed(forceRefreshToken: true);
    return _sendRaw(method, uri, headers: h2, body: body);
  }

  Future<http.Response> _sendRaw(
    String method,
    Uri uri, {
    required Map<String, String> headers,
    String? body,
  }) async {
    final m = method.trim().toUpperCase();

    // body なし
    if (body == null) {
      switch (m) {
        case 'GET':
          return _client.get(uri, headers: headers);
        case 'DELETE':
          return _client.delete(uri, headers: headers);
        case 'POST':
          return _client.post(uri, headers: headers);
        case 'PUT':
          return _client.put(uri, headers: headers);
        default:
          final req = http.Request(m, uri);
          req.headers.addAll(headers);
          final streamed = await _client.send(req);
          return http.Response.fromStream(streamed);
      }
    }

    // body あり
    switch (m) {
      case 'POST':
        return _client.post(uri, headers: headers, body: body);
      case 'PUT':
        return _client.put(uri, headers: headers, body: body);
      case 'DELETE':
        // ✅ http.delete(body) が環境/バージョンで不安定なため Request で送る
        final req = http.Request('DELETE', uri);
        req.headers.addAll(headers);
        req.body = body;
        final streamed = await _client.send(req);
        return http.Response.fromStream(streamed);
      default:
        final req = http.Request(m, uri);
        req.headers.addAll(headers);
        req.body = body;
        final streamed = await _client.send(req);
        return http.Response.fromStream(streamed);
    }
  }

  // ----------------------------
  // DTO + parsing (CartDTO only)
  // ----------------------------

  CartDTO _decodeCart(http.Response res) {
    if (res.statusCode >= 200 && res.statusCode < 300) {
      final map = _decodeJsonMap(res.body);
      return CartDTO.fromJson(map);
    }

    _throwHttpError(res);
    throw StateError('unreachable');
  }

  Map<String, dynamic> _decodeJsonMap(String body) {
    final raw = body.trim().isEmpty ? '{}' : body;
    final v = jsonDecode(raw);
    if (v is Map<String, dynamic>) return v;
    throw FormatException('invalid json response');
  }

  void _throwHttpError(http.Response res) {
    final status = res.statusCode;
    String msg = 'HTTP $status';

    try {
      final m = _decodeJsonMap(res.body);
      final e = (m['error'] ?? '').toString().trim();
      if (e.isNotEmpty) msg = e;
    } catch (_) {
      final s = res.body.trim();
      if (s.isNotEmpty) msg = s;
    }

    throw CartHttpException(statusCode: status, message: msg);
  }

  // ----------------------------
  // URL / headers
  // ----------------------------

  Uri _uri(String path, {Map<String, String>? qp}) {
    final baseRaw = (_apiBase.isNotEmpty ? _apiBase : resolveSnsApiBase())
        .trim();

    if (baseRaw.isEmpty) {
      throw StateError(
        'API base URL is empty (API_BASE_URL / API_BASE / fallback).',
      );
    }

    final base = baseRaw.replaceAll(RegExp(r'\/+$'), '');
    final b = Uri.parse(base);

    final cleanPath = path.startsWith('/') ? path : '/$path';
    final joinedPath = _joinPaths(b.path, cleanPath);

    return Uri(
      scheme: b.scheme,
      userInfo: b.userInfo,
      host: b.host,
      port: b.hasPort ? b.port : null,
      path: joinedPath,
      queryParameters: (qp == null || qp.isEmpty) ? null : qp,
      fragment: b.fragment.isEmpty ? null : b.fragment,
    );
  }

  String _joinPaths(String a, String b) {
    final aa = a.trim();
    final bb = b.trim();
    if (aa.isEmpty || aa == '/') return bb;
    if (bb.isEmpty || bb == '/') return aa;
    if (aa.endsWith('/') && bb.startsWith('/')) return aa + bb.substring(1);
    if (!aa.endsWith('/') && !bb.startsWith('/')) return '$aa/$bb';
    return aa + bb;
  }

  /// ✅ CORS 的に “simple headers” 寄りにする（x- 系などカスタムは入れない）
  /// NOTE: Authorization はここには入れない（authed 時は _headersJsonAuthed で追加）
  Map<String, String> _headersJson() => const {
    'Content-Type': 'application/json; charset=utf-8',
    'Accept': 'application/json',
  };
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
  /// - items は Map で来る前提だが、値が Map / int / string などでも壊れない
  /// - 新形式(Map)はできる限り拾う（IDが欠けていても保持）
  /// - 旧形式(key->qty) も保持（modelId は key を使う）
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
          // castが失敗する可能性があるので “best-effort” で変換
          final m = <String, dynamic>{};
          for (final e in v.entries) {
            m[e.key.toString()] = e.value;
          }
          items[key] = CartItemDTO.fromJson(m, fallbackKey: key);
          continue;
        }

        // Legacy shape: itemKey(or modelId) -> qty
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

  // ✅ backend が /sns/cart で返してくる（表示用）
  final String? title;
  final String? size;
  final String? color;

  // ✅ NEW: read-model fields
  final String? listImage; // backend: listImage
  final int? price; // backend: price (number)
  final String? productName; // backend: productName

  /// ✅ “捨てない”方針のため isValid を厳格にしない
  /// - UI側で「表示に必要な最低条件」を決めて弾く
  /// - ここでは「qty>0 なら概ね有効」くらいにしておく
  bool get isRoughlyValid => qty > 0;

  /// 画面表示のための最低条件（必要ならUI側が使う）
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

  /// ✅ “絶対に落ちない”:
  /// - 型が変でも落ちない（toString + tryParse）
  /// - id 欠損時は fallbackKey（items の key）を modelId などに使える
  factory CartItemDTO.fromJson(
    Map<String, dynamic> json, {
    String? fallbackKey,
  }) {
    final invId = (json['inventoryId'] ?? '').toString().trim();
    final lid = (json['listId'] ?? '').toString().trim();

    // modelId が無い/空なら fallbackKey を採用（legacy混在でも最低限拾う）
    final mid0 = (json['modelId'] ?? '').toString().trim();
    final mid = mid0.isNotEmpty ? mid0 : (fallbackKey ?? '').trim();

    final qty = _toInt(json['qty']);

    final title = _toNullableStr(json['title']);
    final size = _toNullableStr(json['size']);
    final color = _toNullableStr(json['color']);

    // backend: listImage / price / productName
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

class CartHttpException implements Exception {
  CartHttpException({required this.statusCode, required this.message});

  final int statusCode;
  final String message;

  @override
  String toString() =>
      'CartHttpException(statusCode=$statusCode, message=$message)';
}
