import 'package:flutter/material.dart';
import '../../config/app_theme.dart';
import '../widgets/sidebar/sidebar_widget.dart';
import '../widgets/chat/chat_panel.dart';
import '../widgets/monitoring/monitoring_panel.dart';

// Mobile Screen 1: Chat List (uses Sidebar content)
class MobileChatListScreen extends StatelessWidget {
  final Function(String) onChatTap;
  
  const MobileChatListScreen({super.key, required this.onChatTap});

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      backgroundColor: AppTheme.background,
      body: SafeArea(
        child: Sidebar(
          isMobile: true,
          onChatTap: onChatTap,
        ),
      ),
    );
  }
}

// Mobile Screen 2: Individual Chat (uses ChatPanel content with back button)
class MobileChatScreen extends StatelessWidget {
  final String chatTitle;
  final VoidCallback onMonitorTap;
  
  const MobileChatScreen({
    super.key, 
    required this.chatTitle, 
    required this.onMonitorTap,
  });

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      backgroundColor: AppTheme.background,
      body: SafeArea(
        child: ChatPanel(
          isMobile: true,
          onToggleMonitor: onMonitorTap,
          chatTitle: chatTitle,
        ),
      ),
    );
  }
}

// Mobile Screen 3: Monitor (uses MonitoringPanel content with back button)
class MobileMonitorScreen extends StatelessWidget {
  const MobileMonitorScreen({super.key});

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      backgroundColor: AppTheme.background,
      body: SafeArea(
        child: MonitoringPanel(
          isMobile: true,
          onClose: () => Navigator.pop(context),
        ),
      ),
    );
  }
}