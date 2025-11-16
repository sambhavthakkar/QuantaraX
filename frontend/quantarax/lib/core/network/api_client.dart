import 'dart:convert';
import 'package:http/http.dart' as http;
import 'config.dart';

class ApiClient {
  final http.Client _client;
  ApiClient({http.Client? client}) : _client = client ?? http.Client();

  Map<String, String> _headers({bool jsonBody = true}) {
    final h = <String, String>{};
    if (jsonBody) h['Content-Type'] = 'application/json';
    final token = ApiConfig.authToken;
    if (token != null && token.isNotEmpty) {
      h['X-Auth-Token'] = token;
    }
    return h;
  }

  Future<Map<String, dynamic>> createTransfer(String filePath, String recipientId, {int? chunkSize, Map<String, String>? metadata}) async {
    final uri = Uri.parse('${ApiConfig.baseUrl}/api/v1/transfer/create');
    final body = jsonEncode({
      'file_path': filePath,
      'recipient_id': recipientId,
      if (chunkSize != null) 'chunk_size_override': chunkSize,
      if (metadata != null) 'metadata': metadata,
    });
    final res = await _client.post(uri, headers: _headers(), body: body);
    return _decode(res);
  }

  Future<Map<String, dynamic>> acceptTransfer(String token, String outputPath, {String? resumeSessionId}) async {
    final uri = Uri.parse('${ApiConfig.baseUrl}/api/v1/transfer/accept');
    final body = jsonEncode({
      'transfer_token': token,
      'output_path': outputPath,
      if (resumeSessionId != null) 'resume_session_id': resumeSessionId,
    });
    final res = await _client.post(uri, headers: _headers(), body: body);
    return _decode(res);
  }

  Future<Map<String, dynamic>> getStatus(String sessionId) async {
    final uri = Uri.parse('${ApiConfig.baseUrl}/api/v1/transfer/$sessionId/status');
    final res = await _client.get(uri, headers: _headers(jsonBody: false));
    return _decode(res);
  }

  Future<Map<String, dynamic>> getKeys() async {
    final uri = Uri.parse('${ApiConfig.baseUrl}/api/v1/keys');
    final res = await _client.get(uri, headers: _headers(jsonBody: false));
    return _decode(res);
  }

  Future<Map<String, dynamic>> listTransfers({String? state, int? limit, int? offset}) async {
    final q = <String, String>{};
    if (state != null) q['state'] = state;
    if (limit != null) q['limit'] = '$limit';
    if (offset != null) q['offset'] = '$offset';
    final uri = Uri.parse('${ApiConfig.baseUrl}/api/v1/transfers').replace(queryParameters: q.isEmpty ? null : q);
    final res = await _client.get(uri, headers: _headers(jsonBody: false));
    return _decode(res);
  }

  Future<void> pauseTransfer(String sessionId) async {
    final uri = Uri.parse('${ApiConfig.baseUrl}/api/v1/transfer/$sessionId/pause');
    final res = await _client.post(uri, headers: _headers());
    _decode(res);
  }

  Future<void> cancelTransfer(String sessionId) async {
    final uri = Uri.parse('${ApiConfig.baseUrl}/api/v1/transfer/$sessionId/cancel');
    final res = await _client.post(uri, headers: _headers());
    _decode(res);
  }

  Map<String, dynamic> _decode(http.Response res) {
    final body = res.body.isNotEmpty ? jsonDecode(res.body) : {};
    if (res.statusCode >= 200 && res.statusCode < 300) return body as Map<String, dynamic>;
    // Normalize error model
    if (body is Map && body.containsKey('code')) {
      throw ApiException(body['code']?.toString() ?? 'ERROR', body['message']?.toString() ?? 'Request failed');
    }
    throw ApiException('HTTP_${res.statusCode}', 'Request failed');
  }
}

class ApiException implements Exception {
  final String code;
  final String message;
  ApiException(this.code, this.message);
  @override
  String toString() => 'ApiException($code): $message';
}
