import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import '../../providers/post_provider.dart';
import '../../providers/user_provider.dart';
import '../../widgets/post_card.dart';
import '../../widgets/create_post_dialog.dart';

class HomeScreen extends StatefulWidget {
  const HomeScreen({Key? key}) : super(key: key);

  @override
  State<HomeScreen> createState() => _HomeScreenState();
}

class _HomeScreenState extends State<HomeScreen> {
  int _currentIndex = 0;
  late PageController _pageController;

  @override
  void initState() {
    super.initState();
    _pageController = PageController();
    
    // 初期データ読み込み
    WidgetsBinding.instance.addPostFrameCallback((_) {
      context.read<PostProvider>().loadPosts();
      context.read<UserProvider>().loadCurrentUser();
    });
  }

  void _onTabTapped(int index) {
    if (index == 2) {
      // 投稿ボタンがタップされた場合
      _showCreatePostDialog();
    } else {
      // 通常のタブナビゲーション
      final actualIndex = index > 2 ? index - 1 : index;
      setState(() {
        _currentIndex = actualIndex;
      });
      _pageController.animateToPage(
        actualIndex,
        duration: const Duration(milliseconds: 300),
        curve: Curves.easeInOut,
      );
    }
  }

  void _showCreatePostDialog() {
    showDialog(
      context: context,
      builder: (context) => const CreatePostDialog(),
    );
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      body: PageView(
        controller: _pageController,
        onPageChanged: (index) {
          setState(() {
            _currentIndex = index;
          });
        },
        children: const [
          _FeedTab(),
          _ExploreTab(),
          _NotificationsTab(),
          _ProfileTab(),
        ],
      ),
      bottomNavigationBar: BottomNavigationBar(
        type: BottomNavigationBarType.fixed,
        currentIndex: _currentIndex >= 2 ? _currentIndex + 1 : _currentIndex,
        onTap: _onTabTapped,
        selectedItemColor: Colors.blue,
        unselectedItemColor: Colors.grey,
        items: const [
          BottomNavigationBarItem(
            icon: Icon(Icons.home),
            label: 'Home',
          ),
          BottomNavigationBarItem(
            icon: Icon(Icons.explore),
            label: 'Explore',
          ),
          BottomNavigationBarItem(
            icon: Icon(
              Icons.add_circle,
              size: 32,
            ),
            label: 'Post',
          ),
          BottomNavigationBarItem(
            icon: Icon(Icons.notifications),
            label: 'Notifications',
          ),
          BottomNavigationBarItem(
            icon: Icon(Icons.person),
            label: 'Profile',
          ),
        ],
      ),
    );
  }
}

class _FeedTab extends StatelessWidget {
  const _FeedTab();

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: const Text('Narratives'),
        elevation: 0,
      ),
      body: Consumer<PostProvider>(
        builder: (context, postProvider, child) {
          if (postProvider.isLoading && postProvider.posts.isEmpty) {
            return const Center(child: CircularProgressIndicator());
          }

          if (postProvider.posts.isEmpty) {
            return const Center(
              child: Text('No posts yet'),
            );
          }

          return RefreshIndicator(
            onRefresh: () => postProvider.refreshPosts(),
            child: ListView.builder(
              itemCount: postProvider.posts.length,
              itemBuilder: (context, index) {
                final post = postProvider.posts[index];
                return PostCard(post: post); // ← ここで投稿データを表示
              },
            ),
          );
        },
      ),
    );
  }
}

class _ExploreTab extends StatelessWidget {
  const _ExploreTab();

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: const Text('Explore'),
        elevation: 0,
      ),
      body: const Center(
        child: Text('Explore Tab - Coming Soon'),
      ),
    );
  }
}

class _NotificationsTab extends StatelessWidget {
  const _NotificationsTab();

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: const Text('Notifications'),
        elevation: 0,
      ),
      body: const Center(
        child: Text('Notifications Tab - Coming Soon'),
      ),
    );
  }
}

class _ProfileTab extends StatelessWidget {
  const _ProfileTab();

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: const Text('Profile'),
        elevation: 0,
      ),
      body: const ProfileEditForm(),
    );
  }
}

class ProfileEditForm extends StatefulWidget {
  const ProfileEditForm({Key? key}) : super(key: key);

  @override
  State<ProfileEditForm> createState() => _ProfileEditFormState();
}

class _ProfileEditFormState extends State<ProfileEditForm> {
  final _formKey = GlobalKey<FormState>();
  final _firstNameController = TextEditingController();
  final _firstNameKatakanaController = TextEditingController();
  final _lastNameController = TextEditingController();
  final _lastNameKatakanaController = TextEditingController();
  final _emailController = TextEditingController();
  
  // Avatar fields
  final _avatarNameController = TextEditingController();
  final _iconUrlController = TextEditingController();
  final _bioController = TextEditingController();
  final _linkController = TextEditingController();
  
  String _selectedRole = 'user';
  bool _isLoading = false;

  final List<String> _roles = ['user', 'admin', 'moderator', 'premium'];

  @override
  void initState() {
    super.initState();
    _loadUserData();
  }

  void _loadUserData() async {
    final userProvider = context.read<UserProvider>();
    await userProvider.loadCurrentUser();
    
    final userData = userProvider.currentUser;
    if (userData != null) {
      setState(() {
        // User data
        _firstNameController.text = userData['firstName'] ?? '';
        _firstNameKatakanaController.text = userData['firstNameKatakana'] ?? '';
        _lastNameController.text = userData['lastName'] ?? '';
        _lastNameKatakanaController.text = userData['lastNameKatakana'] ?? '';
        _emailController.text = userData['emailAddress'] ?? '';
        _selectedRole = userData['role'] ?? 'user';
        
        // Avatar data
        _avatarNameController.text = userData['avatarName'] ?? '';
        _iconUrlController.text = userData['iconUrl'] ?? '';
        _bioController.text = userData['bio'] ?? '';
        _linkController.text = userData['link'] ?? '';
      });
    }
  }

  Future<void> _saveProfile() async {
    if (!_formKey.currentState!.validate()) {
      return;
    }

    setState(() {
      _isLoading = true;
    });

    try {
      final userProvider = context.read<UserProvider>();
      await userProvider.updateProfile({
        // User data
        'firstName': _firstNameController.text,
        'firstNameKatakana': _firstNameKatakanaController.text,
        'lastName': _lastNameController.text,
        'lastNameKatakana': _lastNameKatakanaController.text,
        'emailAddress': _emailController.text,
        'role': _selectedRole,
        
        // Avatar data
        'avatarName': _avatarNameController.text,
        'iconUrl': _iconUrlController.text,
        'bio': _bioController.text,
        'link': _linkController.text,
      });

      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          const SnackBar(
            content: Text('プロフィールを保存しました'),
            backgroundColor: Colors.green,
          ),
        );
      }
    } catch (e) {
      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(
            content: Text('保存に失敗しました: $e'),
            backgroundColor: Colors.red,
          ),
        );
      }
    } finally {
      setState(() {
        _isLoading = false;
      });
    }
  }

  @override
  Widget build(BuildContext context) {
    return Consumer<UserProvider>(
      builder: (context, userProvider, child) {
        if (userProvider.isLoading && userProvider.currentUser == null) {
          return const Center(child: CircularProgressIndicator());
        }

        return SingleChildScrollView(
          padding: const EdgeInsets.all(16.0),
          child: Form(
            key: _formKey,
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                // Profile Header with Avatar
                Card(
                  child: Padding(
                    padding: const EdgeInsets.all(16.0),
                    child: Column(
                      children: [
                        Row(
                          children: [
                            CircleAvatar(
                              radius: 40,
                              backgroundImage: _iconUrlController.text.isNotEmpty 
                                  ? NetworkImage(_iconUrlController.text)
                                  : null,
                              child: _iconUrlController.text.isEmpty
                                  ? Text(
                                      '${_firstNameController.text.isNotEmpty ? _firstNameController.text[0] : 'U'}${_lastNameController.text.isNotEmpty ? _lastNameController.text[0] : ''}',
                                      style: const TextStyle(fontSize: 24),
                                    )
                                  : null,
                            ),
                            const SizedBox(width: 16),
                            Expanded(
                              child: Column(
                                crossAxisAlignment: CrossAxisAlignment.start,
                                children: [
                                  Text(
                                    _avatarNameController.text.isNotEmpty
                                        ? _avatarNameController.text
                                        : '${_lastNameController.text} ${_firstNameController.text}',
                                    style: const TextStyle(fontSize: 18, fontWeight: FontWeight.bold),
                                  ),
                                  Text(
                                    _emailController.text,
                                    style: const TextStyle(color: Colors.grey),
                                  ),
                                  if (_bioController.text.isNotEmpty) ...[
                                    const SizedBox(height: 4),
                                    Text(
                                      _bioController.text,
                                      style: const TextStyle(fontSize: 14),
                                      maxLines: 2,
                                      overflow: TextOverflow.ellipsis,
                                    ),
                                  ],
                                ],
                              ),
                            ),
                          ],
                        ),
                        if (_linkController.text.isNotEmpty) ...[
                          const SizedBox(height: 12),
                          Row(
                            children: [
                              const Icon(Icons.link, size: 16, color: Colors.blue),
                              const SizedBox(width: 4),
                              Expanded(
                                child: Text(
                                  _linkController.text,
                                  style: const TextStyle(color: Colors.blue),
                                  overflow: TextOverflow.ellipsis,
                                ),
                              ),
                            ],
                          ),
                        ],
                      ],
                    ),
                  ),
                ),
                
                const SizedBox(height: 20),
                
                // Basic Profile Information
                const Text(
                  'Profile Information',
                  style: TextStyle(fontSize: 18, fontWeight: FontWeight.bold),
                ),
                const SizedBox(height: 16),
                
                Row(
                  children: [
                    Expanded(
                      child: TextFormField(
                        controller: _lastNameController,
                        decoration: const InputDecoration(
                          labelText: 'Last Name',
                          border: OutlineInputBorder(),
                        ),
                        validator: (value) {
                          if (value == null || value.isEmpty) {
                            return 'Please enter last name';
                          }
                          return null;
                        },
                        onChanged: (value) => setState(() {}),
                      ),
                    ),
                    const SizedBox(width: 16),
                    Expanded(
                      child: TextFormField(
                        controller: _firstNameController,
                        decoration: const InputDecoration(
                          labelText: 'First Name',
                          border: OutlineInputBorder(),
                        ),
                        validator: (value) {
                          if (value == null || value.isEmpty) {
                            return 'Please enter first name';
                          }
                          return null;
                        },
                        onChanged: (value) => setState(() {}),
                      ),
                    ),
                  ],
                ),
                
                const SizedBox(height: 16),
                
                Row(
                  children: [
                    Expanded(
                      child: TextFormField(
                        controller: _lastNameKatakanaController,
                        decoration: const InputDecoration(
                          labelText: '苗字（カナ）',
                          border: OutlineInputBorder(),
                        ),
                        validator: (value) {
                          if (value == null || value.isEmpty) {
                            return '苗字（カナ）を入力してください';
                          }
                          return null;
                        },
                      ),
                    ),
                    const SizedBox(width: 16),
                    Expanded(
                      child: TextFormField(
                        controller: _firstNameKatakanaController,
                        decoration: const InputDecoration(
                          labelText: '名前（カナ）',
                          border: OutlineInputBorder(),
                        ),
                        validator: (value) {
                          if (value == null || value.isEmpty) {
                            return '名前（カナ）を入力してください';
                          }
                          return null;
                        },
                      ),
                    ),
                  ],
                ),
                
                const SizedBox(height: 16),
                
                TextFormField(
                  controller: _emailController,
                  decoration: const InputDecoration(
                    labelText: 'メールアドレス',
                    border: OutlineInputBorder(),
                  ),
                  keyboardType: TextInputType.emailAddress,
                  validator: (value) {
                    if (value == null || value.isEmpty) {
                      return 'メールアドレスを入力してください';
                    }
                    if (!RegExp(r'^[\w-\.]+@([\w-]+\.)+[\w-]{2,4}$').hasMatch(value)) {
                      return '正しいメールアドレスを入力してください';
                    }
                    return null;
                  },
                  onChanged: (value) => setState(() {}),
                ),
                
                const SizedBox(height: 16),
                
                DropdownButtonFormField<String>(
                  value: _selectedRole,
                  decoration: const InputDecoration(
                    labelText: '権限',
                    border: OutlineInputBorder(),
                  ),
                  items: _roles.map((role) {
                    return DropdownMenuItem(
                      value: role,
                      child: Text(_getRoleDisplayName(role)),
                    );
                  }).toList(),
                  onChanged: (value) {
                    setState(() {
                      _selectedRole = value!;
                    });
                  },
                ),
                
                const SizedBox(height: 32),
                
                // Avatar Information
                const Text(
                  'Avatar Information',
                  style: TextStyle(fontSize: 18, fontWeight: FontWeight.bold),
                ),
                const SizedBox(height: 16),
                
                TextFormField(
                  controller: _avatarNameController,
                  decoration: const InputDecoration(
                    labelText: 'アバター名',
                    border: OutlineInputBorder(),
                    hintText: '表示名を入力してください',
                  ),
                  maxLength: 30,
                  validator: (value) {
                    if (value != null && value.length > 30) {
                      return 'アバター名は30文字以内で入力してください';
                    }
                    return null;
                  },
                  onChanged: (value) => setState(() {}),
                ),
                
                const SizedBox(height: 16),
                
                TextFormField(
                  controller: _iconUrlController,
                  decoration: const InputDecoration(
                    labelText: 'アイコンURL',
                    border: OutlineInputBorder(),
                    hintText: 'https://example.com/avatar.jpg',
                  ),
                  keyboardType: TextInputType.url,
                  validator: (value) {
                    if (value != null && value.isNotEmpty) {
                      if (!Uri.tryParse(value)?.hasAbsolutePath == true) {
                        return '正しいURLを入力してください';
                      }
                    }
                    return null;
                  },
                  onChanged: (value) => setState(() {}),
                ),
                
                const SizedBox(height: 16),
                
                TextFormField(
                  controller: _bioController,
                  decoration: const InputDecoration(
                    labelText: '自己紹介文',
                    border: OutlineInputBorder(),
                    hintText: 'あなたについて教えてください',
                  ),
                  maxLines: 3,
                  maxLength: 100,
                  validator: (value) {
                    if (value != null && value.length > 100) {
                      return '自己紹介文は100文字以内で入力してください';
                    }
                    return null;
                  },
                ),
                
                const SizedBox(height: 16),
                
                TextFormField(
                  controller: _linkController,
                  decoration: const InputDecoration(
                    labelText: '外部リンク',
                    border: OutlineInputBorder(),
                    hintText: 'https://your-website.com',
                    prefixIcon: Icon(Icons.link),
                  ),
                  keyboardType: TextInputType.url,
                  validator: (value) {
                    if (value != null && value.isNotEmpty) {
                      if (!Uri.tryParse(value)?.hasAbsolutePath == true) {
                        return '正しいURLを入力してください';
                      }
                    }
                    return null;
                  },
                ),
                
                const SizedBox(height: 32),
                
                SizedBox(
                  width: double.infinity,
                  child: ElevatedButton(
                    onPressed: _isLoading ? null : _saveProfile,
                    style: ElevatedButton.styleFrom(
                      padding: const EdgeInsets.symmetric(vertical: 16),
                    ),
                    child: _isLoading
                        ? const CircularProgressIndicator()
                        : const Text('Save Profile'),
                  ),
                ),
              ],
            ),
          ),
        );
      },
    );
  }

  String _getRoleDisplayName(String role) {
    switch (role) {
      case 'user':
        return '一般ユーザー';
      case 'admin':
        return '管理者';
      case 'moderator':
        return 'モデレーター';
      case 'premium':
        return 'プレミアムユーザー';
      default:
        return role;
    }
  }

  @override
  void dispose() {
    _firstNameController.dispose();
    _firstNameKatakanaController.dispose();
    _lastNameController.dispose();
    _lastNameKatakanaController.dispose();
    _emailController.dispose();
    _avatarNameController.dispose();
    _iconUrlController.dispose();
    _bioController.dispose();
    _linkController.dispose();
    super.dispose();
  }
}