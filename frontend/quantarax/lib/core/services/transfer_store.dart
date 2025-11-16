import 'dart:async';
import 'dart:math';
import '../network/api_client.dart';
import '../network/sse_service.dart';

class TransferInfo {
  final String sessionId;
  String? fileName;
  int? fileSize;
  int? totalChunks;
  int? chunksTransferred;
  double progressPercent = 0;
  double rateMbps = 0;
  int etaSeconds = 0;
  double? rttMs;
  int? streams;
  double? lossRatePct;
  final List<double> speedHistoryMbPerSec = <double>[]; // MB/s for chart
  DateTime lastUpdate = DateTime.now();

  TransferInfo(this.sessionId);
}

class TransferStore {
  static final TransferStore instance = TransferStore._internal();
  TransferStore._internal();

  final _api = ApiClient();
  final Map<String, TransferInfo> _items = {};
  final Map<String, StreamSubscription> _sseSubs = {};
  final Map<String, Timer> _pollTimers = {};
  final _controller = StreamController<List<TransferInfo>>.broadcast();

  Stream<List<TransferInfo>> observe() => _controller.stream;

  List<TransferInfo> get snapshot => _items.values.toList(growable: false);

  void _emit() {
    _controller.add(snapshot);
  }

  void track(String sessionId, {String? fileName, int? fileSize, int? totalChunks}) {
    final info = _items.putIfAbsent(sessionId, () => TransferInfo(sessionId));
    info.fileName ??= fileName;
    info.fileSize ??= fileSize;
    info.totalChunks ??= totalChunks;
    _emit();
    _start(sessionId);
  }

  void _start(String sessionId) {
    _sseSubs[sessionId]?.cancel();
    _pollTimers[sessionId]?.cancel();

    final sse = SseService();
    _sseSubs[sessionId] = sse.subscribe(sessionId: sessionId).listen((ev) {
      final info = _items[sessionId];
      if (info == null) return;
      info.progressPercent = (ev['progress_percent'] as num?)?.toDouble() ?? info.progressPercent;
      final md = (ev['metadata'] as Map?) ?? {};
      info.rateMbps = double.tryParse((md['transfer_rate_mbps'] ?? info.rateMbps).toString()) ?? info.rateMbps;
      info.etaSeconds = int.tryParse((md['eta_seconds'] ?? info.etaSeconds).toString()) ?? info.etaSeconds;
      info.chunksTransferred = int.tryParse((md['chunk_index'] ?? info.chunksTransferred).toString()) ?? info.chunksTransferred;
      info.totalChunks = int.tryParse((md['total_chunks'] ?? info.totalChunks).toString()) ?? info.totalChunks;
      info.rttMs = double.tryParse((md['rtt_ms'] ?? info.rttMs ?? '').toString()) ?? info.rttMs;
      info.streams = int.tryParse((md['streams'] ?? info.streams ?? '').toString()) ?? info.streams;
      info.lossRatePct = double.tryParse((md['loss_rate_pct'] ?? info.lossRatePct ?? '').toString()) ?? info.lossRatePct;
      // Keep MB/s chart history (convert Mbps to MB/s)
      final mbps = info.rateMbps;
      final mBps = mbps / 8.0;
      info.speedHistoryMbPerSec.add(mBps);
      if (info.speedHistoryMbPerSec.length > 30) info.speedHistoryMbPerSec.removeAt(0);
      info.lastUpdate = DateTime.now();
      _emit();
    });

    _pollTimers[sessionId] = Timer.periodic(const Duration(seconds: 2), (_) async {
      try {
        final st = await _api.getStatus(sessionId);
        final info = _items[sessionId] ?? TransferInfo(sessionId);
        _items[sessionId] = info;
        info.progressPercent = (st['progress_percent'] as num?)?.toDouble() ?? info.progressPercent;
        info.rateMbps = (st['transfer_rate_mbps'] as num?)?.toDouble() ?? info.rateMbps;
        info.etaSeconds = (st['estimated_time_remaining'] as num?)?.toInt() ?? info.etaSeconds;
        info.chunksTransferred = (st['chunks_transferred'] as num?)?.toInt() ?? info.chunksTransferred;
        info.totalChunks = (st['total_chunks'] as num?)?.toInt() ?? info.totalChunks;
        // Optional diagnostics if status includes them in future
        info.rttMs = (st['rtt_ms'] as num?)?.toDouble() ?? info.rttMs;
        info.streams = (st['streams'] as num?)?.toInt() ?? info.streams;
        info.lossRatePct = (st['loss_rate_pct'] as num?)?.toDouble() ?? info.lossRatePct;
        // MB/s history
        final mBps = (info.rateMbps) / 8.0;
        info.speedHistoryMbPerSec.add(mBps);
        if (info.speedHistoryMbPerSec.length > 30) info.speedHistoryMbPerSec.removeAt(0);
        info.lastUpdate = DateTime.now();
        _emit();
      } catch (_) {}
    });
  }

  void dispose() {
    for (final s in _sseSubs.values) { s.cancel(); }
    for (final t in _pollTimers.values) { t.cancel(); }
    _controller.close();
  }
}
