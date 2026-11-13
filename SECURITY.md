# Security policy

## Supported versions

justz is in early development. Only the latest release gets security fixes.

## Reporting a vulnerability

Please don't open a public issue for a security problem.

Use GitHub's private reporting instead: go to the [Security tab](https://github.com/igorlino/justz/security) and choose "Report a vulnerability". That sends the report to the maintainer only.

A good report says what you found, how to reproduce it, and which version you tried. I'll acknowledge it within a few days and keep you posted while it's fixed. You're welcome to be credited in the release notes if you'd like.

## What counts

Things we'd want to hear about:

- Running a command or writing a file the user didn't ask for.
- Writing outside the paths justz documents, or following a symlink out of a repository.
- Leaking secrets, for example through logs, process arguments or generated files.
- Tampered release artifacts, install scripts or checksums.

justz never reads, stores or injects credentials, so a bug that makes it do so is a vulnerability.

## Verifying releases

Releases include SHA256 checksums and build provenance attestations. Once the first release is out, you can check an artifact with `gh attestation verify <file> --repo igorlino/justz`.
