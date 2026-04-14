# VIGIL Security Policy

## Reporting a Vulnerability

We take security vulnerabilities seriously. If you discover a security vulnerability in VIGIL, please report it responsibly.

### How to Report

**DO NOT** create a public GitHub issue for security vulnerabilities.

Instead, please report vulnerabilities by:

1. **Email**: Send details to security@vigil.local
2. **GitHub Security**: Use GitHub's private vulnerability reporting feature at https://github.com/wezzels/vigil/security/advisories

### What to Include

Please include the following information:

- **Description** of the vulnerability
- **Steps to reproduce** the issue
- **Potential impact** of the vulnerability
- **Affected versions** (if known)
- **Suggested fix** (if available)
- **Your contact information** for follow-up

## Response SLA

We are committed to responding to security reports promptly:

| Severity | Initial Response | Status Update | Resolution Target |
|----------|------------------|---------------|-------------------|
| Critical | 24 hours | 48 hours | 7 days |
| High | 48 hours | 72 hours | 14 days |
| Medium | 72 hours | 1 week | 30 days |
| Low | 1 week | 2 weeks | 90 days |

### Severity Definitions

#### Critical
- Remote code execution
- SQL injection with data exfiltration
- Authentication bypass
- Privilege escalation to admin

#### High
- Information disclosure of sensitive data
- Cross-site scripting (XSS) with session hijacking
- Denial of service (DoS) with no mitigation
- Insecure deserialization

#### Medium
- Cross-site scripting (XSS) limited scope
- Insecure direct object references
- Information disclosure of non-sensitive data
- Local privilege escalation

#### Low
- Minor information leakage
- DoS with mitigation available
- Security misconfiguration

## Disclosure Policy

### Coordinated Disclosure

We follow a coordinated disclosure process:

1. **Report Received**: Acknowledge receipt within SLA
2. **Triage**: Validate and assess severity
3. **Fix Development**: Develop and test fix
4. **Pre-Notification**: Notify reporter of fix timing
5. **Release**: Publish security advisory and fix
6. **Public Disclosure**: After appropriate delay

### Disclosure Timeline

| Time | Action |
|------|--------|
| Day 0 | Vulnerability reported |
| Day 1 | Initial response, triage begins |
| Day 3 | Severity assessment complete |
| Day 7-14 | Fix developed and tested |
| Day 14-21 | Security advisory prepared |
| Day 21+ | Public disclosure (with reporter approval) |

### Public Disclosure Conditions

We will delay public disclosure for up to 90 days to allow:

- Fix development and testing
- Deployment to production systems
- Coordination with affected third parties

## Security Best Practices

### For Developers

1. **Never commit secrets** to version control
2. **Use parameterized queries** for all database operations
3. **Validate and sanitize** all user inputs
4. **Use TLS 1.2+** for all network communication
5. **Enable authentication** for all endpoints
6. **Log security events** appropriately
7. **Follow least privilege** principle

### For Operators

1. **Keep software updated** with security patches
2. **Monitor audit logs** for suspicious activity
3. **Use strong passwords** and rotate regularly
4. **Enable mTLS** for service-to-service communication
5. **Apply network policies** to restrict traffic
6. **Use secrets management** (Vault, etc.)
7. **Run security scans** regularly

## Supported Versions

| Version | Supported | Security Fixes |
|---------|-----------|----------------|
| 1.x | ✅ Yes | Active |
| 0.x | ✅ Yes | Active |
| < 0.1 | ❌ No | End of life |

## Security Architecture

### Authentication

- **mTLS**: Service-to-service authentication
- **JWT**: User authentication with refresh tokens
- **API Keys**: Programmatic access

### Authorization

- **RBAC**: Role-based access control
- **Roles**: viewer, operator, supervisor, admin
- **Permissions**: Resource-action pairs

### Encryption

- **Transport**: TLS 1.2+ (TLS 1.3 preferred)
- **At Rest**: AES-256-GCM
- **Key Management**: Vault Transit engine

### Audit

- **Events**: All security events logged
- **Retention**: 90 days minimum
- **Integrity**: Append-only with signatures

## Security Contacts

| Role | Contact |
|------|---------|
| Security Team | security@vigil.local |
| Security Lead | lead-security@vigil.local |
| Incident Response | incident-response@vigil.local |

## Credits

We would like to thank the following individuals for responsibly disclosing security vulnerabilities:

- (No disclosures yet)

---

**Last Updated**: 2026-04-14
**Policy Version**: 1.0