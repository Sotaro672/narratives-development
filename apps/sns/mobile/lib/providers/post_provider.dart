import 'package:flutter/material.dart';
import 'package:graphql_flutter/graphql_flutter.dart';
import '../services/gcs_service.dart';

class PostProvider extends ChangeNotifier {
  bool _isLoading = false;
  List<Map<String, dynamic>> _posts = [];
  String? _error;
  GraphQLClient? _client;
  final GCSService _gcsService = GCSService();
  
  bool get isLoading => _isLoading;
  List<Map<String, dynamic>> get posts => _posts;
  String? get error => _error;
  
  void setGraphQLClient(GraphQLClient client) {
    _client = client;
    _gcsService.setGraphQLClient(client);
  }
  
  // GraphQL Queries
  static const String getAllPostsQuery = '''
    query GetAllPosts {
      posts {
        id
        content
        mediaUrl
        createdAt
        likesCount
        commentsCount
        author {
          id
          name
          iconUrl
        }
        isLiked
        isBookmarked
      }
    }
  ''';
  
  static const String createPostMutation = '''
    mutation CreatePost(\$content: String!, \$mediaUrl: String) {
      createPost(input: {
        content: \$content
        mediaUrl: \$mediaUrl
      }) {
        id
        content
        mediaUrl
        createdAt
        author {
          id
          name
          iconUrl
        }
      }
    }
  ''';
  
  Future<void> loadPosts() async {
    _isLoading = true;
    _error = null;
    notifyListeners();
    
    try {
      if (_client != null) {
        final QueryOptions options = QueryOptions(
          document: gql(getAllPostsQuery),
        );
        
        final QueryResult result = await _client!.query(options);
        
        if (result.hasException) {
          throw result.exception!;
        }
        
        if (result.data != null && result.data!['posts'] != null) {
          _posts = List<Map<String, dynamic>>.from(
            result.data!['posts'].map((post) => Map<String, dynamic>.from(post))
          );
        }
      } else {
        // Fallback to demo data if GraphQL client is not available
        await _loadDemoPosts();
      }
    } catch (e) {
      _error = e.toString();
      // Load demo posts as fallback
      await _loadDemoPosts();
    } finally {
      _isLoading = false;
      notifyListeners();
    }
  }
  
  Future<void> _loadDemoPosts() async {
    await Future.delayed(const Duration(seconds: 1));
    
    _posts = [
      {
        'id': '1',
        'content': 'Welcome to Narratives SNS! Network packages are offline but app works locally.',
        'author': {'name': 'System', 'iconUrl': ''},
        'mediaUrl': '',
        'createdAt': DateTime.now().toIso8601String(),
        'likesCount': 12,
        'commentsCount': 3,
        'isLiked': false,
        'isBookmarked': false,
      },
      {
        'id': '2',
        'content': 'This demonstrates offline functionality with core Flutter only.',
        'author': {'name': 'Demo User', 'iconUrl': ''},
        'mediaUrl': '',
        'createdAt': DateTime.now().subtract(const Duration(hours: 2)).toIso8601String(),
        'likesCount': 8,
        'commentsCount': 1,
        'isLiked': true,
        'isBookmarked': false,
      },
      {
        'id': '3',
        'content': 'GraphQL integration ready! Backend connection established.',
        'author': {'name': 'GraphQL Service', 'iconUrl': ''},
        'mediaUrl': '',
        'createdAt': DateTime.now().subtract(const Duration(hours: 4)).toIso8601String(),
        'likesCount': 15,
        'commentsCount': 5,
        'isLiked': false,
        'isBookmarked': true,
      },
    ];
  }
  
  Future<void> refreshPosts() async {
    await loadPosts();
  }
  
  Future<void> createPost({required String text, String? mediaUrl}) async {
    if (text.trim().isEmpty) {
      throw Exception('Please enter post content');
    }
    
    _isLoading = true;
    notifyListeners();
    
    try {
      if (_client != null) {
        final MutationOptions options = MutationOptions(
          document: gql(createPostMutation),
          variables: {
            'content': text.trim(),
            'mediaUrl': mediaUrl?.trim(),
          },
        );
        
        final QueryResult result = await _client!.mutate(options);
        
        if (result.hasException) {
          throw result.exception!;
        }
        
        // Reload posts to show the new post
        await loadPosts();
      } else {
        // Fallback to local creation
        await _createLocalPost(text, mediaUrl);
      }
    } catch (e) {
      _error = e.toString();
      // Fallback to local creation
      await _createLocalPost(text, mediaUrl);
    } finally {
      _isLoading = false;
      notifyListeners();
    }
  }
  
  Future<void> createPostWithImage({
    required String text,
    required String userId,
    String? mediaUrl,
    bool pickImage = false,
  }) async {
    if (text.trim().isEmpty) {
      throw Exception('Please enter post content');
    }
    
    _isLoading = true;
    notifyListeners();
    
    try {
      String? imageUrl = mediaUrl;
      
      // Pick and upload image if requested
      if (pickImage) {
        imageUrl = await _gcsService.pickAndUploadPostImage(userId);
      }
      
      if (_client != null) {
        final MutationOptions options = MutationOptions(
          document: gql(createPostMutation),
          variables: {
            'content': text.trim(),
            'mediaUrl': imageUrl,
          },
        );
        
        final QueryResult result = await _client!.mutate(options);
        
        if (result.hasException) {
          throw result.exception!;
        }
        
        await loadPosts();
      } else {
        await _createLocalPost(text, imageUrl);
      }
    } catch (e) {
      _error = e.toString();
      throw e;
    } finally {
      _isLoading = false;
      notifyListeners();
    }
  }
  
  Future<void> _createLocalPost(String text, String? mediaUrl) async {
    await Future.delayed(const Duration(milliseconds: 500));
    
    final newPost = {
      'id': DateTime.now().millisecondsSinceEpoch.toString(),
      'content': text.trim(),
      'author': {'name': 'Current User', 'iconUrl': ''},
      'mediaUrl': mediaUrl ?? '',
      'createdAt': DateTime.now().toIso8601String(),
      'likesCount': 0,
      'commentsCount': 0,
      'isLiked': false,
      'isBookmarked': false,
    };
    
    _posts.insert(0, newPost);
  }
  
  Future<void> updatePost({
    required String postId,
    required String text,
    String? mediaUrl,
  }) async {
    if (text.trim().isEmpty) {
      throw Exception('Please enter post content');
    }
    
    _isLoading = true;
    notifyListeners();
    
    try {
      await Future.delayed(const Duration(milliseconds: 500));
      
      final index = _posts.indexWhere((post) => post['id'] == postId);
      if (index == -1) {
        throw Exception('Post not found');
      }
      
      _posts[index] = {
        ..._posts[index],
        'content': text.trim(),
        'mediaUrl': mediaUrl?.trim() ?? _posts[index]['mediaUrl'],
        'updatedAt': DateTime.now().toIso8601String(),
      };
    } catch (e) {
      _error = e.toString();
      throw e;
    } finally {
      _isLoading = false;
      notifyListeners();
    }
  }
  
  Future<void> deletePost(String postId) async {
    _isLoading = true;
    _error = null;
    notifyListeners();
    
    try {
      // Simulate network delay
      await Future.delayed(const Duration(milliseconds: 500));
      
      _posts.removeWhere((post) => post['id'] == postId);
    } catch (e) {
      _error = e.toString();
      throw e;
    } finally {
      _isLoading = false;
      notifyListeners();
    }
  }
  
  void clearError() {
    _error = null;
    notifyListeners();
  }
}
