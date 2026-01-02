// frontend/sns/lib/features/cart/infrastructure/cart_repository_http.dart
import 'dart:convert';

import 'package:firebase_auth/firebase_auth.dart';
import 'package:http/http.dart' as http;

import '../../inventory/infrastructure/inventory_repository_http.dart';

/// Buyer-facing Cart repository (HTTP).
///
/// Backend endpoints (CartHandler):
/// - GET    /sns/cart?avatarId=...
/// - POST   /sns/cart/items           body: {avatarId, inventoryId, listId, modelId, qty}
/// - PUT    /sns/cart/items           body: {avatarId, inventoryId, listId, modelId, qty}
/// - DELETE /sns/cart/items           body: {avatarId, inventoryId, listId, modelId}
/// - DELETE /sns/cart?avatarId=...
///
/// Backend endpoints (Read-model):
/// - GET    /sns/cart/query?avatarId=...                 (sns/cart_query.go)
/// - GET    /sns/preview?avatarId=...&itemKey=...        (sns/preview_query.go)  ※ itemKey 必須の想定
///
/// NOTE:
/// - Web(CORS) ではカスタムヘッダ（例: x-avatar-id）を送ると preflight が走り、
///   backend 側で Access-Control-Allow-Headers に許可が無いとブロックされます。
/// - そのため avatarId は **query/body のみ**で送ります（ヘッダには入れない）。
/// - /sns/cart/items は backend 実装差異を踏まえ、query には avatarId を付けず **body のみに統一**します。
class CartRepositoryHttp {
  CartRepositoryHttp({http.Client? client, String? apiBase})
    : _client = client ?? http.Client(),
      _apiBase = (apiBase ?? const String.fromEnvironment('API_BASE')).trim();

  final http.Client _client;

  /// Optional override. If empty, resolveSnsApiBase() will be used.
  final String _apiBase;

  void dispose() {
    _client.close();
  }

  // ----------------------------
  // Public API
  // ----------------------------

  /// Basic cart (legacy / minimal shape).
  Future<CartDTO> fetchCart({required String avatarId}) async {
    final aid = avatarId.trim();
    if (aid.isEmpty) throw ArgumentError('avatarId is required');

    final uri = _uri('/sns/cart', qp: {'avatarId': aid});
    final res = await _sendAuthed('GET', uri);

    return _decodeCart(res);
  }

  /// ✅ cart_query.go (read-model)
  ///
  /// 返却 JSON の形は実装差分が出やすいので、CartQueryDTO は raw を保持しつつ
  /// 可能なら rows も best-effort で生成します（UI 側の移行が楽になります）。
  Future<CartQueryDTO> fetchCartQuery({required String avatarId}) async {
    final aid = avatarId.trim();
    if (aid.isEmpty) throw ArgumentError('avatarId is required');

    final uri = _uri('/sns/cart/query', qp: {'avatarId': aid});
    final res = await _sendAuthed('GET', uri);
    return _decodeCartQuery(res);
  }

  /// ✅ preview (read-model)
  ///
  /// 現状のバックエンド設計では /sns/preview は itemKey が必要な想定です。
  /// ただし、フロントが「カート全体のプレビュー」を欲しいケースが多いため、
  /// itemKey が空の場合は /sns/cart/query を叩いてプレビュー互換の形で返します。
  ///
  /// - itemKey != null/empty  -> GET /sns/preview?avatarId=...&itemKey=...
  /// - itemKey is null/empty  -> GET /sns/cart/query?avatarId=...   (fallback)
  ///
  /// あなたが提示したレスポンス:
  /// { avatarId, items: { itemKey: {title,size,color,qty} }, createdAt, updatedAt, expiresAt }
  /// は後者（cart/query）型なので、この fallback で 400 を止めつつ UI を進められます。
  Future<PreviewQueryDTO> fetchPreview({
    required String avatarId,
    String? itemKey,
  }) async {
    final aid = avatarId.trim();
    final ik = (itemKey ?? '').trim();
    if (aid.isEmpty) throw ArgumentError('avatarId is required');

    // ✅ itemKey が無い場合は cart/query のレスポンス（あなたの提示形）をプレビューとして扱う
    final Uri uri;
    if (ik.isEmpty) {
      uri = _uri('/sns/cart/query', qp: {'avatarId': aid});
    } else {
      uri = _uri('/sns/preview', qp: {'avatarId': aid, 'itemKey': ik});
    }

    final res = await _sendAuthed('GET', uri);
    return _decodePreview(res);
  }

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

    // ✅ /sns/cart/items は query に avatarId を付けず、body のみに統一
    final uri = _uri('/sns/cart/items');
    final bodyMap = <String, dynamic>{
      'avatarId': aid,
      'inventoryId': invId,
      'listId': lid,
      'modelId': mid,
      'qty': qty,
    };
    final body = jsonEncode(bodyMap);

    // ignore: avoid_print
    print('[CartRepositoryHttp] POST $uri body=$body');

    final res = await _sendAuthed('POST', uri, body: body);
    return _decodeCart(res);
  }

  /// Sets quantity for a cart item (identified by inventoryId/listId/modelId).
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

    // ignore: avoid_print
    print('[CartRepositoryHttp] PUT $uri body=$body');

    final res = await _sendAuthed('PUT', uri, body: body);
    return _decodeCart(res);
  }

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

    // ignore: avoid_print
    print('[CartRepositoryHttp] DELETE $uri body=$body');

    final res = await _sendAuthed('DELETE', uri, body: body);
    return _decodeCart(res);
  }

  Future<void> clearCart({required String avatarId}) async {
    final aid = avatarId.trim();
    if (aid.isEmpty) throw ArgumentError('avatarId is required');

    final uri = _uri('/sns/cart', qp: {'avatarId': aid});

    // ignore: avoid_print
    print('[CartRepositoryHttp] DELETE $uri (clear)');

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

    // ✅ Authorization は “simple header” ではないので Web では preflight が走る
    // ただし /sns/payment が通っているので、backend 側は Allow-Headers に Authorization が入っている前提。
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

    // ignore: avoid_print
    print(
      '[CartRepositoryHttp] 401 received -> retry with refreshed token. uri=$uri',
    );

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
  // DTO + parsing
  // ----------------------------

  CartDTO _decodeCart(http.Response res) {
    if (res.statusCode >= 200 && res.statusCode < 300) {
      final map = _decodeJsonMap(res.body);
      return CartDTO.fromJson(map);
    }
    _throwHttpError(res);
    throw StateError('unreachable');
  }

  CartQueryDTO _decodeCartQuery(http.Response res) {
    if (res.statusCode >= 200 && res.statusCode < 300) {
      final map = _decodeJsonMap(res.body);

      // ✅ best-effort: server shape varies
      CartDTO? cart;
      final c0 = map['cart'];
      if (c0 is Map) {
        cart = CartDTO.fromJson(c0.cast<String, dynamic>());
      } else if (map['data'] is Map) {
        final d = (map['data'] as Map).cast<String, dynamic>();
        final dc = d['cart'];
        if (dc is Map) {
          cart = CartDTO.fromJson(dc.cast<String, dynamic>());
        } else {
          // sometimes data itself is cart-ish
          cart = CartDTO.fromJson(d);
        }
      } else {
        // flat cart-ish
        cart = CartDTO.fromJson(map);
      }

      // ✅ rows 生成（items が Map のケースも吸収）
      final rows = CartQueryRowDTO.parseListFrom(map);

      return CartQueryDTO(raw: map, cart: cart, rows: rows);
    }
    _throwHttpError(res);
    throw StateError('unreachable');
  }

  PreviewQueryDTO _decodePreview(http.Response res) {
    if (res.statusCode >= 200 && res.statusCode < 300) {
      final map = _decodeJsonMap(res.body);
      return PreviewQueryDTO.fromJson(map);
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

    // ignore: avoid_print
    print('[CartRepositoryHttp] HTTP error status=$status body="${res.body}"');

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
        'API_BASE is not set (use --dart-define=API_BASE=https://...)',
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
// Models
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

  factory CartDTO.fromJson(Map<String, dynamic> json) {
    final aid = (json['avatarId'] ?? json['id'] ?? '').toString().trim();

    final itemsRaw = json['items'];
    final Map<String, CartItemDTO> items = {};
    if (itemsRaw is Map) {
      for (final entry in itemsRaw.entries) {
        final key = entry.key.toString().trim();
        final v = entry.value;

        if (key.isEmpty) continue;

        // New shape: itemKey -> {inventoryId, listId, modelId, qty}
        if (v is Map) {
          final it = CartItemDTO.fromJson(v.cast<String, dynamic>());
          if (it.isValid) items[key] = it;
          continue;
        }

        // Legacy shape: modelId -> qty
        final n = (v is int) ? v : int.tryParse(v.toString());
        if (n != null && n > 0) {
          items[key] = CartItemDTO(
            inventoryId: '',
            listId: '',
            modelId: key,
            qty: n,
          );
        }
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
  });

  final String inventoryId;
  final String listId;
  final String modelId;
  final int qty;

  bool get isValid =>
      inventoryId.trim().isNotEmpty &&
      listId.trim().isNotEmpty &&
      modelId.trim().isNotEmpty &&
      qty > 0;

  static int _toInt(dynamic v) {
    if (v == null) return 0;
    if (v is int) return v;
    if (v is double) return v.toInt();
    if (v is num) return v.toInt();
    final s = v.toString().trim();
    if (s.isEmpty) return 0;
    return int.tryParse(s) ?? 0;
  }

  factory CartItemDTO.fromJson(Map<String, dynamic> json) {
    final invId = (json['inventoryId'] ?? '').toString().trim();
    final lid = (json['listId'] ?? '').toString().trim();
    final mid = (json['modelId'] ?? '').toString().trim();
    final qty = _toInt(json['qty']);

    return CartItemDTO(inventoryId: invId, listId: lid, modelId: mid, qty: qty);
  }
}

/// ✅ cart_query.go の結果を “受け取る” ための DTO（raw 保持 + best-effort 抽出）
class CartQueryDTO {
  CartQueryDTO({required this.raw, required this.cart, required this.rows});

  /// server response (full)
  final Map<String, dynamic> raw;

  /// best-effort extracted cart (may be null if server doesn't include a cart-ish object)
  final CartDTO? cart;

  /// best-effort extracted rows (may be empty)
  final List<CartQueryRowDTO> rows;
}

/// itemKey から inventoryId/listId/modelId を best-effort で復元する
/// - 期待形: inventoryId__listId__modelId
/// - inventoryId 自体が pb__tb の場合、全体は pb__tb__listId__modelId になり得る
class _ItemKeyParts {
  _ItemKeyParts({
    required this.inventoryId,
    required this.listId,
    required this.modelId,
  });

  final String inventoryId;
  final String listId;
  final String modelId;

  static _ItemKeyParts fromItemKey(String itemKey) {
    final k = itemKey.trim();
    if (k.isEmpty) {
      return _ItemKeyParts(inventoryId: '', listId: '', modelId: '');
    }
    final parts = k
        .split('__')
        .map((e) => e.trim())
        .where((e) => e.isNotEmpty)
        .toList();

    // 3 以上なら末尾2つを list/model とみなし、それ以外を inventory に結合
    if (parts.length >= 3) {
      final listId = parts[parts.length - 2];
      final modelId = parts[parts.length - 1];
      final inv = parts.sublist(0, parts.length - 2).join('__');
      return _ItemKeyParts(inventoryId: inv, listId: listId, modelId: modelId);
    }

    // それ以外は復元不能（空で返す）
    return _ItemKeyParts(inventoryId: '', listId: '', modelId: '');
  }
}

/// cart_query / preview の「壊れにくい」行 DTO
/// - 実装差分が出やすいので raw を保持
/// - あなたが提示した items-map 形（itemKey -> {title,size,color,qty}）も rows に変換して吸収
class CartQueryRowDTO {
  CartQueryRowDTO({
    required this.itemKey,
    required this.avatarId,
    required this.inventoryId,
    required this.listId,
    required this.modelId,
    required this.qty,
    required this.raw,
    this.title,
    this.size,
    this.color,
    this.listImage,
    this.productName,
    this.price,
  });

  final String itemKey;
  final String avatarId;
  final String inventoryId;
  final String listId;
  final String modelId;
  final int qty;

  // display-ish (optional)
  final String? title;
  final String? size;
  final String? color;
  final String? listImage;
  final String? productName;
  final int? price;

  /// row raw map (may include inventory/list/model/productBlueprint/etc)
  final Map<String, dynamic> raw;

  static List<CartQueryRowDTO> parseListFrom(Map<String, dynamic> root) {
    String s(dynamic v) => (v ?? '').toString().trim();
    int i(dynamic v) => CartItemDTO._toInt(v);
    int? iN(dynamic v) {
      if (v == null) return null;
      final n = i(v);
      return n == 0 ? null : n;
    }

    // (A) list rows: rows/items/lineItems が List のケース
    dynamic rowsRaw = root['rows'] ?? root['items'] ?? root['lineItems'];
    if (rowsRaw is Map && rowsRaw['rows'] is List) rowsRaw = rowsRaw['rows'];

    if (rowsRaw is! List) {
      // sometimes nested: { data: { rows: [...] } }
      final d = root['data'];
      if (d is Map) {
        rowsRaw = d['rows'] ?? d['items'] ?? d['lineItems'];
      }
    }

    if (rowsRaw is List) {
      final out = <CartQueryRowDTO>[];
      for (final e in rowsRaw) {
        if (e is! Map) continue;
        final m = e.cast<String, dynamic>();

        final itemKey = s(m['itemKey'] ?? m['key'] ?? m['id']);
        final avatarId = s(m['avatarId'] ?? root['avatarId']);
        var inventoryId = s(m['inventoryId']);
        var listId = s(m['listId']);
        var modelId = s(m['modelId']);

        if (inventoryId.isEmpty || listId.isEmpty || modelId.isEmpty) {
          final p = _ItemKeyParts.fromItemKey(itemKey);
          if (inventoryId.isEmpty) inventoryId = p.inventoryId;
          if (listId.isEmpty) listId = p.listId;
          if (modelId.isEmpty) modelId = p.modelId;
        }

        final qty = i(m['qty'] ?? m['quantity']);

        final title = s(m['title']);
        final size = s(m['size']);
        final color = s(m['color']);
        final listImage = s(
          m['listImage'] ?? m['image'] ?? m['imageUrl'] ?? m['imageURL'],
        );
        final productName = s(m['productName']);
        final price = iN(m['price']);

        out.add(
          CartQueryRowDTO(
            itemKey: itemKey,
            avatarId: avatarId,
            inventoryId: inventoryId,
            listId: listId,
            modelId: modelId,
            qty: qty,
            raw: m,
            title: title.isEmpty ? null : title,
            size: size.isEmpty ? null : size,
            color: color.isEmpty ? null : color,
            listImage: listImage.isEmpty ? null : listImage,
            productName: productName.isEmpty ? null : productName,
            price: price,
          ),
        );
      }
      return out;
    }

    // (B) あなたが提示した形: { avatarId, items: { itemKey: {...} }, createdAt... }
    final items = root['items'];
    if (items is Map) {
      final avatarId = s(root['avatarId'] ?? root['aid']);

      final out = <CartQueryRowDTO>[];
      for (final entry in items.entries) {
        final key = entry.key.toString().trim();
        if (key.isEmpty) continue;

        final val = entry.value;
        final m = (val is Map)
            ? val.cast<String, dynamic>()
            : <String, dynamic>{'value': val};

        final p = _ItemKeyParts.fromItemKey(key);
        var inventoryId = s(m['inventoryId']);
        var listId = s(m['listId']);
        var modelId = s(m['modelId']);

        if (inventoryId.isEmpty) inventoryId = p.inventoryId;
        if (listId.isEmpty) listId = p.listId;
        if (modelId.isEmpty) modelId = p.modelId;

        final qty = i(m['qty'] ?? m['quantity']);

        final title = s(m['title']);
        final size = s(m['size']);
        final color = s(m['color']);
        final listImage = s(
          m['listImage'] ?? m['image'] ?? m['imageUrl'] ?? m['imageURL'],
        );
        final productName = s(m['productName']);
        final price = iN(m['price']);

        out.add(
          CartQueryRowDTO(
            itemKey: key,
            avatarId: avatarId,
            inventoryId: inventoryId,
            listId: listId,
            modelId: modelId,
            qty: qty,
            raw: m,
            title: title.isEmpty ? null : title,
            size: size.isEmpty ? null : size,
            color: color.isEmpty ? null : color,
            listImage: listImage.isEmpty ? null : listImage,
            productName: productName.isEmpty ? null : productName,
            price: price,
          ),
        );
      }
      return out;
    }

    // (C) single-item preview shape: { avatarId, itemKey, title, size, color, qty, ... }
    final itemKey = s(root['itemKey'] ?? root['key'] ?? root['id']);
    if (itemKey.isNotEmpty) {
      final avatarId = s(root['avatarId'] ?? root['aid']);
      final p = _ItemKeyParts.fromItemKey(itemKey);

      final inventoryId = s(root['inventoryId']).isNotEmpty
          ? s(root['inventoryId'])
          : p.inventoryId;
      final listId = s(root['listId']).isNotEmpty
          ? s(root['listId'])
          : p.listId;
      final modelId = s(root['modelId']).isNotEmpty
          ? s(root['modelId'])
          : p.modelId;

      final qty = i(root['qty'] ?? root['quantity']);

      final title = s(root['title']);
      final size = s(root['size']);
      final color = s(root['color']);
      final listImage = s(
        root['listImage'] ??
            root['image'] ??
            root['imageUrl'] ??
            root['imageURL'],
      );
      final productName = s(root['productName']);
      final price = iN(root['price']);

      return [
        CartQueryRowDTO(
          itemKey: itemKey,
          avatarId: avatarId,
          inventoryId: inventoryId,
          listId: listId,
          modelId: modelId,
          qty: qty,
          raw: root,
          title: title.isEmpty ? null : title,
          size: size.isEmpty ? null : size,
          color: color.isEmpty ? null : color,
          listImage: listImage.isEmpty ? null : listImage,
          productName: productName.isEmpty ? null : productName,
          price: price,
        ),
      ];
    }

    return const [];
  }
}

/// ✅ preview/query の結果を “受け取る” ための DTO
/// - いまの運用では itemKey が無い場合 /sns/cart/query の返却（あなたが貼った形）をそのまま受ける
class PreviewQueryDTO {
  PreviewQueryDTO({
    required this.raw,
    required this.avatarId,
    required this.cart,
    required this.rows,
    required this.total,
    required this.subtotal,
    required this.shippingFee,
    required this.tax,
  });

  final Map<String, dynamic> raw;

  final String avatarId;

  /// best-effort: preview が cart を含む場合に拾う（多くのケースで null でもOK）
  final CartDTO? cart;

  /// best-effort: preview rows
  final List<CartQueryRowDTO> rows;

  /// best-effort: totals (implementation-dependent)
  final int? total;
  final int? subtotal;
  final int? shippingFee;
  final int? tax;

  factory PreviewQueryDTO.fromJson(Map<String, dynamic> json) {
    String s(dynamic v) => (v ?? '').toString().trim();
    int? iN(dynamic v) {
      if (v == null) return null;
      final n = CartItemDTO._toInt(v);
      return n == 0 ? null : n;
    }

    final data = (json['data'] is Map)
        ? (json['data'] as Map).cast<String, dynamic>()
        : null;

    final root = data ?? json;

    final avatarId = s(root['avatarId'] ?? root['aid'] ?? json['avatarId']);

    // cart (rare)
    CartDTO? cart;
    final c0 = root['cart'];
    if (c0 is Map) {
      cart = CartDTO.fromJson(c0.cast<String, dynamic>());
    }

    // ✅ rows: list rows / items-map / single-item すべて吸収
    final rows = CartQueryRowDTO.parseListFrom(root);

    // totals (names vary)
    final totals = (root['totals'] is Map)
        ? (root['totals'] as Map).cast<String, dynamic>()
        : null;

    final subtotal = iN(
      totals?['subtotal'] ?? root['subtotal'] ?? root['itemsSubtotal'],
    );
    final shippingFee = iN(
      totals?['shippingFee'] ?? root['shippingFee'] ?? root['shipping'],
    );
    final tax = iN(totals?['tax'] ?? root['tax']);
    final total = iN(totals?['total'] ?? root['total']);

    return PreviewQueryDTO(
      raw: json,
      avatarId: avatarId,
      cart: cart,
      rows: rows,
      total: total,
      subtotal: subtotal,
      shippingFee: shippingFee,
      tax: tax,
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
