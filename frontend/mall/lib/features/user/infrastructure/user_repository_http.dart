// frontend/mall/lib/features/user/infrastructure/user_repository_http.dart
import 'dart:convert';

import 'package:flutter/foundation.dart';
import 'package:dio/dio.dart';
import 'package:firebase_auth/firebase_auth.dart';

// ✅ 共通 resolver（API_BASE / API_BASE_URL / fallback を吸収）
import '../../../app/config/api_base.dart';

/// User DTO (backend/internal/domain/user/entity.go に準拠: email/phone なし)
class UserDTO {
  UserDTO({
    required this.id,
    this.firstName,
    this.firstNameKana,
    this.lastNameKana,
    this.lastName,
    this.createdAt,
    this.updatedAt,
    this.deletedAt,
  });

  final String id;
  final String? firstName;
  final String? firstNameKana;
  final String? lastNameKana;
  final String? lastName;

  final DateTime? createdAt;
  final DateTime? updatedAt;
  final DateTime? deletedAt;

  static String? _optS(dynamic v) {
    final s = (v ?? '').toString().trim();
    return s.isEmpty ? null : s;
  }

  static DateTime? _parseDate(dynamic v) {
    if (v == null) {
      return null;
    }
    if (v is DateTime) {
      return v.toUtc();
    }
    if (v is String) {
      final t = DateTime.tryParse(v.trim());
      return t?.toUtc();
    }
    return null;
  }

  factory UserDTO.fromJson(Map<String, dynamic> json) {
    String s(dynamic v) => (v ?? '').toString().trim();

    return UserDTO(
      id: s(json['id']),
      firstName: _optS(json['first_name']),
      firstNameKana: _optS(json['first_name_kana']),
      lastNameKana: _optS(json['last_name_kana']),
      lastName: _optS(json['last_name']),
      createdAt: _parseDate(json['createdAt']),
      updatedAt: _parseDate(json['updatedAt']),
      deletedAt: _parseDate(json['deletedAt']),
    );
  }

  Map<String, dynamic> toJson() {
    return <String, dynamic>{
      'id': id,
      if (firstName != null) 'first_name': firstName,
      if (firstNameKana != null) 'first_name_kana': firstNameKana,
      if (lastNameKana != null) 'last_name_kana': lastNameKana,
      if (lastName != null) 'last_name': lastName,
      if (createdAt != null) 'createdAt': createdAt!.toUtc().toIso8601String(),
      if (updatedAt != null) 'updatedAt': updatedAt!.toUtc().toIso8601String(),
      if (deletedAt != null) 'deletedAt': deletedAt!.toUtc().toIso8601String(),
    };
  }
}

/// Backend へ POST する payload（旧式互換は削除）
/// - id はクライアントから送らない（サーバが Authorization の uid を採用）
class CreateUserBody {
  CreateUserBody({
    this.firstName,
    this.firstNameKana,
    this.lastNameKana,
    this.lastName,
  });

  final String? firstName;
  final String? firstNameKana;
  final String? lastNameKana;
  final String? lastName;

  Map<String, dynamic> toJson() => <String, dynamic>{
    if (firstName != null) 'first_name': firstName!.trim(),
    if (firstNameKana != null) 'first_name_kana': firstNameKana!.trim(),
    if (lastNameKana != null) 'last_name_kana': lastNameKana!.trim(),
    if (lastName != null) 'last_name': lastName!.trim(),
  };
}

/// Backend へ PATCH する payload（部分更新）
class UpdateUserBody {
  UpdateUserBody({
    this.firstName,
    this.firstNameKana,
    this.lastNameKana,
    this.lastName,
  });

  final String? firstName;
  final String? firstNameKana;
  final String? lastNameKana;
  final String? lastName;

  Map<String, dynamic> toJson() => <String, dynamic>{
    if (firstName != null) 'first_name': firstName!.trim(),
    if (firstNameKana != null) 'first_name_kana': firstNameKana!.trim(),
    if (lastNameKana != null) 'last_name_kana': lastNameKana!.trim(),
    if (lastName != null) 'last_name': lastName!.trim(),
  };
}

/// Mall Flutter 用 UserRepository (HTTP)
/// - ✅ /mall/me/users に統一（旧式互換は削除）
/// - baseUrl が ".../mall" 注入でも動くように prefix を自動調整
class UserRepositoryHttp {
  UserRepositoryHttp({Dio? dio, FirebaseAuth? auth, String? baseUrl})
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

    final b = Uri.parse(normalized);
    final basePath = b.path.replaceAll(RegExp(r'\/+$'), '');
    _pathPrefix = (basePath == '/mall' || basePath.endsWith('/mall'))
        ? ''
        : 'mall';

    _dio.options = BaseOptions(
      baseUrl: normalized,
      connectTimeout: const Duration(seconds: 12),
      receiveTimeout: const Duration(seconds: 12),
      sendTimeout: const Duration(seconds: 12),
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
              final t = token.toString().trim();
              if (t.isNotEmpty) {
                options.headers['Authorization'] = 'Bearer $t';
              }
            }
          } catch (e) {
            _log('[UserRepositoryHttp] token error: $e');
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
                final t = token.toString().trim();
                if (t.isNotEmpty) {
                  final opts = e.requestOptions;
                  opts.extra['__retried401'] = true;
                  opts.headers['Authorization'] = 'Bearer $t';

                  _log(
                    '[UserRepositoryHttp] retrying once with refreshed token: ${opts.method} ${opts.uri}',
                  );

                  final res = await _dio.fetch(opts);
                  handler.resolve(res);
                  return;
                }
              }
            } catch (ex) {
              _log('[UserRepositoryHttp] 401 retry failed: $ex');
            }
          }

          handler.next(e);
        },
      ),
    );

    _log('[UserRepositoryHttp] init baseUrl=$normalized prefix=$_pathPrefix');
  }

  final Dio _dio;
  final FirebaseAuth _auth;
  final CancelToken _cancelToken = CancelToken();

  late final String _pathPrefix; // '' or 'mall'

  void dispose() {
    if (!_cancelToken.isCancelled) {
      _cancelToken.cancel('disposed');
    }
    _dio.close(force: true);
  }

  // ----------------------------
  // API (me only)
  // ----------------------------

  static const String _meUsersPath = 'me/users';

  /// GET /mall/me/users
  Future<UserDTO> getMe() async {
    try {
      final res = await _dio.get(_p(_meUsersPath), cancelToken: _cancelToken);
      final data = _asMap(res.data);
      return UserDTO.fromJson(data);
    } on DioException catch (e) {
      _logFailureSummary(e, op: 'GET /$_meUsersPath');
      throw _toException(e, op: 'GET /$_meUsersPath');
    }
  }

  /// POST /mall/me/users
  Future<UserDTO> create(CreateUserBody body) async {
    try {
      final res = await _dio.post(
        _p(_meUsersPath),
        data: body.toJson(),
        cancelToken: _cancelToken,
      );
      final data = _asMap(res.data);
      return UserDTO.fromJson(data);
    } on DioException catch (e) {
      _logFailureSummary(e, op: 'POST /$_meUsersPath');
      throw _toException(e, op: 'POST /$_meUsersPath');
    }
  }

  /// PATCH /mall/me/users
  Future<UserDTO> updateMe(UpdateUserBody body) async {
    try {
      final res = await _dio.patch(
        _p(_meUsersPath),
        data: body.toJson(),
        cancelToken: _cancelToken,
      );
      final data = _asMap(res.data);
      return UserDTO.fromJson(data);
    } on DioException catch (e) {
      _logFailureSummary(e, op: 'PATCH /$_meUsersPath');
      throw _toException(e, op: 'PATCH /$_meUsersPath');
    }
  }

  /// DELETE /mall/me/users
  Future<void> deleteMe() async {
    try {
      await _dio.delete(_p(_meUsersPath), cancelToken: _cancelToken);
    } on DioException catch (e) {
      _logFailureSummary(e, op: 'DELETE /$_meUsersPath');
      throw _toException(e, op: 'DELETE /$_meUsersPath');
    }
  }

  // ----------------------------
  // helpers
  // ----------------------------

  static const bool _envHttpLog = bool.fromEnvironment(
    'ENABLE_HTTP_LOG',
    defaultValue: false,
  );

  bool get _logEnabled => kDebugMode || _envHttpLog;

  String _p(String path) {
    var p = path.trim();
    if (p.startsWith('/')) {
      p = p.substring(1);
    }
    if (p.isEmpty) {
      return p;
    }
    if (_pathPrefix.isEmpty) {
      return '/$p';
    }
    return '/$_pathPrefix/$p';
  }

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
      } catch (_) {}
    }
    throw Exception(
      'Invalid response body: expected object, got ${v.runtimeType}',
    );
  }

  Exception _toException(DioException e, {required String op}) {
    final status = e.response?.statusCode;
    final data = e.response?.data;

    var msg = '$op failed';
    if (status != null) {
      msg += ' (status=$status)';
    }

    try {
      if (data is Map && data['error'] != null) {
        msg += ': ${data['error']}';
        return Exception(msg);
      }
      if (data is String) {
        final decoded = jsonDecode(data);
        if (decoded is Map && decoded['error'] != null) {
          msg += ': ${decoded['error']}';
          return Exception(msg);
        }
      }
    } catch (_) {}

    if (data != null) {
      msg += ': $data';
      return Exception(msg);
    }

    if (e.message != null && e.message!.trim().isNotEmpty) {
      msg += ': ${e.message}';
    }
    return Exception(msg);
  }

  // ----------------------------
  // logging
  // ----------------------------

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
      ..writeln('[UserRepositoryHttp] request')
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
      ..writeln('[UserRepositoryHttp] response')
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
      ..writeln('[UserRepositoryHttp] error')
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
      ..writeln('[UserRepositoryHttp] FAILURE')
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
