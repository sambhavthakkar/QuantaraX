import 'dart:io';
import 'dart:async';
import 'package:file_picker/file_picker.dart';
import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:qr_flutter/qr_flutter.dart';
import '../../../core/network/api_client.dart';
import '../../../core/network/sse_service.dart';

class ApiControls extends StatefulWidget {
  const ApiControls({super.key});

  @override
  State<ApiControls> createState() => _ApiControlsState();
}

class _ApiControlsState extends State<ApiControls> {
  final _api = ApiClient();
  final _sse = SseService();
  String? _sessionId;
  String _status = '';
  Timer? _pollTimer;
  Stream<Map<String, dynamic>>? _events;
  double _progress = 0;
  double _rateMbps = 0;
  int _etaSeconds = 0;
  int? _chunkIndex;
  int? _totalChunks;

  void _startPolling() {
    _stopPolling();
    if (_sessionId == null) return;
    _pollTimer = Timer.periodic(const Duration(seconds: 2), (timer) async {
      try {
        final st = await _api.getStatus(_sessionId!);
        setState(() {
          _progress = (st['progress_percent'] as num?)?.toDouble() ?? _progress;
          _rateMbps = (st['transfer_rate_mbps'] as num?)?.toDouble() ?? _rateMbps;
          _etaSeconds = (st['estimated_time_remaining'] as num?)?.toInt() ?? _etaSeconds;
          _chunkIndex = (st['chunks_transferred'] as num?)?.toInt() ?? _chunkIndex;
          _totalChunks = (st['total_chunks'] as num?)?.toInt() ?? _totalChunks;
        });
      } catch (_) {}
    });
  }

  void _stopPolling() {
    _pollTimer?.cancel();
    _pollTimer = null;
  }

  Future<void> _generateToken() async {
    _stopPolling();
    try {
      // Simple file chooser via native dialog
      final file = await FileSelector.pickFile();
      if (file == null) return;
      final res = await _api.createTransfer(file.path, 'peer-local');
      setState(() {
        _sessionId = res['session_id'] as String?;
        _status = 'Token generated';
        _events = _sse.subscribe(sessionId: _sessionId);
      });
      _startPolling();
      if (!mounted) return;
      showDialog(context: context, builder: (_) => AlertDialog(
        title: const Text('Transfer Token (QR)'),
        content: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            QrImageView(data: (res['qr_code_data'] ?? res['transfer_token']).toString(), size: 200),
            const SizedBox(height: 12),
            SelectableText((res['transfer_token']).toString()),
          ],
        ),
        actions: [TextButton(onPressed: ()=>Navigator.pop(context), child: const Text('Close'))],
      ));
    } catch (e) {
      setState(() { _status = 'Error: $e'; });
    }
  }

  Future<void> _acceptTransfer() async {
    _stopPolling();
    try {
      // Try clipboard first
      String token = (await Clipboard.getData('text/plain'))?.text ?? '';
      if (token.isEmpty) {
        token = await InputDialog.prompt(context, title: 'Enter Transfer Token');
        if (token.isEmpty) return;
      }
      final dir = await DirectorySelector.pickDirectory();
      if (dir == null) return;
      final defaultName = 'received.bin';
      final name = await InputDialog.prompt(context, title: 'Output filename', initial: defaultName, hintText: 'e.g., ${defaultName}');
      if (name.isEmpty) return;
      final outPath = '${dir.path}${Platform.pathSeparator}$name';
      final res = await _api.acceptTransfer(token, outPath);
      setState(() {
        _sessionId = res['session_id'] as String?;
        _status = 'Accepted transfer';
        _events = _sse.subscribe(sessionId: _sessionId);
      });
      _startPolling();
    } catch (e) {
      setState(() { _status = 'Error: $e'; });
    }
  }

  void _pauseTransfer() {
    ScaffoldMessenger.of(context).showSnackBar(
      const SnackBar(content: Text('Pause not supported yet')),
    );
  }

  void _cancelTransfer() {
    ScaffoldMessenger.of(context).showSnackBar(
      const SnackBar(content: Text('Cancel not supported yet')),
    );
  }

  @override
  void dispose() { _sse.close(); _stopPolling(); super.dispose(); }

  @override
  Widget build(BuildContext context) {
    return Card(
      elevation: 2,
      margin: const EdgeInsets.only(bottom: 12),
      child: Padding(
        padding: const EdgeInsets.all(12.0),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Row(children: [
              ElevatedButton.icon(onPressed: _generateToken, icon: const Icon(Icons.qr_code_2), label: const Text('Generate Token')),
              const SizedBox(width: 12),
              ElevatedButton.icon(onPressed: _acceptTransfer, icon: const Icon(Icons.download), label: const Text('Accept Transfer')),
              const Spacer(),
              Tooltip(
                message: 'Pause not available yet',
                child: OutlinedButton.icon(
                  onPressed: (_sessionId != null) ? _pauseTransfer : null,
                  icon: const Icon(Icons.pause),
                  label: const Text('Pause'),
                ),
              ),
              const SizedBox(width: 8),
              Tooltip(
                message: 'Cancel not available yet',
                child: OutlinedButton.icon(
                  onPressed: (_sessionId != null) ? _cancelTransfer : null,
                  icon: const Icon(Icons.cancel),
                  label: const Text('Cancel'),
                ),
              ),
            ]),
            const SizedBox(height: 8),
            if (_sessionId != null) Text('Session: $_sessionId', style: Theme.of(context).textTheme.bodySmall),
            if (_status.isNotEmpty)
              Padding(
                padding: const EdgeInsets.only(top: 4.0),
                child: Text(
                  _status,
                  style: Theme.of(context).textTheme.bodySmall?.copyWith(
                    color: _status.startsWith('Error') ? Theme.of(context).colorScheme.error : Colors.blueGrey,
                  ),
                ),
              ),
            if (_sessionId == null && _status.isEmpty)
              Padding(
                padding: const EdgeInsets.only(top: 4.0),
                child: Text(
                  'Use Generate Token to start a transfer or Accept Transfer to paste a token.',
                  style: Theme.of(context).textTheme.bodySmall?.copyWith(color: Colors.blueGrey),
                ),
              ),
            const SizedBox(height: 8),
            if (_events != null)
              SizedBox(
                height: 160,
                child: StreamBuilder<Map<String, dynamic>>(
                  stream: _events,
                  builder: (context, snap) {
                    if (snap.hasData) {
                      final ev = snap.data!;
                      final pct = (ev['progress_percent'] ?? 0);
                      final md = (ev['metadata'] as Map?) ?? {};
                      _progress = (pct is num) ? pct.toDouble() : _progress;
                      _rateMbps = double.tryParse((md['transfer_rate_mbps'] ?? '0').toString()) ?? _rateMbps;
                      _etaSeconds = int.tryParse((md['eta_seconds'] ?? '0').toString()) ?? _etaSeconds;
                      _chunkIndex = int.tryParse((md['chunk_index'] ?? (_chunkIndex?.toString() ?? '0')).toString()) ?? _chunkIndex;
                      _totalChunks = int.tryParse((md['total_chunks'] ?? (_totalChunks?.toString() ?? '0')).toString()) ?? _totalChunks;
                    }
                    return Column(
                      crossAxisAlignment: CrossAxisAlignment.start,
                      children: [
                        LinearProgressIndicator(value: (_progress > 0) ? (_progress/100.0) : null),
                        const SizedBox(height: 8),
                        Row(children: [
                          Expanded(child: Text('Progress: ${_progress.toStringAsFixed(1)}%')),
                          if (_totalChunks != null)
                            Padding(
                              padding: const EdgeInsets.only(left: 8.0),
                              child: Text('Chunk: ${(_chunkIndex ?? 0)}/${_totalChunks}'),
                            ),
                          const Spacer(),
                          Text('Speed: ${_rateMbps.toStringAsFixed(2)} Mbps'),
                          const SizedBox(width: 12),
                          Text('ETA: ${_etaSeconds}s'),
                        ]),
                      ],
                    );
                  },
                ),
              ),
          ],
        ),
      ),
    );
  }
}

class InputDialog {
  static Future<String> prompt(
    BuildContext context, {
    required String title,
    String? initial,
    String? hintText,
    String okText = 'OK',
  }) async {
    final c = TextEditingController(text: initial ?? '');
    String value = initial ?? '';
    await showDialog(context: context, builder: (_) => AlertDialog(
      title: Text(title),
      content: TextField(controller: c, decoration: InputDecoration(hintText: hintText)),
      actions: [
        TextButton(onPressed: ()=>Navigator.pop(context), child: const Text('Cancel')),
        ElevatedButton(onPressed: (){ value = c.text; Navigator.pop(context); }, child: Text(okText)),
      ],
    ));
    return value;
  }
}

class FileSelector {
  static Future<File?> pickFile() async {
    final res = await FilePicker.platform.pickFiles(allowMultiple: false);
    if (res != null && res.files.isNotEmpty && res.files.first.path != null) {
      return File(res.files.first.path!);
    }
    return null;
  }
}

class DirectorySelector {
  static Future<Directory?> pickDirectory() async {
    final path = await FilePicker.platform.getDirectoryPath();
    if (path == null) return null;
    return Directory(path);
  }
}
