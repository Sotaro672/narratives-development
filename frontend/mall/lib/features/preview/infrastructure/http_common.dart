// frontend/mall/lib/features/preview/infrastructure/http_common.dart
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

String normalizeBaseUrl(String s) {
  var v = s.trim();
  if (v.isEmpty) {
    return v;
  }
  while (v.endsWith('/')) {
    v = v.substring(0, v.length - 1);
  }
  return v;
}

Map<String, String> jsonHeaders() => const {'Accept': 'application/json'};

Map<String, String> jsonPostHeaders() => const {
  'Accept': 'application/json',
  'Content-Type': 'application/json',
};
