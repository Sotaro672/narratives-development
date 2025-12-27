import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';

import '../shell/presentation/layout/app_shell.dart';

// pages
import '../../features/home/presentation/page/home_page.dart';
import '../../features/home/presentation/page/catalog.dart';

/// GoRouter instance
final GoRouter appRouter = GoRouter(
  initialLocation: '/',
  routes: [
    /// ✅ AppShell を必ず Navigator の内側に置く
    ShellRoute(
      builder: (context, state, child) {
        return AppShell(title: _titleFor(state), showBack: true, child: child);
      },
      routes: [
        /// Home
        GoRoute(
          path: '/',
          name: HomePage.pageName, // 'home'
          pageBuilder: (context, state) =>
              const NoTransitionPage(child: HomePage()),
        ),

        /// ✅ Detail (Catalog)
        /// 例: /catalog/:listId
        GoRoute(
          path: '/catalog/:listId',
          name: 'catalog',
          builder: (context, state) {
            final listId = state.pathParameters['listId'] ?? '';
            return CatalogPage(listId: listId); // ← catalog.dart 側の ctor に合わせる
          },
        ),
      ],
    ),
  ],

  /// ルートが見つからない / 例外時も AppShell の内側へ
  errorBuilder: (context, state) => AppShell(
    title: 'Not Found',
    showBack: true,
    child: Center(child: Text(state.error?.toString() ?? 'Not Found')),
  ),
);

String? _titleFor(GoRouterState state) {
  final loc = state.uri.toString();
  if (loc == '/') return null; // Homeはタイトル無しなど
  if (loc.startsWith('/catalog/')) return 'Catalog';
  return null;
}
