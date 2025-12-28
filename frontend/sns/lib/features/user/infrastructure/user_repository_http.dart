// frontend/sns/lib/features/user/infrastructure/user_repository_http.dart
import 'dart:async';

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
/// - API_BASE は `--dart-define=API_BASE=https://...` で渡す想定
class UserRepositoryHttp {
  UserRepositoryHttp({Dio? dio, FirebaseAuth? auth, String? baseUrl})
    : _auth = auth ?? FirebaseAuth.instance,
      _dio = dio ?? Dio() {
    final resolved = (baseUrl ?? const String.fromEnvironment('API_BASE'))
        .trim();
    if (resolved.isEmpty) {
      throw Exception(
        'API_BASE is not set (use --dart-define=API_BASE=https://...)',
      );
    }

    _dio.options = BaseOptions(
      baseUrl: resolved,
      connectTimeout: const Duration(seconds: 12),
      receiveTimeout: const Duration(seconds: 12),
      sendTimeout: const Duration(seconds: 12),
      headers: <String, dynamic>{'Content-Type': 'application/json'},
      // backend 側が /users 等なので、ここは素直に baseUrl を使う
    );

    _dio.interceptors.add(
      InterceptorsWrapper(
        onRequest: (options, handler) async {
          // Firebase ID token を自動付与
          final u = _auth.currentUser;
          if (u != null) {
            final token = await u.getIdToken();
            options.headers['Authorization'] = 'Bearer $token';
          }
          handler.next(options);
        },
      ),
    );
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
      throw _toException(e, op: 'GET /users/$trimmed');
    }
  }

  /// POST /users
  /// - 想定: id は Firebase uid を渡す（users/{uid} にする設計なら handler 側で対応）
  Future<UserDTO> create(CreateUserBody body) async {
    if (body.id.trim().isEmpty) {
      throw ArgumentError('id is empty');
    }

    try {
      final res = await _dio.post('/users', data: body.toJson());
      final data = _asMap(res.data);
      return UserDTO.fromJson(data);
    } on DioException catch (e) {
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
      throw _toException(e, op: 'DELETE /users/$trimmed');
    }
  }

  // ----------------------------
  // helpers
  // ----------------------------

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
    if (status != null) msg += ' (status=$status)';

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
}
