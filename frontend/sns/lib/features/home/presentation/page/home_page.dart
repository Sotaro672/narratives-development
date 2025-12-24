// frontend/sns/lib/features/home/presentation/page/home_page.dart
import 'dart:convert';

import 'package:flutter/material.dart';
import 'package:http/http.dart' as http;

/// Priority:
/// 1) --dart-define=API_BASE_URL=https://...
/// 2) fallback (edit if needed)
const String _fallbackBaseUrl =
    'https://narratives-backend-871263659099.asia-northeast1.run.app';

String _resolveApiBase() {
  const fromDefine = String.fromEnvironment('API_BASE_URL');
  final base = (fromDefine.isNotEmpty ? fromDefine : _fallbackBaseUrl).trim();
  return base.endsWith('/') ? base.substring(0, base.length - 1) : base;
}

String _prettyJsonIfPossible(String raw) {
  try {
    final decoded = jsonDecode(raw);
    return const JsonEncoder.withIndent('  ').convert(decoded);
  } catch (_) {
    return raw; // JSON じゃなければそのまま
  }
}

class HomePage extends StatefulWidget {
  const HomePage({super.key});

  static const String pageName = 'home';

  @override
  State<HomePage> createState() => _HomePageState();
}

class _HomePageState extends State<HomePage> {
  late Future<_HomePayload> _future;

  @override
  void initState() {
    super.initState();
    _future = _load();
  }

  Future<_HomePayload> _load() async {
    final base = _resolveApiBase();
    final uri = Uri.parse(
      '$base/sns/lists',
    ).replace(queryParameters: const {'page': '1', 'perPage': '20'});

    final res = await http.get(
      uri,
      headers: const {'Accept': 'application/json'},
    );

    return _HomePayload(
      url: uri.toString(),
      statusCode: res.statusCode,
      rawBody: res.body,
    );
  }

  void _reload() {
    setState(() {
      _future = _load();
    });
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: const Text('Home'),
        actions: [
          IconButton(
            onPressed: _reload,
            icon: const Icon(Icons.refresh),
            tooltip: 'Reload',
          ),
        ],
      ),
      body: FutureBuilder<_HomePayload>(
        future: _future,
        builder: (context, snap) {
          if (snap.connectionState == ConnectionState.waiting) {
            return const Center(child: CircularProgressIndicator());
          }
          if (snap.hasError) {
            return _ErrorView(error: snap.error, onRetry: _reload);
          }

          final data = snap.data!;
          final pretty = _prettyJsonIfPossible(data.rawBody);

          return Column(
            crossAxisAlignment: CrossAxisAlignment.stretch,
            children: [
              Container(
                padding: const EdgeInsets.all(12),
                color: Theme.of(context).colorScheme.surfaceContainerHighest,
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Text(
                      'GET ${data.url}',
                      style: Theme.of(context).textTheme.labelLarge,
                    ),
                    const SizedBox(height: 4),
                    Text(
                      'Status: ${data.statusCode}',
                      style: Theme.of(context).textTheme.labelMedium,
                    ),
                  ],
                ),
              ),
              const Divider(height: 1),
              Expanded(
                child: SingleChildScrollView(
                  padding: const EdgeInsets.all(12),
                  child: SelectableText(
                    pretty,
                    style: const TextStyle(
                      fontFamily: 'monospace',
                      fontSize: 12,
                      height: 1.35,
                    ),
                  ),
                ),
              ),
            ],
          );
        },
      ),
    );
  }
}

class _HomePayload {
  const _HomePayload({
    required this.url,
    required this.statusCode,
    required this.rawBody,
  });

  final String url;
  final int statusCode;
  final String rawBody;
}

class _ErrorView extends StatelessWidget {
  const _ErrorView({required this.error, required this.onRetry});

  final Object? error;
  final VoidCallback onRetry;

  @override
  Widget build(BuildContext context) {
    return Center(
      child: Padding(
        padding: const EdgeInsets.all(16),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            const Icon(Icons.error_outline, size: 40),
            const SizedBox(height: 12),
            Text(
              'Failed to load',
              style: Theme.of(context).textTheme.titleMedium,
            ),
            const SizedBox(height: 8),
            Text(
              error?.toString() ?? 'unknown error',
              textAlign: TextAlign.center,
            ),
            const SizedBox(height: 12),
            ElevatedButton(onPressed: onRetry, child: const Text('Retry')),
          ],
        ),
      ),
    );
  }
}
