import 'package:flutter/material.dart';

class PostCard extends StatelessWidget {
  final Map<String, dynamic> post;
  
  const PostCard({Key? key, required this.post}) : super(key: key);
  
  @override
  Widget build(BuildContext context) {
    return Card(
      margin: const EdgeInsets.symmetric(horizontal: 16, vertical: 8),
      elevation: 2,
      shape: RoundedRectangleBorder(
        borderRadius: BorderRadius.circular(12),
      ),
      child: Padding(
        padding: const EdgeInsets.all(16),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            // Author header
            Row(
              children: [
                CircleAvatar(
                  radius: 20,
                  backgroundImage: post['author']['iconUrl']?.isNotEmpty == true
                      ? NetworkImage(post['author']['iconUrl'])
                      : null,
                  child: post['author']['iconUrl']?.isNotEmpty != true
                      ? Text(
                          _getInitials(post['author']['name']),
                          style: const TextStyle(fontSize: 14, fontWeight: FontWeight.bold),
                        )
                      : null,
                ),
                const SizedBox(width: 12),
                Expanded(
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      Text(
                        post['author']['name'] ?? 'Unknown User',
                        style: const TextStyle(
                          fontWeight: FontWeight.bold,
                          fontSize: 16,
                        ),
                      ),
                      Text(
                        _formatDateTime(post['createdAt']),
                        style: const TextStyle(
                          color: Colors.grey,
                          fontSize: 12,
                        ),
                      ),
                    ],
                  ),
                ),
                // More options button
                IconButton(
                  onPressed: () => _showMoreOptions(context),
                  icon: const Icon(Icons.more_vert, color: Colors.grey),
                ),
              ],
            ),
            
            const SizedBox(height: 12),
            
            // Post content
            Text(
              post['content'] ?? post['text'] ?? '',
              style: const TextStyle(fontSize: 16),
            ),
            
            // Media content if available
            if (post['mediaUrl']?.isNotEmpty == true) ...[
              const SizedBox(height: 12),
              ClipRRect(
                borderRadius: BorderRadius.circular(8),
                child: Image.network(
                  post['mediaUrl'],
                  width: double.infinity,
                  height: 200,
                  fit: BoxFit.cover,
                  loadingBuilder: (context, child, loadingProgress) {
                    if (loadingProgress == null) return child;
                    return Container(
                      height: 200,
                      color: Colors.grey[200],
                      child: const Center(
                        child: CircularProgressIndicator(),
                      ),
                    );
                  },
                  errorBuilder: (context, error, stackTrace) {
                    return Container(
                      height: 200,
                      color: Colors.grey[300],
                      child: const Center(
                        child: Column(
                          mainAxisAlignment: MainAxisAlignment.center,
                          children: [
                            Icon(Icons.broken_image, size: 50, color: Colors.grey),
                            SizedBox(height: 8),
                            Text('画像を読み込めませんでした', style: TextStyle(color: Colors.grey)),
                          ],
                        ),
                      ),
                    );
                  },
                ),
              ),
            ],
            
            const SizedBox(height: 16),
            
            // Action buttons
            Row(
              mainAxisAlignment: MainAxisAlignment.spaceBetween,
              children: [
                // Like button
                Row(
                  children: [
                    IconButton(
                      onPressed: () => _toggleLike(context),
                      icon: Icon(
                        post['isLiked'] == true ? Icons.favorite : Icons.favorite_border,
                        color: post['isLiked'] == true ? Colors.red : Colors.grey,
                        size: 22,
                      ),
                    ),
                    Text(
                      '${post['likesCount'] ?? 0}',
                      style: const TextStyle(color: Colors.grey, fontSize: 14),
                    ),
                  ],
                ),
                
                // Comment button
                Row(
                  children: [
                    IconButton(
                      onPressed: () => _showComments(context),
                      icon: const Icon(Icons.comment_outlined, color: Colors.grey, size: 22),
                    ),
                    Text(
                      '${post['commentsCount'] ?? 0}',
                      style: const TextStyle(color: Colors.grey, fontSize: 14),
                    ),
                  ],
                ),
                
                // Share button
                IconButton(
                  onPressed: () => _sharePost(context),
                  icon: const Icon(Icons.share_outlined, color: Colors.grey, size: 22),
                ),
                
                // Bookmark button
                IconButton(
                  onPressed: () => _toggleBookmark(context),
                  icon: Icon(
                    post['isBookmarked'] == true ? Icons.bookmark : Icons.bookmark_border,
                    color: post['isBookmarked'] == true ? Colors.blue : Colors.grey,
                    size: 22,
                  ),
                ),
              ],
            ),
          ],
        ),
      ),
    );
  }
  
  String _getInitials(String? name) {
    if (name == null || name.isEmpty) return 'U';
    final words = name.split(' ');
    if (words.length >= 2) {
      return '${words[0][0]}${words[1][0]}'.toUpperCase();
    }
    return name[0].toUpperCase();
  }
  
  String _formatDateTime(String? dateTimeString) {
    if (dateTimeString == null) return '';
    
    try {
      final dateTime = DateTime.parse(dateTimeString);
      final now = DateTime.now();
      final difference = now.difference(dateTime);
      
      if (difference.inMinutes < 1) {
        return 'たった今';
      } else if (difference.inMinutes < 60) {
        return '${difference.inMinutes}分前';
      } else if (difference.inHours < 24) {
        return '${difference.inHours}時間前';
      } else if (difference.inDays < 7) {
        return '${difference.inDays}日前';
      } else {
        return '${dateTime.month}/${dateTime.day}';
      }
    } catch (e) {
      return '';
    }
  }
  
  void _toggleLike(BuildContext context) {
    // TODO: Implement like functionality
    ScaffoldMessenger.of(context).showSnackBar(
      const SnackBar(
        content: Text('いいね機能は実装予定です'),
        duration: Duration(seconds: 1),
      ),
    );
  }
  
  void _showComments(BuildContext context) {
    // TODO: Implement comments functionality
    ScaffoldMessenger.of(context).showSnackBar(
      const SnackBar(
        content: Text('コメント機能は実装予定です'),
        duration: Duration(seconds: 1),
      ),
    );
  }
  
  void _sharePost(BuildContext context) {
    // TODO: Implement share functionality
    ScaffoldMessenger.of(context).showSnackBar(
      const SnackBar(
        content: Text('シェア機能は実装予定です'),
        duration: Duration(seconds: 1),
      ),
    );
  }
  
  void _toggleBookmark(BuildContext context) {
    // TODO: Implement bookmark functionality
    ScaffoldMessenger.of(context).showSnackBar(
      const SnackBar(
        content: Text('ブックマーク機能は実装予定です'),
        duration: Duration(seconds: 1),
      ),
    );
  }
  
  void _showMoreOptions(BuildContext context) {
    showModalBottomSheet(
      context: context,
      shape: const RoundedRectangleBorder(
        borderRadius: BorderRadius.vertical(top: Radius.circular(16)),
      ),
      builder: (context) => Container(
        padding: const EdgeInsets.symmetric(vertical: 20),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            ListTile(
              leading: const Icon(Icons.edit),
              title: const Text('投稿を編集'),
              onTap: () {
                Navigator.pop(context);
                _editPost(context);
              },
            ),
            ListTile(
              leading: const Icon(Icons.delete, color: Colors.red),
              title: const Text('投稿を削除', style: TextStyle(color: Colors.red)),
              onTap: () {
                Navigator.pop(context);
                _deletePost(context);
              },
            ),
            ListTile(
              leading: const Icon(Icons.report),
              title: const Text('投稿を報告'),
              onTap: () {
                Navigator.pop(context);
                _reportPost(context);
              },
            ),
          ],
        ),
      ),
    );
  }
  
  void _editPost(BuildContext context) {
    // TODO: Implement edit functionality
    ScaffoldMessenger.of(context).showSnackBar(
      const SnackBar(
        content: Text('編集機能は実装予定です'),
        duration: Duration(seconds: 1),
      ),
    );
  }
  
  void _deletePost(BuildContext context) {
    showDialog(
      context: context,
      builder: (context) => AlertDialog(
        title: const Text('投稿を削除'),
        content: const Text('この投稿を削除しますか？この操作は元に戻せません。'),
        actions: [
          TextButton(
            onPressed: () => Navigator.pop(context),
            child: const Text('キャンセル'),
          ),
          TextButton(
            onPressed: () {
              Navigator.pop(context);
              // TODO: Implement delete functionality
              ScaffoldMessenger.of(context).showSnackBar(
                const SnackBar(
                  content: Text('削除機能は実装予定です'),
                  duration: Duration(seconds: 1),
                ),
              );
            },
            style: TextButton.styleFrom(foregroundColor: Colors.red),
            child: const Text('削除'),
          ),
        ],
      ),
    );
  }
  
  void _reportPost(BuildContext context) {
    // TODO: Implement report functionality
    ScaffoldMessenger.of(context).showSnackBar(
      const SnackBar(
        content: Text('報告機能は実装予定です'),
        duration: Duration(seconds: 1),
      ),
    );
  }
}
