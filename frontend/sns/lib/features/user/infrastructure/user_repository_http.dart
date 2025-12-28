// frontend/sns/lib/features/user/infrastructure/user_repository_http.dart
import 'dart:async';
import 'dart:convert';

import 'package:flutter/foundation.dart';
import 'package:dio/dio.dart';
import 'package:firebase_auth/firebase_auth.dart';

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

  /// Backend が ISO8601 を返しても、DateTime を返しても吸収するため any で受ける
  final DateTime? createdAt;
  final DateTime? updatedAt;
  final DateTime? deletedAt;

  static DateTime? _parseDate(dynamic v) {
    if (v == null) return null;
    if (v is DateTime) return v.toUtc();
    if (v is String) {
      final t = DateTime.tryParse(v);
      return t?.toUtc();
    }
    return null;
  }

  factory UserDTO.fromJson(Map<String, dynamic> json) {
    return UserDTO(
      id: (json['id'] ?? '').toString().trim(),
      firstName: (json['first_name'] ?? json['firstName'])?.toString(),
      firstNameKana: (json['first_name_kana'] ?? json['firstNameKana'])
          ?.toString(),
      lastNameKana: (json['last_name_kana'] ?? json['lastNameKana'])
          ?.toString(),
      lastName: (json['last_name'] ?? json['lastName'])?.toString(),
      createdAt: _parseDate(json['createdAt'] ?? json['created_at']),
      updatedAt: _parseDate(json['updatedAt'] ?? json['updated_at']),
      deletedAt: _parseDate(json['deletedAt'] ?? json['deleted_at']),
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

/// Backend へ POST する payload
class CreateUserBody {
  CreateUserBody({
    required this.id,
    this.firstName,
    this.firstNameKana,
    this.lastNameKana,
    this.lastName,
  });

  final String id;
  final String? firstName;
  final String? firstNameKana;
  final String? lastNameKana;
  final String? lastName;

  Map<String, dynamic> toJson() => <String, dynamic>{
    'id': id,
    if (firstName != null) 'first_name': firstName,
    if (firstNameKana != null) 'first_name_kana': firstNameKana,
    if (lastNameKana != null) 'last_name_kana': lastNameKana,
    if (lastName != null) 'last_name': lastName,
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
    if (firstName != null) 'first_name': firstName,
    if (firstNameKana != null) 'first_name_kana': firstNameKana,
    if (lastNameKana != null) 'last_name_kana': lastNameKana,
    if (lastName != null) 'last_name': lastName,
  };
}

/// SNS Flutter 用 UserRepository (HTTP)
/// - API_BASE or API_BASE_URL は `--dart-define=...` で渡す想定
class UserRepositoryHttp {
  UserRepositoryHttp({Dio? dio, FirebaseAuth? auth, String? baseUrl})
    : _auth = auth ?? FirebaseAuth.instance,
      _dio = dio ?? Dio() {
    final resolved = _resolveApiBase(override: baseUrl).trim();
    if (resolved.isEmpty) {
      throw Exception(
        'API_BASE is not set (use --dart-define=API_BASE=https://... or API_BASE_URL=https://...)',
      );
    }

    final normalized = resolved.endsWith('/')
        ? resolved.substring(0, resolved.length - 1)
        : resolved;

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

    // ✅ Request/Response logger + Firebase token injector
    _dio.interceptors.add(
      InterceptorsWrapper(
        onRequest: (options, handler) async {
          try {
            // Firebase ID token を自動付与
            final u = _auth.currentUser;
            if (u != null) {
              final token = await u.getIdToken();
              options.headers['Authorization'] = 'Bearer $token';
            }
          } catch (e) {
            // token取得失敗はログだけ（request自体は流す）
            _log('[UserRepositoryHttp] token error: $e');
          }

          _logRequest(options);
          handler.next(options);
        },
        onResponse: (response, handler) {
          _logResponse(response);
          handler.next(response);
        },
        onError: (e, handler) {
          // ✅ 既存の DioError ログ
          _logDioError(e);

          // ✅ “失敗時のログ” を追加（原因調査用）
          _logFailureSummary(e);

          handler.next(e);
        },
      ),
    );

    _log('[UserRepositoryHttp] init baseUrl=$normalized');
  }

  final Dio _dio;
  final FirebaseAuth _auth;

  void dispose() {
    _dio.close(force: true);
  }

  // ----------------------------
  // API
  // ----------------------------

  /// GET /users/{id}
  Future<UserDTO> getById(String id) async {
    final trimmed = id.trim();
    if (trimmed.isEmpty) {
      throw ArgumentError('id is empty');
    }

    try {
      final res = await _dio.get('/users/$trimmed');
      final data = _asMap(res.data);
      return UserDTO.fromJson(data);
    } on DioException catch (e) {
      _logFailureSummary(e, op: 'GET /users/$trimmed');
      throw _toException(e, op: 'GET /users/$trimmed');
    }
  }

  /// POST /users
  Future<UserDTO> create(CreateUserBody body) async {
    if (body.id.trim().isEmpty) {
      throw ArgumentError('id is empty');
    }

    try {
      final res = await _dio.post('/users', data: body.toJson());
      final data = _asMap(res.data);
      return UserDTO.fromJson(data);
    } on DioException catch (e) {
      _logFailureSummary(e, op: 'POST /users');
      throw _toException(e, op: 'POST /users');
    }
  }

  /// PATCH /users/{id}
  Future<UserDTO> update(String id, UpdateUserBody body) async {
    final trimmed = id.trim();
    if (trimmed.isEmpty) {
      throw ArgumentError('id is empty');
    }

    try {
      final res = await _dio.patch('/users/$trimmed', data: body.toJson());
      final data = _asMap(res.data);
      return UserDTO.fromJson(data);
    } on DioException catch (e) {
      _logFailureSummary(e, op: 'PATCH /users/$trimmed');
      throw _toException(e, op: 'PATCH /users/$trimmed');
    }
  }

  /// DELETE /users/{id}
  Future<void> delete(String id) async {
    final trimmed = id.trim();
    if (trimmed.isEmpty) {
      throw ArgumentError('id is empty');
    }

    try {
      await _dio.delete('/users/$trimmed');
    } on DioException catch (e) {
      _logFailureSummary(e, op: 'DELETE /users/$trimmed');
      throw _toException(e, op: 'DELETE /users/$trimmed');
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

  static String _resolveApiBase({String? override}) {
    final o = (override ?? '').trim();
    if (o.isNotEmpty) return o;

    const v1 = String.fromEnvironment('API_BASE_URL'); // newer
    const v2 = String.fromEnvironment('API_BASE'); // legacy
    final raw = (v1.isNotEmpty ? v1 : v2).trim();
    return raw;
  }

  Map<String, dynamic> _asMap(dynamic v) {
    if (v is Map<String, dynamic>) return v;
    if (v is Map) return Map<String, dynamic>.from(v);
    throw Exception(
      'Invalid response body: expected object, got ${v.runtimeType}',
    );
  }

  Exception _toException(DioException e, {required String op}) {
    final status = e.response?.statusCode;
    final data = e.response?.data;

    String msg = '$op failed';
    if (status != null) {
      msg += ' (status=$status)';
    }

    // backend が {"error": "..."} 返す想定
    if (data is Map && data['error'] != null) {
      msg += ': ${data['error']}';
      return Exception(msg);
    }

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

    // JSON payload（Map/List/String）を可能な範囲で出す
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
    b.writeln('[UserRepositoryHttp] request');
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
    b.writeln('[UserRepositoryHttp] response');
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

    final b = StringBuffer();
    b.writeln('[UserRepositoryHttp] error');
    b.writeln('  method=$method');
    b.writeln('  url=$url');
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

  // Failure summary logger (request payload + response headers/body)
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
    b.writeln('[UserRepositoryHttp] FAILURE');
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
