# Security Policy

We take the security of `applecontainer-go` seriously. This document outlines our policy for reporting and handling security vulnerabilities.

## Supported Versions

Currently, security updates and patches are provided for the following versions:

| Version | Supported |
| :--- | :--- |
| v0.x / Main branch | :white_check_mark: |
| < v0.1.0 | :x: |

## Reporting a Vulnerability

If you discover a security vulnerability in this project, please **do not open a public issue**. Instead, report it using one of the following methods:

1. **GitHub Private Vulnerability Report**: Please use the ["Report a vulnerability" button](https://github.com/lynicis/applecontainer-go/security/advisories/new) under the Security tab on GitHub.
2. **Email**: Send a detailed email to **me@lynicis.dev**.

### What to Include in a Report
To help us investigate and address the vulnerability quickly, please include:
- A description of the issue and the potential impact.
- Step-by-step instructions or a minimal proof-of-concept (such as a Go test case) to reproduce the behavior.
- Any details about the environment or configurations where it was observed.

## Our Response Process

1. **Acknowledgement**: We will acknowledge receipt of your vulnerability report within 48 hours.
2. **Investigation & Fix**: We will investigate the issue and, if confirmed, work on a fix/patch. We may contact you for further details or to test the proposed fix.
3. **Disclosure**: Once the fix is ready, we will coordinate the release of a security advisory and a patched version. We request that you do not disclose the vulnerability publicly until a patch is released and we have coordinated disclosure.
