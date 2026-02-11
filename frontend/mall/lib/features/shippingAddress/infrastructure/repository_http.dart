// frontend/mall/lib/features/shippingAddress/infrastructure/repository_http.dart
import 'dart:convert';

import 'package:flutter/foundation.dart';
import 'package:dio/dio.dart';
import 'package:firebase_auth/firebase_auth.dart';

import '../../../app/config/api_base.dart';

/// Domain model for Mall shipping address
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
  final String street2;
  final String country;
  final DateTime createdAt;
  final DateTime updatedAt;

  factory ShippingAddress.fromJson(Map<String, dynamic> json) {
    DateTime parseTime(dynamic v) {
      if (v is String) {
        final s = v.trim();
        if (s.isNotEmpty) {
          return DateTime.parse(s).toUtc();
        }
      }
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
}

/// ✅ Create payload (me only)
/// - docId は usecase がランダム採番（bodyでは送らない）
/// - userId は /me の文脈で auth から確定（bodyでは送らない）
class CreateShippingAddressInput {
  CreateShippingAddressInput({
    required this.zipCode,
    required this.state,
    required this.city,
    required this.street,
    this.street2,
    this.country = 'JP',
  });

  final String zipCode;
  final String state;
  final String city;
  final String street;
  final String? street2;
  final String country;

  Map<String, dynamic> toJson() => {
    'zipCode': zipCode.trim(),
    'state': state.trim(),
    'city': city.trim(),
    'street': street.trim(),
    // backend entity is string, so send "" when null
    'street2': (street2 ?? '').trim(),
    'country': country.trim(),
  };
}

/// ✅ Partial update payload (by docId)
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

  /// street2 は「消す」ケースがあり得るので "" を渡せば消去扱いにできます
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
    put('street2', street2);
    put('country', country);

    return m;
  }
}

class ShippingAddressRepositoryHttp {
  ShippingAddressRepositoryHttp({Dio? dio, FirebaseAuth? auth, String? baseUrl})
    : _auth = auth ?? FirebaseAuth.instance,
      _dio = dio ?? Dio() {
    final resolvedRaw = (baseUrl ?? '').trim().isNotEmpty
        ? baseUrl!.trim()
        : resolveApiBase().trim();

    if (resolvedRaw.isEmpty) {
      throw Exception(
        'API base URL is empty (API_BASE_URL / API_BASE / fallback).',
      );
    }

    final normalized = resolvedRaw.replaceAll(RegExp(r'\/+$'), '');

    // ✅ baseUrl に /mall が含まれる注入も許容
    final b = Uri.parse(normalized);
    final basePath = b.path.replaceAll(RegExp(r'\/+$'), '');
    _pathPrefix = (basePath.endsWith('/mall') || basePath == '/mall')
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

    _dio.interceptors.add(
      InterceptorsWrapper(
        onRequest: (options, handler) async {
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
          _logDioError(e);
          _logFailureSummary(e);

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

  late final String _pathPrefix;

  static const bool _envHttpLog = bool.fromEnvironment(
    'ENABLE_HTTP_LOG',
    defaultValue: false,
  );

  bool get _logEnabled => kDebugMode || _envHttpLog;

  void dispose() {
    if (!_cancelToken.isCancelled) {
      _cancelToken.cancel('disposed');
    }
    _dio.close(force: true);
  }

  // ------------------------------------------------------------
  // auth helpers
  // ------------------------------------------------------------

  void _requireSignedIn() {
    final u = _auth.currentUser;
    final uid = (u?.uid ?? '').trim();
    if (uid.isEmpty) {
      throw Exception('not signed in');
    }
  }

  // ------------------------------------------------------------
  // path helpers
  // ------------------------------------------------------------

  String _p(String path) {
    var raw = path.trim();
    if (raw.isEmpty) {
      return raw;
    }

    raw = raw.replaceAll(RegExp(r'^/+'), '');

    if (_pathPrefix.isEmpty) {
      return '/$raw';
    }

    final prefix = _pathPrefix.replaceAll(RegExp(r'^/+|/+$'), '');
    return '/$prefix/$raw';
  }

  // ------------------------------------------------------------
  // API (me only)
  // - POST は必ず id なし
  // - docId は usecase が採番
  // - userId は auth から確定（bodyでは送らない）
  // ------------------------------------------------------------

  static const String _mePath = 'me/shipping-addresses';

  /// GET /mall/me/shipping-addresses (list)
  Future<List<ShippingAddress>> listMine() async {
    _requireSignedIn();
    final res = await _dio.get(_p(_mePath), cancelToken: _cancelToken);

    final body = res.data;
    if (body is List) {
      return body
          .map((e) => ShippingAddress.fromJson(_asMap(e)))
          .toList(growable: false);
    }
    if (body is String) {
      final decoded = jsonDecode(body);
      if (decoded is List) {
        return decoded
            .map((e) => ShippingAddress.fromJson(_asMap(e)))
            .toList(growable: false);
      }
    }

    throw Exception(
      'Invalid response body: expected array, got ${body.runtimeType}',
    );
  }

  /// GET /mall/me/shipping-addresses/{id}
  Future<ShippingAddress> getById(String id) async {
    _requireSignedIn();
    final tid = id.trim();
    if (tid.isEmpty) {
      throw Exception('invalid id');
    }
    final res = await _dio.get(_p('$_mePath/$tid'), cancelToken: _cancelToken);
    final data = _asMap(res.data);
    return ShippingAddress.fromJson(data);
  }

  /// POST /mall/me/shipping-addresses (create)
  /// ✅ 必ず id なしで叩く
  Future<ShippingAddress> createMine({
    required String zipCode,
    required String state,
    required String city,
    required String street,
    String? street2,
    String country = 'JP',
  }) async {
    _requireSignedIn();

    final payload = CreateShippingAddressInput(
      zipCode: zipCode,
      state: state,
      city: city,
      street: street,
      street2: street2,
      country: country,
    );

    final res = await _dio.post(
      _p(_mePath), // ✅ id を付けない
      data: payload.toJson(),
      cancelToken: _cancelToken,
    );
    final data = _asMap(res.data);
    return ShippingAddress.fromJson(data);
  }

  /// PATCH /mall/me/shipping-addresses/{id} (partial update)
  Future<ShippingAddress> updateById(
    String id,
    UpdateShippingAddressInput inData,
  ) async {
    _requireSignedIn();
    final tid = id.trim();
    if (tid.isEmpty) {
      throw Exception('invalid id');
    }

    final body = inData.toJson();
    if (body.isEmpty) {
      return getById(tid);
    }

    final res = await _dio.patch(
      _p('$_mePath/$tid'),
      data: body,
      cancelToken: _cancelToken,
    );
    final data = _asMap(res.data);
    return ShippingAddress.fromJson(data);
  }

  /// DELETE /mall/me/shipping-addresses/{id}
  Future<void> deleteById(String id) async {
    _requireSignedIn();
    final tid = id.trim();
    if (tid.isEmpty) {
      throw Exception('invalid id');
    }
    await _dio.delete(_p('$_mePath/$tid'), cancelToken: _cancelToken);
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
      final decoded = jsonDecode(v);
      if (decoded is Map<String, dynamic>) {
        return decoded;
      }
      if (decoded is Map) {
        return Map<String, dynamic>.from(decoded);
      }
    }
    throw Exception(
      'Invalid response body: expected object, got ${v.runtimeType}',
    );
  }

  // ------------------------------------------------------------
  // logging
  // ------------------------------------------------------------

  void _log(String msg) {
    if (!_logEnabled) {
      return;
    }
    debugPrint(msg);
  }

  void _logRequest(RequestOptions o) {
    if (!_logEnabled) {
      return;
    }

    final method = o.method.toUpperCase();
    final url = o.uri.toString();

    final headers = <String, dynamic>{};
    o.headers.forEach((k, v) {
      headers[k] = (k.toLowerCase() == 'authorization') ? 'Bearer ***' : v;
    });

    String body = '';
    final d = o.data;
    if (d != null) {
      try {
        body = (d is String)
            ? d
            : (d is Map || d is List)
            ? jsonEncode(d)
            : d.toString();
      } catch (e) {
        body = '(failed to encode body: $e)';
      }
    }

    final b = StringBuffer()
      ..writeln('[ShippingAddressRepositoryHttp] request')
      ..writeln('  method=$method')
      ..writeln('  url=$url')
      ..writeln('  headers=${jsonEncode(headers)}');

    if (body.isNotEmpty) {
      b.writeln('  body=${_truncate(body, 1500)}');
    }

    debugPrint(b.toString());
  }

  void _logResponse(Response r) {
    if (!_logEnabled) {
      return;
    }

    final method = r.requestOptions.method.toUpperCase();
    final url = r.requestOptions.uri.toString();

    String body = '';
    try {
      final d = r.data;
      body = (d == null)
          ? ''
          : (d is String)
          ? d
          : (d is Map || d is List)
          ? jsonEncode(d)
          : d.toString();
    } catch (e) {
      body = '(failed to encode response body: $e)';
    }

    final b = StringBuffer()
      ..writeln('[ShippingAddressRepositoryHttp] response')
      ..writeln('  method=$method')
      ..writeln('  url=$url')
      ..writeln('  status=${r.statusCode}');

    if (body.isNotEmpty) {
      b.writeln('  body=${_truncate(body, 1500)}');
    }

    debugPrint(b.toString());
  }

  void _logDioError(DioException e) {
    if (!_logEnabled) {
      return;
    }

    final o = e.requestOptions;
    final method = o.method.toUpperCase();
    final url = o.uri.toString();
    final status = e.response?.statusCode;

    String resBody = '';
    try {
      final d = e.response?.data;
      resBody = (d == null)
          ? ''
          : (d is String)
          ? d
          : (d is Map || d is List)
          ? jsonEncode(d)
          : d.toString();
    } catch (_) {
      resBody = '(failed to encode error response body)';
    }

    final b = StringBuffer()
      ..writeln('[ShippingAddressRepositoryHttp] error')
      ..writeln('  method=$method')
      ..writeln('  url=$url');

    if (status != null) {
      b.writeln('  status=$status');
    }
    if ((e.message ?? '').trim().isNotEmpty) {
      b.writeln('  message=${e.message}');
    }
    if (resBody.isNotEmpty) {
      b.writeln('  responseBody=${_truncate(resBody, 1500)}');
    }

    debugPrint(b.toString());
  }

  void _logFailureSummary(DioException e, {String? op}) {
    if (!_logEnabled) {
      return;
    }

    final o = e.requestOptions;
    final method = o.method.toUpperCase();
    final url = o.uri.toString();
    final status = e.response?.statusCode;
    final statusLine = status != null ? 'status=$status' : 'status=?';

    final reqHeaders = <String, dynamic>{};
    o.headers.forEach((k, v) {
      reqHeaders[k] = (k.toLowerCase() == 'authorization') ? 'Bearer ***' : v;
    });

    String reqBody = '';
    try {
      final d = o.data;
      reqBody = (d == null)
          ? ''
          : (d is String)
          ? d
          : (d is Map || d is List)
          ? jsonEncode(d)
          : d.toString();
    } catch (ex) {
      reqBody = '(failed to encode request body: $ex)';
    }

    String resBody = '';
    try {
      final d = e.response?.data;
      resBody = (d == null)
          ? ''
          : (d is String)
          ? d
          : (d is Map || d is List)
          ? jsonEncode(d)
          : d.toString();
    } catch (ex) {
      resBody = '(failed to encode response body: $ex)';
    }

    final safeOp = (op ?? '').trim().isEmpty ? '(n/a)' : op!.trim();

    final b = StringBuffer()
      ..writeln('[ShippingAddressRepositoryHttp] FAILURE')
      ..writeln('  op=$safeOp')
      ..writeln('  $statusLine')
      ..writeln('  method=$method')
      ..writeln('  url=$url')
      ..writeln('  dioType=${e.type}')
      ..writeln('  requestHeaders=${_truncate(jsonEncode(reqHeaders), 1500)}');

    if (reqBody.isNotEmpty) {
      b.writeln('  requestBody=${_truncate(reqBody, 1500)}');
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
