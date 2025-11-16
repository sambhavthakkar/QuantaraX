class ApiConfig {
  ApiConfig._();
  static String baseUrl = const String.fromEnvironment('QUANTARAX_API', defaultValue: 'http://127.0.0.1:8080');
  static String? authToken = const String.fromEnvironment('QUANTARAX_AUTH_TOKEN');
}
