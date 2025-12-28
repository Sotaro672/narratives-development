// frontend/sns/lib/features/auth/presentation/page/shipping_address.dart
import 'dart:convert';

import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';
import 'package:firebase_auth/firebase_auth.dart';
import 'package:http/http.dart' as http;

import '../../../../app/shell/presentation/components/header.dart';

/// ✅ 認証メールリンクの着地点 + 配送先住所入力（雛形）
class ShippingAddressPage extends StatefulWidget {
  const ShippingAddressPage({
    super.key,
    this.mode,
    this.oobCode,
    this.continueUrl,
    this.lang,
    this.from,
    this.intent,
  });

  /// Firebase action params
  final String? mode; // e.g. verifyEmail
  final String? oobCode;
  final String? continueUrl;
  final String? lang;

  /// optional app params
  final String? from;
  final String? intent;

  @override
  State<ShippingAddressPage> createState() => _ShippingAddressPageState();
}

class _ShippingAddressPageState extends State<ShippingAddressPage> {
  bool _verifying = false;
  bool _verified = false;
  String? _verifyError;

  // ---- profile form ----
  final _lastNameCtrl = TextEditingController();
  final _lastNameKanaCtrl = TextEditingController();
  final _firstNameCtrl = TextEditingController();
  final _firstNameKanaCtrl = TextEditingController();

  // ---- address form (skeleton) ----
  final _zipCtrl = TextEditingController();
  final _prefCtrl = TextEditingController();
  final _cityCtrl = TextEditingController();
  final _addr1Ctrl = TextEditingController();
  final _addr2Ctrl = TextEditingController();

  bool _zipLoading = false;
  String? _zipError;

  bool _saving = false;
  String? _saveMsg;

  @override
  void initState() {
    super.initState();
    _maybeApplyActionCode();

    // ✅ 郵便番号が変わったら自動で検索（7桁になったタイミング）
    _zipCtrl.addListener(_onZipChanged);
  }

  @override
  void dispose() {
    _zipCtrl.removeListener(_onZipChanged);

    _lastNameCtrl.dispose();
    _lastNameKanaCtrl.dispose();
    _firstNameCtrl.dispose();
    _firstNameKanaCtrl.dispose();

    _zipCtrl.dispose();
    _prefCtrl.dispose();
    _cityCtrl.dispose();
    _addr1Ctrl.dispose();
    _addr2Ctrl.dispose();
    super.dispose();
  }

  String _s(String? v) => (v ?? '').trim();

  /// ✅ “戻る” の遷移先
  /// - from があれば最優先
  /// - continueUrl がアプリ内パスっぽければそこへ（https://.../catalog/... 等）
  /// - それ以外は '/'
  String _backTo() {
    final from = _s(widget.from);
    if (from.isNotEmpty) return from;

    final cu = _s(widget.continueUrl);
    if (cu.isNotEmpty) {
      final uri = Uri.tryParse(cu);
      if (uri != null) {
        final path = uri.path.isEmpty ? '/' : uri.path;
        final query = uri.query;
        return query.isEmpty ? path : '$path?$query';
      }
    }
    return '/';
  }

  Future<void> _maybeApplyActionCode() async {
    final mode = _s(widget.mode);
    final oob = _s(widget.oobCode);

    // ✅ verifyEmail 以外 / oobCode なしは何もしない（通常アクセス扱い）
    if (mode != 'verifyEmail' || oob.isEmpty) return;

    if (mounted) {
      setState(() {
        _verifying = true;
        _verified = false;
        _verifyError = null;
      });
    }

    try {
      await FirebaseAuth.instance.applyActionCode(oob);

      // ログイン中なら emailVerified を更新
      final user = FirebaseAuth.instance.currentUser;
      if (user != null) {
        await user.reload();
      }

      if (mounted) {
        setState(() {
          _verified = true;
        });
      }
    } on FirebaseAuthException catch (e) {
      if (mounted) {
        setState(() {
          _verifyError = _friendlyActionError(e);
        });
      }
    } catch (e) {
      if (mounted) {
        setState(() {
          _verifyError = e.toString();
        });
      }
    } finally {
      if (mounted) {
        setState(() {
          _verifying = false;
        });
      }
    }
  }

  String _friendlyActionError(FirebaseAuthException e) {
    switch (e.code) {
      case 'expired-action-code':
        return 'この認証リンクは期限切れです。もう一度認証メールを送信してください。';
      case 'invalid-action-code':
        return 'この認証リンクは無効です。もう一度認証メールを送信してください。';
      case 'user-disabled':
        return 'このアカウントは無効化されています。';
      default:
        return e.message ?? 'メール認証に失敗しました。';
    }
  }

  // ----------------------------------------------------------------
  // 郵便番号 → 住所自動入力
  // ----------------------------------------------------------------

  /// 例: "123-4567" or "1234567" → "1234567"
  String _normalizeZip(String s) {
    final only = s.replaceAll(RegExp(r'[^0-9]'), '');
    return only;
  }

  String? _lastResolvedZip; // 同じ郵便番号で連続呼び出ししないためのガード

  void _onZipChanged() {
    final zip = _normalizeZip(_zipCtrl.text);

    // 7桁になったら自動検索（同じ値は再検索しない）
    if (zip.length == 7 && zip != _lastResolvedZip) {
      _lastResolvedZip = zip;
      _lookupZipAndFill(zip);
    } else {
      // 途中入力のときはエラー表示を消す
      if (_zipError != null) {
        setState(() => _zipError = null);
      }
    }
  }

  Future<void> _lookupZipAndFill(String zip7) async {
    if (_zipLoading) return;

    if (mounted) {
      setState(() {
        _zipLoading = true;
        _zipError = null;
      });
    }

    try {
      // ✅ 日本の郵便番号API（無料）: https://zipcloud.ibsnet.co.jp/doc/api
      final uri = Uri.parse(
        'https://zipcloud.ibsnet.co.jp/api/search?zipcode=$zip7',
      );
      final res = await http.get(uri);

      if (res.statusCode != 200) {
        throw StateError('住所検索に失敗しました（HTTP ${res.statusCode}）。');
      }

      final json = jsonDecode(res.body) as Map<String, dynamic>;

      // zipcloud: { status: 200, message: null, results: [...] }
      final status = json['status'];
      if (status != 200) {
        final msg = (json['message'] ?? '住所検索に失敗しました。').toString();
        throw StateError(msg);
      }

      final results = json['results'];
      if (results == null) {
        // 見つからない場合は message が入ることが多い
        final msg = (json['message'] ?? '該当する住所が見つかりませんでした。').toString();
        throw StateError(msg);
      }

      final list = results as List<dynamic>;
      if (list.isEmpty) {
        throw StateError('該当する住所が見つかりませんでした。');
      }

      // 先頭を採用（通常は1件）
      final r0 = list.first as Map<String, dynamic>;
      final pref = (r0['address1'] ?? '').toString(); // 都道府県
      final city = (r0['address2'] ?? '').toString(); // 市区町村
      final town = (r0['address3'] ?? '').toString(); // 町域

      // ✅ フィールド反映
      _prefCtrl.text = pref;
      _cityCtrl.text = city;
      // addr1 に町域まで入れておく（番地はユーザーが追記）
      _addr1Ctrl.text = town;

      if (mounted) {
        setState(() {
          _zipError = null;
        });
      }
    } catch (e) {
      if (mounted) {
        setState(() {
          _zipError = e.toString();
        });
      }
    } finally {
      if (mounted) {
        setState(() {
          _zipLoading = false;
        });
      }
    }
  }

  // ----------------------------------------------------------------

  bool get _canSaveAddress {
    // 雛形：必須チェック（必要なら強化）
    return !_saving &&
        _s(_lastNameCtrl.text).isNotEmpty &&
        _s(_firstNameCtrl.text).isNotEmpty &&
        _s(_zipCtrl.text).isNotEmpty &&
        _s(_prefCtrl.text).isNotEmpty &&
        _s(_cityCtrl.text).isNotEmpty &&
        _s(_addr1Ctrl.text).isNotEmpty;
  }

  Future<void> _saveAddressDummy() async {
    if (mounted) {
      setState(() {
        _saveMsg = null;
        _saving = true;
      });
    }

    try {
      // ここで backend / Firestore などへ保存する（雛形のため、いまはダミー保存）
      await Future<void>.delayed(const Duration(milliseconds: 600));

      if (!mounted) return;

      setState(() {
        _saveMsg = '配送先情報を保存しました（ダミー）。';
      });

      // ✅ 保存後に請求先住所入力へ遷移
      // NOTE: ルーティングはプロジェクト側の go_router 定義に合わせてください。
      // ここでは一般的なパスを採用しています。
      context.go('/billing-address');
    } catch (e) {
      if (mounted) {
        setState(() {
          _saveMsg = e.toString();
        });
      }
    } finally {
      if (mounted) {
        setState(() {
          _saving = false;
        });
      }
    }
  }

  @override
  Widget build(BuildContext context) {
    final backTo = _backTo();

    final user = FirebaseAuth.instance.currentUser;
    final loggedIn = user != null;
    final emailVerified = user?.emailVerified ?? false;

    final mode = _s(widget.mode);
    final oob = _s(widget.oobCode);
    final cameFromEmailLink = (mode == 'verifyEmail' && oob.isNotEmpty);

    return Scaffold(
      body: SafeArea(
        child: Column(
          children: [
            // ✅ ここは Shell 外なので、右上 Sign in は出さない。戻るだけ。
            AppHeader(
              title: '配送先住所',
              showBack: true,
              backTo: backTo,
              actions: const [],
              onTapTitle: () => context.go('/'),
            ),

            Expanded(
              child: Center(
                child: ConstrainedBox(
                  constraints: const BoxConstraints(maxWidth: 520),
                  child: SingleChildScrollView(
                    padding: const EdgeInsets.all(16),
                    child: Column(
                      crossAxisAlignment: CrossAxisAlignment.stretch,
                      children: [
                        if (cameFromEmailLink) ...[
                          Text(
                            'メール認証の確認',
                            style: Theme.of(context).textTheme.titleLarge,
                          ),
                          const SizedBox(height: 8),

                          if (_verifying) ...[
                            const Text('確認中です…'),
                            const SizedBox(height: 12),
                            const LinearProgressIndicator(),
                            const SizedBox(height: 16),
                          ] else if (_verifyError != null) ...[
                            _InfoBox(
                              kind: _InfoKind.error,
                              text: _verifyError!,
                            ),
                            const SizedBox(height: 12),
                          ] else if (_verified) ...[
                            const _InfoBox(
                              kind: _InfoKind.ok,
                              text: 'メール認証が完了しました。続けて配送先情報を入力してください。',
                            ),
                            const SizedBox(height: 12),
                          ] else ...[
                            const _InfoBox(
                              kind: _InfoKind.info,
                              text: '認証リンクを確認します。',
                            ),
                            const SizedBox(height: 12),
                          ],
                        ] else ...[
                          const _InfoBox(
                            kind: _InfoKind.info,
                            text: '配送先情報を入力してください。',
                          ),
                          const SizedBox(height: 12),
                        ],

                        if (!loggedIn) ...[
                          const _InfoBox(
                            kind: _InfoKind.info,
                            text: '※ 住所の保存にはサインインが必要です。',
                          ),
                          const SizedBox(height: 8),
                          OutlinedButton(
                            onPressed: () {
                              final from = Uri.encodeComponent(
                                GoRouterState.of(context).uri.toString(),
                              );
                              context.go('/login?from=$from');
                            },
                            child: const Text('サインインへ'),
                          ),
                          const SizedBox(height: 16),
                        ] else if (!emailVerified) ...[
                          const _InfoBox(
                            kind: _InfoKind.info,
                            text: '※ 現在サインイン中ですが、メール認証が未完了の可能性があります。',
                          ),
                          const SizedBox(height: 16),
                        ],

                        Text(
                          'お届け先氏名',
                          style: Theme.of(context).textTheme.titleMedium,
                        ),
                        const SizedBox(height: 8),

                        TextField(
                          controller: _lastNameCtrl,
                          decoration: const InputDecoration(
                            labelText: '苗字',
                            border: OutlineInputBorder(),
                          ),
                        ),
                        const SizedBox(height: 12),

                        TextField(
                          controller: _firstNameCtrl,
                          decoration: const InputDecoration(
                            labelText: '名前',
                            border: OutlineInputBorder(),
                          ),
                        ),
                        const SizedBox(height: 12),

                        TextField(
                          controller: _lastNameKanaCtrl,
                          decoration: const InputDecoration(
                            labelText: '苗字かな',
                            border: OutlineInputBorder(),
                          ),
                        ),
                        const SizedBox(height: 12),

                        TextField(
                          controller: _firstNameKanaCtrl,
                          decoration: const InputDecoration(
                            labelText: '名前かな',
                            border: OutlineInputBorder(),
                          ),
                        ),
                        const SizedBox(height: 16),

                        Text(
                          '配送先住所',
                          style: Theme.of(context).textTheme.titleMedium,
                        ),
                        const SizedBox(height: 8),

                        TextField(
                          controller: _zipCtrl,
                          keyboardType: TextInputType.number,
                          decoration: InputDecoration(
                            labelText: '郵便番号（7桁）',
                            border: const OutlineInputBorder(),
                            helperText: '例: 1000001（ハイフン不要）',
                            suffixIcon: _zipLoading
                                ? const Padding(
                                    padding: EdgeInsets.all(12),
                                    child: SizedBox(
                                      width: 16,
                                      height: 16,
                                      child: CircularProgressIndicator(
                                        strokeWidth: 2,
                                      ),
                                    ),
                                  )
                                : IconButton(
                                    tooltip: '住所を自動入力',
                                    onPressed: () {
                                      final zip = _normalizeZip(_zipCtrl.text);
                                      if (zip.length == 7) {
                                        _lastResolvedZip = zip;
                                        _lookupZipAndFill(zip);
                                      } else {
                                        setState(() {
                                          _zipError = '郵便番号は7桁で入力してください。';
                                        });
                                      }
                                    },
                                    icon: const Icon(Icons.search),
                                  ),
                            errorText:
                                (_zipError == null || _zipError!.trim().isEmpty)
                                ? null
                                : _zipError,
                          ),
                        ),
                        const SizedBox(height: 12),

                        TextField(
                          controller: _prefCtrl,
                          decoration: const InputDecoration(
                            labelText: '都道府県',
                            border: OutlineInputBorder(),
                          ),
                        ),
                        const SizedBox(height: 12),

                        TextField(
                          controller: _cityCtrl,
                          decoration: const InputDecoration(
                            labelText: '市区町村',
                            border: OutlineInputBorder(),
                          ),
                        ),
                        const SizedBox(height: 12),

                        TextField(
                          controller: _addr1Ctrl,
                          decoration: const InputDecoration(
                            labelText: '住所１（町名・番地など）',
                            border: OutlineInputBorder(),
                          ),
                        ),
                        const SizedBox(height: 12),

                        TextField(
                          controller: _addr2Ctrl,
                          decoration: const InputDecoration(
                            labelText: '住所２（建物名・部屋番号など）',
                            border: OutlineInputBorder(),
                          ),
                        ),
                        const SizedBox(height: 16),

                        ElevatedButton(
                          onPressed: _canSaveAddress ? _saveAddressDummy : null,
                          child: _saving
                              ? const SizedBox(
                                  width: 18,
                                  height: 18,
                                  child: CircularProgressIndicator(
                                    strokeWidth: 2,
                                  ),
                                )
                              : const Text('この住所を保存する'),
                        ),

                        if (_saveMsg != null) ...[
                          const SizedBox(height: 12),
                          _InfoBox(
                            kind: _saveMsg!.contains('保存しました')
                                ? _InfoKind.ok
                                : _InfoKind.info,
                            text: _saveMsg!,
                          ),
                        ],
                      ],
                    ),
                  ),
                ),
              ),
            ),
          ],
        ),
      ),
    );
  }
}

enum _InfoKind { info, ok, error }

class _InfoBox extends StatelessWidget {
  const _InfoBox({required this.kind, required this.text});

  final _InfoKind kind;
  final String text;

  @override
  Widget build(BuildContext context) {
    final scheme = Theme.of(context).colorScheme;

    late final Color bg;
    switch (kind) {
      case _InfoKind.ok:
        bg = scheme.primaryContainer.withValues(alpha: 0.55);
        break;
      case _InfoKind.error:
        bg = scheme.errorContainer.withValues(alpha: 0.55);
        break;
      case _InfoKind.info:
        bg = scheme.surfaceContainerHighest.withValues(alpha: 0.55);
        break;
    }

    return Container(
      padding: const EdgeInsets.all(12),
      decoration: BoxDecoration(
        color: bg,
        borderRadius: BorderRadius.circular(12),
      ),
      child: Text(text, style: Theme.of(context).textTheme.bodyMedium),
    );
  }
}
