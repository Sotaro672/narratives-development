// frontend/sns/lib/features/tokenBlueprint/infrastructure/token_blueprint_repository_http.dart
import "dart:convert";

import "package:http/http.dart" as http;

/// Buyer-facing TokenBlueprint repository (HTTP).
///
/// Backend routes (expected):
/// - GET /sns/token-blueprints/{id}/patch  -> Patch JSON
///   (fallback) GET /sns/token-blueprints/{id}
///
/// How to configure API base:
/// - --dart-define=API_BASE=https://your-backend.example.com
/// If omitted, it falls back to Cloud Run default below.
class TokenBlueprintRepositoryHTTP {
  TokenBlueprintRepositoryHTTP({http.Client? client, String? apiBase})
    : _client = client ?? http.Client(),
      _apiBase = _normalizeBaseUrl(apiBase ?? _resolveApiBase());

  final http.Client _client;
  final String _apiBase;

  static String _resolveApiBase() {
    const env = String.fromEnvironment("API_BASE");
    final s = env.trim();
    if (s.isNotEmpty) return s;

    // Fallback (your current Cloud Run backend)
    return "https://narratives-backend-871263659099.asia-northeast1.run.app";
  }

  static String _normalizeBaseUrl(String s) {
    s = s.trim();
    if (s.isEmpty) return s;
    // remove trailing slash
    while (s.endsWith("/")) {
      s = s.substring(0, s.length - 1);
    }
    return s;
  }

  void _log(String msg) {
    // ✅ Flutter/Dart コンソールに出る確認ログ
    // ignore: avoid_print
    print("[TokenBlueprintRepositoryHTTP] $msg");
  }

  String _bodyPreview(String body, {int max = 500}) {
    final t = body.trim();
    if (t.length <= max) return t;
    return "${t.substring(0, max)}...(truncated ${t.length - max} chars)";
  }

  /// Fetch TokenBlueprint patch by ID.
  ///
  /// Returns null if not found (404).
  Future<TokenBlueprintPatch?> fetchPatch(String tokenBlueprintId) async {
    final id = tokenBlueprintId.trim();
    if (id.isEmpty) {
      throw ArgumentError("tokenBlueprintId is empty");
    }

    _log("fetchPatch start tokenBlueprintId=$id apiBase=$_apiBase");

    // Prefer /patch route
    final u1 = Uri.parse("$_apiBase/sns/token-blueprints/$id/patch");
    _log("request GET $u1");
    final r1 = await _client.get(u1, headers: _jsonHeaders());
    _log("response ${r1.statusCode} from $u1 body=${_bodyPreview(r1.body)}");

    if (r1.statusCode == 200) {
      final m = _decodeJsonMap(r1.body);
      _log("json keys (patch) = ${m.keys.toList()}");

      final patch = TokenBlueprintPatch.fromJson(m);

      _log(
        "parsed patch ok "
        "name='${patch.name ?? ""}' "
        "symbol='${patch.symbol ?? ""}' "
        "brandId='${patch.brandId ?? ""}' "
        "brandName='${patch.brandName ?? ""}' "
        "minted=${patch.minted} "
        "hasIconUrl=${(patch.iconUrl ?? '').trim().isNotEmpty}",
      );

      return patch;
    }

    if (r1.statusCode == 404) {
      _log("patch route returned 404. fallback to /sns/token-blueprints/$id");

      // fallback to /{id} route (in case handler uses a simpler route)
      final u2 = Uri.parse("$_apiBase/sns/token-blueprints/$id");
      _log("request GET $u2");
      final r2 = await _client.get(u2, headers: _jsonHeaders());
      _log("response ${r2.statusCode} from $u2 body=${_bodyPreview(r2.body)}");

      if (r2.statusCode == 200) {
        final m = _decodeJsonMap(r2.body);
        _log("json keys (fallback) = ${m.keys.toList()}");

        final patch = TokenBlueprintPatch.fromJson(m);

        _log(
          "parsed patch (fallback) ok "
          "name='${patch.name ?? ""}' "
          "symbol='${patch.symbol ?? ""}' "
          "brandId='${patch.brandId ?? ""}' "
          "brandName='${patch.brandName ?? ""}' "
          "minted=${patch.minted} "
          "hasIconUrl=${(patch.iconUrl ?? '').trim().isNotEmpty}",
        );

        return patch;
      }
      if (r2.statusCode == 404) {
        _log("fallback route also returned 404 -> return null");
        return null;
      }
      throw HttpException(
        "fetchPatch failed (fallback): ${r2.statusCode}",
        body: r2.body,
        url: u2.toString(),
      );
    }

    throw HttpException(
      "fetchPatch failed: ${r1.statusCode}",
      body: r1.body,
      url: u1.toString(),
    );
  }

  Map<String, String> _jsonHeaders() => const {"Accept": "application/json"};

  static Map<String, dynamic> _decodeJsonMap(String body) {
    final raw = jsonDecode(body);
    if (raw is Map<String, dynamic>) return raw;
    if (raw is Map) return raw.map((k, v) => MapEntry("$k", v));
    throw FormatException("JSON is not an object");
  }
}

/// TokenBlueprint patch DTO (buyer-facing).
///
/// Backend JSON (confirmed):
/// {"name": "...", "symbol": "...", "brandId": "...", "minted": true, ...}
class TokenBlueprintPatch {
  const TokenBlueprintPatch({
    this.name,
    this.symbol,
    this.brandId,
    this.brandName,
    this.description,
    this.iconUrl,
    this.minted,
  });

  final String? name;
  final String? symbol;
  final String? brandId;
  final String? brandName;
  final String? description;
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
      if (x == "true") return true;
      if (x == "false") return false;
      return null;
    }

    return TokenBlueprintPatch(
      name: s(json["name"]),
      symbol: s(json["symbol"]),
      brandId: s(json["brandId"]),
      brandName: s(json["brandName"]),
      description: s(json["description"]),
      iconUrl: s(json["iconUrl"]),
      minted: b(json["minted"]),
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
    final u = url == null ? "" : " url=$url";
    final b = body == null
        ? ""
        : " body=${body!.length > 300 ? body!.substring(0, 300) : body}";
    return "HttpException($message$u$b)";
  }
}
