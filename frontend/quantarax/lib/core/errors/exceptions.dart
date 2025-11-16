// Base exception class
abstract class AppException implements Exception {
  final String message;
  final String? code;
  final dynamic details;

  const AppException(this.message, {this.code, this.details});

  @override
  String toString() => 'AppException: $message';
}

// Network related exceptions
class NetworkException extends AppException {
  const NetworkException(super.message, {super.code, super.details});
}

class ConnectionTimeoutException extends NetworkException {
  const ConnectionTimeoutException([String message = 'Connection timeout'])
      : super(message, code: 'CONNECTION_TIMEOUT');
}

class NoInternetException extends NetworkException {
  const NoInternetException([String message = 'No internet connection'])
      : super(message, code: 'NO_INTERNET');
}

// File related exceptions
class FileException extends AppException {
  const FileException(super.message, {super.code, super.details});
}

class FilePickerException extends FileException {
  const FilePickerException(super.message, {super.code, super.details});
}

class FileValidationException extends FileException {
  const FileValidationException(super.message, {super.code, super.details});
}

class FileNotFound extends FileException {
  const FileNotFound([String message = 'File not found'])
      : super(message, code: 'FILE_NOT_FOUND');
}

// Transfer related exceptions
class TransferException extends AppException {
  const TransferException(super.message, {super.code, super.details});
}

class TransferInitializationException extends TransferException {
  const TransferInitializationException(super.message, {super.code, super.details});
}

class TransferInterruptedException extends TransferException {
  const TransferInterruptedException([String message = 'Transfer was interrupted'])
      : super(message, code: 'TRANSFER_INTERRUPTED');
}

// Authentication related exceptions
class AuthException extends AppException {
  const AuthException(super.message, {super.code, super.details});
}

class UnauthorizedException extends AuthException {
  const UnauthorizedException([String message = 'Unauthorized access'])
      : super(message, code: 'UNAUTHORIZED');
}

class TokenExpiredException extends AuthException {
  const TokenExpiredException([String message = 'Token has expired'])
      : super(message, code: 'TOKEN_EXPIRED');
}

// Server related exceptions
class ServerException extends AppException {
  const ServerException(super.message, {super.code, super.details});
}

class InternalServerException extends ServerException {
  const InternalServerException([String message = 'Internal server error'])
      : super(message, code: 'INTERNAL_SERVER_ERROR');
}

class BadRequestException extends ServerException {
  const BadRequestException([String message = 'Bad request'])
      : super(message, code: 'BAD_REQUEST');
}

// Cache related exceptions
class CacheException extends AppException {
  const CacheException(super.message, {super.code, super.details});
}

// Validation exceptions
class ValidationException extends AppException {
  const ValidationException(super.message, {super.code, super.details});
}

// Platform specific exceptions
class PlatformException extends AppException {
  const PlatformException(super.message, {super.code, super.details});
}