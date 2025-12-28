// frontend/sns/lib/features/shippingAddress/infrastructure/shipping_address_repository_http.dart
import 'dart:convert';

import 'package:flutter/foundation.dart';
import 'package:dio/dio.dart';
import 'package:firebase_auth/firebase_auth.dart';

String _resolveApiBase() {
  const v = String.fromEnvironment('API_BASE', defaultValue: '');
  final s = v.trim();
  if (s.isEmpty) {
    throw Exception(
      'API_BASE is not set (use --dart-define=API_BASE=https://...)',
    );
  }
  // Ensure no trailing slash
  return s.endsWith('/') ? s.substring(0, s.length - 1) : s;
}

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
  final String
  street2; // optional in UI, but backend entity currently has string (can be "")
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

/// POST body
class CreateShippingAddressInput {
  CreateShippingAddressInput({
    required this.userId,
    required this.zipCode,
    required this.state,
    required this.city,
    required this.street,
    this.street2,
    this.country = 'JP',
  });

  final String userId;
  final String zipCode;
  final String state;
  final String city;
  final String street;
  final String? street2;
  final String country;

  Map<String, dynamic> toJson() => {
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
class UpdateShippingAddressInput {
  UpdateShippingAddressInput({
    this.zipCode,
    this.state,
    this.city,
    this.street,
    this.street2,
    this.country,
  });

  final String? zipCode;
  final String? state;
  final String? city;
  final String? street;
  final String? street2;
  final String? country;

  Map<String, dynamic> toJson() {
    final m = <String, dynamic>{};

    void put(String k, String? v) {
      if (v == null) {
        return;
      }
      m[k] = v.trim();
    }

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
    final resolved = (baseUrl ?? _resolveApiBase()).trim();
    final normalized = resolved.endsWith('/')
        ? resolved.substring(0, resolved.length - 1)
        : resolved;

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

    // ✅ Request/Response logger + Firebase token injector
    _dio.interceptors.add(
      InterceptorsWrapper(
        onRequest: (options, handler) async {
          try {
            final u = _auth.currentUser;
            if (u != null) {
              final token = await u.getIdToken();
              options.headers['Authorization'] = 'Bearer $token';
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
        onError: (e, handler) {
          // ✅ 既存のエラー詳細ログ
          _logDioError(e);

          // ✅ NEW: 失敗時の “要約ログ” を追加（request/response をまとめて出す）
          _logFailureSummary(e);

          handler.next(e);
        },
      ),
    );
  }

  final Dio _dio;
  final FirebaseAuth _auth;
  final CancelToken _cancelToken = CancelToken();

  void dispose() {
    if (!_cancelToken.isCancelled) {
      _cancelToken.cancel('disposed');
    }
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
  // API
  // ------------------------------------------------------------

  /// GET /shipping-addresses/{id}
  Future<ShippingAddress> getById(String id) async {
    final s = id.trim();
    if (s.isEmpty) {
      throw Exception('invalid id');
    }

    try {
      final res = await _dio.get(
        '/shipping-addresses/$s',
        cancelToken: _cancelToken,
      );
      final data = _asMap(res.data);
      return ShippingAddress.fromJson(data);
    } on DioException catch (e) {
      // ✅ NEW: throw 前にも要約ログ（呼び出し側で握りつぶされる対策）
      _logFailureSummary(e, op: 'GET /shipping-addresses/$s');
      throw _normalizeDioError(e);
    } catch (e) {
      throw _normalizeDioError(e);
    }
  }

  /// POST /shipping-addresses
  Future<ShippingAddress> create(CreateShippingAddressInput inData) async {
    try {
      final res = await _dio.post(
        '/shipping-addresses',
        data: inData.toJson(),
        cancelToken: _cancelToken,
      );
      final data = _asMap(res.data);
      return ShippingAddress.fromJson(data);
    } on DioException catch (e) {
      _logFailureSummary(e, op: 'POST /shipping-addresses');
      throw _normalizeDioError(e);
    } catch (e) {
      throw _normalizeDioError(e);
    }
  }

  /// PATCH /shipping-addresses/{id}
  ///
  /// backend が PUT の場合は `.patch` を `.put` に変更してください。
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
        '/shipping-addresses/$s',
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

  /// DELETE /shipping-addresses/{id}
  Future<void> delete(String id) async {
    final s = id.trim();
    if (s.isEmpty) {
      throw Exception('invalid id');
    }

    try {
      await _dio.delete('/shipping-addresses/$s', cancelToken: _cancelToken);
    } on DioException catch (e) {
      _logFailureSummary(e, op: 'DELETE /shipping-addresses/$s');
      throw _normalizeDioError(e);
    } catch (e) {
      throw _normalizeDioError(e);
    }
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
  // logging (debug only)
  // ------------------------------------------------------------

  void _log(String msg) {
    if (!kDebugMode) {
      return;
    }
    debugPrint(msg);
  }

  void _logRequest(RequestOptions o) {
    if (!kDebugMode) {
      return;
    }

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
    if (!kDebugMode) {
      return;
    }

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
    if (!kDebugMode) {
      return;
    }

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

  // ✅ NEW: Failure summary logger (request headers/body + response headers/body + status)
  void _logFailureSummary(DioException e, {String? op}) {
    if (!kDebugMode) return;

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
