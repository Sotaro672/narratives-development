import 'package:flutter/material.dart';
import '../services/firestore_service.dart';

class PostProvider extends ChangeNotifier {
  final FirestoreService _firestoreService = FirestoreService();
  
  bool _isLoading = false;
  List<Map<String, dynamic>> _posts = [];
  String? _error;
  
  bool get isLoading => _isLoading;
  List<Map<String, dynamic>> get posts => _posts;
  String? get error => _error;
  
  Future<void> loadPosts() async {
    _isLoading = true;
    _error = null;
    notifyListeners();
    
    try {
      _posts = await _firestoreService.getPosts();
    } catch (e) {
      _error = e.toString();
      // Load demo posts if Firestore fails
      _posts = [
        {
          'id': '1',
          'content': 'Welcome to Narratives SNS!',
          'author': {'name': 'System', 'avatar': null, 'iconUrl': ''},
          'mediaUrl': '',
          'createdAt': DateTime.now().toIso8601String(),
          'likesCount': 5,
          'commentsCount': 2,
        },
        {
          'id': '2',
          'content': 'This is a demo post to show the app is working.',
          'author': {'name': 'Demo User', 'avatar': null, 'iconUrl': ''},
          'mediaUrl': '',
          'createdAt': DateTime.now().subtract(const Duration(hours: 1)).toIso8601String(),
          'likesCount': 3,
          'commentsCount': 1,
        },
      ];
    } finally {
      _isLoading = false;
      notifyListeners();
    }
  }
  
  Future<void> refreshPosts() async {
    await loadPosts();
  }
  
  Future<void> createPost({
    required String text,
    String? mediaUrl,
  }) async {
    if (text.trim().isEmpty) {
      throw Exception('投稿内容を入力してください');
    }
    
    if (text.length > 300) {
      throw Exception('投稿は300文字以内で入力してください');
    }
    
    _isLoading = true;
    _error = null;
    notifyListeners();
    
    try {
      await _firestoreService.createPost(
        text: text.trim(),
        mediaUrl: mediaUrl?.trim(),
      );
      
      // Reload posts to show the new post
      await loadPosts();
    } catch (e) {
      _error = e.toString();
      throw e;
    } finally {
      _isLoading = false;
      notifyListeners();
    }
  }
  
  Future<void> updatePost({
    required String postId,
    required String text,
    String? mediaUrl,
  }) async {
    if (text.trim().isEmpty) {
      throw Exception('投稿内容を入力してください');
    }
    
    if (text.length > 300) {
      throw Exception('投稿は300文字以内で入力してください');
    }
    
    _isLoading = true;
    _error = null;
    notifyListeners();
    
    try {
      await _firestoreService.updatePost(
        postId: postId,
        text: text.trim(),
        mediaUrl: mediaUrl?.trim(),
      );
      
      // Reload posts to show the updated post
      await loadPosts();
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
      await _firestoreService.deletePost(postId);
      
      // Remove from local list
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
