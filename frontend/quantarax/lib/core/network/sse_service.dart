import 'dart:async';
import 'dart:convert';
import 'package:http/http.dart' as http;
import 'config.dart';

class SseService {
  http.Client? _client;
  StreamSubscription<http.StreamedResponse>? _responseSubscription;
  StreamController<Map<String, dynamic>>? _controller;
  bool _closing = false;
  String? _lastUrl;
  Map<String, String>? _lastHeaders;

  Stream<Map<String, dynamic>> subscribe({String? sessionId}) {
    _controller?.close();
    _controller = StreamController.broadcast();
    final url = Uri.parse('${ApiConfig.baseUrl}/api/v1/events${sessionId != null ? '?session_id=$sessionId' : ''}').toString();
    final headers = <String, String>{
      if ((ApiConfig.authToken ?? '').isNotEmpty) 'X-Auth-Token': ApiConfig.authToken!,
    };
    _lastUrl = url; _lastHeaders = headers;
    _connect(url, headers, 0);
    return _controller!.stream;
  }

  void _connect(String url, Map<String, String> headers, int attempt) {
    if (_closing) return;
    
    try {
      _client = http.Client();
      final request = http.Request('GET', Uri.parse(url));
      request.headers.addAll(headers);
      
      _responseSubscription = _client!
          .send(request)
          .timeout(const Duration(seconds: 30))
          .asStream()
          .listen(
        (response) {
          if (response.statusCode == 200) {
            _processSseStream(response, attempt);
          } else {
            _scheduleReconnect(attempt + 1);
          }
        },
        onError: (_) {
          _scheduleReconnect(attempt + 1);
        },
        onDone: () {
          _scheduleReconnect(attempt + 1);
        },
      );
    } catch (_) {
      _scheduleReconnect(attempt + 1);
    }
  }
  
  void _processSseStream(http.StreamedResponse response, int attempt) {
    response.stream
        .transform(const Utf8Decoder())
        .transform(const LineSplitter())
        .listen(
      (line) {
        if (_closing) return;
        
        final trimmedLine = line.trim();
        if (trimmedLine.startsWith('data: ')) {
          final data = trimmedLine.substring(6);
          if (data.isNotEmpty) {
            try {
              _controller?.add(_tryParseJson(data));
            } catch (_) {
              // Ignore malformed JSON
            }
          }
        }
      },
      onError: (_) {
        _scheduleReconnect(attempt + 1);
      },
      onDone: () {
        _scheduleReconnect(attempt + 1);
      },
    );
  }

  void _scheduleReconnect(int attempt) {
    if (_closing) return;
    final delay = Duration(milliseconds: 500 * (attempt.clamp(1, 10)));
    Future.delayed(delay, () {
      if (_lastUrl != null && _lastHeaders != null) {
        _connect(_lastUrl!, _lastHeaders!, attempt);
      }
    });
  }

  void close() {
    _closing = true;
    _responseSubscription?.cancel();
    _client?.close();
    _controller?.close();
    _responseSubscription = null;
    _client = null;
    _controller = null;
  }

  Map<String, dynamic> _tryParseJson(String raw) {
    try { return jsonDecode(raw) as Map<String, dynamic>; } catch (_) { return {'raw': raw}; }
  }
}
