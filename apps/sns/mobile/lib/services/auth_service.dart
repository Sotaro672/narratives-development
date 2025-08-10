import 'package:firebase_auth/firebase_auth.dart';
import 'package:flutter_secure_storage/flutter_secure_storage.dart';
import 'package:graphql_flutter/graphql_flutter.dart';
import 'package:flutter/foundation.dart';

class AuthService {
  final FirebaseAuth _auth = FirebaseAuth.instance;
  // Web環境では flutter_secure_storage を使わない
  final FlutterSecureStorage? _storage = kIsWeb ? null : const FlutterSecureStorage();
  
  User? get currentUser => _auth.currentUser;
  Stream<User?> get authStateChanges => _auth.authStateChanges();

  // Email/Password登録
  Future<UserCredential?> signUpWithEmail(String email, String password) async {
    try {
      UserCredential result = await _auth.createUserWithEmailAndPassword(
        email: email,
        password: password,
      );
      
      // GraphQLでユーザープロファイルを作成
      await _createUserProfile(result.user!);
      
      return result;
    } catch (e) {
      throw Exception('アカウント作成に失敗しました: $e');
    }
  }

  // Email/Passwordログイン
  Future<UserCredential?> signInWithEmail(String email, String password) async {
    try {
      UserCredential result = await _auth.signInWithEmailAndPassword(
        email: email,
        password: password,
      );
      
      // トークンを保存
      String? token = await result.user?.getIdToken();
      if (token != null) {
        await _storage!.write(key: 'auth_token', value: token);
      }
      
      return result;
    } catch (e) {
      throw Exception('ログインに失敗しました: $e');
    }
  }

  // ログアウト（Web対応）
  Future<void> signOut() async {
    await _auth.signOut();
    if (!kIsWeb && _storage != null) {
      await _storage!.delete(key: 'auth_token');
    }
  }

  // トークン取得（Web対応）
  Future<String?> getToken() async {
    try {
      User? user = _auth.currentUser;
      if (user != null) {
        String token = await user.getIdToken(true);
        if (!kIsWeb && _storage != null) {
          await _storage!.write(key: 'auth_token', value: token);
        }
        return token;
      }
      if (!kIsWeb && _storage != null) {
        return await _storage!.read(key: 'auth_token');
      }
      return null;
    } catch (e) {
      return null;
    }
  }

  // GraphQLでユーザープロファイル作成
  Future<void> _createUserProfile(User user) async {
    // TODO: GraphQL mutationでユーザープロファイルを作成
    const String createUserMutation = '''
      mutation CreateUser(\$input: CreateUserInput!) {
        createUser(input: \$input) {
          id
          email
          displayName
          createdAt
        }
      }
    ''';
    
    final Map<String, dynamic> variables = {
      'input': {
        'firebaseUid': user.uid,
        'email': user.email,
        'displayName': user.displayName ?? user.email?.split('@')[0],
      }
    };
    
    // GraphQL clientを使用してmutationを実行
    // この部分は実際のGraphQLクライアントの実装に依存
  }
}
