# Production Rollout Checklist - Last9 Terraform Provider v1.0.0

## Pre-Release Validation

### Code Quality
- [x] Run `go fmt ./...` - Code is properly formatted
- [x] Run `go vet ./...` - No issues found
- [x] Run `go build` - Builds successfully
- [x] Run `CGO_ENABLED=0 go test ./...` - All tests pass (macOS requires CGO disabled)

### Security Review
- [x] No hardcoded credentials in source code
- [x] Tokens are not logged in plaintext
- [x] Thread-safe token management with mutex locks
- [x] Proper error wrapping without exposing sensitive data
- [x] Input validation on all user inputs
- [x] Separate delete token scope for destructive operations

### Documentation
- [x] README.md is comprehensive with usage examples
- [x] CHANGELOG.md documents v1.0.0 features
- [x] AUTHENTICATION.md covers all auth methods
- [x] TESTING.md provides test instructions
- [x] Examples provided for all resources

## Resources Implemented

| Resource | Status | Tests | Documentation |
|----------|--------|-------|---------------|
| `last9_dashboard` | :white_check_mark: Complete | :white_check_mark: | :white_check_mark: |
| `last9_alert` | :white_check_mark: Complete | :white_check_mark: | :white_check_mark: |
| `last9_macro` | :white_check_mark: Complete | :white_check_mark: | :white_check_mark: |
| `last9_policy` | :white_check_mark: Complete | :white_check_mark: | :white_check_mark: |
| `last9_drop_rule` | :white_check_mark: Complete | :white_check_mark: | :white_check_mark: |
| `last9_forward_rule` | :white_check_mark: Complete | :white_check_mark: | :white_check_mark: |
| `last9_scheduled_search_alert` | :white_check_mark: Complete | :white_check_mark: | :white_check_mark: |
| `last9_entity` | :white_check_mark: Complete | :white_check_mark: | :white_check_mark: |
| `last9_notification_channel` | :white_check_mark: Complete | :white_check_mark: | :white_check_mark: |

## Data Sources Implemented

| Data Source | Status | Tests | Documentation |
|-------------|--------|-------|---------------|
| `last9_dashboard` | :white_check_mark: Complete | :white_check_mark: | :white_check_mark: |
| `last9_entity` | :white_check_mark: Complete | :white_check_mark: | :white_check_mark: |
| `last9_notification_destination` | :white_check_mark: Complete | :white_check_mark: | :white_check_mark: |

## Release Process

### GitHub Repository Setup
- [ ] Repository is public: `github.com/last9/terraform-provider`
- [ ] GPG key configured for signing releases
- [ ] GitHub Secrets configured:
  - [ ] `GPG_PRIVATE_KEY` - GPG key for signing
  - [ ] `PASSPHRASE` - GPG key passphrase
  - [ ] `LAST9_REFRESH_TOKEN` - For acceptance tests
  - [ ] `LAST9_ORG` - For acceptance tests

### Release Artifacts
- [x] `.goreleaser.yml` configured for multi-platform builds
- [x] GitHub Actions workflow for releases (`.github/workflows/release.yml`)
- [x] GitHub Actions workflow for tests (`.github/workflows/test.yml`)

### Create Release
1. Ensure all changes are committed:
   ```bash
   git add .
   git commit -m "Prepare v1.0.0 release"
   ```

2. Create and push the release tag:
   ```bash
   git tag -a v1.0.0 -m "Release v1.0.0 - First stable release"
   git push origin main
   git push origin v1.0.0
   ```

3. GitHub Actions will automatically:
   - Build binaries for all platforms (linux/darwin/windows, amd64/arm64)
   - Create signed checksums
   - Create GitHub release with artifacts

### Terraform Registry Publication

1. **Connect Repository to Terraform Registry**:
   - Go to https://registry.terraform.io/
   - Sign in with GitHub
   - Click "Publish" → "Provider"
   - Select `last9/terraform-provider` repository
   - Accept terms and publish

2. **Registry Requirements**:
   - [x] Repository name follows pattern: `terraform-provider-{name}` ✓
   - [x] Contains valid Go module
   - [x] Releases are tagged with `vX.Y.Z` format
   - [x] Releases include signed checksums
   - [ ] GPG public key added to Terraform Registry

3. **After Publication**:
   - Provider will be available at: `registry.terraform.io/last9/last9`
   - Users can install with:
     ```hcl
     terraform {
       required_providers {
         last9 = {
           source  = "last9/last9"
           version = "~> 1.0"
         }
       }
     }
     ```

## Post-Release Tasks

### Internal
- [ ] Update docs.last9.io with provider documentation
- [ ] Create internal announcement
- [ ] Update Linear ticket ENG-364 to Done

### Customer Communication
- [ ] Prepare release announcement
- [ ] Update getting started guides
- [ ] Create migration guide from Python IaC (l9iac)

### Monitoring
- [ ] Monitor GitHub issues for bug reports
- [ ] Monitor Terraform Registry for download stats
- [ ] Set up alerts for critical issues

## Known Limitations

1. **macOS Test Workaround**: Tests require `CGO_ENABLED=0` on macOS due to dyld LC_UUID issue
2. **Delete Operations**: Require separate delete token/refresh token with delete scope
3. **Rate Limiting**: No built-in rate limiting; API enforces limits

## Rollback Plan

If critical issues are found:

1. Create new patch release (v1.0.1) with fix
2. If severe, mark GitHub release as pre-release
3. Update Terraform Registry to deprecate version (if needed)
4. Communicate via GitHub issues and internal channels

## Success Criteria

- [ ] Provider available on Terraform Registry
- [ ] All core resources functional (9 resources, 3 data sources)
- [ ] Documentation complete and accessible
- [ ] Zero critical bugs after 1-week monitoring period
- [ ] Positive feedback from initial users

---

**Linear Ticket**: [ENG-364](https://linear.app/last9/issue/ENG-364/release-v10-of-last9-terraform-provider)
**Target Release Date**: January 2026
**Owner**: @prathamesh @aditya
