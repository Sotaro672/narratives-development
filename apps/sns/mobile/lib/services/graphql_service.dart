import 'package:graphql_flutter/graphql_flutter.dart';
import 'auth_service.dart';
import 'dart:io';

class GraphQLService {
  late ValueNotifier<GraphQLClient> client;
  final AuthService _authService = AuthService();

  GraphQLService() {
    _initializeClient();
  }

  void _initializeClient() {
    // ローカル開発環境での接続設定
    String endpoint;
    
    if (Platform.isAndroid) {
      // Android Emulator: 10.0.2.2 
      // Android 実機: PCのIPアドレスを使用（例: 192.168.1.100:8080）
      endpoint = 'http://10.0.2.2:8080/graphql'; 
    } else if (Platform.isIOS) {
      // iOS Simulator: localhost
      // iOS 実機: PCのIPアドレスを使用
      endpoint = 'http://localhost:8080/graphql';
    } else {
      // Web やその他のプラットフォーム
      endpoint = 'http://localhost:8080/graphql';
    }

    print('GraphQL endpoint: $endpoint'); // デバッグ用

    final HttpLink httpLink = HttpLink(
      endpoint,
      defaultHeaders: {
        'Content-Type': 'application/json',
        'Accept': 'application/json',
      },
    );

    final AuthLink authLink = AuthLink(
      getToken: () async {
        try {
          String? token = await _authService.getToken();
          return token != null ? 'Bearer $token' : null;
        } catch (e) {
          print('Error getting auth token: $e');
          return null;
        }
      },
    );

    final Link link = authLink.concat(httpLink);

    client = ValueNotifier(
      GraphQLClient(
        cache: GraphQLCache(store: HiveStore()),
        link: link,
        defaultPolicies: DefaultPolicies(
          watchQuery: Policies(
            errorPolicy: ErrorPolicy.all,
            fetchPolicy: FetchPolicy.cacheAndNetwork,
          ),
          query: Policies(
            errorPolicy: ErrorPolicy.all,
            fetchPolicy: FetchPolicy.cacheFirst,
          ),
        ),
      ),
    );
  }

  // ユーザー一覧取得
  Future<QueryResult> getUsers() async {
    const String getUsersQuery = '''
      query GetUsers {
        users {
          id
          email
          displayName
          avatarUrl
          createdAt
        }
      }
    ''';

    return await client.value.query(
      QueryOptions(document: gql(getUsersQuery)),
    );
  }

  // 投稿一覧取得
  Future<QueryResult> getPosts({int? limit, String? cursor}) async {
    const String getPostsQuery = '''
      query GetPosts(\$limit: Int, \$cursor: String) {
        posts(limit: \$limit, cursor: \$cursor) {
          edges {
            node {
              id
              content
              imageUrls
              author {
                id
                displayName
                avatarUrl
              }
              likesCount
              commentsCount
              createdAt
            }
            cursor
          }
          pageInfo {
            hasNextPage
            endCursor
          }
        }
      }
    ''';

    return await client.value.query(
      QueryOptions(
        document: gql(getPostsQuery),
        variables: {
          'limit': limit ?? 20,
          'cursor': cursor,
        },
      ),
    );
  }

  // 投稿作成
  Future<QueryResult> createPost({
    required String content,
    List<String>? imageUrls,
  }) async {
    const String createPostMutation = '''
      mutation CreatePost(\$input: CreatePostInput!) {
        createPost(input: \$input) {
          id
          content
          imageUrls
          author {
            id
            displayName
            avatarUrl
          }
          createdAt
        }
      }
    ''';

    return await client.value.mutate(
      MutationOptions(
        document: gql(createPostMutation),
        variables: {
          'input': {
            'content': content,
            'imageUrls': imageUrls ?? [],
          }
        },
      ),
    );
  }

  // いいね切り替え
  Future<QueryResult> toggleLike(String postId) async {
    const String toggleLikeMutation = '''
      mutation ToggleLike(\$postId: ID!) {
        toggleLike(postId: \$postId) {
          success
          likesCount
        }
      }
    ''';

    return await client.value.mutate(
      MutationOptions(
        document: gql(toggleLikeMutation),
        variables: {'postId': postId},
      ),
    );
  }
}
