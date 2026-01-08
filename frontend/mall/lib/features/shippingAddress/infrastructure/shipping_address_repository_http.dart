// frontend/mall/lib/features/shippingAddress/infrastructure/shipping_address_repository_http.dart
import 'dart:convert';

import 'package:flutter/foundation.dart';
import 'package:dio/dio.dart';
import 'package:firebase_auth/firebase_auth.dart';

// ✅ 共通 resolver を使う（fallback/環境変数名のブレを防ぐ）
import '../../../app/config/api_base.dart';

/// Domain-ish model for SNS shipping address (matches backend shippingAddress.entity.go)
class ShippingAddress {
  ShippingAddress({
    required this.id,
    required this.userId,
    required this.zipCode,
    required this.state,
    required this.city,
    required this.street,
    required this.street2,
    required this.country,
    required this.createdAt,
    required this.updatedAt,
  });

  final String id;
  final String userId;
  final String zipCode;
  final String state;
  final String city;
  final String street;

  /// optional in UI, but backend entity currently has string (can be "")
  final String street2;

  final String country;
  final DateTime createdAt;
  final DateTime updatedAt;

  factory ShippingAddress.fromJson(Map<String, dynamic> json) {
    DateTime parseTime(dynamic v) {
      if (v is String) {
        final s = v.trim();
        if (s.isNotEmpty) return DateTime.parse(s).toUtc();
      }
      // Firestore Timestamp etc are not expected on HTTP boundary; fallback
      return DateTime.fromMillisecondsSinceEpoch(0, isUtc: true);
    }

    String s(dynamic v) => (v ?? '').toString().trim();

    return ShippingAddress(
      id: s(json['id']),
      userId: s(json['userId']),
      zipCode: s(json['zipCode']),
      state: s(json['state']),
      city: s(json['city']),
      street: s(json['street']),
      street2: s(json['street2']),
      country: s(json['country']),
      createdAt: parseTime(json['createdAt']),
      updatedAt: parseTime(json['updatedAt']),
    );
  }

  Map<String, dynamic> toJson() => {
    'id': id,
    'userId': userId,
    'zipCode': zipCode,
    'state': state,
    'city': city,
    'street': street,
    'street2': street2,
    'country': country,
    'createdAt': createdAt.toUtc().toIso8601String(),
    'updatedAt': updatedAt.toUtc().toIso8601String(),
  };
}

/// ✅ Upsert body (create/update)
/// - docId = Firebase UID
/// - backend: PATCH /mall/shipping-addresses/{id} does upsert
class UpsertShippingAddressInput {
  UpsertShippingAddressInput({
    required this.id, // ✅ uid (docId)
    required this.userId, // ✅ uid (redundant but needed for backend upsert-create path)
    required this.zipCode,
    required this.state,
    required this.city,
    required this.street,
    this.street2,
    this.country = 'JP',
  });

  final String id;
  final String userId;
  final String zipCode;
  final String state;
  final String city;
  final String street;
  final String? street2;
  final String country;

  Map<String, dynamic> toJson() => {
    // ✅ backend 側 upsert 用
    'id': id.trim(),
    'userId': userId.trim(),
    'zipCode': zipCode.trim(),
    'state': state.trim(),
    'city': city.trim(),
    'street': street.trim(),
    // backend entity is string, so we send "" when null
    'street2': (street2 ?? '').trim(),
    'country': country.trim(),
  };
}

/// PATCH body (only non-null fields will be sent)
/// ✅ backend の “not_found -> upsert-create” 分岐で userId が必要なので送れるようにする
class UpdateShippingAddressInput {
  UpdateShippingAddressInput({
    this.userId,
    this.zipCode,
    this.state,
    this.city,
    this.street,
    this.street2,
    this.country,
  });

  final String? userId; // ✅ add
  final String? zipCode;
  final String? state;
  final String? city;
  final String? street;
  final String? street2;
  final String? country;

  Map<String, dynamic> toJson() {
    final m = <String, dynamic>{};

    void put(String k, String? v) {
      if (v == null) return;
      m[k] = v.trim();
    }

    // ✅ 初回作成 fallback のため
    put('userId', userId);

    put('zipCode', zipCode);
    put('state', state);
    put('city', city);
    put('street', street);

    // street2 は「消す」ケースがあり得るので、呼び出し側で "" を渡せば消去扱いにできます
    put('street2', street2);

    put('country', country);

    return m;
  }
}

class ShippingAddressRepositoryHttp {
  ShippingAddressRepositoryHttp({Dio? dio, FirebaseAuth? auth, String? baseUrl})
    : _auth = auth ?? FirebaseAuth.instance,
      _dio = dio ?? Dio() {
    // ✅ baseUrl 優先。無ければ共通 resolver を使う（fallback あり）
    final resolvedRaw = (baseUrl ?? '').trim().isNotEmpty
        ? baseUrl!.trim()
        : resolveApiBase().trim();

    if (resolvedRaw.isEmpty) {
      throw Exception(
        'API base URL is empty (API_BASE_URL / API_BASE / fallback).',
      );
    }

    final normalized = resolvedRaw.replaceAll(RegExp(r'\/+$'), '');

    // ✅ baseUrl に /sns が含まれる注入も許容
    // - baseUrl が ".../sns" の場合: 以降の path は "shipping-addresses/..." にする
    // - baseUrl が domain だけの場合: 以降の path は "mall/shipping-addresses/..." にする
    final b = Uri.parse(normalized);
    final basePath = b.path.replaceAll(RegExp(r'\/+$'), '');
    _pathPrefix = basePath.endsWith('/mall') || basePath == '/mall'
        ? ''
        : 'mall';

    _dio.options = BaseOptions(
      baseUrl: normalized,
      connectTimeout: const Duration(seconds: 12),
      receiveTimeout: const Duration(seconds: 20),
      sendTimeout: const Duration(seconds: 20),
      headers: <String, dynamic>{
        'Content-Type': 'application/json',
        'Accept': 'application/json',
      },
    );

    // ✅ Request/Response logger + Firebase token injector + 401 retry
    _dio.interceptors.add(
      InterceptorsWrapper(
        onRequest: (options, handler) async {
          // ✅ token inject (non-force)
          try {
            final u = _auth.currentUser;
            if (u != null) {
              final token = await u.getIdToken(false);
              final t = (token ?? '').toString().trim();
              if (t.isNotEmpty) {
                options.headers['Authorization'] = 'Bearer $t';
              }
            }
          } catch (e) {
            _log('[ShippingAddressRepositoryHttp] token error: $e');
          }

          _logRequest(options);
          handler.next(options);
        },
        onResponse: (response, handler) {
          _logResponse(response);
          handler.next(response);
        },
        onError: (e, handler) async {
          // ✅ 既存のエラー詳細ログ
          _logDioError(e);

          // ✅ 失敗時の “要約ログ”
          _logFailureSummary(e);

          // ✅ 401 のときは 1 回だけ forceRefreshToken=true でリトライ
          final status = e.response?.statusCode;
          final alreadyRetried = e.requestOptions.extra['__retried401'] == true;

          if (status == 401 && !alreadyRetried) {
            try {
              final u = _auth.currentUser;
              if (u != null) {
                final token = await u.getIdToken(true);
                final t = (token ?? '').toString().trim();
                if (t.isNotEmpty) {
                  final opts = e.requestOptions;
                  opts.extra['__retried401'] = true;
                  opts.headers['Authorization'] = 'Bearer $t';

                  _log(
                    '[ShippingAddressRepositoryHttp] retrying once with refreshed token: ${opts.method} ${opts.uri}',
                  );

                  final res = await _dio.fetch(opts);
                  handler.resolve(res);
                  return;
                }
              }
            } catch (ex) {
              _log('[ShippingAddressRepositoryHttp] 401 retry failed: $ex');
              // fallthrough
            }
          }

          handler.next(e);
        },
      ),
    );

    _log(
      '[ShippingAddressRepositoryHttp] init baseUrl=$normalized prefix=$_pathPrefix',
    );
  }

  final Dio _dio;
  final FirebaseAuth _auth;
  final CancelToken _cancelToken = CancelToken();

  late final String _pathPrefix; // '' or 'sns'

  // ✅ release でもログを出したい場合: --dart-define=ENABLE_HTTP_LOG=true
  static const bool _envHttpLog = bool.fromEnvironment(
    'ENABLE_HTTP_LOG',
    defaultValue: false,
  );

  bool get _logEnabled => kDebugMode || _envHttpLog;

  void dispose() {
    if (!_cancelToken.isCancelled) {
      _cancelToken.cancel('disposed');
    }
    // ⚠️ Repository を短命に大量生成するなら close(force:true) は避けたいが、
    // 現状方針を崩さず維持する（必要なら DI で singleton 化してください）
    _dio.close(force: true);
  }

  Exception _normalizeDioError(Object e) {
    if (e is DioException) {
      final status = e.response?.statusCode;
      final data = e.response?.data;
      final msg = e.message ?? 'dio_exception';
      return Exception(
        'HTTP error${status != null ? ' ($status)' : ''}: ${data ?? msg}',
      );
    }
    return Exception(e.toString());
  }

  // ------------------------------------------------------------
  // auth helpers
  // ------------------------------------------------------------

  String _requireUid() {
    final u = _auth.currentUser;
    final uid = (u?.uid ?? '').trim();
    if (uid.isEmpty) {
      throw Exception('not signed in');
    }
    return uid;
  }

  // ------------------------------------------------------------
  // path helpers
  // ------------------------------------------------------------

  String _p(String path) {
    var p = path.trim();
    if (p.startsWith('/')) p = p.substring(1);
    if (p.isEmpty) return p;

    if (_pathPrefix.isEmpty) return p;
    return '$_pathPrefix/$p';
  }

  // ------------------------------------------------------------
  // API (SNS)
  // ------------------------------------------------------------

  /// ✅ Get "my" shipping address (docId = uid)
  Future<ShippingAddress> getMine() async {
    final uid = _requireUid();
    return getById(uid);
  }

  /// GET /mall/shipping-addresses/{id}
  Future<ShippingAddress> getById(String id) async {
    final s = id.trim();
    if (s.isEmpty) {
      throw Exception('invalid id');
    }

    try {
      final res = await _dio.get(
        _p('shipping-addresses/$s'),
        cancelToken: _cancelToken,
      );
      final data = _asMap(res.data);
      return ShippingAddress.fromJson(data);
    } on DioException catch (e) {
      _logFailureSummary(e, op: 'GET /shipping-addresses/$s');
      throw _normalizeDioError(e);
    } catch (e) {
      throw _normalizeDioError(e);
    }
  }

  /// ✅ UPSERT (create/update) for "my" shipping address
  /// - backend 仕様に合わせて PATCH /mall/shipping-addresses/{uid} を基本にする
  Future<ShippingAddress> upsertMine(UpsertShippingAddressInput inData) async {
    final uid = _requireUid();

    // ✅ docId=uid を強制
    final fixed = UpsertShippingAddressInput(
      id: uid,
      userId: uid,
      zipCode: inData.zipCode,
      state: inData.state,
      city: inData.city,
      street: inData.street,
      street2: inData.street2,
      country: inData.country,
    );

    try {
      final res = await _dio.patch(
        _p('shipping-addresses/$uid'),
        data: fixed.toJson(),
        cancelToken: _cancelToken,
      );
      final data = _asMap(res.data);
      return ShippingAddress.fromJson(data);
    } on DioException catch (e) {
      _logFailureSummary(e, op: 'PATCH /shipping-addresses/$uid (upsertMine)');
      throw _normalizeDioError(e);
    } catch (e) {
      throw _normalizeDioError(e);
    }
  }

  /// ⚠️ 互換: 旧 create API 名
  /// - 実体は PATCH(upsert) に変更
  Future<ShippingAddress> create(UpsertShippingAddressInput inData) async {
    return upsertMine(inData);
  }

  /// PATCH /mall/shipping-addresses/{id}
  Future<ShippingAddress> updateMine(UpdateShippingAddressInput inData) async {
    final uid = _requireUid();

    // ✅ not_found -> create 分岐で userId が必要になる可能性があるので、ここで補完
    final fixed = UpdateShippingAddressInput(
      userId: (inData.userId ?? uid),
      zipCode: inData.zipCode,
      state: inData.state,
      city: inData.city,
      street: inData.street,
      street2: inData.street2,
      country: inData.country,
    );

    return update(uid, fixed);
  }

  /// PATCH /mall/shipping-addresses/{id}
  Future<ShippingAddress> update(
    String id,
    UpdateShippingAddressInput inData,
  ) async {
    final s = id.trim();
    if (s.isEmpty) {
      throw Exception('invalid id');
    }

    final body = inData.toJson();
    if (body.isEmpty) {
      return getById(s);
    }

    try {
      final res = await _dio.patch(
        _p('shipping-addresses/$s'),
        data: body,
        cancelToken: _cancelToken,
      );
      final data = _asMap(res.data);
      return ShippingAddress.fromJson(data);
    } on DioException catch (e) {
      _logFailureSummary(e, op: 'PATCH /shipping-addresses/$s');
      throw _normalizeDioError(e);
    } catch (e) {
      throw _normalizeDioError(e);
    }
  }

  /// DELETE /mall/shipping-addresses/{id}
  Future<void> delete(String id) async {
    final s = id.trim();
    if (s.isEmpty) {
      throw Exception('invalid id');
    }

    try {
      await _dio.delete(_p('shipping-addresses/$s'), cancelToken: _cancelToken);
    } on DioException catch (e) {
      _logFailureSummary(e, op: 'DELETE /shipping-addresses/$s');
      throw _normalizeDioError(e);
    } catch (e) {
      throw _normalizeDioError(e);
    }
  }

  /// ✅ Delete "my" shipping address (docId=uid)
  Future<void> deleteMine() async {
    final uid = _requireUid();
    return delete(uid);
  }

  // ------------------------------------------------------------
  // helpers
  // ------------------------------------------------------------

  Map<String, dynamic> _asMap(dynamic v) {
    if (v is Map<String, dynamic>) {
      return v;
    }
    if (v is Map) {
      return Map<String, dynamic>.from(v);
    }
    if (v is String) {
      try {
        final decoded = jsonDecode(v);
        if (decoded is Map<String, dynamic>) {
          return decoded;
        }
        if (decoded is Map) {
          return Map<String, dynamic>.from(decoded);
        }
      } catch (_) {
        // ignore
      }
    }
    throw Exception(
      'Invalid response body: expected object, got ${v.runtimeType}',
    );
  }

  // ------------------------------------------------------------
  // logging (debug or ENABLE_HTTP_LOG=true)
  // ------------------------------------------------------------

  void _log(String msg) {
    if (!_logEnabled) return;
    debugPrint(msg);
  }

  void _logRequest(RequestOptions o) {
    if (!_logEnabled) return;

    final method = o.method.toUpperCase();
    final url = o.uri.toString();

    // Authorization は伏せる
    final headers = <String, dynamic>{};
    o.headers.forEach((k, v) {
      if (k.toLowerCase() == 'authorization') {
        headers[k] = 'Bearer ***';
      } else {
        headers[k] = v;
      }
    });

    String body = '';
    final d = o.data;
    if (d != null) {
      try {
        if (d is String) {
          body = d;
        } else if (d is Map || d is List) {
          body = jsonEncode(d);
        } else {
          body = d.toString();
        }
      } catch (e) {
        body = '(failed to encode body: $e)';
      }
    }

    final b = StringBuffer();
    b.writeln('[ShippingAddressRepositoryHttp] request');
    b.writeln('  method=$method');
    b.writeln('  url=$url');
    b.writeln('  headers=${jsonEncode(headers)}');
    if (body.isNotEmpty) {
      b.writeln('  body=${_truncate(body, 1500)}');
    }
    debugPrint(b.toString());
  }

  void _logResponse(Response r) {
    if (!_logEnabled) return;

    final method = r.requestOptions.method.toUpperCase();
    final url = r.requestOptions.uri.toString();

    String body = '';
    try {
      final d = r.data;
      if (d == null) {
        body = '';
      } else if (d is String) {
        body = d;
      } else if (d is Map || d is List) {
        body = jsonEncode(d);
      } else {
        body = d.toString();
      }
    } catch (e) {
      body = '(failed to encode response body: $e)';
    }

    final b = StringBuffer();
    b.writeln('[ShippingAddressRepositoryHttp] response');
    b.writeln('  method=$method');
    b.writeln('  url=$url');
    b.writeln('  status=${r.statusCode}');
    if (body.isNotEmpty) {
      b.writeln('  body=${_truncate(body, 1500)}');
    }
    debugPrint(b.toString());
  }

  void _logDioError(DioException e) {
    if (!_logEnabled) return;

    final o = e.requestOptions;
    final method = o.method.toUpperCase();
    final url = o.uri.toString();
    final status = e.response?.statusCode;

    String resBody = '';
    try {
      final d = e.response?.data;
      if (d == null) {
        resBody = '';
      } else if (d is String) {
        resBody = d;
      } else if (d is Map || d is List) {
        resBody = jsonEncode(d);
      } else {
        resBody = d.toString();
      }
    } catch (_) {
      resBody = '(failed to encode error response body)';
    }

    // request body
    String reqBody = '';
    try {
      final d = o.data;
      if (d == null) {
        reqBody = '';
      } else if (d is String) {
        reqBody = d;
      } else if (d is Map || d is List) {
        reqBody = jsonEncode(d);
      } else {
        reqBody = d.toString();
      }
    } catch (_) {
      reqBody = '(failed to encode request body)';
    }

    final b = StringBuffer();
    b.writeln('[ShippingAddressRepositoryHttp] error');
    b.writeln('  method=$method');
    b.writeln('  url=$url');
    if (status != null) {
      b.writeln('  status=$status');
    }
    if ((e.message ?? '').trim().isNotEmpty) {
      b.writeln('  message=${e.message}');
    }
    if (reqBody.isNotEmpty) {
      b.writeln('  requestBody=${_truncate(reqBody, 1500)}');
    }
    if (resBody.isNotEmpty) {
      b.writeln('  responseBody=${_truncate(resBody, 1500)}');
    }
    debugPrint(b.toString());
  }

  // Failure summary logger (request headers/body + response headers/body + status)
  void _logFailureSummary(DioException e, {String? op}) {
    if (!_logEnabled) return;

    final o = e.requestOptions;
    final method = o.method.toUpperCase();
    final url = o.uri.toString();

    final status = e.response?.statusCode;
    final statusLine = status != null ? 'status=$status' : 'status=?';

    // --- request headers (mask auth) ---
    final reqHeaders = <String, dynamic>{};
    o.headers.forEach((k, v) {
      if (k.toLowerCase() == 'authorization') {
        reqHeaders[k] = 'Bearer ***';
      } else {
        reqHeaders[k] = v;
      }
    });

    // --- request body ---
    String reqBody = '';
    try {
      final d = o.data;
      if (d == null) {
        reqBody = '';
      } else if (d is String) {
        reqBody = d;
      } else if (d is Map || d is List) {
        reqBody = jsonEncode(d);
      } else {
        reqBody = d.toString();
      }
    } catch (ex) {
      reqBody = '(failed to encode request body: $ex)';
    }

    // --- response headers ---
    final resHeaders = <String, dynamic>{};
    try {
      final h = e.response?.headers.map;
      if (h != null) {
        h.forEach((k, v) {
          resHeaders[k] = v;
        });
      }
    } catch (_) {
      // ignore
    }

    // --- response body ---
    String resBody = '';
    try {
      final d = e.response?.data;
      if (d == null) {
        resBody = '';
      } else if (d is String) {
        resBody = d;
      } else if (d is Map || d is List) {
        resBody = jsonEncode(d);
      } else {
        resBody = d.toString();
      }
    } catch (ex) {
      resBody = '(failed to encode response body: $ex)';
    }

    final b = StringBuffer();
    b.writeln('[ShippingAddressRepositoryHttp] FAILURE');
    if ((op ?? '').trim().isNotEmpty) {
      b.writeln('  op=$op');
    }
    b.writeln('  $statusLine');
    b.writeln('  method=$method');
    b.writeln('  url=$url');
    b.writeln('  dioType=${e.type}');
    if ((e.message ?? '').trim().isNotEmpty) {
      b.writeln('  message=${e.message}');
    }
    b.writeln('  requestHeaders=${_truncate(jsonEncode(reqHeaders), 1500)}');
    if (reqBody.isNotEmpty) {
      b.writeln('  requestBody=${_truncate(reqBody, 1500)}');
    }
    if (resHeaders.isNotEmpty) {
      b.writeln('  responseHeaders=${_truncate(jsonEncode(resHeaders), 1500)}');
    }
    if (resBody.isNotEmpty) {
      b.writeln('  responseBody=${_truncate(resBody, 2000)}');
    }
    debugPrint(b.toString());
  }

  String _truncate(String s, int max) {
    final t = s.trim();
    if (t.length <= max) {
      return t;
    }
    return '${t.substring(0, max)}...(truncated ${t.length - max} chars)';
  }
}
