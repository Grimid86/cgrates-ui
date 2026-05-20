# Phase 1.1: MFA / TOTP Specification

## Status
**In Progress** — started 2026-05-20

## Context
Current MFA implementation in `admin-gateway` is a stub:
- `SetupMFA` returns a hardcoded secret (`JBSWY3DPEHPK3PXP`)
- `VerifyMFA` always returns `{"verified": true}` without checking the code
- `Login` sets `mfaVerified = true` unconditionally when `mfa_code` is non-empty
- Backup codes are generated but never validated

This is a **critical security vulnerability** that must be fixed before production.

## Scope
This document covers the complete TOTP (RFC 6238) implementation for staff portals (Admin, Operator). SelfCare subscriber PIN is separate and out of scope.

## Goals
1. Generate cryptographically secure TOTP secrets using `crypto/rand`
2. Validate TOTP codes during login and MFA setup
3. Support backup codes (one-time recovery codes)
4. Provide QR code URL for authenticator apps (Google Authenticator, Authy, etc.)
5. Encrypt MFA secrets at rest (AES-256-GCM or similar)

## Non-Goals
- SMS-based MFA (out of scope for Phase 1)
- Hardware security keys / WebAuthn (future phase)
- Push notifications

## Architecture

### Libraries
| Library | Version | Purpose |
|---------|---------|---------|
| `github.com/pquerna/otp/totp` | v1.5.0 | TOTP generation & validation per RFC 6238 |
| `golang.org/x/crypto/bcrypt` | existing | Backup code hashing |

### Data Model Changes

#### `users` table (already exists)
- `mfa_secret` VARCHAR(255) — TOTP secret (encrypted at rest)
- `mfa_enabled` BOOLEAN — flag
- `mfa_backup_codes` JSONB — array of bcrypt-hashed backup codes

#### `models.User` struct changes
```go
type User struct {
    // ... existing fields ...
    MFASecret       string     `json:"-" db:"mfa_secret"`
    MFABackupCodes  []string   `json:"-" db:"mfa_backup_codes"`
}
```

### Flow: Setup MFA
```
1. Authenticated user calls POST /auth/mfa/setup
2. Backend generates random TOTP secret (totp.Generate)
3. Backend generates 8 random backup codes + hashes them with bcrypt
4. Backend stores encrypted secret + hashed backup codes in DB
5. Backend returns:
   {
     "secret": "base32-secret",        // raw secret for manual entry
     "qr_code_url": "otpauth://...",   // URL for QR generation
     "backup_codes": ["xxxx-xxxx", ...] // plaintext (shown once)
   }
6. Frontend displays QR code + backup codes
7. User MUST verify a TOTP code to activate MFA (see Flow: Verify MFA)
```

### Flow: Verify MFA (Activation)
```
1. User scans QR code and enters first TOTP code
2. Frontend sends POST /auth/mfa/verify { "code": "123456" }
3. Backend validates code against stored secret
4. If valid: set mfa_enabled = true, return { "verified": true }
5. If invalid: return 400, do NOT enable MFA
```

### Flow: Login with MFA
```
1. User sends POST /auth/login { "email": "...", "password": "...", "mfa_code": "123456" }
2. Backend authenticates email+password
3. If user.mfa_enabled == true:
   a. If mfa_code is empty → 403 "mfa code required"
   b. Validate mfa_code against TOTP secret
   c. If TOTP invalid → try backup codes (bcrypt compare against JSONB array)
   d. If backup code matches → remove it from array (one-time use) + allow login
   e. If nothing matches → 401 "invalid mfa code"
4. Create portal session with mfa_verified = true in JWT claims
```

### Flow: Disable MFA
```
1. Authenticated user calls POST /auth/mfa/disable
2. Backend clears mfa_secret, mfa_enabled = false, mfa_backup_codes = []
3. Return 204
```

## API Changes

### POST /auth/mfa/setup
**Response 200:**
```json
{
  "secret": "JBSWY3DPEHPK3PXP",
  "qr_code_url": "otpauth://totp/UI-Bill:user@example.com?secret=JBSWY3DPEHPK3PXP&issuer=UI-Bill",
  "backup_codes": ["a1b2-c3d4", "e5f6-g7h8", "i9j0-k1l2", "m3n4-o5p6", "q7r8-s9t0", "u1v2-w3x4", "y5z6-a7b8", "c9d0-e1f2"]
}
```

### POST /auth/mfa/verify
**Request:**
```json
{ "code": "123456" }
```
**Response 200:**
```json
{ "verified": true }
```
**Response 400:**
```json
{ "message": "invalid mfa code" }
```

### POST /auth/login (updated)
**Request:**
```json
{
  "email": "admin@test.com",
  "password": "TestPass123!",
  "mfa_code": "123456"
}
```
**Response 403 (MFA required but not provided):**
```json
{ "message": "mfa code required" }
```

## Implementation Checklist

### Backend
- [x] Add `github.com/pquerna/otp` to go.mod
- [x] Update `models.User` with `MFASecret` and `MFABackupCodes` fields
- [x] Update `StaffAuth.Authenticate` query to select `mfa_secret` and `mfa_backup_codes`
- [x] Implement `SetupMFA` — real TOTP generation + backup codes
- [x] Implement `VerifyMFA` — real TOTP validation
- [x] Implement `Login` MFA validation (TOTP + backup codes)
- [x] Implement `DisableMFA` — clear all MFA data
- [x] Add helper: `pkg/security/EncryptAES` and `DecryptAES` (AES-256-GCM)
- [x] Fix `audit_log` CHECK constraint to include `'system'` portal type (was blocking DB updates when audit trigger fired without context)
- [x] Add JWT middleware to `/auth/mfa/verify` endpoint (was missing, causing 401)

### Frontend (Admin UI)
- [ ] Display QR code from `qr_code_url` (using `qrcode.react` or similar)
- [ ] Show backup codes with "copy to clipboard" and warning "Save these codes"
- [ ] MFA verification step during setup (enter code to confirm)
- [ ] Login form: handle 403 "mfa code required" by showing MFA input field
- [ ] Handle backup code input (fallback when TOTP unavailable)

### Testing
- [ ] Unit tests for TOTP generation/validation
- [ ] Integration test: setup → verify → login → disable
- [ ] Test backup code consumption (one-time use)

## Security Considerations
1. **Encryption at rest**: `mfa_secret` must be encrypted before storing in DB. Use AES-256-GCM with a key from `SECURITY_CSRFSecret` or a dedicated `MFA_ENCRYPTION_KEY` env var.
2. **Backup codes**: Must be shown only once during setup. Store only bcrypt hashes.
3. **Rate limiting**: MFA verification endpoints must be rate-limited (e.g., 5 attempts per minute) to prevent brute force.
4. **Time skew**: TOTP validation should allow ±1 time step (30-second window) to accommodate clock drift.

## Decisions
- **Library choice**: `pquerna/otp` is the de-facto standard for Go TOTP (RFC 6238 compliant, actively maintained).
- **No QR image generation on backend**: Backend returns `otpauth://` URL; frontend generates QR image using JS library. This avoids adding image generation dependencies to Go backend.
- **Backup code format**: 8 codes, each 8 chars alphanumeric with dash (e.g., `a1b2-c3d4`), generated via `crypto/rand`.

## Related Files
- `backend/admin-gateway/handlers/handlers.go` — MFA handlers
- `backend/operator-gateway/handlers/handlers.go` — Login MFA check (if Operator enables MFA later)
- `backend/pkg/models/models.go` — User struct
- `backend/pkg/auth/staff.go` — Authentication logic
- `backend/pkg/security/security.go` — Encryption helpers
- `frontend/admin-ui/src/pages/LoginPage.jsx` — Login form
- `frontend/admin-ui/src/pages/ProfilePage.jsx` or new `MFASetupPage.jsx`
