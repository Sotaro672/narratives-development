// frontend/mall/lib/app/routing/router.dart
import 'dart:async';

import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';
import 'package:firebase_auth/firebase_auth.dart';

import '../shell/presentation/layout/app_shell.dart';
import '../shell/presentation/components/footer.dart';

// ✅ route defs（package: に統一）
import 'package:mall/app/routing/routes.dart';

// ✅ redirect / stores（package: に統一）
import 'package:mall/app/routing/navigation.dart';

// ✅ app routes 분離（ShellRoute / GoRoute ツリー）
import 'package:mall/app/routing/app_routes.dart';

// ✅ header meta/actions 分離
import 'package:mall/app/routing/app_scaffold_meta.dart';
import 'package:mall/app/routing/header/header_actions.dart';

// ✅ FIX: HomePage を解決するために list.dart を import（package: に統一）
import 'package:mall/features/list/presentation/page/list.dart';

// ✅ NEW: avatar name store (for header title refresh)（package: に統一）
import 'package:mall/app/routing/avatar_name_store.dart';

GoRouter buildRouter({required bool firebaseReady, Object? initError}) {
  if (firebaseReady) return buildAppRouter();
  return buildPublicOnlyRouter(
    initError: initError ?? Exception('Firebase init failed'),
  );
}

GoRouter buildAppRouter() {
  // ✅ auth state changes should trigger redirect/UI refresh
  final authRefresh = GoRouterRefreshStream(
    FirebaseAuth.instance.userChanges(),
  );

  return GoRouter(
    initialLocation: AppRoutePath.home,
    refreshListenable: Listenable.merge([
      authRefresh,
      AvatarIdStore.I,
      AvatarNameStore.I, // ✅ backend avatarName -> header title refresh
      // NavStore.I,
    ]),
    redirect: (context, state) async => appRedirect(context, state),
    debugLogDiagnostics: true,

    routes: buildAppRoutes(firebaseReady: true),

    errorBuilder: (context, state) => AppShell(
      title: 'Not Found',
      showBack: true,
      actions: buildHeaderActionsFor(
        state,
        allowLogin: true,
        firebaseReady: true,
      ),
      footer: const SignedInFooter(),
      child: Center(child: Text(state.error?.toString() ?? 'Not Found')),
    ),
  );
}

GoRouter buildPublicOnlyRouter({required Object initError}) {
  return GoRouter(
    initialLocation: AppRoutePath.home,
    debugLogDiagnostics: true,
    routes: [
      GoRoute(
        path: AppRoutePath.login,
        name: AppRouteName.login,
        pageBuilder: (context, state) =>
            NoTransitionPage(child: _FirebaseInitErrorPage(error: initError)),
      ),
      GoRoute(
        path: AppRoutePath.createAccount,
        name: AppRouteName.createAccount,
        pageBuilder: (context, state) =>
            NoTransitionPage(child: _FirebaseInitErrorPage(error: initError)),
      ),
      GoRoute(
        path: AppRoutePath.shippingAddress,
        name: AppRouteName.shippingAddress,
        pageBuilder: (context, state) =>
            NoTransitionPage(child: _FirebaseInitErrorPage(error: initError)),
      ),
      GoRoute(
        path: AppRoutePath.billingAddress,
        name: AppRouteName.billingAddress,
        pageBuilder: (context, state) =>
            NoTransitionPage(child: _FirebaseInitErrorPage(error: initError)),
      ),
      GoRoute(
        path: AppRoutePath.avatarCreate,
        name: AppRouteName.avatarCreate,
        pageBuilder: (context, state) =>
            NoTransitionPage(child: _FirebaseInitErrorPage(error: initError)),
      ),
      GoRoute(
        path: AppRoutePath.avatarEdit,
        name: AppRouteName.avatarEdit,
        pageBuilder: (context, state) =>
            NoTransitionPage(child: _FirebaseInitErrorPage(error: initError)),
      ),
      GoRoute(
        path: AppRoutePath.avatar,
        name: AppRouteName.avatar,
        pageBuilder: (context, state) =>
            NoTransitionPage(child: _FirebaseInitErrorPage(error: initError)),
      ),
      GoRoute(
        path: AppRoutePath.cart,
        name: AppRouteName.cart,
        pageBuilder: (context, state) =>
            NoTransitionPage(child: _FirebaseInitErrorPage(error: initError)),
      ),
      GoRoute(
        path: AppRoutePath.preview,
        name: AppRouteName.preview,
        pageBuilder: (context, state) =>
            NoTransitionPage(child: _FirebaseInitErrorPage(error: initError)),
      ),
      GoRoute(
        path: AppRoutePath.payment,
        name: AppRouteName.payment,
        pageBuilder: (context, state) =>
            NoTransitionPage(child: _FirebaseInitErrorPage(error: initError)),
      ),
      GoRoute(
        path: AppRoutePath.walletContents,
        name: AppRouteName.walletContents,
        pageBuilder: (context, state) =>
            NoTransitionPage(child: _FirebaseInitErrorPage(error: initError)),
      ),
      GoRoute(
        path: AppRoutePath.userEdit,
        name: AppRouteName.userEdit,
        pageBuilder: (context, state) =>
            NoTransitionPage(child: _FirebaseInitErrorPage(error: initError)),
      ),
      ShellRoute(
        builder: (context, state, child) {
          return AppShell(
            title: resolveTitleFor(state),
            showBack: resolveShowBackFor(state),
            actions: buildHeaderActionsFor(
              state,
              allowLogin: true,
              firebaseReady: false,
            ),
            footer: null,
            child: child,
          );
        },
        routes: [
          GoRoute(
            path: AppRoutePath.home,
            name: AppRouteName.home,
            pageBuilder: _PublicHomePageBuilder.pageBuilder,
          ),
        ],
      ),
    ],
    errorBuilder: (context, state) => AppShell(
      title: 'Not Found',
      showBack: true,
      actions: buildHeaderActionsFor(
        state,
        allowLogin: true,
        firebaseReady: false,
      ),
      footer: null,
      child: Center(child: Text(state.error?.toString() ?? 'Not Found')),
    ),
  );
}

/// ------------------------------------------------------------
/// ✅ GoRouter の refresh 用（auth stream を ChangeNotifier に変換）
class GoRouterRefreshStream extends ChangeNotifier {
  GoRouterRefreshStream(Stream<dynamic> stream) {
    _sub = stream.listen((_) => notifyListeners());
  }

  late final StreamSubscription<dynamic> _sub;

  @override
  void dispose() {
    _sub.cancel();
    super.dispose();
  }
}

/// ------------------------------------------------------------
/// ✅ public-only Home を routes に載せるための wrapper
class _PublicHomePageBuilder {
  static Page<void> pageBuilder(BuildContext context, GoRouterState state) {
    return const NoTransitionPage(child: _PublicHomePage());
  }
}

class _PublicHomePage extends StatelessWidget {
  const _PublicHomePage();

  @override
  Widget build(BuildContext context) {
    return const HomePage();
  }
}

class _FirebaseInitErrorPage extends StatelessWidget {
  const _FirebaseInitErrorPage({required this.error});
  final Object error;

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      body: SafeArea(
        child: Center(
          child: ConstrainedBox(
            constraints: const BoxConstraints(maxWidth: 640),
            child: Padding(
              padding: const EdgeInsets.all(16),
              child: Column(
                mainAxisSize: MainAxisSize.min,
                children: [
                  const Icon(Icons.warning_amber_rounded, size: 48),
                  const SizedBox(height: 12),
                  Text(
                    'Firebase is not configured for Web.',
                    style: Theme.of(context).textTheme.titleLarge,
                    textAlign: TextAlign.center,
                  ),
                  const SizedBox(height: 8),
                  Text(error.toString(), textAlign: TextAlign.center),
                  const SizedBox(height: 16),
                  const Text(
                    'Fix:\n'
                    '1) Run: flutterfire configure (recommended)\n'
                    'or\n'
                    '2) Provide Firebase web settings via --dart-define.',
                    textAlign: TextAlign.center,
                  ),
                  const SizedBox(height: 16),
                  OutlinedButton(
                    onPressed: () => context.go(AppRoutePath.home),
                    child: const Text('Back to Home'),
                  ),
                ],
              ),
            ),
          ),
        ),
      ),
    );
  }
}
