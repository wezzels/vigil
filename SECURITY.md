# Security Policy

## Supported Versions

| Version | Supported          |
| ------- | ------------------ |
| 0.0.x   | :white_check_mark: |

## Reporting a Vulnerability

Please report security vulnerabilities to security@wezzel.com.

Do NOT create a public GitHub issue for security vulnerabilities.

### What to Include

When reporting a vulnerability, please include:

1. **Description** of the vulnerability
2. **Steps to reproduce** the issue
3. **Impact** assessment
4. **Suggested fix** (if available)

### Response Timeline

- **Initial Response**: Within 24 hours
- **Triage**: Within 72 hours
- **Fix Development**: Depends on severity
- **Disclosure**: After fix is released

## Security Measures

### Code Scanning

We use automated security scanning:

- **gosec**: Static analysis for Go code
- **govulncheck**: Go vulnerability checker
- **Dependency Scanning**: Automated CVE scanning

### Best Practices

1. **Input Validation**: All external inputs are validated
2. **Error Handling**: Errors are handled explicitly
3. **No Hardcoded Secrets**: Secrets are loaded from environment
4. **Minimal Permissions**: Services run with minimal privileges
5. **Audit Logging**: Security-relevant events are logged

### Reporting Non-Security Bugs

For non-security bugs, please use [GitHub Issues](https://github.com/wezzels/vigil/issues).

## Security Configuration

### Environment Variables

Sensitive configuration is loaded from environment variables:

```bash
# Kafka
KAFKA_BROKERS=kafka:9092
KAFKA_SASL_USERNAME=xxx
KAFKA_SASL_PASSWORD=xxx

# PostgreSQL
DATABASE_URL=postgresql://xxx
DATABASE_PASSWORD=xxx

# Redis
REDIS_URL=redis://xxx
REDIS_PASSWORD=xxx
```

### Network Security

- All inter-service communication uses TLS
- Services authenticate via mTLS where supported
- External endpoints require authentication

## Dependency Management

We regularly update dependencies to address security vulnerabilities:

```bash
# Check for vulnerabilities
make security

# Update dependencies
go get -u ./...
go mod tidy
```