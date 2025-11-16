import 'package:flutter/material.dart';
import 'package:flutter_svg/flutter_svg.dart';
import 'package:file_picker/file_picker.dart';
import 'package:flutter/services.dart';
import 'package:qr_flutter/qr_flutter.dart';
import '../../../core/network/api_client.dart';
import 'package:gradient_borders/box_borders/gradient_box_border.dart';
import 'package:simple_gradient_text/simple_gradient_text.dart';
import '../../../config/app_theme.dart';
import '../../../core/services/transfer_store.dart';

class Sidebar extends StatefulWidget {
  final bool isMobile;
  final Function(String)? onChatTap;
  const Sidebar({super.key, this.isMobile = false, this.onChatTap});

  @override
  State<Sidebar> createState() => _SidebarState();
}

class _SidebarState extends State<Sidebar> {
  final _api = ApiClient();

  Future<void> _handleGenerateToken() async {
    try {
      final result = await FilePicker.platform.pickFiles(allowMultiple: false);
      if (result == null || result.files.isEmpty) return;
      final file = result.files.first;
      final res = await _api.createTransfer(file.path!, 'peer-local');
      try {
        final sessionId = res['session_id']?.toString();
        final manifest = res['manifest'] as Map?;
        if (sessionId != null) {
          final fileName = manifest?['file_name']?.toString();
          final fileSize = (manifest?['file_size'] as num?)?.toInt();
          final totalChunks = (manifest?['total_chunks'] as num?)?.toInt();
          TransferStore.instance.track(sessionId, fileName: fileName, fileSize: fileSize, totalChunks: totalChunks);
          // Track in TransferStore for Monitoring/Chat
          // ignore: avoid_print
          // print('Tracking session $sessionId');
          // Import locally to avoid global refactor
          // ignore: invalid_use_of_internal_member
          // ignore: library_prefixes
          // We'll import at top-level properly below
        }
      } catch (_) {}
      if (!mounted) return;
      await showDialog(context: context, builder: (_) => AlertDialog(
        title: const Text('Transfer Token'),
        content: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            QrImageView(data: (res['qr_code_data'] ?? res['transfer_token']).toString(), size: 200),
            const SizedBox(height: 12),
            SelectableText((res['transfer_token']).toString()),
          ],
        ),
        actions: [
          TextButton(onPressed: ()=>Navigator.pop(context), child: const Text('Close')),
          TextButton(onPressed: (){ Clipboard.setData(ClipboardData(text: (res['transfer_token']).toString())); Navigator.pop(context); }, child: const Text('Copy')),
        ],
      ));
      ScaffoldMessenger.of(context).showSnackBar(const SnackBar(content: Text('Token generated')));
    } catch (e) {
      if (!mounted) return;
      ScaffoldMessenger.of(context).showSnackBar(SnackBar(content: Text('Generate failed: $e')));
    }
  }

  Future<void> _handleAcceptToken() async {
    try {
      String token = (await Clipboard.getData('text/plain'))?.text ?? '';
      if (token.isEmpty) {
        token = await _prompt(context, title: 'Enter Transfer Token');
        if (token.isEmpty) return;
      }
      final dirPath = await FilePicker.platform.getDirectoryPath();
      if (dirPath == null) return;
      final defaultName = 'received.bin';
      final name = await _prompt(context, title: 'Output filename', initial: defaultName, hintText: 'e.g., $defaultName');
      if (name.isEmpty) return;
      final outPath = '$dirPath/$name';
      final acc = await _api.acceptTransfer(token, outPath);
      try {
        final sessionId = acc['session_id']?.toString();
        final manifest = acc['manifest'] as Map?;
        if (sessionId != null) {
          final fileName = manifest?['file_name']?.toString();
          final fileSize = (manifest?['file_size'] as num?)?.toInt();
          final totalChunks = (manifest?['total_chunks'] as num?)?.toInt();
          TransferStore.instance.track(sessionId, fileName: fileName, fileSize: fileSize, totalChunks: totalChunks);
        }
      } catch (_) {}
      if (!mounted) return;
      ScaffoldMessenger.of(context).showSnackBar(const SnackBar(content: Text('Transfer accepted. Check Monitoring for progress.')));
    } catch (e) {
      if (!mounted) return;
      ScaffoldMessenger.of(context).showSnackBar(SnackBar(content: Text('Accept failed: $e')));
    }
  }

  Future<String> _prompt(BuildContext context, {required String title, String? initial, String? hintText}) async {
    final c = TextEditingController(text: initial ?? '');
    String value = initial ?? '';
    await showDialog(context: context, builder: (_) => AlertDialog(
      title: Text(title),
      content: TextField(controller: c, decoration: InputDecoration(hintText: hintText)),
      actions: [
        TextButton(onPressed: ()=>Navigator.pop(context), child: const Text('Cancel')),
        ElevatedButton(onPressed: (){ value = c.text; Navigator.pop(context); }, child: const Text('OK')),
      ],
    ));
    return value;
  }


  @override
  Widget build(BuildContext context) {
    return Container(
      width: widget.isMobile ? double.infinity : 360,
      color: AppTheme.sidebarBg,
      child: Column(
        children: [
          // Header with logo
          Container(
            padding: const EdgeInsets.all(20),
            child: Row(
              children: [
                SvgPicture.asset('assets/quantarax-logo.svg', height: 28),
                const SizedBox(width: 12),
                Text('QuantaraX', style: AppTheme.titleStyle),
              ],
            ),
          ),

          // Search Bar
          Container(
            margin: const EdgeInsets.symmetric(horizontal: 16),
            child: Container(
              padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 12),
              decoration: BoxDecoration(
                color: AppTheme.surface,
                borderRadius: BorderRadius.circular(12),
                border: Border.all(color: AppTheme.border),
              ),
              child: Row(
                children: [
                  Icon(Icons.search, color: AppTheme.textMuted, size: 20),
                  const SizedBox(width: 12),
                  Expanded(
                    child: TextField(
                      decoration: InputDecoration(
                        isDense: true,
                        hintText: 'Search',
                        hintStyle: AppTheme.bodyStyle.copyWith(
                          color: AppTheme.textMuted,
                        ),
                        border: InputBorder.none,
                      ),
                      style: AppTheme.bodyStyle.copyWith(color: AppTheme.textPrimary),
                      onChanged: (q) {
                        // TODO: wire to filtering logic
                      },
                      onSubmitted: (q) {
                        // TODO: execute search action
                      },
                    ),
                  ),
                ],
              ),
            ),
          ),

          const SizedBox(height: 16),

          // Chats Section
          Container(
            margin: const EdgeInsets.symmetric(horizontal: 16),
            child: Row(
              children: [
                Text(
                  'Chats',
                  style: AppTheme.smallStyle.copyWith(
                    color: AppTheme.textMuted,
                  ),
                ),
              ],
            ),
          ),

          const SizedBox(height: 16),

          // Chat List
          Expanded(
            child: SingleChildScrollView(
              padding: const EdgeInsets.only(bottom: 16),
              child: Column(
                children: [
                  _buildChatItem(
                    title: 'Project Trackshift',
                    status: '27% Completed',
                    timestamp: '19:27',
                    isActive: true,
                  ),
                  _buildChatItem(
                    title: 'Media Client',
                    status: '100% Completed',
                    timestamp: '00:21',
                    isActive: false,
                  ),
                  // Add more chat items as needed...
                ],
              ),
            ),
          ),

          // Footer with action buttons
          Container(
            padding: const EdgeInsets.all(16),
            child: Column(
              children: [
                Row(
                  children: [
                    Expanded(
                      child: _buildActionButton(
                        label: 'Scan QR',
                        icon: SvgPicture.asset(
                          'assets/icons/scan-qr-icon.svg',
                          colorFilter: const ColorFilter.mode(
                            Colors.white,
                            BlendMode.srcIn,
                          ),
                        ),
                        variant: 'gradient',
                        onPressed: () {
                          _handleAcceptToken();
                        },
                      ),
                    ),
                    const SizedBox(width: 8),
                    Expanded(
                      child: _buildActionButton(
                        label: 'Generate Token',
                        icon: SvgPicture.asset(
                          'assets/icons/generate-token-icon.svg',
                          colorFilter: const ColorFilter.mode(
                            Colors.white,
                            BlendMode.srcIn,
                          ),
                        ),
                        variant: 'gradient',
                        onPressed: () {
                          _handleGenerateToken();
                        },
                      ),
                    ),
                  ],
                ),
                const SizedBox(height: 16),
                // User Profile
                Container(
                  padding: const EdgeInsets.all(12),
                  decoration: BoxDecoration(
                    color: AppTheme.surface,
                    borderRadius: BorderRadius.circular(12),
                    border: Border.all(color: AppTheme.border),
                  ),
                  child: Row(
                    children: [
                      CircleAvatar(
                        radius: 16,
                        backgroundColor: AppTheme.primary,
                        child: Text(
                          'ST',
                          style: AppTheme.smallStyle.copyWith(
                            color: Colors.black,
                            fontWeight: FontWeight.w600,
                          ),
                        ),
                      ),
                      const SizedBox(width: 12),
                      Expanded(
                        child: Text(
                          'Sambhav Thakkar',
                          style: AppTheme.bodyStyle,
                        ),
                      ),
                    ],
                  ),
                ),
              ],
            ),
          ),
        ],
      ),
    );
  }

  Widget _buildChatItem({
    required String title,
    required String status,
    required String timestamp,
    required bool isActive,
  }) {
    return Container(
      margin: const EdgeInsets.symmetric(horizontal: 16, vertical: 4),
      child: Material(
        color: Colors.transparent,
        child: InkWell(
          borderRadius: BorderRadius.circular(12),
          onTap: () {
            if (widget.onChatTap != null) {
              widget.onChatTap!(title);
            }
          },
          child: Ink(
            padding: const EdgeInsets.all(16),
            decoration: BoxDecoration(
              color: isActive ? AppTheme.surface : Colors.transparent,
              borderRadius: BorderRadius.circular(12),
              border: isActive
                  ? GradientBoxBorder(gradient: AppTheme.primaryGradient, width: 1)
                  : null,
            ),
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Row(
                  mainAxisAlignment: MainAxisAlignment.spaceBetween,
                  children: [
                    Expanded(child: Text(title, style: AppTheme.bodyStyle)),
                    Text(timestamp, style: AppTheme.smallStyle),
                  ],
                ),
                const SizedBox(height: 4),
                GradientText(
                  status,
                  style: AppTheme.bodyStyle,
                  colors: status.contains('100%')
                      ? AppTheme.primaryGradient.colors
                      : [
                          AppTheme.textSecondary,
                          AppTheme.textSecondary,
                        ], // Provide two identical colors
                ),
              ],
            ),
          ),
        ),
      ),
    );
  }

  Widget _buildActionButton({
    required String label,
    required Widget icon,
    required String variant,
    VoidCallback? onPressed,
  }) {
    return Material(
      color: Colors.transparent,
      child: InkWell(
        borderRadius: BorderRadius.circular(42),
        onTap: onPressed,
        child: Ink(
          padding: const EdgeInsets.symmetric(vertical: 12, horizontal: 16),
          decoration: BoxDecoration(
            gradient: variant == 'gradient' ? AppTheme.primaryGradient : null,
            color: variant == 'glow' ? AppTheme.primary.withOpacity(0.1) : null,
            borderRadius: BorderRadius.circular(42),
            boxShadow: variant == 'glow'
                ? [AppTheme.blueGlow.copyWith(blurRadius: 8)]
                : null,
          ),
          child: Row(
            mainAxisSize: MainAxisSize.min,
            children: [
              SizedBox(width: 18, height: 18, child: icon),
              const SizedBox(width: 8),
              Flexible(
                child: Text(
                  label,
                  style: AppTheme.smallStyle.copyWith(
                    color: Colors.white,
                    fontWeight: FontWeight.w500,
                  ),
                ),
              ),
            ],
          ),
        ),
      ),
    );
  }
}
