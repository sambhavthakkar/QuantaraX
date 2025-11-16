import 'package:equatable/equatable.dart';

class ChatModel extends Equatable {
  final String id;
  final String title;
  final String subtitle;
  final String timestamp;
  final String status;
  final double progress;
  final bool isActive;
  final String? avatarUrl;
  final String? lastMessage;
  final DateTime lastActivity;

  const ChatModel({
    required this.id,
    required this.title,
    required this.subtitle,
    required this.timestamp,
    required this.status,
    required this.progress,
    this.isActive = false,
    this.avatarUrl,
    this.lastMessage,
    required this.lastActivity,
  });

  factory ChatModel.fromJson(Map<String, dynamic> json) {
    return ChatModel(
      id: json['id'] as String,
      title: json['title'] as String,
      subtitle: json['subtitle'] as String,
      timestamp: json['timestamp'] as String,
      status: json['status'] as String,
      progress: (json['progress'] as num).toDouble(),
      isActive: json['isActive'] as bool? ?? false,
      avatarUrl: json['avatarUrl'] as String?,
      lastMessage: json['lastMessage'] as String?,
      lastActivity: DateTime.parse(json['lastActivity'] as String),
    );
  }

  Map<String, dynamic> toJson() {
    return {
      'id': id,
      'title': title,
      'subtitle': subtitle,
      'timestamp': timestamp,
      'status': status,
      'progress': progress,
      'isActive': isActive,
      'avatarUrl': avatarUrl,
      'lastMessage': lastMessage,
      'lastActivity': lastActivity.toIso8601String(),
    };
  }

  ChatModel copyWith({
    String? id,
    String? title,
    String? subtitle,
    String? timestamp,
    String? status,
    double? progress,
    bool? isActive,
    String? avatarUrl,
    String? lastMessage,
    DateTime? lastActivity,
  }) {
    return ChatModel(
      id: id ?? this.id,
      title: title ?? this.title,
      subtitle: subtitle ?? this.subtitle,
      timestamp: timestamp ?? this.timestamp,
      status: status ?? this.status,
      progress: progress ?? this.progress,
      isActive: isActive ?? this.isActive,
      avatarUrl: avatarUrl ?? this.avatarUrl,
      lastMessage: lastMessage ?? this.lastMessage,
      lastActivity: lastActivity ?? this.lastActivity,
    );
  }

  @override
  List<Object?> get props => [
        id,
        title,
        subtitle,
        timestamp,
        status,
        progress,
        isActive,
        avatarUrl,
        lastMessage,
        lastActivity,
      ];
}