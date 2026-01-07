// frontend/sns/lib/features/tokenBlueprint/infrastructure/token_blueprint_repository_http.dart
import 'dart:convert';

import 'package:http/http.dart' as http;

// ✅ 共通 resolver を使う（env名/fallbackのブレを防ぐ）
import '../../../app/config/api_base.dart';

class TokenBlueprintRepositoryHttp {
  TokenBlueprintRepositoryHttp({http.Client? client, String? apiBase})
    : _client = client ?? http.Client(),
      _apiBase = _normalizeBaseUrl(
        (apiBase ?? '').trim().isNotEmpty
            ? apiBase!.trim()
            : resolveSnsApiBase(),
      ) {
    if (_apiBase.trim().isEmpty) {
      throw StateError(
        'API base URL is empty (API_BASE_URL / API_BASE / fallback).',
      );
    }
  }

  final http.Client _client;
  final String _apiBase;

  void dispose() {
    _client.close();
  }

  static String _normalizeBaseUrl(String s) {
    s = s.trim();
    if (s.isEmpty) return s;
    while (s.endsWith('/')) {
      s = s.substring(0, s.length - 1);
    }
    return s;
  }

  Map<String, String> _jsonHeaders() => const {'Accept': 'application/json'};

  static Map<String, dynamic> _decodeJsonMap(String body) {
    final rawBody = body.trim();
    if (rawBody.isEmpty) return <String, dynamic>{};

    final raw = jsonDecode(rawBody);
    if (raw is Map<String, dynamic>) return raw;

    if (raw is Map) {
      final out = <String, dynamic>{};
      for (final e in raw.entries) {
        out[e.key.toString()] = e.value;
      }
      return out;
    }

    throw const FormatException('JSON is not an object');
  }

  /// ✅ wrapper 吸収: {data:{...}} を許容
  static Map<String, dynamic> _unwrapData(Map<String, dynamic> m) {
    final data = m['data'];
    if (data is Map<String, dynamic>) return data;
    if (data is Map) {
      final out = <String, dynamic>{};
      for (final e in data.entries) {
        out[e.key.toString()] = e.value;
      }
      return out;
    }
    return m;
  }

  Future<TokenBlueprintPatch?> fetchPatch(String tokenBlueprintId) async {
    final id = tokenBlueprintId.trim();
    if (id.isEmpty) {
      throw ArgumentError('tokenBlueprintId is empty');
    }

    // 1) primary: /patch
    final u1 = Uri.parse('$_apiBase/sns/token-blueprints/$id/patch');
    final r1 = await _client.get(u1, headers: _jsonHeaders());

    if (r1.statusCode == 200) {
      final m = _unwrapData(_decodeJsonMap(r1.body));
      return TokenBlueprintPatch.fromJson(m);
    }

    // 404 の時だけ fallback
    if (r1.statusCode == 404) {
      final u2 = Uri.parse('$_apiBase/sns/token-blueprints/$id');
      final r2 = await _client.get(u2, headers: _jsonHeaders());

      if (r2.statusCode == 200) {
        final m = _unwrapData(_decodeJsonMap(r2.body));
        return TokenBlueprintPatch.fromJson(m);
      }
      if (r2.statusCode == 404) {
        return null;
      }

      throw HttpException(
        'fetchPatch failed (fallback): ${r2.statusCode}',
        body: r2.body,
        url: u2.toString(),
      );
    }

    throw HttpException(
      'fetchPatch failed: ${r1.statusCode}',
      body: r1.body,
      url: u1.toString(),
    );
  }
}

class TokenBlueprintPatch {
  const TokenBlueprintPatch({
    this.name,
    this.symbol,
    this.brandId,
    this.brandName,
    this.companyId,
    this.companyName,
    this.description,
    this.iconUrl,
    this.minted,
  });

  final String? name;
  final String? symbol;

  final String? brandId;
  final String? brandName;

  // ✅ NEW: company
  final String? companyId;
  final String? companyName;

  final String? description;

  /// ✅ 正: "iconUrl" のみ（名揺れ吸収しない）
  final String? iconUrl;

  final bool? minted;

  factory TokenBlueprintPatch.fromJson(Map<String, dynamic> json) {
    String? s(dynamic v) {
      if (v == null) return null;
      final x = v.toString().trim();
      return x.isEmpty ? null : x;
    }

    bool? b(dynamic v) {
      if (v == null) return null;
      if (v is bool) return v;
      final x = v.toString().trim().toLowerCase();
      if (x == 'true') return true;
      if (x == 'false') return false;
      return null;
    }

    return TokenBlueprintPatch(
      name: s(json['name']),
      symbol: s(json['symbol']),
      brandId: s(json['brandId']),
      brandName: s(json['brandName']),
      companyId: s(json['companyId']),
      companyName: s(json['companyName']),
      description: s(json['description']),
      iconUrl: s(json['iconUrl']),
      minted: b(json['minted']),
    );
  }
}

class HttpException implements Exception {
  HttpException(this.message, {this.url, this.body});

  final String message;
  final String? url;
  final String? body;

  @override
  String toString() {
    final u = url == null ? '' : ' url=$url';
    final b = body == null
        ? ''
        : ' body=${body!.length > 300 ? body!.substring(0, 300) : body}';
    return 'HttpException($message$u$b)';
  }
}
